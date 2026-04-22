package supervisor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/akashdas0307/kognis-core/core/internal/config"
	"github.com/akashdas0307/kognis-core/core/internal/eventbus"
	"github.com/akashdas0307/kognis-core/core/internal/registry"
	"github.com/akashdas0307/kognis-core/core/internal/router"
)

// missedHeartbeatThreshold is the number of consecutive missed heartbeats
// before a plugin is considered UNRESPONSIVE (SPEC 04 Section 4.7).
const missedHeartbeatThreshold = 3

// Supervisor manages plugin lifecycle: registration, health monitoring, and restart.
type Supervisor struct {
	mu       sync.Mutex
	registry *registry.Registry
	router   *router.Router
	bus      *eventbus.Bus
	cfg      config.SupervisorConfig

	cancelFuncs map[string]context.CancelFunc
}

// New creates a new plugin supervisor.
func New(reg *registry.Registry, rtr *router.Router, bus *eventbus.Bus, cfg config.SupervisorConfig) *Supervisor {
	return &Supervisor{
		registry:    reg,
		router:      rtr,
		bus:         bus,
		cfg:         cfg,
		cancelFuncs: make(map[string]context.CancelFunc),
	}
}

// Run starts the supervisor loop. It blocks until the context is cancelled.
func (s *Supervisor) Run(ctx context.Context) error {
	log.Printf("supervisor: starting (heartbeat=%ds, timeout=%ds, grace=%ds, max_restarts=%d)",
		s.cfg.HeartbeatIntervalSec, s.cfg.RegistrationTimeoutSec,
		s.cfg.ShutdownGracePeriodSec, s.cfg.MaxRestartAttempts)

	// Subscribe to registration events
	regSubject := eventbus.PluginLifecycleSubject()
	sub, err := s.bus.Subscribe(regSubject, func(msg *nats.Msg) {
		s.handleRegistration(msg)
	})
	if err != nil {
		return fmt.Errorf("subscribe to registration events: %w", err)
	}
	defer func() { _ = sub.Unsubscribe() }()

	// Subscribe to health pulse events
	healthSubject := eventbus.HealthSubject(">")
	healthSub, err := s.bus.Subscribe(healthSubject, func(msg *nats.Msg) {
		s.handleHealthPulse(msg)
	})
	if err != nil {
		return fmt.Errorf("subscribe to health events: %w", err)
	}
	defer func() { _ = healthSub.Unsubscribe() }()

	// Start periodic health check ticker
	heartbeatInterval := time.Duration(s.cfg.HeartbeatIntervalSec) * time.Second
	healthTicker := time.NewTicker(heartbeatInterval)
	defer healthTicker.Stop()

	go func() {
		for {
			select {
			case <-healthTicker.C:
				s.checkHeartbeatTimeouts()
				s.checkPluginHealth()
			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()

	log.Println("supervisor: shutting down...")
	s.shutdownAll()
	log.Println("supervisor: shutdown complete")

	return nil
}

// handleRegistration processes a plugin registration message.
func (s *Supervisor) handleRegistration(msg *nats.Msg) {
	var entry registry.PluginEntry
	if err := json.Unmarshal(msg.Data, &entry); err != nil {
		log.Printf("supervisor: invalid registration message: %v", err)
		return
	}

	// Skip messages that don't have required fields (e.g., lifecycle events
	// from HandshakeManager that aren't actual registration requests).
	if entry.ID == "" || entry.Name == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Skip if already registered (e.g., via gRPC handshake path).
	if _, ok := s.registry.Get(entry.ID); ok {
		return
	}

	entry.State = registry.StateRegistered
	if err := s.registry.Register(&entry); err != nil {
		log.Printf("supervisor: registration failed for %s: %v", entry.ID, err)
		return
	}

	log.Printf("supervisor: plugin %s (%s v%s) registered", entry.ID, entry.Name, entry.Version)

	// Transition to starting state
	if err := s.registry.UpdateState(entry.ID, registry.StateStarting); err != nil {
		log.Printf("supervisor: failed to update state for %s: %v", entry.ID, err)
	}

	// Publish lifecycle event
	if err := s.bus.Publish(eventbus.PluginLifecycleSubject(), []byte(
		fmt.Sprintf(`{"event":"registered","plugin_id":"%s","timestamp":"%s"}`, entry.ID, time.Now().UTC().Format(time.RFC3339)),
	)); err != nil {
		log.Printf("supervisor: failed to publish lifecycle event: %v", err)
	}

	// Acknowledge registration
	reply := fmt.Sprintf(`{"plugin_id":"%s","state":"REGISTERED"}`, entry.ID)
	if msg.Reply != "" {
		if err := s.bus.Publish(msg.Reply, []byte(reply)); err != nil {
			log.Printf("supervisor: failed to publish registration ack: %v", err)
		}
	}
}

// handleHealthPulse processes a health pulse from a plugin.
func (s *Supervisor) handleHealthPulse(msg *nats.Msg) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var pulse struct {
		PluginID string `json:"plugin_id"`
		Status   string `json:"status"`
	}
	if err := json.Unmarshal(msg.Data, &pulse); err != nil {
		log.Printf("supervisor: invalid health pulse: %v", err)
		return
	}

	plugin, ok := s.registry.Get(pulse.PluginID)
	if !ok {
		log.Printf("supervisor: health pulse from unknown plugin %s", pulse.PluginID)
		return
	}

	// Update state based on pulse status (SPEC 18)
	switch pulse.Status {
	case "HEALTHY":
		// Record heartbeat and reset missed count (SPEC 04 Section 4.7)
		if err := s.registry.RecordHeartbeat(plugin.ID); err != nil {
			log.Printf("supervisor: failed to record heartbeat for %s: %v", plugin.ID, err)
		}

		// If plugin has any restart history, reset on recovery (SPEC 08 Section 8.4).
		// Check RestartCount > 0 rather than previous state, because the state
		// may have already been transitioned to HEALTHY_ACTIVE before this pulse.
		if plugin.RestartCount > 0 {
			if err := s.registry.ResetRestartCount(plugin.ID); err != nil {
				log.Printf("supervisor: failed to reset restart count for %s: %v", plugin.ID, err)
			}
			log.Printf("supervisor: plugin %s recovered, reset restart count", plugin.ID)

			// Publish recovery event
			if err := s.bus.Publish(eventbus.HealthSubject(plugin.ID), []byte(
				fmt.Sprintf(`{"event":"recovered","plugin_id":"%s","timestamp":"%s"}`, plugin.ID, time.Now().UTC().Format(time.RFC3339)),
			)); err != nil {
				log.Printf("supervisor: failed to publish recovery event for %s: %v", plugin.ID, err)
			}
		}
		if err := s.registry.UpdateState(plugin.ID, registry.StateHealthyActive); err != nil {
			log.Printf("supervisor: failed to update state for %s: %v", plugin.ID, err)
		}

	case "DEGRADED":
		_ = s.registry.UpdateState(plugin.ID, registry.StateUnhealthy) //nolint:errcheck // best-effort state update
	case "ERROR":
		_ = s.registry.UpdateState(plugin.ID, registry.StateUnhealthy) //nolint:errcheck // best-effort state update
	case "CRITICAL":
		_ = s.registry.UpdateState(plugin.ID, registry.StateUnresponsive) //nolint:errcheck // best-effort state update
	case "UNRESPONSIVE":
		_ = s.registry.UpdateState(plugin.ID, registry.StateUnresponsive) //nolint:errcheck // best-effort state update
	}
}

// checkHeartbeatTimeouts checks all plugins for missed heartbeats and
// transitions those exceeding the threshold to UNRESPONSIVE (SPEC 04 Section 4.7).
func (s *Supervisor) checkHeartbeatTimeouts() {
	s.mu.Lock()
	defer s.mu.Unlock()

	timeoutIDs := s.registry.CheckHeartbeatTimeouts(missedHeartbeatThreshold)
	for _, id := range timeoutIDs {
		plugin, ok := s.registry.Get(id)
		if !ok {
			continue
		}
		// Only transition if not already in a terminal or unresponsive state
		if plugin.State == registry.StateHealthyActive || plugin.State == registry.StateUnhealthy {
			log.Printf("supervisor: plugin %s missed %d heartbeats, marking UNRESPONSIVE", id, missedHeartbeatThreshold)
			if err := s.registry.UpdateState(id, registry.StateUnresponsive); err != nil {
				log.Printf("supervisor: failed to update state for %s: %v", id, err)
			}

			// Publish health event
			if err := s.bus.Publish(eventbus.HealthSubject(id), []byte(
				fmt.Sprintf(`{"event":"heartbeat_timeout","plugin_id":"%s","missed":%d,"timestamp":"%s"}`, id, missedHeartbeatThreshold, time.Now().UTC().Format(time.RFC3339)),
			)); err != nil {
				log.Printf("supervisor: failed to publish timeout event for %s: %v", id, err)
			}
		}
	}

	// Increment missed heartbeats for all active/unhealthy plugins
	for _, plugin := range s.registry.List() {
		if plugin.State == registry.StateHealthyActive || plugin.State == registry.StateUnhealthy {
			_, _ = s.registry.IncrementMissedHeartbeats(plugin.ID) //nolint:errcheck // best-effort increment
		}
	}
}

// checkPluginHealth scans all plugins for unresponsive ones and restarts them.
func (s *Supervisor) checkPluginHealth() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, plugin := range s.registry.List() {
		switch plugin.State {
		case registry.StateUnresponsive, registry.StateUnhealthy:
			s.restartPlugin(plugin.ID)
		}
	}
}

// spawnProcess starts a new plugin process from an entrypoint string.
func (s *Supervisor) spawnProcess(entrypoint string) error {
	parts := strings.Fields(entrypoint)
	if len(parts) == 0 {
		return fmt.Errorf("empty entrypoint")
	}

	cmd := exec.Command(parts[0], parts[1:]...)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	log.Printf("supervisor: spawned process with PID %d: %s", cmd.Process.Pid, entrypoint)
	return nil
}

// restartPlugin attempts to restart an unresponsive plugin using the SPEC 08
// backoff schedule and circuit breaker logic.
func (s *Supervisor) restartPlugin(pluginID string) {
	// Check circuit breaker BEFORE incrementing (SPEC 08 Section 8.4).
	// If already at or above max, open the circuit immediately.
	if s.registry.ShouldCircuitOpen(pluginID, s.cfg.MaxRestartAttempts) {
		log.Printf("supervisor: plugin %s exceeded max restart attempts (%d), opening circuit breaker", pluginID, s.cfg.MaxRestartAttempts)
		_ = s.registry.UpdateState(pluginID, registry.StateCircuitOpen) //nolint:errcheck // best-effort state update

		// Publish circuit open event
		plugin, _ := s.registry.Get(pluginID)
		restartCount := 0
		if plugin != nil {
			restartCount = plugin.RestartCount
		}
		if err := s.bus.Publish(eventbus.PluginLifecycleSubject(), []byte(
			fmt.Sprintf(`{"event":"circuit_open","plugin_id":"%s","restart_count":%d,"timestamp":"%s"}`, pluginID, restartCount, time.Now().UTC().Format(time.RFC3339)),
		)); err != nil {
			log.Printf("supervisor: failed to publish circuit open event: %v", err)
		}
		return
	}

	// Record the restart attempt in the registry
	newCount, err := s.registry.RecordRestartAttempt(pluginID)
	if err != nil {
		log.Printf("supervisor: failed to record restart attempt for %s: %v", pluginID, err)
		return
	}

	// Get backoff duration per SPEC 08 schedule
	delay := registry.BackoffDuration(newCount)

	log.Printf("supervisor: restarting plugin %s (attempt %d, backoff %v)", pluginID, newCount, delay)
	_ = s.registry.UpdateState(pluginID, registry.StateStarting) //nolint:errcheck // best-effort state update

	// If the plugin has an entrypoint defined, spawn a new process.
	if plugin, ok := s.registry.Get(pluginID); ok && plugin.Entrypoint != "" {
		go func(entrypoint string, d time.Duration) {
			if d > 0 {
				time.Sleep(d)
			}
			if err := s.spawnProcess(entrypoint); err != nil {
				log.Printf("supervisor: failed to spawn process for %s: %v", pluginID, err)
			}
		}(plugin.Entrypoint, delay)
	}

	// Publish restart command
	subject := fmt.Sprintf("kognis.plugin.%s.restart", pluginID)
	if err := s.bus.Publish(subject, []byte(fmt.Sprintf(`{"plugin_id":"%s","attempt":%d,"backoff":"%s"}`, pluginID, newCount, delay))); err != nil {
		log.Printf("supervisor: failed to publish restart command: %v", err)
	}

	// Publish lifecycle event
	if err := s.bus.Publish(eventbus.PluginLifecycleSubject(), []byte(
		fmt.Sprintf(`{"event":"restarting","plugin_id":"%s","attempt":%d,"backoff":"%s","timestamp":"%s"}`, pluginID, newCount, delay, time.Now().UTC().Format(time.RFC3339)),
	)); err != nil {
		log.Printf("supervisor: failed to publish restarting event: %v", err)
	}
}

// shutdownAll gracefully shuts down all registered plugins.
func (s *Supervisor) shutdownAll() {
	s.mu.Lock()
	defer s.mu.Unlock()

	gracePeriod := time.Duration(s.cfg.ShutdownGracePeriodSec) * time.Second

	for _, plugin := range s.registry.List() {
		_ = s.registry.UpdateState(plugin.ID, registry.StateShuttingDown) //nolint:errcheck // best-effort state update
		log.Printf("supervisor: sending shutdown to plugin %s", plugin.ID)

		subject := fmt.Sprintf("kognis.plugin.%s.shutdown", plugin.ID)
		if err := s.bus.Publish(subject, []byte(fmt.Sprintf(`{"plugin_id":"%s","grace_period":"%s"}`, plugin.ID, gracePeriod))); err != nil {
			log.Printf("supervisor: failed to publish shutdown command for %s: %v", plugin.ID, err)
		}
	}

	// Allow grace period for plugins to shut down
	time.Sleep(gracePeriod)

	// Mark remaining plugins as shut down
	for _, plugin := range s.registry.List() {
		if plugin.State != registry.StateShutDown {
			_ = s.registry.UpdateState(plugin.ID, registry.StateShutDown) //nolint:errcheck // best-effort state update
		}
	}
}