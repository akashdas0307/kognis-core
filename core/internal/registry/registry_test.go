package registry

import (
	"errors"
	"testing"
)

func TestNew(t *testing.T) {
	r := New()
	if r == nil {
		t.Fatal("New() returned nil")
	}
	if len(r.plugins) != 0 {
		t.Fatalf("expected empty registry, got %d entries", len(r.plugins))
	}
}

func TestRegister(t *testing.T) {
	r := New()
	entry := &PluginEntry{
		ID:      "test-plugin",
		Name:    "TestPlugin",
		Version: "1.0.0",
		State:   StateRegistered,
	}

	if err := r.Register(entry); err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	got, ok := r.Get("test-plugin")
	if !ok {
		t.Fatal("Get() returned false after Register()")
	}
	if got.ID != "test-plugin" {
		t.Fatalf("expected ID test-plugin, got %s", got.ID)
	}
}

func TestRegisterCapabilityConflict(t *testing.T) {
	r := New()
	p1 := &PluginEntry{
		ID:           "p1",
		Capabilities: []string{"shared.cap"},
	}
	p2 := &PluginEntry{
		ID:           "p2",
		Capabilities: []string{"shared.cap"},
	}

	if err := r.Register(p1); err != nil {
		t.Fatalf("p1 registration failed: %v", err)
	}

	err := r.Register(p2)
	if err == nil {
		t.Fatal("expected error for duplicate capability, got nil")
	}
	if !errors.Is(err, ErrCapabilityConflict) {
		t.Fatalf("expected ErrCapabilityConflict, got %v", err)
	}
}

func TestUpdateStateValidation(t *testing.T) {
	r := New()
	entry := &PluginEntry{
		ID:    "p1",
		State: StateRegistered,
	}
	r.Register(entry)

	// Valid transition: REGISTERED -> STARTING
	if err := r.UpdateState("p1", StateStarting); err != nil {
		t.Fatalf("expected valid transition REGISTERED -> STARTING, got %v", err)
	}

	// Invalid transition: STARTING -> SHUT_DOWN (must go through HEALTHY_ACTIVE or similar, though SPEC 08 might vary, let's stick to our map)
	err := r.UpdateState("p1", StateShutDown)
	if err == nil {
		t.Fatal("expected error for invalid transition STARTING -> SHUT_DOWN, got nil")
	}
	if !errors.Is(err, ErrInvalidStateTransition) {
		t.Fatalf("expected ErrInvalidStateTransition, got %v", err)
	}

	// Valid transition: STARTING -> HEALTHY_ACTIVE
	if err := r.UpdateState("p1", StateHealthyActive); err != nil {
		t.Fatalf("expected valid transition STARTING -> HEALTHY_ACTIVE, got %v", err)
	}

	// Valid transition: HEALTHY_ACTIVE -> SHUTTING_DOWN
	if err := r.UpdateState("p1", StateShuttingDown); err != nil {
		t.Fatalf("expected valid transition HEALTHY_ACTIVE -> SHUTTING_DOWN, got %v", err)
	}

	// Valid transition: SHUTTING_DOWN -> SHUT_DOWN
	if err := r.UpdateState("p1", StateShutDown); err != nil {
		t.Fatalf("expected valid transition SHUTTING_DOWN -> SHUT_DOWN, got %v", err)
	}
}

func TestRemoveCleansCapabilities(t *testing.T) {
	r := New()
	p1 := &PluginEntry{
		ID:           "p1",
		Capabilities: []string{"cap1"},
	}
	r.Register(p1)

	if len(r.FindByCapability("cap1")) != 1 {
		t.Fatal("expected 1 provider for cap1")
	}

	r.Remove("p1")

	if len(r.FindByCapability("cap1")) != 0 {
		t.Fatal("expected 0 providers for cap1 after removal")
	}

	// Should be able to register another plugin with same capability now
	p2 := &PluginEntry{
		ID:           "p2",
		Capabilities: []string{"cap1"},
	}
	if err := r.Register(p2); err != nil {
		t.Fatalf("failed to register p2 after p1 removal: %v", err)
	}
}

func TestIsValidTransition(t *testing.T) {
	tests := []struct {
		from   PluginState
		to     PluginState
		wanted bool
	}{
		{StateRegistered, StateStarting, true},
		{StateStarting, StateHealthyActive, true},
		{StateHealthyActive, StateUnhealthy, true},
		{StateHealthyActive, StateShuttingDown, true},
		{StateShuttingDown, StateShutDown, true},
		{StateRegistered, StateHealthyActive, false},
		{StateHealthyActive, StateStarting, false},
		{StateDead, StateStarting, false},
		{StateHealthyActive, StateHealthyActive, true},
	}

	for _, tt := range tests {
		if got := IsValidTransition(tt.from, tt.to); got != tt.wanted {
			t.Errorf("IsValidTransition(%s, %s) = %v, want %v", tt.from, tt.to, got, tt.wanted)
		}
	}
}
