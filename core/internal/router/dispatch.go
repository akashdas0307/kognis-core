package router

import (
	"fmt"
	"sort"
	"time"

	"github.com/akashdas0307/kognis-core/core/internal/registry"
)

// CompiledDispatchTable is the enhanced dispatch table built from loaded
// pipelines and the current registry state. It includes health-based filtering,
// priority ordering, and slot-level fallback policies.
type CompiledDispatchTable struct {
	PipelineID string
	Slots      []CompiledSlot
	BuiltAt    time.Time
	Version    int // incremented on each rebuild
}

// CompiledSlot represents a single slot within a compiled dispatch table,
// including its providers ordered by priority and fallback policies from the
// pipeline spec.
type CompiledSlot struct {
	SlotName    string
	Capability  string
	Providers   []CompiledProvider
	OnEmpty     string // skip|fail|buffer (from pipeline spec)
	OnAllFailed string // skip|fail|retry (from pipeline spec)
}

// CompiledProvider represents a single plugin provider within a compiled slot,
// captured at compile time with its priority and health state.
type CompiledProvider struct {
	PluginID     string
	Priority     int
	LatencyClass string
	State        registry.PluginState // current state at compile time
}

// CompileEnhancedTable builds an enhanced dispatch table for a single pipeline.
// It filters providers by health state (per SPEC 08 Section 8.3: only HEALTHY_ACTIVE
// and STARTING plugins receive dispatches; UNHEALTHY plugins are included but flagged),
// orders them by priority (lower number = higher priority), and carries through
// OnEmpty/OnAllFailed policies from the pipeline spec.
func CompileEnhancedTable(r *Router) *CompiledDispatchTable {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// If no pipelines loaded, return nil
	if len(r.pipelines) == 0 {
		return nil
	}

	// Compile the first pipeline found — for single-pipeline use.
	// Use CompileAllPipelines for multi-pipeline scenarios.
	for _, spec := range r.pipelines {
		return compilePipelineTable(r, spec)
	}
	return nil
}

// CompileAllPipelines compiles enhanced dispatch tables for all loaded pipelines.
func CompileAllPipelines(r *Router) []*CompiledDispatchTable {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.pipelines) == 0 {
		return nil
	}

	tables := make([]*CompiledDispatchTable, 0, len(r.pipelines))
	for _, spec := range r.pipelines {
		dt := compilePipelineTable(r, spec)
		if dt != nil {
			tables = append(tables, dt)
		}
	}
	return tables
}

// compilePipelineTable builds the compiled dispatch table for a single pipeline spec.
func compilePipelineTable(r *Router, spec *PipelineSpec) *CompiledDispatchTable {
	slots := make([]CompiledSlot, 0, len(spec.Slots))

	for _, slotSpec := range spec.Slots {
		plugins := r.registry.FindByPipelineSlot(spec.Name, slotSpec.Name)

		// Build compiled providers list with health-based filtering.
		// Per SPEC 08 Section 8.3:
		//   - HEALTHY_ACTIVE: primary dispatch target
		//   - STARTING: receives dispatches (may be slow)
		//   - UNHEALTHY: included but flagged (degraded expectations)
		//   - UNRESPONSIVE, DEAD, CIRCUIT_OPEN, SHUTTING_DOWN, SHUT_DOWN, REGISTERED, UNREGISTERED: excluded
		providers := make([]CompiledProvider, 0, len(plugins))
		for _, p := range plugins {
			if isProviderEligible(p.State) {
				priority := slotPriority(p, spec.Name, slotSpec.Name)
				providers = append(providers, CompiledProvider{
					PluginID:     p.ID,
					Priority:     priority,
					LatencyClass: p.LatencyClass,
					State:        p.State,
				})
			}
		}

		// Sort providers by priority (lower number = higher priority)
		sort.SliceStable(providers, func(i, j int) bool {
			return providers[i].Priority < providers[j].Priority
		})

		slots = append(slots, CompiledSlot{
			SlotName:    slotSpec.Name,
			Capability:  slotSpec.Capability,
			Providers:   providers,
			OnEmpty:     slotSpec.OnEmpty,
			OnAllFailed: slotSpec.OnAllFailed,
		})
	}

	return &CompiledDispatchTable{
		PipelineID: spec.Name,
		Slots:      slots,
		BuiltAt:    time.Now(),
		Version:    1,
	}
}

// isProviderEligible returns true if a plugin in the given state is eligible
// for dispatch per SPEC 08 Section 8.3.
//
// Eligible states:
//   - HEALTHY_ACTIVE: primary target
//   - STARTING: may receive dispatches
//   - UNHEALTHY: included with degraded expectations (flagged)
//
// Excluded states:
//   - UNRESPONSIVE, DEAD, CIRCUIT_OPEN, SHUTTING_DOWN, SHUT_DOWN,
//     REGISTERED, UNREGISTERED
func isProviderEligible(state registry.PluginState) bool {
	switch state {
	case registry.StateHealthyActive,
		registry.StateStarting,
		registry.StateUnhealthy:
		return true
	default:
		return false
	}
}

// slotPriority returns the priority a plugin has for a specific pipeline/slot
// combination from its SlotRegistrations. Returns 0 if no explicit priority
// is found (0 is treated as lowest priority / fallback).
func slotPriority(p *registry.PluginEntry, pipelineName, slotName string) int {
	for _, sr := range p.SlotRegistrations {
		if sr.Pipeline == pipelineName && sr.Slot == slotName {
			return sr.Priority
		}
	}
	return 0
}

// ResolveSlot resolves a specific slot from a compiled dispatch table by name.
// Returns an error if the slot is not found.
func ResolveSlot(pipelineName, slotName string, dt *CompiledDispatchTable) (*CompiledSlot, error) {
	if dt == nil {
		return nil, fmt.Errorf("dispatch table is nil")
	}

	for i := range dt.Slots {
		if dt.Slots[i].SlotName == slotName {
			return &dt.Slots[i], nil
		}
	}

	return nil, fmt.Errorf("slot %s not found in pipeline %s dispatch table", slotName, pipelineName)
}

// IsSlotAvailable returns true if the slot has at least one HEALTHY_ACTIVE provider.
// This is the primary availability check — a slot with only STARTING or UNHEALTHY
// providers is not considered "available" in the strong sense.
func IsSlotAvailable(slot *CompiledSlot) bool {
	if slot == nil {
		return false
	}

	for _, p := range slot.Providers {
		if p.State == registry.StateHealthyActive {
			return true
		}
	}
	return false
}

// NextProvider returns the next available provider for a slot, skipping any
// plugin IDs in the excludeIDs set (for failover — exclude already-tried plugins).
//
// It prefers HEALTHY_ACTIVE providers first, then STARTING, then UNHEALTHY
// (degraded). Among each health tier, providers are already ordered by priority.
//
// Returns an error if no eligible provider is found.
func NextProvider(slot *CompiledSlot, excludeIDs ...string) (*CompiledProvider, error) {
	if slot == nil {
		return nil, fmt.Errorf("slot is nil")
	}

	exclude := make(map[string]bool, len(excludeIDs))
	for _, id := range excludeIDs {
		exclude[id] = true
	}

	// Try HEALTHY_ACTIVE providers first (in priority order)
	if p := findProviderByState(slot.Providers, registry.StateHealthyActive, exclude); p != nil {
		return p, nil
	}

	// Then try STARTING providers
	if p := findProviderByState(slot.Providers, registry.StateStarting, exclude); p != nil {
		return p, nil
	}

	// Finally try UNHEALTHY providers (degraded expectations, per SPEC 08 Section 8.3)
	if p := findProviderByState(slot.Providers, registry.StateUnhealthy, exclude); p != nil {
		return p, nil
	}

	return nil, fmt.Errorf("no available provider for slot %s (excluded: %v)", slot.SlotName, excludeIDs)
}

// findProviderByState returns the first provider matching the given state that
// is not in the exclude set.
func findProviderByState(providers []CompiledProvider, state registry.PluginState, exclude map[string]bool) *CompiledProvider {
	for i := range providers {
		if providers[i].State == state && !exclude[providers[i].PluginID] {
			return &providers[i]
		}
	}
	return nil
}

// RebuildDispatchTable compiles and caches the dispatch tables for all loaded
// pipelines. If a previous table exists for a pipeline, its version is incremented.
// Returns the list of newly compiled tables.
func (r *Router) RebuildDispatchTable() []*CompiledDispatchTable {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.pipelines) == 0 {
		return nil
	}

	tables := make([]*CompiledDispatchTable, 0, len(r.pipelines))

	for _, spec := range r.pipelines {
		dt := compilePipelineTable(r, spec)

		// Increment version if a previous table exists for this pipeline
		if prev, ok := r.lastDispatchTable[spec.Name]; ok {
			dt.Version = prev.Version + 1
		}

		r.lastDispatchTable[spec.Name] = dt
		tables = append(tables, dt)
	}

	return tables
}

// GetDispatchTable returns the cached compiled dispatch table for a pipeline,
// or nil if none has been built yet.
func (r *Router) GetDispatchTable(pipelineName string) *CompiledDispatchTable {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.lastDispatchTable == nil {
		return nil
	}
	return r.lastDispatchTable[pipelineName]
}