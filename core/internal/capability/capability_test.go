package capability

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/akashdas0307/kognis-core/core/internal/config"
	"github.com/akashdas0307/kognis-core/core/internal/eventbus"
	"github.com/akashdas0307/kognis-core/core/internal/registry"
)

// --- Test port assignments (avoid conflicts with other test packages) ---

const (
	testBusPortSubscribe1   = 15241
	testBusPortSubscribe2   = 15242
	testBusPortStatusChange = 15243
)

// --- Helpers ---

// newTestBusOnPort creates an event bus on the given port for tests.
func newTestBusOnPort(t *testing.T, port int) *eventbus.Bus {
	t.Helper()
	cfg := config.NATSConfig{
		ServerName: "cap-test",
		Port:       port,
		DataDir:    t.TempDir(),
	}
	bus, err := eventbus.New(cfg)
	if err != nil {
		t.Fatalf("create event bus: %v", err)
	}
	t.Cleanup(func() { bus.Close() })
	return bus
}

// newStoreWithBusOnPort creates a Store with a real event bus on the given port.
func newStoreWithBusOnPort(t *testing.T, port int) (*Store, *registry.Registry, *eventbus.Bus) {
	t.Helper()
	reg := registry.New()
	bus := newTestBusOnPort(t, port)
	store := NewStore(reg, bus)
	return store, reg, bus
}

// newStoreWithoutBus creates a Store with a nil bus for tests that don't
// need eventbus.
func newStoreWithoutBus(t *testing.T) (*Store, *registry.Registry) {
	t.Helper()
	reg := registry.New()
	store := NewStore(reg, nil)
	return store, reg
}

// registerHealthyPlugin is a test helper that registers a plugin in the
// HEALTHY_ACTIVE state with the given capabilities.
func registerHealthyPlugin(t *testing.T, reg *registry.Registry, id string, caps []string) {
	t.Helper()
	p := &registry.PluginEntry{
		ID:            id,
		Name:          id,
		Version:       "0.1.0",
		State:         registry.StateHealthyActive,
		Capabilities:  caps,
		HandshakeStep: 1,
	}
	if err := reg.Register(p); err != nil {
		t.Fatalf("register plugin %s: %v", id, err)
	}
}

// --- Tests ---

func TestNewStore(t *testing.T) {
	store, _ := newStoreWithoutBus(t)
	if store == nil {
		t.Fatal("NewStore returned nil")
	}
	if store.caps == nil {
		t.Fatal("caps map is nil")
	}
	if store.reg == nil {
		t.Fatal("registry reference is nil")
	}
}

func TestQueryAvailable_NotFound(t *testing.T) {
	store, _ := newStoreWithoutBus(t)
	if store.QueryAvailable("nonexistent") {
		t.Error("expected QueryAvailable to return false for unknown capability")
	}
}

func TestQueryAvailable_AfterSync(t *testing.T) {
	store, reg := newStoreWithoutBus(t)
	registerHealthyPlugin(t, reg, "p1", []string{"vision.detect", "audio.listen"})
	store.SyncFromRegistry()

	if !store.QueryAvailable("vision.detect") {
		t.Error("expected vision.detect to be available after sync")
	}
	if !store.QueryAvailable("audio.listen") {
		t.Error("expected audio.listen to be available after sync")
	}
}

func TestQueryAvailable_UnhealthyPlugin(t *testing.T) {
	store, reg := newStoreWithoutBus(t)

	p := &registry.PluginEntry{
		ID:            "p-unhealthy",
		Name:          "unhealthy",
		Version:       "0.1.0",
		State:         registry.StateUnhealthy,
		Capabilities:  []string{"cap.broken"},
		HandshakeStep: 1,
	}
	if err := reg.Register(p); err != nil {
		t.Fatalf("register: %v", err)
	}
	store.SyncFromRegistry()

	if store.QueryAvailable("cap.broken") {
		t.Error("expected cap.broken to be unavailable for unhealthy plugin")
	}
}

func TestFindProviders_Unknown(t *testing.T) {
	store, _ := newStoreWithoutBus(t)
	ids := store.FindProviders("no.such.cap")
	if ids != nil {
		t.Errorf("expected nil for unknown capability, got %v", ids)
	}
}

func TestFindProviders_Known(t *testing.T) {
	store, reg := newStoreWithoutBus(t)
	registerHealthyPlugin(t, reg, "p1", []string{"cap.alpha"})
	store.SyncFromRegistry()

	ids := store.FindProviders("cap.alpha")
	if len(ids) != 1 || ids[0] != "p1" {
		t.Errorf("expected [p1], got %v", ids)
	}
}

func TestFindProviders_ReturnsCopy(t *testing.T) {
	store, reg := newStoreWithoutBus(t)
	registerHealthyPlugin(t, reg, "p1", []string{"cap.alpha"})
	store.SyncFromRegistry()

	ids := store.FindProviders("cap.alpha")
	ids[0] = "mutated" // should not affect internal state

	ids2 := store.FindProviders("cap.alpha")
	if ids2[0] != "p1" {
		t.Error("FindProviders returned a reference, not a copy")
	}
}

func TestGetSchema_NotFound(t *testing.T) {
	store, _ := newStoreWithoutBus(t)
	_, err := store.GetSchema("no.such.cap")
	if err == nil {
		t.Fatal("expected error for unknown capability")
	}
	if !isErrCapabilityNotFound(err) {
		t.Errorf("expected ErrCapabilityNotFound, got %v", err)
	}
}

func TestGetSchema_NoSchemaSet(t *testing.T) {
	store, reg := newStoreWithoutBus(t)
	registerHealthyPlugin(t, reg, "p1", []string{"cap.no-schema"})
	store.SyncFromRegistry()

	schema, err := store.GetSchema("cap.no-schema")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema != nil {
		t.Errorf("expected nil schema, got %v", schema)
	}
}

func TestGetSchema_WithSchema(t *testing.T) {
	store, reg := newStoreWithoutBus(t)
	registerHealthyPlugin(t, reg, "p1", []string{"cap.schema"})
	store.SyncFromRegistry()

	want := map[string]interface{}{
		"params":   map[string]interface{}{"type": "object"},
		"response": map[string]interface{}{"type": "string"},
	}
	if err := store.SetSchema("cap.schema", want); err != nil {
		t.Fatalf("SetSchema: %v", err)
	}

	got, err := store.GetSchema("cap.schema")
	if err != nil {
		t.Fatalf("GetSchema: %v", err)
	}
	if got["params"] == nil || got["response"] == nil {
		t.Errorf("schema missing expected keys: %v", got)
	}
}

func TestSetSchema_NotFound(t *testing.T) {
	store, _ := newStoreWithoutBus(t)
	err := store.SetSchema("no.such.cap", map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for unknown capability")
	}
	if !isErrCapabilityNotFound(err) {
		t.Errorf("expected ErrCapabilityNotFound, got %v", err)
	}
}

func TestGetSchema_ReturnsCopy(t *testing.T) {
	store, reg := newStoreWithoutBus(t)
	registerHealthyPlugin(t, reg, "p1", []string{"cap.copy"})
	store.SyncFromRegistry()

	schema := map[string]interface{}{"key": "value"}
	if err := store.SetSchema("cap.copy", schema); err != nil {
		t.Fatalf("SetSchema: %v", err)
	}

	got, _ := store.GetSchema("cap.copy")
	got["key"] = "mutated" // should not affect internal state

	got2, _ := store.GetSchema("cap.copy")
	if got2["key"] == "mutated" {
		t.Error("GetSchema returned a reference, not a copy")
	}
}

func TestListForLLM_ExposedToSpecific(t *testing.T) {
	store, reg := newStoreWithoutBus(t)

	p1 := &registry.PluginEntry{
		ID:            "llm-brain",
		Name:          "brain",
		Version:       "0.1.0",
		State:         registry.StateHealthyActive,
		Capabilities:  []string{"cap.reason"},
		LLMExposedTo:  []string{"llm-brain"}, // only self
		HandshakeStep: 1,
	}
	p2 := &registry.PluginEntry{
		ID:            "llm-mouth",
		Name:          "mouth",
		Version:       "0.1.0",
		State:         registry.StateHealthyActive,
		Capabilities:  []string{"cap.speak"},
		LLMExposedTo:  []string{"llm-brain"},
		HandshakeStep: 1,
	}
	reg.Register(p1)
	reg.Register(p2)
	store.SyncFromRegistry()

	// llm-brain should see cap.reason (its own) and cap.speak (exposed to it)
	entries := store.ListForLLM("llm-brain")
	ids := capIDsFromEntries(entries)
	if !containsString(ids, "cap.reason") {
		t.Error("llm-brain should see cap.reason")
	}
	if !containsString(ids, "cap.speak") {
		t.Error("llm-brain should see cap.speak (exposed to it)")
	}

	// llm-mouth should NOT see cap.speak (it is only exposed to llm-brain)
	// nor cap.reason (only exposed to llm-brain)
	entries2 := store.ListForLLM("llm-mouth")
	ids2 := capIDsFromEntries(entries2)
	if containsString(ids2, "cap.reason") {
		t.Error("llm-mouth should NOT see cap.reason")
	}
	if containsString(ids2, "cap.speak") {
		t.Error("llm-mouth should NOT see cap.speak (only exposed to llm-brain)")
	}
}

func TestListForLLM_EmptyExposedToMeansAll(t *testing.T) {
	store, reg := newStoreWithoutBus(t)

	p1 := &registry.PluginEntry{
		ID:            "p-open",
		Name:          "open",
		Version:       "0.1.0",
		State:         registry.StateHealthyActive,
		Capabilities:  []string{"cap.public"},
		LLMExposedTo:  nil, // empty = exposed to all
		HandshakeStep: 1,
	}
	reg.Register(p1)
	store.SyncFromRegistry()

	entries := store.ListForLLM("any-plugin")
	ids := capIDsFromEntries(entries)
	if !containsString(ids, "cap.public") {
		t.Error("empty LLMExposedTo should expose to everyone")
	}
}

func TestListForLLM_NoMatch(t *testing.T) {
	store, reg := newStoreWithoutBus(t)

	p1 := &registry.PluginEntry{
		ID:            "p-restricted",
		Name:          "restricted",
		Version:       "0.1.0",
		State:         registry.StateHealthyActive,
		Capabilities:  []string{"cap.secret"},
		LLMExposedTo:  []string{"p-restricted"}, // only self
		HandshakeStep: 1,
	}
	reg.Register(p1)
	store.SyncFromRegistry()

	entries := store.ListForLLM("stranger")
	if len(entries) != 0 {
		t.Errorf("expected no entries for stranger, got %d", len(entries))
	}
}

func TestSyncFromRegistry_PreservesSchema(t *testing.T) {
	store, reg := newStoreWithoutBus(t)
	registerHealthyPlugin(t, reg, "p1", []string{"cap.persistent"})
	store.SyncFromRegistry()

	// Set a schema
	schema := map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
	if err := store.SetSchema("cap.persistent", schema); err != nil {
		t.Fatalf("SetSchema: %v", err)
	}

	// Re-sync (e.g. after a heartbeat or another plugin registers)
	registerHealthyPlugin(t, reg, "p2", []string{"cap.other"})
	store.SyncFromRegistry()

	// The schema for cap.persistent should survive
	got, err := store.GetSchema("cap.persistent")
	if err != nil {
		t.Fatalf("GetSchema after re-sync: %v", err)
	}
	if got == nil {
		t.Fatal("schema was lost after re-sync")
	}
	if got["type"] != "object" {
		t.Errorf("schema corrupted after re-sync: %v", got)
	}
}

func TestSyncFromRegistry_RemovedCapability(t *testing.T) {
	store, reg := newStoreWithoutBus(t)
	registerHealthyPlugin(t, reg, "p1", []string{"cap.vanish"})
	store.SyncFromRegistry()

	if !store.QueryAvailable("cap.vanish") {
		t.Fatal("cap.vanish should exist after first sync")
	}

	// Remove the plugin (and its capability)
	reg.Remove("p1")
	store.SyncFromRegistry()

	if store.QueryAvailable("cap.vanish") {
		t.Error("cap.vanish should be gone after plugin removed and re-synced")
	}
}

func TestSyncFromRegistry_StatusChangeDetected(t *testing.T) {
	store, reg, _ := newStoreWithBusOnPort(t, testBusPortStatusChange)

	registerHealthyPlugin(t, reg, "p1", []string{"cap.watcher"})
	store.SyncFromRegistry()

	// Now change plugin state to UNHEALTHY — capability becomes unavailable
	if err := reg.UpdateState("p1", registry.StateUnhealthy); err != nil {
		t.Fatalf("UpdateState: %v", err)
	}
	store.SyncFromRegistry()

	if store.QueryAvailable("cap.watcher") {
		t.Error("cap.watcher should be unavailable after plugin goes unhealthy")
	}
}

func TestSubscribeChanges_EventFired(t *testing.T) {
	store, reg, _ := newStoreWithBusOnPort(t, testBusPortSubscribe1)

	var received struct {
		mu     sync.Mutex
		events []changeEvent
	}

	sub, err := store.SubscribeChanges(func(capID string, status string) {
		received.mu.Lock()
		received.events = append(received.events, changeEvent{CapID: capID, Status: status})
		received.mu.Unlock()
	})
	if err != nil {
		t.Fatalf("SubscribeChanges: %v", err)
	}
	t.Cleanup(func() { sub.Unsubscribe() })

	// Register a plugin and sync — this should fire a "new capability" event.
	registerHealthyPlugin(t, reg, "p1", []string{"cap.new"})
	store.SyncFromRegistry()

	// Give NATS a moment to deliver.
	waitForEvents(t, &received, 1, 2*time.Second)

	received.mu.Lock()
	defer received.mu.Unlock()
	if len(received.events) < 1 {
		t.Fatalf("expected at least 1 change event, got %d", len(received.events))
	}
	found := false
	for _, evt := range received.events {
		if evt.CapID == "cap.new" && evt.Status == "available" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("missing cap.new/available event; got %v", received.events)
	}
}

func TestSubscribeChanges_RemovalEvent(t *testing.T) {
	store, reg, _ := newStoreWithBusOnPort(t, testBusPortSubscribe2)

	var received struct {
		mu     sync.Mutex
		events []changeEvent
	}

	sub, err := store.SubscribeChanges(func(capID string, status string) {
		received.mu.Lock()
		received.events = append(received.events, changeEvent{CapID: capID, Status: status})
		received.mu.Unlock()
	})
	if err != nil {
		t.Fatalf("SubscribeChanges: %v", err)
	}
	t.Cleanup(func() { sub.Unsubscribe() })

	registerHealthyPlugin(t, reg, "p1", []string{"cap.vanish"})
	store.SyncFromRegistry()

	// Remove plugin and sync — should fire "unavailable" event.
	reg.Remove("p1")
	store.SyncFromRegistry()

	waitForEvents(t, &received, 2, 2*time.Second)

	received.mu.Lock()
	defer received.mu.Unlock()
	found := false
	for _, evt := range received.events {
		if evt.CapID == "cap.vanish" && evt.Status == "unavailable" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("missing cap.vanish/unavailable removal event; got %v", received.events)
	}
}

func TestSubscribeChanges_NilBus(t *testing.T) {
	store, _ := newStoreWithoutBus(t)

	_, err := store.SubscribeChanges(func(capID string, status string) {})
	if err == nil {
		t.Fatal("expected error when subscribing with nil bus")
	}
}

func TestCapabilityEntry_LatencyClass(t *testing.T) {
	store, reg := newStoreWithoutBus(t)

	p := &registry.PluginEntry{
		ID:            "p-rt",
		Name:          "realtime",
		Version:       "0.1.0",
		State:         registry.StateHealthyActive,
		Capabilities:  []string{"cap.rt"},
		LatencyClass:  "realtime",
		HandshakeStep: 1,
	}
	reg.Register(p)
	store.SyncFromRegistry()

	ids := store.FindProviders("cap.rt")
	if len(ids) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(ids))
	}

	// Verify the enriched entry has latency class
	store.mu.RLock()
	entry := store.caps["cap.rt"]
	store.mu.RUnlock()

	if entry.LatencyClass != "realtime" {
		t.Errorf("expected LatencyClass=realtime, got %s", entry.LatencyClass)
	}
}

func TestCapabilityEntry_LLMExposedTo(t *testing.T) {
	store, reg := newStoreWithoutBus(t)

	p := &registry.PluginEntry{
		ID:            "p-expose",
		Name:          "expose",
		Version:       "0.1.0",
		State:         registry.StateHealthyActive,
		Capabilities:  []string{"cap.exposed"},
		LLMExposedTo:  []string{"llm-a", "llm-b"},
		HandshakeStep: 1,
	}
	reg.Register(p)
	store.SyncFromRegistry()

	store.mu.RLock()
	entry := store.caps["cap.exposed"]
	store.mu.RUnlock()

	if len(entry.LLMExposedTo) != 2 {
		t.Errorf("expected 2 LLMExposedTo entries, got %d", len(entry.LLMExposedTo))
	}
}

func TestSyncFromRegistry_MultiplePlugins(t *testing.T) {
	store, reg := newStoreWithoutBus(t)

	registerHealthyPlugin(t, reg, "p1", []string{"cap.a", "cap.b"})
	registerHealthyPlugin(t, reg, "p2", []string{"cap.c"})
	store.SyncFromRegistry()

	if !store.QueryAvailable("cap.a") {
		t.Error("cap.a should be available")
	}
	if !store.QueryAvailable("cap.b") {
		t.Error("cap.b should be available")
	}
	if !store.QueryAvailable("cap.c") {
		t.Error("cap.c should be available")
	}

	providersA := store.FindProviders("cap.a")
	if len(providersA) != 1 || providersA[0] != "p1" {
		t.Errorf("cap.a providers: expected [p1], got %v", providersA)
	}
}

// --- Internal test helpers ---

func isErrCapabilityNotFound(err error) bool {
	return errors.Is(err, registry.ErrCapabilityNotFound)
}

func capIDsFromEntries(entries []CapabilityEntry) []string {
	ids := make([]string, len(entries))
	for i, e := range entries {
		ids[i] = e.ID
	}
	return ids
}

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// waitForEvents polls until at least minEvents are recorded or timeout.
func waitForEvents(t *testing.T, received *struct {
	mu     sync.Mutex
	events []changeEvent
}, minEvents int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		received.mu.Lock()
		count := len(received.events)
		received.mu.Unlock()
		if count >= minEvents {
			return
		}
		if time.Now().After(deadline) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}