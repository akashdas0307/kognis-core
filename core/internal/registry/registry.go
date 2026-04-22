package registry

import (
	"errors"
	"fmt"
	"sync"
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

var (
	ErrCapabilityConflict = errors.New("CAPABILITY_CONFLICT")
	ErrInvalidStateTransition = errors.New("INVALID_STATE_TRANSITION")
)

// PluginEntry holds a registered plugin's metadata and state.
type PluginEntry struct {
	ID                string
	Name              string
	Version           string
	State             PluginState
	Capabilities      []string
	SlotRegistrations []SlotRegistration
	PID               int
	EventBusToken     string
	LatencyClass      string
	LLMExposedTo      []string
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
	capabilities map[string][]string // capability_id -> plugin_ids
}

// New creates a new empty plugin registry.
func New() *Registry {
	return &Registry{
		plugins:      make(map[string]*PluginEntry),
		capabilities: make(map[string][]string),
	}
}

// Register adds a plugin to the registry.
func (r *Registry) Register(entry *PluginEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[entry.ID]; exists {
		return fmt.Errorf("plugin %s already registered", entry.ID)
	}

	// Check for capability conflicts
	for _, capID := range entry.Capabilities {
		if providers, exists := r.capabilities[capID]; exists && len(providers) > 0 {
			return fmt.Errorf("%w: capability %s already registered by %v", ErrCapabilityConflict, capID, providers)
		}
	}

	// Register plugin
	r.plugins[entry.ID] = entry

	// Register capabilities
	for _, capID := range entry.Capabilities {
		r.capabilities[capID] = append(r.capabilities[capID], entry.ID)
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
	return nil
}

// Remove removes a plugin from the registry.
func (r *Registry) Remove(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Clean up capabilities
	if p, ok := r.plugins[id]; ok {
		for _, capID := range p.Capabilities {
			providers := r.capabilities[capID]
			newProviders := make([]string, 0, len(providers))
			for _, pid := range providers {
				if pid != id {
					newProviders = append(newProviders, pid)
				}
			}
			if len(newProviders) == 0 {
				delete(r.capabilities, capID)
			} else {
				r.capabilities[capID] = newProviders
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

	providerIDs := r.capabilities[capID]
	result := make([]*PluginEntry, 0, len(providerIDs))
	for _, id := range providerIDs {
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
