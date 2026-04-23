package registry

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// PluginState represents the lifecycle state of a plugin.
type PluginState string

const (
	StateUnregistered  PluginState = "UNREGISTERED"
	StateRegistered    PluginState = "REGISTERED"
	StateStarting      PluginState = "STARTING"
	StateHealthyActive PluginState = "HEALTHY_ACTIVE"
	StateUnhealthy     PluginState = "UNHEALTHY"
	StateUnresponsive  PluginState = "UNRESPONSIVE"
	StateCircuitOpen   PluginState = "CIRCUIT_OPEN"
	StateDead          PluginState = "DEAD"
	StateShuttingDown  PluginState = "SHUTTING_DOWN"
	StateShutDown      PluginState = "SHUT_DOWN"
)

// KGN-* error codes per SPEC 07
var (
	// Existing errors (preserved for backward compatibility)
	ErrCapabilityConflict     = errors.New("CAPABILITY_CONFLICT")
	ErrInvalidStateTransition = errors.New("INVALID_STATE_TRANSITION")

	// SPEC 07 — LIFECYCLE errors
	ErrRegistrationTimeout = errors.New("KGN-LIFECYCLE-REGISTRATION_TIMEOUT-ERROR")
	ErrUnresponsive        = errors.New("KGN-LIFECYCLE-UNRESPONSIVE-ERROR")
	ErrStartupFailed       = errors.New("KGN-LIFECYCLE-STARTUP_FAILED-ERROR")
	ErrShutdownTimeout      = errors.New("KGN-LIFECYCLE-SHUTDOWN_TIMEOUT-WARNING")
	ErrMaxRestartsExceeded  = errors.New("KGN-LIFECYCLE-MAX_RESTARTS_EXCEEDED-CRITICAL")

	// SPEC 07 — CAPABILITY errors
	ErrCapabilityNotFound    = errors.New("KGN-CAPABILITY-NOT_FOUND-ERROR")
	ErrCapabilityUnavailable = errors.New("KGN-CAPABILITY-UNAVAILABLE-ERROR")

	// SPEC 07 — PERMISSION errors
	ErrPermissionDenied    = errors.New("KGN-PERMISSION-DENIED-ERROR")
	ErrBypassUnauthorized  = errors.New("KGN-PERMISSION-BYPASS_UNAUTHORIZED-ERROR")
)

// validEmergencyBypassTypes is the fixed registry of authorized bypass types per SPEC 14.
var validEmergencyBypassTypes = map[string]bool{
	"safety_sound_detected": true,
	"health_critical":       true,
	"creator_emergency":     true,
	"physical_hazard":      true,
}

// CapabilityEntry tracks a capability's providers and availability status (SPEC 05).
type CapabilityEntry struct {
	ID          string
	ProviderIDs []string
	Status      string // "available" or "unavailable"
}

// PluginEntry holds a registered plugin's metadata and state.
type PluginEntry struct {
	ID                string
	Name              string
	Version           string
	State             PluginState
	Capabilities      []string
	SlotRegistrations []SlotRegistration
	PID               int
	RuntimeID         string               // unique ID for this specific run (SPEC 04 Step 2)
	EventBusToken     string
	LatencyClass      string
	LLMExposedTo      []string

	// Extended fields per SPEC 04/08 compliance (M-013)
	RegisteredAt         time.Time            // when plugin registered
	LastHeartbeat        time.Time            // last heartbeat timestamp
	MissedHeartbeats     int                  // consecutive missed heartbeats (3 = UNRESPONSIVE per SPEC 04 Section 4.7)
	RestartCount         int                  // how many restart attempts
	LastRestartAt        time.Time            // when last restart happened
	EmergencyBypassTypes []string             // authorized emergency bypass types from manifest (SPEC 14)
	ManifestHash         string               // hash of manifest for integrity checking
	HandshakeStep        int                  // current step in 4-step handshake (1-4, SPEC 04 Section 4.2)
	SubscribedTopics     []string             // topics plugin subscribed to after registration
	ConfigBundle         map[string]string    // config received from core in REGISTER_ACK
	ShutdownRequestedAt  time.Time            // when shutdown was requested
	Entrypoint           string               // command to start the plugin (SPEC 02 runtime.entrypoint)
	Path                 string               // filesystem path to the plugin directory
}

// SlotRegistration records a plugin's registration for a pipeline slot.
type SlotRegistration struct {
	Pipeline string
	Slot     string
	Priority int
}

// Registry maintains the live index of all registered plugins.
type Registry struct {
	mu           sync.RWMutex
	plugins      map[string]*PluginEntry
	capabilities map[string]*CapabilityEntry // capability_id -> CapabilityEntry
}

// New creates a new empty plugin registry.
func New() *Registry {
	return &Registry{
		plugins:      make(map[string]*PluginEntry),
		capabilities: make(map[string]*CapabilityEntry),
	}
}

// Register adds a plugin to the registry.
func (r *Registry) Register(entry *PluginEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[entry.ID]; exists {
		return fmt.Errorf("plugin %s already registered", entry.ID)
	}

	// Check for capability conflicts — a capability can only have one provider at a time
	for _, capID := range entry.Capabilities {
		if capEntry, exists := r.capabilities[capID]; exists && len(capEntry.ProviderIDs) > 0 {
			return fmt.Errorf("%w: capability %s already registered by %v", ErrCapabilityConflict, capID, capEntry.ProviderIDs)
		}
	}

	// Set registration timestamp
	entry.RegisteredAt = time.Now()

	// Initialize handshake step to 1 (SPEC 04 Section 4.2)
	if entry.HandshakeStep == 0 {
		entry.HandshakeStep = 1
	}

	// Validate emergency bypass types at registration (SPEC 14 Section 14.4)
	for _, bt := range entry.EmergencyBypassTypes {
		if !validEmergencyBypassTypes[bt] {
			return fmt.Errorf("%w: bypass type %s is not a valid emergency bypass type", ErrBypassUnauthorized, bt)
		}
	}

	// Register plugin
	r.plugins[entry.ID] = entry

	// Register capabilities — determine initial status from plugin state
	status := capabilityStatusForState(entry.State)
	for _, capID := range entry.Capabilities {
		if existing, exists := r.capabilities[capID]; exists {
			existing.ProviderIDs = append(existing.ProviderIDs, entry.ID)
			existing.Status = status
		} else {
			r.capabilities[capID] = &CapabilityEntry{
				ID:          capID,
				ProviderIDs: []string{entry.ID},
				Status:      status,
			}
		}
	}

	return nil
}

// Get retrieves a plugin by ID.
func (r *Registry) Get(id string) (*PluginEntry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.plugins[id]
	return p, ok
}

// UpdateState transitions a plugin's state with strict validation.
// It also updates capability status based on the new state per SPEC 08 Section 8.3.
func (r *Registry) UpdateState(id string, newState PluginState) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.plugins[id]
	if !ok {
		return fmt.Errorf("plugin %s not found", id)
	}

	if !IsValidTransition(p.State, newState) {
		return fmt.Errorf("%w: from %s to %s", ErrInvalidStateTransition, p.State, newState)
	}

	p.State = newState

	// Update capability status based on new plugin state (SPEC 08 Section 8.3)
	newCapStatus := capabilityStatusForState(newState)
	for _, capID := range p.Capabilities {
		if capEntry, exists := r.capabilities[capID]; exists {
			capEntry.Status = newCapStatus
		}
	}

	return nil
}

// Remove removes a plugin from the registry.
func (r *Registry) Remove(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Clean up capabilities
	if p, ok := r.plugins[id]; ok {
		for _, capID := range p.Capabilities {
			if capEntry, exists := r.capabilities[capID]; exists {
				newProviders := make([]string, 0, len(capEntry.ProviderIDs))
				for _, pid := range capEntry.ProviderIDs {
					if pid != id {
						newProviders = append(newProviders, pid)
					}
				}
				if len(newProviders) == 0 {
					delete(r.capabilities, capID)
				} else {
					capEntry.ProviderIDs = newProviders
				}
			}
		}
	}

	delete(r.plugins, id)
}

// List returns all registered plugins.
func (r *Registry) List() []*PluginEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*PluginEntry, 0, len(r.plugins))
	for _, p := range r.plugins {
		result = append(result, p)
	}
	return result
}

// FindByCapability returns plugins that provide the given capability.
func (r *Registry) FindByCapability(capID string) []*PluginEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	capEntry, exists := r.capabilities[capID]
	if !exists {
		return nil
	}

	result := make([]*PluginEntry, 0, len(capEntry.ProviderIDs))
	for _, id := range capEntry.ProviderIDs {
		if p, ok := r.plugins[id]; ok {
			result = append(result, p)
		}
	}
	return result
}

// FindByPipelineSlot returns plugins registered for a specific pipeline slot.
func (r *Registry) FindByPipelineSlot(pipeline, slot string) []*PluginEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*PluginEntry
	for _, p := range r.plugins {
		for _, sr := range p.SlotRegistrations {
			if sr.Pipeline == pipeline && sr.Slot == slot {
				result = append(result, p)
				break
			}
		}
	}
	return result
}

// SetCapabilityStatus sets the status of a capability (SPEC 05).
// Status must be "available" or "unavailable".
func (r *Registry) SetCapabilityStatus(capID, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if status != "available" && status != "unavailable" {
		return fmt.Errorf("invalid capability status %q: must be \"available\" or \"unavailable\"", status)
	}

	capEntry, exists := r.capabilities[capID]
	if !exists {
		return fmt.Errorf("%w: capability %s not found", ErrCapabilityNotFound, capID)
	}

	capEntry.Status = status
	return nil
}

// FindAvailableCapabilities returns all capability IDs with status "available".
func (r *Registry) FindAvailableCapabilities() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []string
	for _, capEntry := range r.capabilities {
		if capEntry.Status == "available" {
			result = append(result, capEntry.ID)
		}
	}
	return result
}

// RecordHeartbeat updates LastHeartbeat and resets MissedHeartbeats to 0 (SPEC 04 Section 4.7).
func (r *Registry) RecordHeartbeat(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.plugins[id]
	if !ok {
		return fmt.Errorf("plugin %s not found", id)
	}

	p.LastHeartbeat = time.Now()
	p.MissedHeartbeats = 0
	return nil
}

// IncrementMissedHeartbeats increments MissedHeartbeats and returns the new count
// (SPEC 04 Section 4.7: 3 missed = UNRESPONSIVE).
func (r *Registry) IncrementMissedHeartbeats(id string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.plugins[id]
	if !ok {
		return 0, fmt.Errorf("plugin %s not found", id)
	}

	p.MissedHeartbeats++
	return p.MissedHeartbeats, nil
}

// CheckHeartbeatTimeouts returns IDs of plugins with MissedHeartbeats >= maxMissed
// (default 3 per SPEC 04 Section 4.7).
func (r *Registry) CheckHeartbeatTimeouts(maxMissed int) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []string
	for id, p := range r.plugins {
		if p.MissedHeartbeats >= maxMissed {
			result = append(result, id)
		}
	}
	return result
}

// RecordRestartAttempt increments RestartCount and updates LastRestartAt (SPEC 08 Section 8.4).
// Returns the new restart count.
func (r *Registry) RecordRestartAttempt(id string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.plugins[id]
	if !ok {
		return 0, fmt.Errorf("plugin %s not found", id)
	}

	p.RestartCount++
	p.LastRestartAt = time.Now()
	return p.RestartCount, nil
}

// ShouldCircuitOpen returns true if RestartCount >= maxRestarts (default 5 per SPEC 08 Section 8.4).
func (r *Registry) ShouldCircuitOpen(id string, maxRestarts int) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.plugins[id]
	if !ok {
		return false
	}

	return p.RestartCount >= maxRestarts
}

// BackoffDuration returns the backoff duration for a given restart attempt count
// per SPEC 08 Section 8.4 schedule:
//   Attempt 1: 0 (immediate)
//   Attempt 2: 30s
//   Attempt 3: 2m
//   Attempt 4: 5m
//   Attempt 5: 15m
func BackoffDuration(restartCount int) time.Duration {
	switch restartCount {
	case 1:
		return 0
	case 2:
		return 30 * time.Second
	case 3:
		return 2 * time.Minute
	case 4:
		return 5 * time.Minute
	case 5:
		return 15 * time.Minute
	default:
		// Beyond 5: 1-hour cooldown per SPEC 08
		return 1 * time.Hour
	}
}

// ResetRestartCount resets the restart count after successful recovery.
func (r *Registry) ResetRestartCount(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.plugins[id]
	if !ok {
		return fmt.Errorf("plugin %s not found", id)
	}

	p.RestartCount = 0
	p.LastRestartAt = time.Time{} // zero value
	return nil
}

// ValidateEmergencyBypass checks if a plugin is authorized for a specific bypass type (SPEC 14).
func (r *Registry) ValidateEmergencyBypass(pluginID, bypassType string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.plugins[pluginID]
	if !ok {
		return fmt.Errorf("plugin %s not found", pluginID)
	}

	// Check that bypass type itself is valid
	if !validEmergencyBypassTypes[bypassType] {
		return fmt.Errorf("%w: bypass type %s is not a valid emergency bypass type", ErrBypassUnauthorized, bypassType)
	}

	// Check that plugin is authorized for this bypass type
	for _, bt := range p.EmergencyBypassTypes {
		if bt == bypassType {
			return nil
		}
	}

	return fmt.Errorf("%w: plugin %s is not authorized for bypass type %s", ErrBypassUnauthorized, pluginID, bypassType)
}

// AdvanceHandshake moves to the next handshake step (1->2->3->4) per SPEC 04 Section 4.2.
// Returns an error if already at step 4 (handshake complete).
func (r *Registry) AdvanceHandshake(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.plugins[id]
	if !ok {
		return fmt.Errorf("plugin %s not found", id)
	}

	if p.HandshakeStep >= 4 {
		return fmt.Errorf("handshake already complete for plugin %s (step 4)", id)
	}

	p.HandshakeStep++
	return nil
}

// GetHandshakeStep returns the current handshake step (1-4) per SPEC 04 Section 4.2.
func (r *Registry) GetHandshakeStep(id string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.plugins[id]
	if !ok {
		return 0, fmt.Errorf("plugin %s not found", id)
	}

	return p.HandshakeStep, nil
}

// RequestShutdown sets plugin state to SHUTTING_DOWN and records ShutdownRequestedAt
// per SPEC 04 Section 4.3.
func (r *Registry) RequestShutdown(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.plugins[id]
	if !ok {
		return fmt.Errorf("plugin %s not found", id)
	}

	if !IsValidTransition(p.State, StateShuttingDown) {
		return fmt.Errorf("%w: from %s to %s", ErrInvalidStateTransition, p.State, StateShuttingDown)
	}

	p.State = StateShuttingDown
	p.ShutdownRequestedAt = time.Now()

	// Update capability status to unavailable
	for _, capID := range p.Capabilities {
		if capEntry, exists := r.capabilities[capID]; exists {
			capEntry.Status = "unavailable"
		}
	}

	return nil
}

// capabilityStatusForState maps plugin state to capability status per SPEC 08 Section 8.3.
// HEALTHY_ACTIVE -> "available"
// UNHEALTHY/UNRESPONSIVE/DEAD/CIRCUIT_OPEN -> "unavailable"
func capabilityStatusForState(state PluginState) string {
	switch state {
	case StateHealthyActive:
		return "available"
	case StateUnhealthy, StateUnresponsive, StateDead, StateCircuitOpen:
		return "unavailable"
	default:
		// STARTING, REGISTERED, etc. — not yet available for dispatch
		return "unavailable"
	}
}

// IsValidTransition checks if a state transition is allowed per SPEC 08.
func IsValidTransition(from, to PluginState) bool {
	if from == to {
		return true
	}

	transitions := map[PluginState][]PluginState{
		StateUnregistered: {StateRegistered},
		StateRegistered:   {StateStarting},
		StateStarting:     {StateHealthyActive, StateUnresponsive},
		StateHealthyActive: {StateUnhealthy, StateUnresponsive, StateShuttingDown},
		StateUnhealthy:     {StateHealthyActive, StateUnresponsive},
		StateUnresponsive:  {StateStarting, StateCircuitOpen},
		StateCircuitOpen:   {StateStarting, StateDead},
		StateShuttingDown:  {StateShutDown},
		// StateDead and StateShutDown are terminal for most paths
	}

	allowed, ok := transitions[from]
	if !ok {
		return false
	}

	for _, a := range allowed {
		if a == to {
			return true
		}
	}

	return false
}