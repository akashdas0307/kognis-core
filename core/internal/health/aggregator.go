package health

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/kognis-framework/kognis-core/core/internal/eventbus"
	"github.com/kognis-framework/kognis-core/core/internal/registry"
)

// Pulse represents a health pulse from a plugin.
type Pulse struct {
	PluginID    string    `json:"plugin_id"`
	Timestamp   time.Time `json:"timestamp"`
	State       string    `json:"state"`
	LatencyMS   int       `json:"latency_ms"`
	MemoryMB    int       `json:"memory_mb"`
	CustomData  string    `json:"custom_data,omitempty"`
}

// Aggregator collects and processes health pulses from all plugins.
type Aggregator struct {
	mu       sync.RWMutex
	registry *registry.Registry
	bus      *eventbus.Bus
	pulses   map[string]*Pulse
}

// NewAggregator creates a new health pulse aggregator.
func NewAggregator(reg *registry.Registry, bus *eventbus.Bus) *Aggregator {
	a := &Aggregator{
		registry: reg,
		bus:      bus,
		pulses:   make(map[string]*Pulse),
	}

	// Subscribe to health pulse events
	bus.Subscribe("kognis.plugin.health", func(msg *nats.Msg) {
		var pulse Pulse
		if err := json.Unmarshal(msg.Data, &pulse); err != nil {
			log.Printf("health: invalid pulse: %v", err)
			return
		}
		a.recordPulse(&pulse)
	})

	return a
}

// recordPulse stores a health pulse and updates the registry state.
func (a *Aggregator) recordPulse(pulse *Pulse) {
	a.mu.Lock()
	defer a.mu.Unlock()

	pulse.Timestamp = time.Now()
	a.pulses[pulse.PluginID] = pulse

	// Map pulse state to registry state
	switch pulse.State {
	case "HEALTHY":
		a.registry.UpdateState(pulse.PluginID, registry.StateHealthyActive)
	case "UNHEALTHY":
		a.registry.UpdateState(pulse.PluginID, registry.StateUnhealthy)
	case "UNRESPONSIVE":
		a.registry.UpdateState(pulse.PluginID, registry.StateUnresponsive)
	}
}

// GetPulse returns the latest pulse for a plugin.
func (a *Aggregator) GetPulse(pluginID string) (*Pulse, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	p, ok := a.pulses[pluginID]
	return p, ok
}

// AllPulses returns all recorded pulses.
func (a *Aggregator) AllPulses() map[string]*Pulse {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make(map[string]*Pulse, len(a.pulses))
	for k, v := range a.pulses {
		result[k] = v
	}
	return result
}