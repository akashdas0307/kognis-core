package health

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/akashdas0307/kognis-core/core/internal/eventbus"
	"github.com/akashdas0307/kognis-core/core/internal/registry"
)

// DefaultMaxHistory is the default number of historical pulses retained per plugin.
const DefaultMaxHistory = 100

// Pulse represents a health pulse from a plugin per SPEC 18 Section 18.1.
type Pulse struct {
	PluginID        string                 `json:"plugin_id"`
	Timestamp       time.Time              `json:"timestamp"`
	Status          string                 `json:"status"`            // HEALTHY|DEGRADED|ERROR|CRITICAL|UNRESPONSIVE
	Metrics         map[string]interface{} `json:"metrics"`           // queue_depth, latency_p50/p99, error_count, memory
	CurrentActivity string                 `json:"current_activity"`
	LastDispatchAt  *time.Time             `json:"last_dispatch_at,omitempty"`
	Alerts          []Alert                `json:"alerts,omitempty"`
}

// Alert represents an alert embedded in a health pulse per SPEC 18.
type Alert struct {
	Severity string `json:"severity"` // warning|error|critical
	Code     string `json:"code"`
	Message  string `json:"message"`
}

// DerivedMetrics holds computed metrics derived from pulse history per SPEC 18 Section 18.2.
type DerivedMetrics struct {
	PluginID      string
	UptimePercent float64
	ErrorRate     float64 // errors per minute
	AvgLatencyP50 float64
	AvgLatencyP99 float64
}

// Aggregator collects and processes health pulses from all plugins per SPEC 18.
type Aggregator struct {
	mu         sync.RWMutex
	registry   *registry.Registry
	bus        *eventbus.Bus
	pulses     map[string]*Pulse  // latest pulse per plugin
	history    map[string][]Pulse // pulse history per plugin
	maxHistory int                // max history entries per plugin
}

// NewAggregator creates a new health pulse aggregator per SPEC 18.
func NewAggregator(reg *registry.Registry, bus *eventbus.Bus) *Aggregator {
	a := &Aggregator{
		registry:   reg,
		bus:        bus,
		pulses:     make(map[string]*Pulse),
		history:    make(map[string][]Pulse),
		maxHistory: DefaultMaxHistory,
	}

	// Subscribe to health pulse events from all plugins: kognis.health.>
	if _, err := bus.Subscribe(eventbus.HealthSubject(">"), func(msg *nats.Msg) {
		var pulse Pulse
		if err := json.Unmarshal(msg.Data, &pulse); err != nil {
			log.Printf("health: invalid pulse: %v", err)
			return
		}
		// Skip messages that are not actual health pulses (e.g., supervisor notifications)
		if pulse.Status == "" {
			return
		}
		a.RecordPulse(&pulse)
	}); err != nil {
		log.Printf("health: failed to subscribe to health events: %v", err)
	}

	return a
}

// RecordPulse stores a health pulse, updates registry state, and maintains history
// per SPEC 18 Section 18.2.
func (a *Aggregator) RecordPulse(pulse *Pulse) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Set timestamp if not provided
	if pulse.Timestamp.IsZero() {
		pulse.Timestamp = time.Now()
	}

	pluginID := pulse.PluginID

	// Get previous pulse status for state change detection
	var prevStatus string
	if prev, ok := a.pulses[pluginID]; ok {
		prevStatus = prev.Status
	}

	// Store as latest pulse
	a.pulses[pluginID] = pulse

	// Append to history
	a.history[pluginID] = append(a.history[pluginID], *pulse)

	// Auto-prune if history exceeds maxHistory
	if len(a.history[pluginID]) > a.maxHistory {
		a.history[pluginID] = a.history[pluginID][len(a.history[pluginID])-a.maxHistory:]
	}

	// Map pulse status to registry state and update
	newState := mapStatusToState(pulse.Status)
	if newState != "" {
		prevState := mapStatusToState(prevStatus)
		if err := a.registry.UpdateState(pluginID, newState); err != nil {
			log.Printf("health: failed to update state for %s: %v", pluginID, err)
		} else if prevStatus != "" && prevStatus != pulse.Status {
			// Publish state change when the pulse status actually changes.
			// PublishState internally skips when old and new state values are equal.
			a.bus.PublishState(pluginID, "health_status", prevState, newState) //nolint:errcheck // state change publish is best-effort
		}
	}
}

// mapStatusToState maps a health pulse status to a registry PluginState per SPEC 18.
func mapStatusToState(status string) registry.PluginState {
	switch status {
	case "HEALTHY":
		return registry.StateHealthyActive
	case "DEGRADED":
		return registry.StateUnhealthy
	case "ERROR":
		return registry.StateUnhealthy
	case "CRITICAL":
		return registry.StateUnresponsive
	case "UNRESPONSIVE":
		return registry.StateUnresponsive
	default:
		return ""
	}
}

// GetPulse returns the latest pulse for a plugin.
func (a *Aggregator) GetPulse(pluginID string) (*Pulse, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	p, ok := a.pulses[pluginID]
	return p, ok
}

// GetHistory returns the pulse history for a plugin per SPEC 18 Section 18.2.
func (a *Aggregator) GetHistory(pluginID string) []Pulse {
	a.mu.RLock()
	defer a.mu.RUnlock()

	hist, ok := a.history[pluginID]
	if !ok {
		return nil
	}
	result := make([]Pulse, len(hist))
	copy(result, hist)
	return result
}

// GetDerivedMetrics returns computed metrics derived from pulse history per SPEC 18 Section 18.2.
func (a *Aggregator) GetDerivedMetrics(pluginID string) *DerivedMetrics {
	a.mu.RLock()
	defer a.mu.RUnlock()

	hist, ok := a.history[pluginID]
	if !ok || len(hist) == 0 {
		return nil
	}

	dm := &DerivedMetrics{PluginID: pluginID}

	healthyCount := 0
	errorCount := 0
	var totalP50, totalP99 float64
	p50Count, p99Count := 0, 0

	for _, p := range hist {
		switch p.Status {
		case "HEALTHY":
			healthyCount++
		case "ERROR", "CRITICAL":
			errorCount++
		}

		if p.Metrics != nil {
			if v, ok := toFloat(p.Metrics["latency_p50"]); ok {
				totalP50 += v
				p50Count++
			}
			if v, ok := toFloat(p.Metrics["latency_p99"]); ok {
				totalP99 += v
				p99Count++
			}
		}
	}

	total := len(hist)
	dm.UptimePercent = float64(healthyCount) / float64(total) * 100.0

	// Error rate: errors per minute over the history window
	if total > 1 {
		durationMin := hist[total-1].Timestamp.Sub(hist[0].Timestamp).Minutes()
		if durationMin > 0 {
			dm.ErrorRate = float64(errorCount) / durationMin
		}
	}

	if p50Count > 0 {
		dm.AvgLatencyP50 = totalP50 / float64(p50Count)
	}
	if p99Count > 0 {
		dm.AvgLatencyP99 = totalP99 / float64(p99Count)
	}

	return dm
}

// GetAlerts returns the current alerts for a plugin from its latest pulse.
func (a *Aggregator) GetAlerts(pluginID string) []Alert {
	a.mu.RLock()
	defer a.mu.RUnlock()

	p, ok := a.pulses[pluginID]
	if !ok || len(p.Alerts) == 0 {
		return nil
	}
	result := make([]Alert, len(p.Alerts))
	copy(result, p.Alerts)
	return result
}

// SystemHealthStatus returns the health status of all known plugins.
func (a *Aggregator) SystemHealthStatus() map[string]string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make(map[string]string, len(a.pulses))
	for id, p := range a.pulses {
		result[id] = p.Status
	}
	return result
}

// PruneHistory trims history to maxHistory entries per plugin.
func (a *Aggregator) PruneHistory() {
	a.mu.Lock()
	defer a.mu.Unlock()

	for pluginID, hist := range a.history {
		if len(hist) > a.maxHistory {
			a.history[pluginID] = hist[len(hist)-a.maxHistory:]
		}
	}
}

// SetMaxHistory sets the maximum number of history entries per plugin.
func (a *Aggregator) SetMaxHistory(n int) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if n > 0 {
		a.maxHistory = n
	}
}

// AllPulses returns all recorded latest pulses.
func (a *Aggregator) AllPulses() map[string]*Pulse {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make(map[string]*Pulse, len(a.pulses))
	for k, v := range a.pulses {
		result[k] = v
	}
	return result
}

// toFloat extracts a float64 from an interface{} value.
// Handles float64, float32, int, int32, int64.
func toFloat(v interface{}) (float64, bool) {
	switch f := v.(type) {
	case float64:
		return f, true
	case float32:
		return float64(f), true
	case int:
		return float64(f), true
	case int32:
		return float64(f), true
	case int64:
		return float64(f), true
	default:
		return 0, false
	}
}