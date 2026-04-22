package registry

import (
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

// PluginEntry holds a registered plugin's metadata and state.
type PluginEntry struct {
	ID           string
	Name         string
	Version      string
	State        PluginState
	Capabilities []string
	SlotRegistrations []SlotRegistration
}

// SlotRegistration records a plugin's registration for a pipeline slot.
type SlotRegistration struct {
	Pipeline string
	Slot     string
	Priority int
}

// Registry maintains the live index of all registered plugins.
type Registry struct {
	mu      sync.RWMutex
	plugins map[string]*PluginEntry
}

// New creates a new empty plugin registry.
func New() *Registry {
	return &Registry{
		plugins: make(map[string]*PluginEntry),
	}
}

// Register adds a plugin to the registry.
func (r *Registry) Register(entry *PluginEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[entry.ID]; exists {
		return fmt.Errorf("plugin %s already registered", entry.ID)
	}

	r.plugins[entry.ID] = entry
	return nil
}

// Get retrieves a plugin by ID.
func (r *Registry) Get(id string) (*PluginEntry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.plugins[id]
	return p, ok
}

// UpdateState transitions a plugin's state.
func (r *Registry) UpdateState(id string, state PluginState) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.plugins[id]
	if !ok {
		return fmt.Errorf("plugin %s not found", id)
	}

	p.State = state
	return nil
}

// Remove removes a plugin from the registry.
func (r *Registry) Remove(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

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

	var result []*PluginEntry
	for _, p := range r.plugins {
		for _, c := range p.Capabilities {
			if c == capID {
				result = append(result, p)
				break
			}
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