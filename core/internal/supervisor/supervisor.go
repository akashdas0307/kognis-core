package supervisor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/akashdas0307/kognis-core/core/internal/config"
	"github.com/akashdas0307/kognis-core/core/internal/eventbus"
	"github.com/akashdas0307/kognis-core/core/internal/registry"
	"github.com/akashdas0307/kognis-core/core/internal/router"
)

// Supervisor manages plugin lifecycle: registration, health monitoring, and restart.
type Supervisor struct {
	mu       sync.Mutex
	registry *registry.Registry
	router   *router.Router
	bus      *eventbus.Bus
	cfg      config.SupervisorConfig

	restartCounts map[string]int
	lastRestart   map[string]time.Time
	cancelFuncs   map[string]context.CancelFunc
}

// New creates a new plugin supervisor.
func New(reg *registry.Registry, rtr *router.Router, bus *eventbus.Bus, cfg config.SupervisorConfig) *Supervisor {
	return &Supervisor{
		registry:      reg,
		router:        rtr,
		bus:           bus,
		cfg:           cfg,
		restartCounts: make(map[string]int),
		lastRestart:   make(map[string]time.Time),
		cancelFuncs:   make(map[string]context.CancelFunc),
	}
}

// Run starts the supervisor loop. It blocks until the context is cancelled.
func (s *Supervisor) Run(ctx context.Context) error {
	log.Printf("supervisor: starting (heartbeat=%ds, timeout=%ds, grace=%ds, max_restarts=%d)",
		s.cfg.HeartbeatIntervalSec, s.cfg.RegistrationTimeoutSec,
		s.cfg.ShutdownGracePeriodSec, s.cfg.MaxRestartAttempts)

	// Subscribe to registration events
	sub, err := s.bus.Subscribe("kognis.plugin.register", func(msg *nats.Msg) {
		s.handleRegistration(msg)
	})
	if err != nil {
		return fmt.Errorf("subscribe to registration events: %w", err)
	}
	defer sub.Unsubscribe()

	// Subscribe to health pulse events
	healthSub, err := s.bus.Subscribe("kognis.plugin.health", func(msg *nats.Msg) {
		s.handleHealthPulse(msg)
	})
	if err != nil {
		return fmt.Errorf("subscribe to health events: %w", err)
	}
	defer healthSub.Unsubscribe()

	// Start periodic health check ticker
	heartbeatInterval := time.Duration(s.cfg.HeartbeatIntervalSec) * time.Second
	healthTicker := time.NewTicker(heartbeatInterval)
	defer healthTicker.Stop()

	go func() {
		for {
			select {
			case <-healthTicker.C:
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

	s.mu.Lock()
	defer s.mu.Unlock()

	entry.State = registry.StateRegistered
	if err := s.registry.Register(&entry); err != nil {
		log.Printf("supervisor: registration failed for %s: %v", entry.ID, err)
		return
	}

	log.Printf("supervisor: plugin %s (%s v%s) registered", entry.ID, entry.Name, entry.Version)

	// Transition to starting state
	s.registry.UpdateState(entry.ID, registry.StateStarting)

	// Acknowledge registration
	reply := fmt.Sprintf(`{"plugin_id":"%s","state":"REGISTERED"}`, entry.ID)
	if msg.Reply != "" {
		s.bus.Publish(msg.Reply, []byte(reply))
	}
}

// handleHealthPulse processes a health pulse from a plugin.
func (s *Supervisor) handleHealthPulse(msg *nats.Msg) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var pulse struct {
		PluginID string `json:"plugin_id"`
		State    string `json:"state"`
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

	// Update state based on pulse
	switch pulse.State {
	case "HEALTHY":
		s.registry.UpdateState(plugin.ID, registry.StateHealthyActive)
		s.restartCounts[plugin.ID] = 0 // reset restart count on healthy pulse
	case "UNHEALTHY":
		s.registry.UpdateState(plugin.ID, registry.StateUnhealthy)
	case "UNRESPONSIVE":
		s.registry.UpdateState(plugin.ID, registry.StateUnresponsive)
	}
}

// checkPluginHealth scans all plugins for unresponsive ones.
func (s *Supervisor) checkPluginHealth() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, plugin := range s.registry.List() {
		switch plugin.State {
		case registry.StateUnresponsive:
			s.restartPlugin(plugin.ID)
		case registry.StateUnhealthy:
			log.Printf("supervisor: plugin %s is unhealthy, monitoring", plugin.ID)
		}
	}
}

// restartPlugin attempts to restart an unresponsive plugin.
func (s *Supervisor) restartPlugin(pluginID string) {
	count := s.restartCounts[pluginID]
	if count >= s.cfg.MaxRestartAttempts {
		log.Printf("supervisor: plugin %s exceeded max restart attempts (%d), marking DEAD", pluginID, s.cfg.MaxRestartAttempts)
		s.registry.UpdateState(pluginID, registry.StateDead)
		return
	}

	delay := backoffDuration(count)
	s.restartCounts[pluginID] = count + 1
	s.lastRestart[pluginID] = time.Now()

	log.Printf("supervisor: restarting plugin %s (attempt %d, backoff %v)", pluginID, count+1, delay)
	s.registry.UpdateState(pluginID, registry.StateStarting)

	// Publish restart command
	subject := fmt.Sprintf("kognis.plugin.%s.restart", pluginID)
	s.bus.Publish(subject, []byte(fmt.Sprintf(`{"plugin_id":"%s","attempt":%d}`, pluginID, count+1)))
}

// shutdownAll gracefully shuts down all registered plugins.
func (s *Supervisor) shutdownAll() {
	s.mu.Lock()
	defer s.mu.Unlock()

	gracePeriod := time.Duration(s.cfg.ShutdownGracePeriodSec) * time.Second

	for _, plugin := range s.registry.List() {
		s.registry.UpdateState(plugin.ID, registry.StateShuttingDown)
		log.Printf("supervisor: sending shutdown to plugin %s", plugin.ID)

		subject := fmt.Sprintf("kognis.plugin.%s.shutdown", plugin.ID)
		s.bus.Publish(subject, []byte(fmt.Sprintf(`{"plugin_id":"%s","grace_period":"%s"}`, plugin.ID, gracePeriod)))
	}

	// Allow grace period for plugins to shut down
	time.Sleep(gracePeriod)

	// Mark remaining plugins as shut down
	for _, plugin := range s.registry.List() {
		if plugin.State != registry.StateShutDown {
			s.registry.UpdateState(plugin.ID, registry.StateShutDown)
		}
	}
}