package controlplane

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/kognis-framework/kognis-core/core/internal/eventbus"
	"github.com/kognis-framework/kognis-core/core/internal/registry"
)

// HandshakeRequest is the initial message from a plugin during registration.
type HandshakeRequest struct {
	PluginID     string   `json:"plugin_id"`
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Capabilities []string `json:"capabilities"`
}

// HandshakeResponse is the core daemon's response to a plugin handshake.
type HandshakeResponse struct {
	PluginID     string `json:"plugin_id"`
	State        string `json:"state"`
	EventBusURL  string `json:"event_bus_url"`
	ControlPlane string `json:"control_plane"`
	Error        string `json:"error,omitempty"`
}

// PerformHandshake executes the single handshake protocol with a connecting plugin.
func PerformHandshake(req *HandshakeRequest, reg *registry.Registry, bus *eventbus.Bus, socketPath string) (*HandshakeResponse, error) {
	if req.PluginID == "" || req.Name == "" || req.Version == "" {
		return nil, fmt.Errorf("handshake requires plugin_id, name, and version")
	}

	entry := &registry.PluginEntry{
		ID:           req.PluginID,
		Name:         req.Name,
		Version:      req.Version,
		State:        registry.StateRegistered,
		Capabilities: req.Capabilities,
	}

	if err := reg.Register(entry); err != nil {
		return &HandshakeResponse{
			PluginID: req.PluginID,
			Error:    err.Error(),
		}, err
	}

	log.Printf("handshake: plugin %s registered successfully", req.PluginID)

	// Publish registration event
	event, _ := json.Marshal(map[string]string{
		"plugin_id": req.PluginID,
		"state":     "REGISTERED",
	})
	bus.Publish("kognis.plugin.register", event)

	return &HandshakeResponse{
		PluginID:     req.PluginID,
		State:        "REGISTERED",
		EventBusURL:  "nats://127.0.0.1:4222",
		ControlPlane: socketPath,
	}, nil
}