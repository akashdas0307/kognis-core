package controlplane

import (
	"testing"

	"github.com/kognis-framework/kognis-core/core/internal/registry"
)

func TestPerformHandshake(t *testing.T) {
	reg := registry.New()
	req := &HandshakeRequest{
		PluginID:     "test-plugin",
		Name:        "TestPlugin",
		Version:     "1.0.0",
		Capabilities: []string{"COGNITION", "PERCEPTION"},
	}

	resp, err := PerformHandshake(req, reg, nil, "/tmp/test.sock")
	if err != nil {
		t.Fatalf("PerformHandshake() failed: %v", err)
	}
	if resp.PluginID != "test-plugin" {
		t.Fatalf("expected plugin_id test-plugin, got %s", resp.PluginID)
	}
	if resp.State != "REGISTERED" {
		t.Fatalf("expected state REGISTERED, got %s", resp.State)
	}

	// Verify plugin is in registry
	entry, ok := reg.Get("test-plugin")
	if !ok {
		t.Fatal("plugin not found in registry after handshake")
	}
	if entry.Name != "TestPlugin" {
		t.Fatalf("expected name TestPlugin, got %s", entry.Name)
	}
}

func TestPerformHandshakeMissingFields(t *testing.T) {
	reg := registry.New()

	tests := []struct {
		name string
		req  *HandshakeRequest
	}{
		{"missing ID", &HandshakeRequest{Name: "N", Version: "1.0.0"}},
		{"missing name", &HandshakeRequest{PluginID: "id", Version: "1.0.0"}},
		{"missing version", &HandshakeRequest{PluginID: "id", Name: "N"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := PerformHandshake(tt.req, reg, nil, "/tmp/test.sock")
			if err == nil {
				t.Fatal("expected error for missing handshake fields, got nil")
			}
		})
	}
}

func TestPerformHandshakeDuplicate(t *testing.T) {
	reg := registry.New()
	req := &HandshakeRequest{
		PluginID: "dup-plugin",
		Name:    "Dup",
		Version: "1.0.0",
	}

	PerformHandshake(req, reg, nil, "/tmp/test.sock")
	_, err := PerformHandshake(req, reg, nil, "/tmp/test.sock")
	if err == nil {
		t.Fatal("expected error for duplicate handshake, got nil")
	}
}