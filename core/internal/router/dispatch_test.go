package router

import (
	"testing"

	"github.com/akashdas0307/kognis-core/core/internal/registry"
)

// helper: register a plugin with a given state and slot registrations
func registerPluginWithState(reg *registry.Registry, id string, state registry.PluginState, slots []registry.SlotRegistration) *registry.PluginEntry {
	p := &registry.PluginEntry{
		ID:                id,
		State:             state,
		SlotRegistrations: slots,
		LatencyClass:      "normal",
	}
	// Register first (always REGISTERED state), then transition to desired state
	reg.Register(p)
	if state != registry.StateRegistered {
		// Walk through the state machine to reach the desired state
		switch state {
		case registry.StateStarting:
			reg.UpdateState(id, registry.StateStarting)
		case registry.StateHealthyActive:
			reg.UpdateState(id, registry.StateStarting)
			reg.UpdateState(id, registry.StateHealthyActive)
		case registry.StateUnhealthy:
			reg.UpdateState(id, registry.StateStarting)
			reg.UpdateState(id, registry.StateHealthyActive)
			reg.UpdateState(id, registry.StateUnhealthy)
		case registry.StateUnresponsive:
			reg.UpdateState(id, registry.StateStarting)
			reg.UpdateState(id, registry.StateHealthyActive)
			reg.UpdateState(id, registry.StateUnresponsive)
		case registry.StateCircuitOpen:
			reg.UpdateState(id, registry.StateStarting)
			reg.UpdateState(id, registry.StateHealthyActive)
			reg.UpdateState(id, registry.StateUnresponsive)
			reg.UpdateState(id, registry.StateCircuitOpen)
		case registry.StateDead:
			reg.UpdateState(id, registry.StateStarting)
			reg.UpdateState(id, registry.StateHealthyActive)
			reg.UpdateState(id, registry.StateUnresponsive)
			reg.UpdateState(id, registry.StateCircuitOpen)
			reg.UpdateState(id, registry.StateDead)
		case registry.StateShuttingDown:
			reg.UpdateState(id, registry.StateStarting)
			reg.UpdateState(id, registry.StateHealthyActive)
			reg.UpdateState(id, registry.StateShuttingDown)
		case registry.StateShutDown:
			reg.UpdateState(id, registry.StateStarting)
			reg.UpdateState(id, registry.StateHealthyActive)
			reg.UpdateState(id, registry.StateShuttingDown)
			reg.UpdateState(id, registry.StateShutDown)
		}
	}
	return p
}

// helper: create a basic valid pipeline spec
func testPipelineSpec(name string) *PipelineSpec {
	return &PipelineSpec{
		Name:            name,
		PipelineVersion: 1,
		Slots: []SlotSpec{
			{
				Name:           "entry",
				Capability:     "PERCEPTION",
				Required:       true,
				ValidEntryPoint: true,
				TimeoutSeconds: 5,
				OnEmpty:        "fail",
				OnAllFailed:    "retry",
			},
			{
				Name:           "process",
				Capability:     "COGNITION",
				Required:       false,
				ValidEntryPoint: false,
				OnEmpty:        "skip",
				OnAllFailed:    "skip",
			},
		},
	}
}

// --- CompileEnhancedTable tests ---

func TestCompileEnhancedTableWithHealthyPlugins(t *testing.T) {
	reg := registry.New()
	r := New(reg, nil)

	registerPluginWithState(reg, "plugin-1", registry.StateHealthyActive, []registry.SlotRegistration{
		{Pipeline: "test-pipeline", Slot: "entry", Priority: 1},
	})
	registerPluginWithState(reg, "plugin-2", registry.StateHealthyActive, []registry.SlotRegistration{
		{Pipeline: "test-pipeline", Slot: "entry", Priority: 2},
		{Pipeline: "test-pipeline", Slot: "process", Priority: 1},
	})

	r.LoadPipeline(testPipelineSpec("test-pipeline"))

	dt := CompileEnhancedTable(r)
	if dt == nil {
		t.Fatal("expected non-nil dispatch table")
	}
	if dt.PipelineID != "test-pipeline" {
		t.Fatalf("expected PipelineID test-pipeline, got %s", dt.PipelineID)
	}
	if len(dt.Slots) != 2 {
		t.Fatalf("expected 2 slots, got %d", len(dt.Slots))
	}
	if dt.Version != 1 {
		t.Fatalf("expected Version 1, got %d", dt.Version)
	}
	if dt.BuiltAt.IsZero() {
		t.Fatal("expected BuiltAt to be set")
	}
}

func TestCompileEnhancedTableNoPipelines(t *testing.T) {
	reg := registry.New()
	r := New(reg, nil)

	dt := CompileEnhancedTable(r)
	if dt != nil {
		t.Fatal("expected nil dispatch table with no pipelines")
	}
}

func TestCompileFiltersOutNonActivePlugins(t *testing.T) {
	reg := registry.New()
	r := New(reg, nil)

	// Register plugins in various states
	registerPluginWithState(reg, "healthy-plugin", registry.StateHealthyActive, []registry.SlotRegistration{
		{Pipeline: "filter-pipeline", Slot: "entry", Priority: 1},
	})
	registerPluginWithState(reg, "starting-plugin", registry.StateStarting, []registry.SlotRegistration{
		{Pipeline: "filter-pipeline", Slot: "entry", Priority: 2},
	})
	registerPluginWithState(reg, "unhealthy-plugin", registry.StateUnhealthy, []registry.SlotRegistration{
		{Pipeline: "filter-pipeline", Slot: "entry", Priority: 3},
	})
	// These should be filtered out:
	registerPluginWithState(reg, "unresponsive-plugin", registry.StateUnresponsive, []registry.SlotRegistration{
		{Pipeline: "filter-pipeline", Slot: "entry", Priority: 4},
	})
	registerPluginWithState(reg, "dead-plugin", registry.StateDead, []registry.SlotRegistration{
		{Pipeline: "filter-pipeline", Slot: "entry", Priority: 5},
	})
	registerPluginWithState(reg, "circuit-open-plugin", registry.StateCircuitOpen, []registry.SlotRegistration{
		{Pipeline: "filter-pipeline", Slot: "entry", Priority: 6},
	})
	registerPluginWithState(reg, "shutting-down-plugin", registry.StateShuttingDown, []registry.SlotRegistration{
		{Pipeline: "filter-pipeline", Slot: "entry", Priority: 7},
	})
	registerPluginWithState(reg, "shut-down-plugin", registry.StateShutDown, []registry.SlotRegistration{
		{Pipeline: "filter-pipeline", Slot: "entry", Priority: 8},
	})

	spec := &PipelineSpec{
		Name:            "filter-pipeline",
		PipelineVersion: 1,
		Slots: []SlotSpec{
			{Name: "entry", Required: true, ValidEntryPoint: true, TimeoutSeconds: 5},
		},
	}
	r.LoadPipeline(spec)

	dt := CompileEnhancedTable(r)
	if dt == nil {
		t.Fatal("expected non-nil dispatch table")
	}

	slot, err := ResolveSlot("filter-pipeline", "entry", dt)
	if err != nil {
		t.Fatalf("ResolveSlot failed: %v", err)
	}

	// Only HEALTHY_ACTIVE, STARTING, and UNHEALTHY should be included (3 providers)
	if len(slot.Providers) != 3 {
		t.Fatalf("expected 3 eligible providers (HEALTHY_ACTIVE + STARTING + UNHEALTHY), got %d", len(slot.Providers))
	}

	// Verify the specific plugin IDs
	expectedIDs := map[string]bool{
		"healthy-plugin":   true,
		"starting-plugin":  true,
		"unhealthy-plugin": true,
	}
	for _, p := range slot.Providers {
		if !expectedIDs[p.PluginID] {
			t.Errorf("unexpected provider %s in dispatch table", p.PluginID)
		}
	}

	// Verify excluded states are not present
	excludedIDs := map[string]bool{
		"unresponsive-plugin":  true,
		"dead-plugin":         true,
		"circuit-open-plugin": true,
		"shutting-down-plugin": true,
		"shut-down-plugin":    true,
	}
	for _, p := range slot.Providers {
		if excludedIDs[p.PluginID] {
			t.Errorf("excluded provider %s should not be in dispatch table", p.PluginID)
		}
	}
}

func TestPriorityOrdering(t *testing.T) {
	reg := registry.New()
	r := New(reg, nil)

	// Register plugins with different priorities (lower = higher priority)
	registerPluginWithState(reg, "low-priority", registry.StateHealthyActive, []registry.SlotRegistration{
		{Pipeline: "priority-pipeline", Slot: "entry", Priority: 10},
	})
	registerPluginWithState(reg, "high-priority", registry.StateHealthyActive, []registry.SlotRegistration{
		{Pipeline: "priority-pipeline", Slot: "entry", Priority: 1},
	})
	registerPluginWithState(reg, "mid-priority", registry.StateHealthyActive, []registry.SlotRegistration{
		{Pipeline: "priority-pipeline", Slot: "entry", Priority: 5},
	})

	spec := &PipelineSpec{
		Name:            "priority-pipeline",
		PipelineVersion: 1,
		Slots: []SlotSpec{
			{Name: "entry", Required: true, ValidEntryPoint: true, TimeoutSeconds: 5},
		},
	}
	r.LoadPipeline(spec)

	dt := CompileEnhancedTable(r)
	slot, err := ResolveSlot("priority-pipeline", "entry", dt)
	if err != nil {
		t.Fatalf("ResolveSlot failed: %v", err)
	}

	if len(slot.Providers) != 3 {
		t.Fatalf("expected 3 providers, got %d", len(slot.Providers))
	}

	// Should be ordered: high(1), mid(5), low(10)
	if slot.Providers[0].PluginID != "high-priority" {
		t.Fatalf("expected first provider high-priority (priority 1), got %s (priority %d)", slot.Providers[0].PluginID, slot.Providers[0].Priority)
	}
	if slot.Providers[1].PluginID != "mid-priority" {
		t.Fatalf("expected second provider mid-priority (priority 5), got %s (priority %d)", slot.Providers[1].PluginID, slot.Providers[1].Priority)
	}
	if slot.Providers[2].PluginID != "low-priority" {
		t.Fatalf("expected third provider low-priority (priority 10), got %s (priority %d)", slot.Providers[2].PluginID, slot.Providers[2].Priority)
	}
}

// --- ResolveSlot tests ---

func TestResolveSlotFound(t *testing.T) {
	reg := registry.New()
	r := New(reg, nil)

	registerPluginWithState(reg, "plugin-1", registry.StateHealthyActive, []registry.SlotRegistration{
		{Pipeline: "resolve-pipeline", Slot: "entry", Priority: 1},
	})

	r.LoadPipeline(testPipelineSpec("resolve-pipeline"))

	dt := CompileEnhancedTable(r)
	slot, err := ResolveSlot("resolve-pipeline", "entry", dt)
	if err != nil {
		t.Fatalf("ResolveSlot failed: %v", err)
	}
	if slot.SlotName != "entry" {
		t.Fatalf("expected slot name entry, got %s", slot.SlotName)
	}
	if slot.Capability != "PERCEPTION" {
		t.Fatalf("expected capability PERCEPTION, got %s", slot.Capability)
	}
}

func TestResolveSlotNotFound(t *testing.T) {
	reg := registry.New()
	r := New(reg, nil)

	r.LoadPipeline(testPipelineSpec("resolve-pipeline"))

	dt := CompileEnhancedTable(r)
	_, err := ResolveSlot("resolve-pipeline", "nonexistent", dt)
	if err == nil {
		t.Fatal("expected error for nonexistent slot, got nil")
	}
}

func TestResolveSlotNilTable(t *testing.T) {
	_, err := ResolveSlot("pipeline", "slot", nil)
	if err == nil {
		t.Fatal("expected error for nil dispatch table, got nil")
	}
}

// --- IsSlotAvailable tests ---

func TestIsSlotAvailableWithHealthyProvider(t *testing.T) {
	slot := &CompiledSlot{
		SlotName: "entry",
		Providers: []CompiledProvider{
			{PluginID: "p1", Priority: 1, State: registry.StateHealthyActive},
		},
	}
	if !IsSlotAvailable(slot) {
		t.Fatal("expected slot to be available with HEALTHY_ACTIVE provider")
	}
}

func TestIsSlotAvailableWithOnlyStartingProvider(t *testing.T) {
	slot := &CompiledSlot{
		SlotName: "entry",
		Providers: []CompiledProvider{
			{PluginID: "p1", Priority: 1, State: registry.StateStarting},
		},
	}
	if IsSlotAvailable(slot) {
		t.Fatal("expected slot to NOT be available with only STARTING provider")
	}
}

func TestIsSlotAvailableWithOnlyUnhealthyProvider(t *testing.T) {
	slot := &CompiledSlot{
		SlotName: "entry",
		Providers: []CompiledProvider{
			{PluginID: "p1", Priority: 1, State: registry.StateUnhealthy},
		},
	}
	if IsSlotAvailable(slot) {
		t.Fatal("expected slot to NOT be available with only UNHEALTHY provider")
	}
}

func TestIsSlotAvailableEmpty(t *testing.T) {
	slot := &CompiledSlot{
		SlotName:  "entry",
		Providers: []CompiledProvider{},
	}
	if IsSlotAvailable(slot) {
		t.Fatal("expected slot to NOT be available with no providers")
	}
}

func TestIsSlotAvailableNil(t *testing.T) {
	if IsSlotAvailable(nil) {
		t.Fatal("expected false for nil slot")
	}
}

// --- NextProvider tests ---

func TestNextProviderBasic(t *testing.T) {
	slot := &CompiledSlot{
		SlotName: "entry",
		Providers: []CompiledProvider{
			{PluginID: "p1", Priority: 1, State: registry.StateHealthyActive},
			{PluginID: "p2", Priority: 2, State: registry.StateHealthyActive},
		},
	}

	provider, err := NextProvider(slot)
	if err != nil {
		t.Fatalf("NextProvider failed: %v", err)
	}
	if provider.PluginID != "p1" {
		t.Fatalf("expected first provider p1, got %s", provider.PluginID)
	}
}

func TestNextProviderFailover(t *testing.T) {
	slot := &CompiledSlot{
		SlotName: "entry",
		Providers: []CompiledProvider{
			{PluginID: "p1", Priority: 1, State: registry.StateHealthyActive},
			{PluginID: "p2", Priority: 2, State: registry.StateHealthyActive},
			{PluginID: "p3", Priority: 3, State: registry.StateHealthyActive},
		},
	}

	// First call returns p1
	p1, err := NextProvider(slot)
	if err != nil {
		t.Fatalf("NextProvider failed: %v", err)
	}
	if p1.PluginID != "p1" {
		t.Fatalf("expected p1, got %s", p1.PluginID)
	}

	// Exclude p1 (simulating failover)
	p2, err := NextProvider(slot, "p1")
	if err != nil {
		t.Fatalf("NextProvider with exclude failed: %v", err)
	}
	if p2.PluginID != "p2" {
		t.Fatalf("expected p2 after excluding p1, got %s", p2.PluginID)
	}

	// Exclude p1 and p2
	p3, err := NextProvider(slot, "p1", "p2")
	if err != nil {
		t.Fatalf("NextProvider with excludes failed: %v", err)
	}
	if p3.PluginID != "p3" {
		t.Fatalf("expected p3 after excluding p1,p2, got %s", p3.PluginID)
	}

	// Exclude all
	_, err = NextProvider(slot, "p1", "p2", "p3")
	if err == nil {
		t.Fatal("expected error when all providers excluded, got nil")
	}
}

func TestNextProviderHealthTiers(t *testing.T) {
	slot := &CompiledSlot{
		SlotName: "entry",
		Providers: []CompiledProvider{
			// Priority order already set; NextProvider prefers by health tier
			{PluginID: "unhealthy-p", Priority: 1, State: registry.StateUnhealthy},
			{PluginID: "starting-p", Priority: 2, State: registry.StateStarting},
			{PluginID: "healthy-p", Priority: 3, State: registry.StateHealthyActive},
		},
	}

	// HEALTHY_ACTIVE should be preferred despite higher priority number
	provider, err := NextProvider(slot)
	if err != nil {
		t.Fatalf("NextProvider failed: %v", err)
	}
	if provider.PluginID != "healthy-p" {
		t.Fatalf("expected healthy-p to be preferred (HEALTHY_ACTIVE tier), got %s", provider.PluginID)
	}

	// Exclude healthy-p, should fall back to STARTING
	provider, err = NextProvider(slot, "healthy-p")
	if err != nil {
		t.Fatalf("NextProvider failed: %v", err)
	}
	if provider.PluginID != "starting-p" {
		t.Fatalf("expected starting-p (STARTING tier), got %s", provider.PluginID)
	}

	// Exclude healthy-p and starting-p, should fall back to UNHEALTHY
	provider, err = NextProvider(slot, "healthy-p", "starting-p")
	if err != nil {
		t.Fatalf("NextProvider failed: %v", err)
	}
	if provider.PluginID != "unhealthy-p" {
		t.Fatalf("expected unhealthy-p (UNHEALTHY tier), got %s", provider.PluginID)
	}
}

func TestNextProviderNilSlot(t *testing.T) {
	_, err := NextProvider(nil)
	if err == nil {
		t.Fatal("expected error for nil slot, got nil")
	}
}

func TestNextProviderNoProviders(t *testing.T) {
	slot := &CompiledSlot{
		SlotName:  "entry",
		Providers: []CompiledProvider{},
	}
	_, err := NextProvider(slot)
	if err == nil {
		t.Fatal("expected error for slot with no providers, got nil")
	}
}

// --- OnEmpty/OnAllFailed values carried through ---

func TestOnEmptyOnAllFailedCarriedThrough(t *testing.T) {
	reg := registry.New()
	r := New(reg, nil)

	spec := &PipelineSpec{
		Name:            "policy-pipeline",
		PipelineVersion: 1,
		Slots: []SlotSpec{
			{
				Name:           "entry",
				Required:       true,
				ValidEntryPoint: true,
				TimeoutSeconds: 5,
				OnEmpty:        "fail",
				OnAllFailed:    "retry",
			},
			{
				Name:           "optional",
				Required:       false,
				ValidEntryPoint: false,
				OnEmpty:        "skip",
				OnAllFailed:    "skip",
			},
			{
				Name:           "buffer-slot",
				Required:       false,
				ValidEntryPoint: false,
				OnEmpty:        "buffer",
				OnAllFailed:    "fail",
			},
		},
	}
	r.LoadPipeline(spec)

	registerPluginWithState(reg, "plugin-1", registry.StateHealthyActive, []registry.SlotRegistration{
		{Pipeline: "policy-pipeline", Slot: "entry", Priority: 1},
	})

	tables := CompileAllPipelines(r)
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}

	dt := tables[0]
	if dt.PipelineID != "policy-pipeline" {
		t.Fatalf("expected PipelineID policy-pipeline, got %s", dt.PipelineID)
	}

	// Check first slot: OnEmpty=fail, OnAllFailed=retry
	entrySlot, err := ResolveSlot("policy-pipeline", "entry", dt)
	if err != nil {
		t.Fatalf("ResolveSlot entry failed: %v", err)
	}
	if entrySlot.OnEmpty != "fail" {
		t.Fatalf("expected OnEmpty=fail, got %s", entrySlot.OnEmpty)
	}
	if entrySlot.OnAllFailed != "retry" {
		t.Fatalf("expected OnAllFailed=retry, got %s", entrySlot.OnAllFailed)
	}

	// Check second slot: OnEmpty=skip, OnAllFailed=skip
	optionalSlot, err := ResolveSlot("policy-pipeline", "optional", dt)
	if err != nil {
		t.Fatalf("ResolveSlot optional failed: %v", err)
	}
	if optionalSlot.OnEmpty != "skip" {
		t.Fatalf("expected OnEmpty=skip, got %s", optionalSlot.OnEmpty)
	}
	if optionalSlot.OnAllFailed != "skip" {
		t.Fatalf("expected OnAllFailed=skip, got %s", optionalSlot.OnAllFailed)
	}

	// Check third slot: OnEmpty=buffer, OnAllFailed=fail
	bufferSlot, err := ResolveSlot("policy-pipeline", "buffer-slot", dt)
	if err != nil {
		t.Fatalf("ResolveSlot buffer-slot failed: %v", err)
	}
	if bufferSlot.OnEmpty != "buffer" {
		t.Fatalf("expected OnEmpty=buffer, got %s", bufferSlot.OnEmpty)
	}
	if bufferSlot.OnAllFailed != "fail" {
		t.Fatalf("expected OnAllFailed=fail, got %s", bufferSlot.OnAllFailed)
	}
}

// --- CompileAllPipelines tests ---

func TestCompileAllPipelinesMultiple(t *testing.T) {
	reg := registry.New()
	r := New(reg, nil)

	registerPluginWithState(reg, "plugin-a", registry.StateHealthyActive, []registry.SlotRegistration{
		{Pipeline: "pipeline-a", Slot: "entry", Priority: 1},
	})
	registerPluginWithState(reg, "plugin-b", registry.StateHealthyActive, []registry.SlotRegistration{
		{Pipeline: "pipeline-b", Slot: "entry", Priority: 1},
	})

	r.LoadPipeline(&PipelineSpec{
		Name:            "pipeline-a",
		PipelineVersion: 1,
		Slots: []SlotSpec{
			{Name: "entry", Required: true, ValidEntryPoint: true, TimeoutSeconds: 5},
		},
	})
	r.LoadPipeline(&PipelineSpec{
		Name:            "pipeline-b",
		PipelineVersion: 1,
		Slots: []SlotSpec{
			{Name: "entry", Required: true, ValidEntryPoint: true, TimeoutSeconds: 5},
		},
	})

	tables := CompileAllPipelines(r)
	if len(tables) != 2 {
		t.Fatalf("expected 2 tables, got %d", len(tables))
	}

	pipelineIDs := map[string]bool{}
	for _, dt := range tables {
		pipelineIDs[dt.PipelineID] = true
	}
	if !pipelineIDs["pipeline-a"] || !pipelineIDs["pipeline-b"] {
		t.Fatalf("expected pipeline-a and pipeline-b, got %v", pipelineIDs)
	}
}

// --- RebuildDispatchTable tests ---

func TestRebuildDispatchTable(t *testing.T) {
	reg := registry.New()
	r := New(reg, nil)

	registerPluginWithState(reg, "plugin-1", registry.StateHealthyActive, []registry.SlotRegistration{
		{Pipeline: "rebuild-pipeline", Slot: "entry", Priority: 1},
	})

	r.LoadPipeline(testPipelineSpec("rebuild-pipeline"))

	// First rebuild
	tables := r.RebuildDispatchTable()
	if len(tables) != 1 {
		t.Fatalf("expected 1 table on first rebuild, got %d", len(tables))
	}
	if tables[0].Version != 1 {
		t.Fatalf("expected version 1 on first rebuild, got %d", tables[0].Version)
	}

	// Verify cached table is accessible
	cached := r.GetDispatchTable("rebuild-pipeline")
	if cached == nil {
		t.Fatal("expected cached dispatch table, got nil")
	}
	if cached.Version != 1 {
		t.Fatalf("expected cached version 1, got %d", cached.Version)
	}

	// Second rebuild should increment version
	tables = r.RebuildDispatchTable()
	if len(tables) != 1 {
		t.Fatalf("expected 1 table on second rebuild, got %d", len(tables))
	}
	if tables[0].Version != 2 {
		t.Fatalf("expected version 2 on second rebuild, got %d", tables[0].Version)
	}

	// Third rebuild
	tables = r.RebuildDispatchTable()
	if tables[0].Version != 3 {
		t.Fatalf("expected version 3 on third rebuild, got %d", tables[0].Version)
	}
}

func TestRebuildDispatchTableNoPipelines(t *testing.T) {
	reg := registry.New()
	r := New(reg, nil)

	tables := r.RebuildDispatchTable()
	if tables != nil {
		t.Fatalf("expected nil with no pipelines, got %v", tables)
	}
}

func TestGetDispatchTableNotBuilt(t *testing.T) {
	reg := registry.New()
	r := New(reg, nil)

	// No pipelines loaded, no table built
	dt := r.GetDispatchTable("nonexistent")
	if dt != nil {
		t.Fatal("expected nil for unbuilt dispatch table")
	}
}

// --- Mixed health states in a slot ---

func TestCompileMixedHealthPlugins(t *testing.T) {
	reg := registry.New()
	r := New(reg, nil)

	// Mix of healthy, starting, unhealthy, and excluded states
	registerPluginWithState(reg, "healthy", registry.StateHealthyActive, []registry.SlotRegistration{
		{Pipeline: "mixed-pipeline", Slot: "entry", Priority: 1},
	})
	registerPluginWithState(reg, "starting", registry.StateStarting, []registry.SlotRegistration{
		{Pipeline: "mixed-pipeline", Slot: "entry", Priority: 2},
	})
	registerPluginWithState(reg, "unhealthy", registry.StateUnhealthy, []registry.SlotRegistration{
		{Pipeline: "mixed-pipeline", Slot: "entry", Priority: 3},
	})
	registerPluginWithState(reg, "dead", registry.StateDead, []registry.SlotRegistration{
		{Pipeline: "mixed-pipeline", Slot: "entry", Priority: 4},
	})

	spec := &PipelineSpec{
		Name:            "mixed-pipeline",
		PipelineVersion: 1,
		Slots: []SlotSpec{
			{Name: "entry", Required: true, ValidEntryPoint: true, TimeoutSeconds: 5},
		},
	}
	r.LoadPipeline(spec)

	dt := CompileEnhancedTable(r)
	slot, err := ResolveSlot("mixed-pipeline", "entry", dt)
	if err != nil {
		t.Fatalf("ResolveSlot failed: %v", err)
	}

	// Should have 3 providers (healthy, starting, unhealthy) — dead excluded
	if len(slot.Providers) != 3 {
		t.Fatalf("expected 3 eligible providers, got %d", len(slot.Providers))
	}

	// Verify all 3 states are present
	states := map[registry.PluginState]bool{}
	for _, p := range slot.Providers {
		states[p.State] = true
	}
	if !states[registry.StateHealthyActive] {
		t.Error("expected HEALTHY_ACTIVE provider")
	}
	if !states[registry.StateStarting] {
		t.Error("expected STARTING provider")
	}
	if !states[registry.StateUnhealthy] {
		t.Error("expected UNHEALTHY provider")
	}
	if states[registry.StateDead] {
		t.Error("DEAD provider should have been filtered out")
	}

	// IsSlotAvailable should be true because we have a HEALTHY_ACTIVE provider
	if !IsSlotAvailable(slot) {
		t.Error("expected slot to be available (has HEALTHY_ACTIVE provider)")
	}
}

// --- CompiledProvider fields ---

func TestCompiledProviderFields(t *testing.T) {
	reg := registry.New()
	r := New(reg, nil)

	registerPluginWithState(reg, "latency-plugin", registry.StateHealthyActive, []registry.SlotRegistration{
		{Pipeline: "fields-pipeline", Slot: "entry", Priority: 5},
	})

	// Set latency class
	p, _ := reg.Get("latency-plugin")
	p.LatencyClass = "fast"

	spec := &PipelineSpec{
		Name:            "fields-pipeline",
		PipelineVersion: 1,
		Slots: []SlotSpec{
			{Name: "entry", Required: true, ValidEntryPoint: true, TimeoutSeconds: 5, Capability: "PERCEPTION"},
		},
	}
	r.LoadPipeline(spec)

	dt := CompileEnhancedTable(r)
	slot, err := ResolveSlot("fields-pipeline", "entry", dt)
	if err != nil {
		t.Fatalf("ResolveSlot failed: %v", err)
	}

	if len(slot.Providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(slot.Providers))
	}

	provider := slot.Providers[0]
	if provider.PluginID != "latency-plugin" {
		t.Fatalf("expected PluginID latency-plugin, got %s", provider.PluginID)
	}
	if provider.Priority != 5 {
		t.Fatalf("expected Priority 5, got %d", provider.Priority)
	}
	if provider.LatencyClass != "fast" {
		t.Fatalf("expected LatencyClass fast, got %s", provider.LatencyClass)
	}
	if provider.State != registry.StateHealthyActive {
		t.Fatalf("expected State HEALTHY_ACTIVE, got %s", provider.State)
	}

	// Check slot capability carried through
	if slot.Capability != "PERCEPTION" {
		t.Fatalf("expected Capability PERCEPTION, got %s", slot.Capability)
	}
}