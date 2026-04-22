package registry

import (
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
	if got.Name != "TestPlugin" {
		t.Fatalf("expected Name TestPlugin, got %s", got.Name)
	}
}

func TestRegisterDuplicate(t *testing.T) {
	r := New()
	entry := &PluginEntry{ID: "dup", Name: "Dup", Version: "1.0.0", State: StateRegistered}

	if err := r.Register(entry); err != nil {
		t.Fatalf("first Register() failed: %v", err)
	}

	if err := r.Register(entry); err == nil {
		t.Fatal("expected error for duplicate registration, got nil")
	}
}

func TestGetNotFound(t *testing.T) {
	r := New()
	_, ok := r.Get("nonexistent")
	if ok {
		t.Fatal("expected false for nonexistent plugin, got true")
	}
}

func TestUpdateState(t *testing.T) {
	r := New()
	entry := &PluginEntry{ID: "state-test", Name: "ST", Version: "1.0.0", State: StateRegistered}
	r.Register(entry)

	if err := r.UpdateState("state-test", StateStarting); err != nil {
		t.Fatalf("UpdateState() failed: %v", err)
	}

	got, _ := r.Get("state-test")
	if got.State != StateStarting {
		t.Fatalf("expected state STARTING, got %s", got.State)
	}
}

func TestUpdateStateNotFound(t *testing.T) {
	r := New()
	err := r.UpdateState("nonexistent", StateHealthyActive)
	if err == nil {
		t.Fatal("expected error for updating nonexistent plugin, got nil")
	}
}

func TestRemove(t *testing.T) {
	r := New()
	entry := &PluginEntry{ID: "remove-me", Name: "RM", Version: "1.0.0", State: StateRegistered}
	r.Register(entry)

	r.Remove("remove-me")

	_, ok := r.Get("remove-me")
	if ok {
		t.Fatal("expected plugin to be removed, but Get() returned true")
	}
}

func TestList(t *testing.T) {
	r := New()
	r.Register(&PluginEntry{ID: "a", Name: "A", Version: "1.0.0", State: StateRegistered})
	r.Register(&PluginEntry{ID: "b", Name: "B", Version: "1.0.0", State: StateRegistered})

	list := r.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(list))
	}
}

func TestFindByCapability(t *testing.T) {
	r := New()
	r.Register(&PluginEntry{
		ID:           "perceiver",
		Name:         "Perceiver",
		Version:      "1.0.0",
		State:        StateRegistered,
		Capabilities: []string{"PERCEPTION", "VISION"},
	})
	r.Register(&PluginEntry{
		ID:           "thinker",
		Name:         "Thinker",
		Version:      "1.0.0",
		State:        StateRegistered,
		Capabilities: []string{"COGNITION"},
	})

	result := r.FindByCapability("PERCEPTION")
	if len(result) != 1 {
		t.Fatalf("expected 1 plugin with PERCEPTION, got %d", len(result))
	}
	if result[0].ID != "perceiver" {
		t.Fatalf("expected perceiver, got %s", result[0].ID)
	}

	none := r.FindByCapability("NONEXISTENT")
	if len(none) != 0 {
		t.Fatalf("expected 0 plugins, got %d", len(none))
	}
}

func TestFindByPipelineSlot(t *testing.T) {
	r := New()
	r.Register(&PluginEntry{
		ID:      "slot-plugin",
		Name:    "SlotPlugin",
		Version: "1.0.0",
		State:   StateRegistered,
		SlotRegistrations: []SlotRegistration{
			{Pipeline: "PERCEPTION_TO_COGNITION", Slot: "perceive", Priority: 1},
		},
	})

	result := r.FindByPipelineSlot("PERCEPTION_TO_COGNITION", "perceive")
	if len(result) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(result))
	}

	none := r.FindByPipelineSlot("WRONG", "slot")
	if len(none) != 0 {
		t.Fatalf("expected 0 plugins, got %d", len(none))
	}
}