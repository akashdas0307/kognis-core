package router

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/akashdas0307/kognis-core/core/internal/config"
	"github.com/akashdas0307/kognis-core/core/internal/envelope"
	"github.com/akashdas0307/kognis-core/core/internal/eventbus"
	"github.com/akashdas0307/kognis-core/core/internal/registry"
)

// testPortCounter provides unique ports for parallel tests.
var testPortCounter int64 = 15300

// nextTestPort returns a unique port number for test isolation.
func nextTestPort() int {
	return int(atomic.AddInt64(&testPortCounter, 1))
}

// newTestBus creates an eventbus.Bus for testing with a unique port.
func newTestBus(t *testing.T) *eventbus.Bus {
	t.Helper()
	port := nextTestPort()
	cfg := config.NATSConfig{
		Port:       port,
		DataDir:    t.TempDir(),
		ServerName: fmt.Sprintf("test-server-%d", port),
	}
	bus, err := eventbus.New(cfg)
	if err != nil {
		t.Fatalf("failed to create test eventbus (port %d): %v", port, err)
	}
	t.Cleanup(func() { bus.Close() })
	return bus
}

// setupMessageRouter creates a MessageRouter with a loaded pipeline and registered plugins.
func setupMessageRouter(t *testing.T) (*MessageRouter, *registry.Registry, *eventbus.Bus) {
	t.Helper()

	reg := registry.New()
	bus := newTestBus(t)
	rtr := New(reg, bus)

	// Register plugins with slot registrations.
	plugin1 := &registry.PluginEntry{
		ID:    "plugin-perception",
		State: registry.StateHealthyActive,
		SlotRegistrations: []registry.SlotRegistration{
			{Pipeline: "PERCEPTION_TO_COGNITION", Slot: "perceive", Priority: 1},
		},
	}
	plugin2 := &registry.PluginEntry{
		ID:    "plugin-preprocess",
		State: registry.StateHealthyActive,
		SlotRegistrations: []registry.SlotRegistration{
			{Pipeline: "PERCEPTION_TO_COGNITION", Slot: "preprocess", Priority: 1},
		},
	}
	plugin3 := &registry.PluginEntry{
		ID:    "plugin-perception-backup",
		State: registry.StateHealthyActive,
		SlotRegistrations: []registry.SlotRegistration{
			{Pipeline: "PERCEPTION_TO_COGNITION", Slot: "perceive", Priority: 2},
		},
	}
	reg.Register(plugin1)
	reg.Register(plugin2)
	reg.Register(plugin3)

	// Load pipeline.
	spec := &PipelineSpec{
		Name:            "PERCEPTION_TO_COGNITION",
		PipelineVersion: 1,
		Slots: []SlotSpec{
			{Name: "perceive", Capability: "PERCEPTION", Required: true, ValidEntryPoint: true, TimeoutSeconds: 5},
			{Name: "preprocess", Capability: "PERCEPTION", Required: false, TimeoutSeconds: 10},
		},
	}
	if err := rtr.LoadPipeline(spec); err != nil {
		t.Fatalf("failed to load pipeline: %v", err)
	}

	mr := NewMessageRouter(rtr, reg, bus)
	return mr, reg, bus
}

// validEnvelope creates a valid test envelope.
func validEnvelope() *envelope.Envelope {
	return &envelope.Envelope{
		ID:       "msg-001",
		Type:     envelope.TypePerception,
		Source:   "test-source",
		Pipeline: "PERCEPTION_TO_COGNITION",
		HopCount: 0,
		MaxHops:  10,
		Priority: "normal",
		Timestamp: time.Now(),
		Payload:  json.RawMessage(`{"data": "test"}`),
	}
}

func TestRouteMessage_ValidEnvelope(t *testing.T) {
	mr, _, bus := setupMessageRouter(t)

	// Subscribe to the slot subject to verify the message is published.
	var received []byte
	var recvMu sync.Mutex
	_, err := bus.Subscribe(eventbus.SlotSubject("PERCEPTION_TO_COGNITION", "perceive"), func(msg *nats.Msg) {
		recvMu.Lock()
		received = msg.Data
		recvMu.Unlock()
	})
	if err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}

	env := validEnvelope()
	if err := mr.RouteMessage(env); err != nil {
		t.Fatalf("RouteMessage() failed: %v", err)
	}

	// Wait briefly for the message to be received.
	time.Sleep(50 * time.Millisecond)

	recvMu.Lock()
	if len(received) == 0 {
		recvMu.Unlock()
		t.Fatal("expected message to be published on slot subject, got nothing")
	}
	recvMu.Unlock()

	// Verify inflight tracking.
	if mr.GetInflightCount() != 1 {
		t.Fatalf("expected 1 inflight message, got %d", mr.GetInflightCount())
	}

	mr.mu.RLock()
	im, ok := mr.inflight["msg-001"]
	mr.mu.RUnlock()
	if !ok {
		t.Fatal("expected msg-001 in inflight tracking")
	}
	if im.Status != StatusAwaitingACK {
		t.Fatalf("expected AWAITING_ACK, got %s", im.Status)
	}
	if im.Pipeline != "PERCEPTION_TO_COGNITION" {
		t.Fatalf("expected pipeline PERCEPTION_TO_COGNITION, got %s", im.Pipeline)
	}
	if im.Slot != "perceive" {
		t.Fatalf("expected slot perceive, got %s", im.Slot)
	}
	if im.PluginID != "plugin-perception" {
		t.Fatalf("expected plugin-perception, got %s", im.PluginID)
	}
}

func TestRouteMessage_InvalidEnvelope(t *testing.T) {
	mr, _, _ := setupMessageRouter(t)

	env := &envelope.Envelope{
		// Missing ID, Type, Source — should fail validation.
		Pipeline: "PERCEPTION_TO_COGNITION",
	}

	if err := mr.RouteMessage(env); err == nil {
		t.Fatal("expected error for invalid envelope, got nil")
	}
}

func TestRouteMessage_HopCountExceeded(t *testing.T) {
	mr, _, _ := setupMessageRouter(t)

	env := &envelope.Envelope{
		ID:       "msg-hop-exceeded",
		Type:     envelope.TypePerception,
		Source:   "test-source",
		Pipeline: "PERCEPTION_TO_COGNITION",
		HopCount: 10,
		MaxHops:  10,
		Priority: "normal",
		Timestamp: time.Now(),
		Payload:  json.RawMessage(`{}`),
	}

	if err := mr.RouteMessage(env); err == nil {
		t.Fatal("expected error for hop count exceeded, got nil")
	}
}

func TestRouteMessage_NoPipeline(t *testing.T) {
	mr, _, _ := setupMessageRouter(t)

	env := &envelope.Envelope{
		ID:       "msg-no-pipeline",
		Type:     envelope.TypePerception,
		Source:   "test-source",
		Pipeline: "", // empty pipeline
		HopCount: 0,
		MaxHops:  10,
		Priority: "normal",
		Timestamp: time.Now(),
		Payload:  json.RawMessage(`{}`),
	}

	if err := mr.RouteMessage(env); err == nil {
		t.Fatal("expected error for missing pipeline, got nil")
	}
}

func TestHandleACK(t *testing.T) {
	mr, _, _ := setupMessageRouter(t)

	env := validEnvelope()
	if err := mr.RouteMessage(env); err != nil {
		t.Fatalf("RouteMessage() failed: %v", err)
	}

	if err := mr.HandleACK("msg-001", "plugin-perception", 200); err != nil {
		t.Fatalf("HandleACK() failed: %v", err)
	}

	mr.mu.RLock()
	im := mr.inflight["msg-001"]
	mr.mu.RUnlock()

	if im.Status != StatusProcessing {
		t.Fatalf("expected PROCESSING, got %s", im.Status)
	}
	if im.PluginID != "plugin-perception" {
		t.Fatalf("expected plugin-perception, got %s", im.PluginID)
	}
}

func TestHandleACK_WrongStatus(t *testing.T) {
	mr, _, _ := setupMessageRouter(t)

	env := validEnvelope()
	if err := mr.RouteMessage(env); err != nil {
		t.Fatalf("RouteMessage() failed: %v", err)
	}

	// ACK the message first.
	if err := mr.HandleACK("msg-001", "plugin-perception", 200); err != nil {
		t.Fatalf("HandleACK() failed: %v", err)
	}

	// Try to ACK again — should fail because status is PROCESSING.
	if err := mr.HandleACK("msg-001", "plugin-perception", 300); err == nil {
		t.Fatal("expected error for double ACK, got nil")
	}
}

func TestHandleComplete(t *testing.T) {
	mr, _, _ := setupMessageRouter(t)

	env := validEnvelope()
	if err := mr.RouteMessage(env); err != nil {
		t.Fatalf("RouteMessage() failed: %v", err)
	}

	// ACK the message first.
	if err := mr.HandleACK("msg-001", "plugin-perception", 100); err != nil {
		t.Fatalf("HandleACK() failed: %v", err)
	}

	// Complete the message — should route to next slot.
	result := []byte(`{"processed": true}`)
	if err := mr.HandleComplete("msg-001", result); err != nil {
		t.Fatalf("HandleComplete() failed: %v", err)
	}

	// The original inflight entry for "perceive" should be removed.
	// A new one for "preprocess" should exist (with a different slot).
	mr.mu.RLock()
	im, ok := mr.inflight["msg-001"]
	mr.mu.RUnlock()

	if !ok {
		// If the envelope was routed to the next slot, a new inflight entry
		// should exist. If no more slots, it would be removed.
		// With our pipeline having "preprocess" after "perceive",
		// the message should be re-routed.
		t.Fatal("expected msg-001 to be re-routed to next slot")
	}

	if im.Slot != "preprocess" {
		t.Fatalf("expected slot preprocess, got %s", im.Slot)
	}
}

func TestHandleComplete_LastSlot(t *testing.T) {
	// Create a pipeline with a single slot.
	reg := registry.New()
	bus := newTestBus(t)
	rtr := New(reg, bus)

	reg.Register(&registry.PluginEntry{
		ID:    "plugin-single",
		State: registry.StateHealthyActive,
		SlotRegistrations: []registry.SlotRegistration{
			{Pipeline: "single-slot-pipeline", Slot: "only", Priority: 1},
		},
	})

	spec := &PipelineSpec{
		Name:            "single-slot-pipeline",
		PipelineVersion: 1,
		Slots: []SlotSpec{
			{Name: "only", Required: true, ValidEntryPoint: true, TimeoutSeconds: 5},
		},
	}
	rtr.LoadPipeline(spec)

	mr := NewMessageRouter(rtr, reg, bus)

	env := &envelope.Envelope{
		ID:       "msg-single",
		Type:     envelope.TypeCognition,
		Source:   "test-source",
		Pipeline: "single-slot-pipeline",
		HopCount: 0,
		MaxHops:  10,
		Priority: "normal",
		Timestamp: time.Now(),
		Payload:  json.RawMessage(`{}`),
	}

	if err := mr.RouteMessage(env); err != nil {
		t.Fatalf("RouteMessage() failed: %v", err)
	}

	if err := mr.HandleACK("msg-single", "plugin-single", 50); err != nil {
		t.Fatalf("HandleACK() failed: %v", err)
	}

	if err := mr.HandleComplete("msg-single", []byte(`done`)); err != nil {
		t.Fatalf("HandleComplete() failed: %v", err)
	}

	// Message should be removed from inflight since it was the last slot.
	mr.mu.RLock()
	_, ok := mr.inflight["msg-single"]
	mr.mu.RUnlock()

	if ok {
		t.Fatal("expected msg-001 to be removed from inflight after completing last slot")
	}
}

func TestHandleFailed_RetrySafe(t *testing.T) {
	mr, _, _ := setupMessageRouter(t)

	env := validEnvelope()
	if err := mr.RouteMessage(env); err != nil {
		t.Fatalf("RouteMessage() failed: %v", err)
	}

	// Verify initial provider is plugin-perception (priority 1).
	mr.mu.RLock()
	im := mr.inflight["msg-001"]
	mr.mu.RUnlock()

	if im.PluginID != "plugin-perception" {
		t.Fatalf("expected initial provider plugin-perception, got %s", im.PluginID)
	}
	if im.providerIndex != 0 {
		t.Fatalf("expected providerIndex 0, got %d", im.providerIndex)
	}

	// Fail with retrySafe=true — should try next provider.
	if err := mr.HandleFailed("msg-001", "KGN-CAPABILITY-UNAVAILABLE-ERROR", true); err != nil {
		t.Fatalf("HandleFailed() failed: %v", err)
	}

	mr.mu.RLock()
	im = mr.inflight["msg-001"]
	mr.mu.RUnlock()

	if im.Status != StatusAwaitingACK {
		t.Fatalf("expected AWAITING_ACK after retry, got %s", im.Status)
	}
	if im.PluginID != "plugin-perception-backup" {
		t.Fatalf("expected retry with plugin-perception-backup, got %s", im.PluginID)
	}
	if im.providerIndex != 1 {
		t.Fatalf("expected providerIndex 1, got %d", im.providerIndex)
	}
}

func TestHandleFailed_NoRetry(t *testing.T) {
	mr, _, _ := setupMessageRouter(t)

	env := validEnvelope()
	if err := mr.RouteMessage(env); err != nil {
		t.Fatalf("RouteMessage() failed: %v", err)
	}

	// Fail with retrySafe=false — should mark as FAILED immediately.
	if err := mr.HandleFailed("msg-001", "KGN-PERMISSION-DENIED-ERROR", false); err != nil {
		t.Fatalf("HandleFailed() failed: %v", err)
	}

	mr.mu.RLock()
	im := mr.inflight["msg-001"]
	mr.mu.RUnlock()

	if im.Status != StatusFailed {
		t.Fatalf("expected FAILED, got %s", im.Status)
	}
}

func TestHandleFailed_AllProvidersExhausted(t *testing.T) {
	mr, _, _ := setupMessageRouter(t)

	env := validEnvelope()
	if err := mr.RouteMessage(env); err != nil {
		t.Fatalf("RouteMessage() failed: %v", err)
	}

	// First failure — retry with backup.
	if err := mr.HandleFailed("msg-001", "KGN-CAPABILITY-UNAVAILABLE-ERROR", true); err != nil {
		t.Fatalf("HandleFailed() first retry failed: %v", err)
	}

	// Second failure — no more providers.
	if err := mr.HandleFailed("msg-001", "KGN-CAPABILITY-UNAVAILABLE-ERROR", true); err != nil {
		t.Fatalf("HandleFailed() second failure: %v", err)
	}

	mr.mu.RLock()
	im := mr.inflight["msg-001"]
	mr.mu.RUnlock()

	if im.Status != StatusFailed {
		t.Fatalf("expected FAILED after all providers exhausted, got %s", im.Status)
	}
}

func TestCheckTimeouts(t *testing.T) {
	mr, _, _ := setupMessageRouter(t)

	env := validEnvelope()
	if err := mr.RouteMessage(env); err != nil {
		t.Fatalf("RouteMessage() failed: %v", err)
	}

	// Manually set DispatchedAt to over 500ms ago to simulate ACK timeout.
	mr.mu.Lock()
	im := mr.inflight["msg-001"]
	im.DispatchedAt = time.Now().Add(-600 * time.Millisecond)
	mr.mu.Unlock()

	mr.CheckTimeouts()

	mr.mu.RLock()
	im = mr.inflight["msg-001"]
	mr.mu.RUnlock()

	if im.Status != StatusTimeout {
		t.Fatalf("expected TIMEOUT, got %s", im.Status)
	}
}

func TestCheckTimeouts_ProcessingDeadline(t *testing.T) {
	mr, _, _ := setupMessageRouter(t)

	env := validEnvelope()
	if err := mr.RouteMessage(env); err != nil {
		t.Fatalf("RouteMessage() failed: %v", err)
	}

	// ACK the message so it's in PROCESSING state.
	if err := mr.HandleACK("msg-001", "plugin-perception", 200); err != nil {
		t.Fatalf("HandleACK() failed: %v", err)
	}

	// Manually set Deadline to the past to simulate processing timeout.
	mr.mu.Lock()
	im := mr.inflight["msg-001"]
	im.Deadline = time.Now().Add(-1 * time.Second)
	mr.mu.Unlock()

	mr.CheckTimeouts()

	mr.mu.RLock()
	im = mr.inflight["msg-001"]
	mr.mu.RUnlock()

	if im.Status != StatusTimeout {
		t.Fatalf("expected TIMEOUT, got %s", im.Status)
	}
}

func TestCheckTimeouts_NoTimeout(t *testing.T) {
	mr, _, _ := setupMessageRouter(t)

	env := validEnvelope()
	if err := mr.RouteMessage(env); err != nil {
		t.Fatalf("RouteMessage() failed: %v", err)
	}

	// Check immediately — should NOT timeout.
	mr.CheckTimeouts()

	mr.mu.RLock()
	im := mr.inflight["msg-001"]
	mr.mu.RUnlock()

	if im.Status != StatusAwaitingACK {
		t.Fatalf("expected AWAITING_ACK (not yet timed out), got %s", im.Status)
	}
}

func TestGetInflightCount(t *testing.T) {
	mr, _, _ := setupMessageRouter(t)

	if mr.GetInflightCount() != 0 {
		t.Fatalf("expected 0 inflight, got %d", mr.GetInflightCount())
	}

	env1 := validEnvelope()
	if err := mr.RouteMessage(env1); err != nil {
		t.Fatalf("RouteMessage() env1 failed: %v", err)
	}

	if mr.GetInflightCount() != 1 {
		t.Fatalf("expected 1 inflight, got %d", mr.GetInflightCount())
	}

	env2 := &envelope.Envelope{
		ID:       "msg-002",
		Type:     envelope.TypeCognition,
		Source:   "test-source",
		Pipeline: "PERCEPTION_TO_COGNITION",
		HopCount: 0,
		MaxHops:  10,
		Priority: "normal",
		Timestamp: time.Now(),
		Payload:  json.RawMessage(`{}`),
	}
	if err := mr.RouteMessage(env2); err != nil {
		t.Fatalf("RouteMessage() env2 failed: %v", err)
	}

	if mr.GetInflightCount() != 2 {
		t.Fatalf("expected 2 inflight, got %d", mr.GetInflightCount())
	}

	// Mark one as failed — it should not be counted.
	mr.HandleFailed("msg-001", "ERROR", false)

	if mr.GetInflightCount() != 1 {
		t.Fatalf("expected 1 inflight after one failed, got %d", mr.GetInflightCount())
	}
}

func TestHandleACK_NotFound(t *testing.T) {
	mr, _, _ := setupMessageRouter(t)

	if err := mr.HandleACK("nonexistent-msg", "plugin-x", 100); err == nil {
		t.Fatal("expected error for unknown message, got nil")
	}
}

func TestHandleComplete_NotFound(t *testing.T) {
	mr, _, _ := setupMessageRouter(t)

	if err := mr.HandleComplete("nonexistent-msg", []byte{}); err == nil {
		t.Fatal("expected error for unknown message, got nil")
	}
}

func TestHandleFailed_NotFound(t *testing.T) {
	mr, _, _ := setupMessageRouter(t)

	if err := mr.HandleFailed("nonexistent-msg", "ERROR", true); err == nil {
		t.Fatal("expected error for unknown message, got nil")
	}
}