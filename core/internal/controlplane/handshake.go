package controlplane

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/akashdas0307/kognis-core/core/internal/eventbus"
	"github.com/akashdas0307/kognis-core/core/internal/registry"
)

// HandshakeStep constants per SPEC 04 Section 4.2.
type HandshakeStep int

const (
	StepRegister HandshakeStep = 1 // Plugin sends REGISTER_REQUEST
	StepAck      HandshakeStep = 2 // Core responds with REGISTER_ACK
	StepReady    HandshakeStep = 3 // Plugin sends READY after connecting to event bus
	StepActive   HandshakeStep = 4 // Core marks plugin as HEALTHY_ACTIVE
)

// Timeout constants per SPEC 04 Section 4.2.
const (
	// Step1to2Timeout is the maximum time for core to validate and respond
	// after receiving REGISTER_REQUEST (5 seconds per SPEC 04).
	Step1to2Timeout = 5 * time.Second
	// Step2to3Timeout is the maximum time for plugin to connect to event bus
	// and send READY (10 seconds per SPEC 04).
	Step2to3Timeout = 10 * time.Second
	// Step3to4Timeout is the maximum time for core to mark plugin as
	// HEALTHY_ACTIVE (2 seconds per SPEC 04).
	Step3to4Timeout = 2 * time.Second
)

// HandshakeRequest is the initial message from a plugin during registration
// (Step 1: REGISTER_REQUEST per SPEC 04 Section 4.2).
type HandshakeRequest struct {
	PluginID             string   `json:"plugin_id"`
	Name                 string   `json:"name"`
	Version              string   `json:"version"`
	Capabilities         []string `json:"capabilities"`
	ManifestHash         string   `json:"manifest_hash"`
	EmergencyBypassTypes []string `json:"emergency_bypass_types"`
	PID                  int      `json:"pid"`
	Entrypoint           string   `json:"entrypoint"`
}

// HandshakeResponse is the core daemon's response to a plugin handshake
// (Step 2: REGISTER_ACK per SPEC 04 Section 4.2).
type HandshakeResponse struct {
	PluginID            string            `json:"plugin_id"`
	PluginIDRuntime     string            `json:"plugin_id_runtime"`
	State               string            `json:"state"`
	EventBusURL         string            `json:"event_bus_url"`
	EventBusToken       string            `json:"event_bus_token"`
	ControlPlane        string            `json:"control_plane"`
	ConfigBundle        map[string]string `json:"config_bundle"`
	PeerCapabilities    []string          `json:"peer_capabilities_snapshot"`
	Error               string            `json:"error,omitempty"`
}

// ReadyMessage is sent by the plugin after connecting to the event bus
// (Step 3: READY per SPEC 04 Section 4.2).
type ReadyMessage struct {
	PluginID         string   `json:"plugin_id"`
	SubscribedTopics []string `json:"subscribed_topics"`
	HealthEndpoint   string   `json:"health_endpoint"`
}

// ShutdownRequest is sent by the core to initiate graceful shutdown
// per SPEC 04 Section 4.3.
type ShutdownRequest struct {
	PluginID    string        `json:"plugin_id"`
	GracePeriod time.Duration `json:"grace_period"`
	Reason      string        `json:"reason,omitempty"`
}

// pendingHandshake tracks a handshake in progress for timeout enforcement.
type pendingHandshake struct {
	step      HandshakeStep
	startedAt time.Time
	pluginID  string
}

// HandshakeManager orchestrates the 4-step handshake protocol per SPEC 04 Section 4.2.
type HandshakeManager struct {
	registry   *registry.Registry
	bus        *eventbus.Bus
	socketPath string
	config     handshakeConfig

	mu      sync.RWMutex
	pending map[string]*pendingHandshake // pluginID -> pending handshake state
}

// handshakeConfig holds optional configuration for the HandshakeManager.
type handshakeConfig struct {
	EventBusURL string
}

// NewHandshakeManager creates a new HandshakeManager.
func NewHandshakeManager(reg *registry.Registry, bus *eventbus.Bus, socketPath string) *HandshakeManager {
	return &HandshakeManager{
		registry:   reg,
		bus:        bus,
		socketPath:  socketPath,
		config: handshakeConfig{
			EventBusURL: "nats://127.0.0.1:4222",
		},
		pending: make(map[string]*pendingHandshake),
	}
}

// SetEventBusURL overrides the default event bus URL returned in handshake responses.
func (hm *HandshakeManager) SetEventBusURL(url string) {
	hm.config.EventBusURL = url
}

// generateEventBusToken generates a simple UUID-style token for event bus authentication.
func generateEventBusToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// StartHandshake initiates steps 1->2 of the handshake protocol.
// Step 1: Plugin sends REGISTER_REQUEST {manifest, pid, version}.
// Step 2: Core validates manifest, assigns runtime ID, responds with
// REGISTER_ACK {plugin_id_runtime, event_bus_token, config_bundle, peer_capabilities_snapshot}.
func (hm *HandshakeManager) StartHandshake(req *HandshakeRequest) (*HandshakeResponse, error) {
	// Validate required fields
	if req.PluginID == "" || req.Name == "" || req.Version == "" {
		return nil, fmt.Errorf("handshake requires plugin_id, name, and version")
	}

	// IDEMPOTENCY: If already registered and in Step 2+, check if we can return existing info
	if existing, ok := hm.registry.Get(req.PluginID); ok {
		step, _ := hm.registry.GetHandshakeStep(req.PluginID)
		if step >= int(StepAck) && existing.State != registry.StateShuttingDown && existing.State != registry.StateShutDown {
			// If metadata matches, return the existing registration response
			if existing.Name == req.Name && existing.Version == req.Version && existing.ManifestHash == req.ManifestHash {
				log.Printf("handshake: idempotent register for plugin %s (already at step %d)", req.PluginID, step)
				return &HandshakeResponse{
					PluginID:         req.PluginID,
					PluginIDRuntime:  existing.RuntimeID,
					State:            string(existing.State),
					EventBusURL:      hm.config.EventBusURL,
					EventBusToken:    existing.EventBusToken,
					ControlPlane:     hm.socketPath,
					ConfigBundle:     existing.ConfigBundle,
					PeerCapabilities: hm.registry.FindAvailableCapabilities(),
				}, nil
			}
		}
	}

	// Build the registry entry
	entry := &registry.PluginEntry{
		ID:                  req.PluginID,
		Name:                req.Name,
		Version:             req.Version,
		State:               registry.StateRegistered,
		Capabilities:        req.Capabilities,
		PID:                 req.PID,
		ManifestHash:        req.ManifestHash,
		EmergencyBypassTypes: req.EmergencyBypassTypes,
		HandshakeStep:       int(StepRegister),
		ConfigBundle:        make(map[string]string),
		Entrypoint:           req.Entrypoint,
	}

	// Register the plugin (validates capability conflicts and emergency bypass types)
	if err := hm.registry.Register(entry); err != nil {
		return &HandshakeResponse{
			PluginID: req.PluginID,
			Error:    err.Error(),
		}, err
	}

	// Advance from Step 1 (Register) to Step 2 (Ack)
	if err := hm.registry.AdvanceHandshake(req.PluginID); err != nil {
		return &HandshakeResponse{
			PluginID: req.PluginID,
			Error:    fmt.Sprintf("advance handshake: %v", err),
		}, err
	}

	// Generate runtime ID and event bus token
	runtimeID := fmt.Sprintf("%s-%s", req.PluginID, generateEventBusToken()[:8])
	eventBusToken := generateEventBusToken()

	// Store the runtime ID and event bus token in the registry entry
	if p, ok := hm.registry.Get(req.PluginID); ok {
		p.RuntimeID = runtimeID
		p.EventBusToken = eventBusToken
	}

	// Get peer capabilities snapshot
	peerCapabilities := hm.registry.FindAvailableCapabilities()

	// Track the pending handshake for timeout enforcement
	hm.mu.Lock()
	hm.pending[req.PluginID] = &pendingHandshake{
		step:      StepAck,
		startedAt: time.Now(),
		pluginID:  req.PluginID,
	}
	hm.mu.Unlock()

	log.Printf("handshake: plugin %s registered (step 1->2), runtime_id=%s", req.PluginID, runtimeID)

	// Publish registration lifecycle event
	if hm.bus != nil {
		_ = hm.bus.PublishJSON(eventbus.PluginLifecycleSubject(), map[string]string{
			"plugin_id": req.PluginID,
			"event":     "registered",
			"step":      "2",
		})
	}

	return &HandshakeResponse{
		PluginID:         req.PluginID,
		PluginIDRuntime:  runtimeID,
		State:            "REGISTERED",
		EventBusURL:      hm.config.EventBusURL,
		EventBusToken:    eventBusToken,
		ControlPlane:     hm.socketPath,
		ConfigBundle:     entry.ConfigBundle,
		PeerCapabilities: peerCapabilities,
	}, nil
}

// CompleteHandshake completes steps 3->4 of the handshake protocol.
// Step 3: Plugin connects to NATS, subscribes to topics, sends READY.
// Step 4: Core marks plugin as HEALTHY_ACTIVE, recomputes dispatch,
// broadcasts plugin.joined.
func (hm *HandshakeManager) CompleteHandshake(pluginID string, readyMsg *ReadyMessage) error {
	// Verify plugin exists
	if _, ok := hm.registry.Get(pluginID); !ok {
		return fmt.Errorf("plugin %s not found in registry", pluginID)
	}

	// Verify plugin is at the correct handshake step (must be at Step 2: Ack)
	currentStep, err := hm.registry.GetHandshakeStep(pluginID)
	if err != nil {
		return fmt.Errorf("get handshake step for %s: %w", pluginID, err)
	}

	// IDEMPOTENCY: If already at Step 4 (Active), return success
	if currentStep == int(StepActive) {
		log.Printf("handshake: idempotent ready for plugin %s (already ACTIVE)", pluginID)
		return nil
	}

	if currentStep != int(StepAck) {
		return fmt.Errorf("plugin %s at handshake step %d, expected step %d (Ack) before Ready",
			pluginID, currentStep, StepAck)
	}

	// Validate ReadyMessage fields
	if readyMsg.PluginID != pluginID {
		return fmt.Errorf("ready message plugin_id %s does not match expected %s",
			readyMsg.PluginID, pluginID)
	}

	// Advance from Step 2 (Ack) to Step 3 (Ready) — plugin has sent READY
	if err := hm.registry.AdvanceHandshake(pluginID); err != nil {
		return fmt.Errorf("advance handshake to step 3: %w", err)
	}

	// Store subscribed topics in the registry entry
	if p, ok := hm.registry.Get(pluginID); ok {
		p.SubscribedTopics = readyMsg.SubscribedTopics
	}

	// Record heartbeat since the plugin just communicated
	if err := hm.registry.RecordHeartbeat(pluginID); err != nil {
		log.Printf("handshake: warning: failed to record heartbeat for %s: %v", pluginID, err)
	}

	// Advance from Step 3 (Ready) to Step 4 (Active)
	if err := hm.registry.AdvanceHandshake(pluginID); err != nil {
		return fmt.Errorf("advance handshake to step 4: %w", err)
	}

	// Transition to HEALTHY_ACTIVE via the valid state path:
	// REGISTERED -> STARTING -> HEALTHY_ACTIVE
	if err := hm.registry.UpdateState(pluginID, registry.StateStarting); err != nil {
		return fmt.Errorf("transition %s to STARTING: %w", pluginID, err)
	}
	if err := hm.registry.UpdateState(pluginID, registry.StateHealthyActive); err != nil {
		return fmt.Errorf("transition %s to HEALTHY_ACTIVE: %w", pluginID, err)
	}

	// Remove from pending handshakes
	hm.mu.Lock()
	delete(hm.pending, pluginID)
	hm.mu.Unlock()

	// Broadcast plugin.joined event (Step 4)
	if hm.bus != nil {
		_ = hm.bus.PublishJSON(eventbus.PluginLifecycleSubject(), map[string]interface{}{
			"plugin_id":       pluginID,
			"event":           "joined",
			"step":            "4",
			"state":           "HEALTHY_ACTIVE",
			"topics":          readyMsg.SubscribedTopics,
			"health_endpoint": readyMsg.HealthEndpoint,
		})
	}

	log.Printf("handshake: plugin %s now HEALTHY_ACTIVE (step 3->4)", pluginID)

	return nil
}

// CheckTimeouts checks for handshakes that have exceeded their timeout thresholds
// and transitions stuck plugins to UNRESPONSIVE. This should be called periodically
// by the supervisor.
func (hm *HandshakeManager) CheckTimeouts() {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	now := time.Now()
	var timedOut []string

	for pluginID, pending := range hm.pending {
		var timeout time.Duration
		switch pending.step {
		case StepRegister:
			// Step 1->2: core must validate within 5 seconds
			timeout = Step1to2Timeout
		case StepAck:
			// Step 2->3: plugin must connect to event bus within 10 seconds
			timeout = Step2to3Timeout
		case StepReady:
			// Step 3->4: core must mark active within 2 seconds
			timeout = Step3to4Timeout
		default:
			continue
		}

		if now.Sub(pending.startedAt) > timeout {
			timedOut = append(timedOut, pluginID)
		}
	}

	for _, pluginID := range timedOut {
		log.Printf("handshake: timeout for plugin %s, transitioning to UNRESPONSIVE", pluginID)

		// Transition to UNRESPONSIVE via valid state path.
		// The registry requires: REGISTERED -> STARTING -> UNRESPONSIVE
		if entry, ok := hm.registry.Get(pluginID); ok {
			if entry.State == registry.StateRegistered {
				if err := hm.registry.UpdateState(pluginID, registry.StateStarting); err != nil {
					log.Printf("handshake: error transitioning %s to STARTING: %v", pluginID, err)
				}
			}
		}
		if err := hm.registry.UpdateState(pluginID, registry.StateUnresponsive); err != nil {
			log.Printf("handshake: error transitioning %s to UNRESPONSIVE: %v", pluginID, err)
		}

		// Publish timeout lifecycle event
		if hm.bus != nil {
			_ = hm.bus.PublishJSON(eventbus.PluginLifecycleSubject(), map[string]string{
				"plugin_id": pluginID,
				"event":     "handshake_timeout",
				"state":     "UNRESPONSIVE",
			})
		}

		// Remove from pending
		delete(hm.pending, pluginID)
	}
}

// InitiateShutdown begins the graceful shutdown protocol per SPEC 04 Section 4.3.
// Step 1 (SHUTDOWN_REQUEST): Core sends shutdown request to the plugin with a grace period.
// The plugin is transitioned to SHUTTING_DOWN state.
func (hm *HandshakeManager) InitiateShutdown(pluginID string, gracePeriod time.Duration) error {
	// Verify plugin exists
	entry, ok := hm.registry.Get(pluginID)
	if !ok {
		return fmt.Errorf("plugin %s not found in registry", pluginID)
	}

	// Request shutdown via registry (validates state transition, sets SHUTTING_DOWN)
	if err := hm.registry.RequestShutdown(pluginID); err != nil {
		return fmt.Errorf("request shutdown for %s: %w", pluginID, err)
	}

	log.Printf("handshake: shutdown requested for plugin %s (grace=%s)", pluginID, gracePeriod)

	// Broadcast SHUTDOWN_REQUEST event
	if hm.bus != nil {
		_ = hm.bus.PublishJSON(eventbus.PluginLifecycleSubject(), map[string]interface{}{
			"plugin_id":    pluginID,
			"event":        "shutdown_request",
			"grace_period": gracePeriod.String(),
			"capabilities": entry.Capabilities,
		})
	}

	return nil
}

// ConfirmShutdown completes the graceful shutdown protocol per SPEC 04 Section 4.3.
// This is called after the plugin signals it is ready to shut down.
// Step 3 (CONFIRMED): Core confirms the shutdown.
// Step 4 (plugin.left broadcast): Core broadcasts plugin.left lifecycle event.
func (hm *HandshakeManager) ConfirmShutdown(pluginID string) error {
	// Verify plugin exists and is shutting down
	entry, ok := hm.registry.Get(pluginID)
	if !ok {
		return fmt.Errorf("plugin %s not found in registry", pluginID)
	}

	if entry.State != registry.StateShuttingDown {
		return fmt.Errorf("plugin %s is not in SHUTTING_DOWN state (current: %s)",
			pluginID, entry.State)
	}

	// Step 3: Transition to SHUT_DOWN (CONFIRMED)
	if err := hm.registry.UpdateState(pluginID, registry.StateShutDown); err != nil {
		return fmt.Errorf("transition %s to SHUT_DOWN: %w", pluginID, err)
	}

	// Step 4: Broadcast plugin.left event
	if hm.bus != nil {
		_ = hm.bus.PublishJSON(eventbus.PluginLifecycleSubject(), map[string]interface{}{
			"plugin_id":    pluginID,
			"event":        "left",
			"state":        "SHUT_DOWN",
			"capabilities": entry.Capabilities,
		})
	}

	log.Printf("handshake: plugin %s shutdown confirmed (plugin.left broadcast)", pluginID)

	return nil
}