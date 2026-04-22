package bypass

import (
	"encoding/json"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/akashdas0307/kognis-core/core/internal/config"
	"github.com/akashdas0307/kognis-core/core/internal/eventbus"
	"github.com/akashdas0307/kognis-core/core/internal/registry"
)

// testPortCounter ensures each test bus gets a unique port to avoid conflicts.
var testPortCounter atomic.Int64

// findFreePort returns a currently available TCP port number.
func findFreePort() int {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

// newTestBus creates an eventbus.Bus for testing with a unique port.
func newTestBus(t *testing.T) *eventbus.Bus {
	t.Helper()
	port := findFreePort()
	cfg := config.NATSConfig{
		ServerName: "test-bypass-bus",
		Port:       port,
		DataDir:    t.TempDir(),
	}
	bus, err := eventbus.New(cfg)
	if err != nil {
		t.Fatalf("create eventbus: %v", err)
	}
	t.Cleanup(func() { bus.Close() })
	return bus
}

// newTestRegistry creates a registry with a pre-registered plugin that has
// the given emergency bypass types.
func newTestRegistry(bypassTypes []string) *registry.Registry {
	reg := registry.New()
	reg.Register(&registry.PluginEntry{
		ID:                  "test-plugin",
		Name:                "Test Plugin",
		Version:             "1.0.0",
		State:               registry.StateHealthyActive,
		Capabilities:        []string{"test-cap"},
		EmergencyBypassTypes: bypassTypes,
	})
	return reg
}

func TestHandleBypass_AuthorizedPlugin(t *testing.T) {
	reg := newTestRegistry([]string{"safety_sound_detected"})
	bus := newTestBus(t)
	ch := NewChannel(reg, bus)

	req := BypassRequest{
		PluginID:   "test-plugin",
		BypassType: BypassTypeSafetySoundDetected,
		Payload:    map[string]string{"sound": "fire_alarm"},
		Timestamp:  time.Now(),
	}

	resp, err := ch.HandleBypass(req)
	if err != nil {
		t.Fatalf("HandleBypass returned error: %v", err)
	}
	if !resp.Accepted {
		t.Errorf("expected accepted=true, got false; reason=%s", resp.Reason)
	}
}

func TestHandleBypass_UnauthorizedPlugin(t *testing.T) {
	// Plugin is registered but not authorized for health_critical
	reg := newTestRegistry([]string{"safety_sound_detected"})
	bus := newTestBus(t)
	ch := NewChannel(reg, bus)

	req := BypassRequest{
		PluginID:   "test-plugin",
		BypassType: BypassTypeHealthCritical,
		Payload:    nil,
		Timestamp:  time.Now(),
	}

	resp, err := ch.HandleBypass(req)
	if err != nil {
		t.Fatalf("HandleBypass returned error: %v", err)
	}
	if resp.Accepted {
		t.Error("expected accepted=false for unauthorized bypass type, got true")
	}
	if resp.Reason == "" {
		t.Error("expected non-empty reason for rejected bypass")
	}
}

func TestHandleBypass_InvalidBypassType(t *testing.T) {
	reg := newTestRegistry([]string{"safety_sound_detected"})
	bus := newTestBus(t)
	ch := NewChannel(reg, bus)

	req := BypassRequest{
		PluginID:   "test-plugin",
		BypassType: "invalid_type",
		Payload:    nil,
		Timestamp:  time.Now(),
	}

	resp, err := ch.HandleBypass(req)
	if err != nil {
		t.Fatalf("HandleBypass returned error: %v", err)
	}
	if resp.Accepted {
		t.Error("expected accepted=false for invalid bypass type, got true")
	}
}

func TestHandleBypass_NonexistentPlugin(t *testing.T) {
	reg := newTestRegistry([]string{"safety_sound_detected"})
	bus := newTestBus(t)
	ch := NewChannel(reg, bus)

	req := BypassRequest{
		PluginID:   "nonexistent-plugin",
		BypassType: BypassTypeSafetySoundDetected,
		Payload:    nil,
		Timestamp:  time.Now(),
	}

	resp, err := ch.HandleBypass(req)
	if err != nil {
		t.Fatalf("HandleBypass returned error: %v", err)
	}
	if resp.Accepted {
		t.Error("expected accepted=false for nonexistent plugin, got true")
	}
}

func TestRegisterHandler(t *testing.T) {
	reg := newTestRegistry(nil)
	bus := newTestBus(t)
	ch := NewChannel(reg, bus)

	err := ch.RegisterHandler(BypassTypeSafetySoundDetected, func(req BypassRequest) {})
	if err != nil {
		t.Fatalf("RegisterHandler returned error: %v", err)
	}

	// Verify invalid type is rejected
	err = ch.RegisterHandler("not_a_real_type", func(req BypassRequest) {})
	if err == nil {
		t.Error("expected error when registering handler for invalid bypass type")
	}
}

func TestHandleBypass_DispatchesToHandler(t *testing.T) {
	reg := newTestRegistry([]string{"creator_emergency"})
	bus := newTestBus(t)
	ch := NewChannel(reg, bus)

	var mu sync.Mutex
	var received BypassRequest
	ch.RegisterHandler(BypassTypeCreatorEmergency, func(req BypassRequest) {
		mu.Lock()
		received = req
		mu.Unlock()
	})

	req := BypassRequest{
		PluginID:   "test-plugin",
		BypassType: BypassTypeCreatorEmergency,
		Payload:    "urgent_message",
		Timestamp:  time.Now(),
	}

	resp, err := ch.HandleBypass(req)
	if err != nil {
		t.Fatalf("HandleBypass returned error: %v", err)
	}
	if !resp.Accepted {
		t.Fatalf("expected accepted=true, got false; reason=%s", resp.Reason)
	}

	mu.Lock()
	if received.PluginID != "test-plugin" {
		t.Errorf("handler received plugin_id=%q, want %q", received.PluginID, "test-plugin")
	}
	if received.BypassType != BypassTypeCreatorEmergency {
		t.Errorf("handler received bypass_type=%q, want %q", received.BypassType, BypassTypeCreatorEmergency)
	}
	mu.Unlock()
}

func TestHandleBypass_PublishesEvent(t *testing.T) {
	reg := newTestRegistry([]string{"physical_hazard"})
	bus := newTestBus(t)
	ch := NewChannel(reg, bus)

	// Subscribe to the emergency bypass subject to capture the published event
	var captured []byte
	capturedCh := make(chan struct{}, 1)
	_, err := bus.Subscribe(EmergencyBypassSubject, func(msg *nats.Msg) {
		captured = msg.Data
		select {
		case capturedCh <- struct{}{}:
		default:
		}
	})
	if err != nil {
		t.Fatalf("subscribe to emergency subject: %v", err)
	}

	req := BypassRequest{
		PluginID:   "test-plugin",
		BypassType: BypassTypePhysicalHazard,
		Payload:    "smoke_detected",
		Timestamp:  time.Now(),
	}

	resp, err := ch.HandleBypass(req)
	if err != nil {
		t.Fatalf("HandleBypass returned error: %v", err)
	}
	if !resp.Accepted {
		t.Fatalf("expected accepted=true, got false; reason=%s", resp.Reason)
	}

	// Wait for the event to be published
	select {
	case <-capturedCh:
		var published BypassRequest
		if err := json.Unmarshal(captured, &published); err != nil {
			t.Fatalf("unmarshal published event: %v", err)
		}
		if published.PluginID != "test-plugin" {
			t.Errorf("published event plugin_id=%q, want %q", published.PluginID, "test-plugin")
		}
		if published.BypassType != BypassTypePhysicalHazard {
			t.Errorf("published event bypass_type=%q, want %q", published.BypassType, BypassTypePhysicalHazard)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for published event on kognis.emergency.bypass")
	}
}

func TestAllValidBypassTypes(t *testing.T) {
	types := []string{
		BypassTypeSafetySoundDetected,
		BypassTypeHealthCritical,
		BypassTypeCreatorEmergency,
		BypassTypePhysicalHazard,
	}

	for _, bt := range types {
		if !validBypassTypes[bt] {
			t.Errorf("bypass type %q should be valid", bt)
		}
	}

	// Verify an invalid type is not valid
	if validBypassTypes["not_valid"] {
		t.Error("invalid bypass type should not be in valid set")
	}
}

func TestStart_SubscribesToSubject(t *testing.T) {
	reg := newTestRegistry([]string{"safety_sound_detected", "health_critical", "creator_emergency", "physical_hazard"})
	bus := newTestBus(t)
	ch := NewChannel(reg, bus)

	if err := ch.Start(); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	// Publish a bypass request on the request subject and verify it's handled
	respCh := make(chan *BypassResponse, 1)

	// Subscribe to a reply subject to get the response
	replySubject := "kognis.emergency.bypass.reply.test"
	_, err := bus.Subscribe(replySubject, func(msg *nats.Msg) {
		var resp BypassResponse
		if err := json.Unmarshal(msg.Data, &resp); err != nil {
			return
		}
		select {
		case respCh <- &resp:
		default:
		}
	})
	if err != nil {
		t.Fatalf("subscribe to reply subject: %v", err)
	}

	req := BypassRequest{
		PluginID:   "test-plugin",
		BypassType: BypassTypeHealthCritical,
		Payload:    "low_battery",
		Timestamp:  time.Now(),
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	// Publish with reply subject so the Channel handler can respond
	if err := bus.Conn().PublishRequest(EmergencyBypassRequestSubject, replySubject, data); err != nil {
		t.Fatalf("publish request: %v", err)
	}

	// Wait for response
	select {
	case resp := <-respCh:
		if !resp.Accepted {
			t.Errorf("expected accepted=true via NATS flow, got false; reason=%s", resp.Reason)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for bypass response via NATS")
	}
}