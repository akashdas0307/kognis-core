package controlplane

import (
	"testing"
	"time"

	"github.com/akashdas0307/kognis-core/core/internal/registry"
)

func newTestManager() *HandshakeManager {
	reg := registry.New()
	return NewHandshakeManager(reg, nil, "/tmp/kognis-test.sock")
}

// --- HandshakeManager Constructor ---

func TestNewHandshakeManager(t *testing.T) {
	hm := newTestManager()
	if hm == nil {
		t.Fatal("NewHandshakeManager returned nil")
	}
	if hm.socketPath != "/tmp/kognis-test.sock" {
		t.Fatalf("expected socketPath /tmp/kognis-test.sock, got %s", hm.socketPath)
	}
	if hm.config.EventBusURL != "nats://127.0.0.1:4222" {
		t.Fatalf("expected default EventBusURL nats://127.0.0.1:4222, got %s", hm.config.EventBusURL)
	}
	if len(hm.pending) != 0 {
		t.Fatalf("expected empty pending map, got %d entries", len(hm.pending))
	}
}

func TestSetEventBusURL(t *testing.T) {
	hm := newTestManager()
	hm.SetEventBusURL("nats://custom:5222")
	if hm.config.EventBusURL != "nats://custom:5222" {
		t.Fatalf("expected EventBusURL nats://custom:5222, got %s", hm.config.EventBusURL)
	}
}

// --- Step 1->2: StartHandshake ---

func TestStartHandshakeSuccess(t *testing.T) {
	hm := newTestManager()
	req := &HandshakeRequest{
		PluginID:             "test-plugin",
		Name:                "TestPlugin",
		Version:             "1.0.0",
		Capabilities:        []string{"COGNITION", "PERCEPTION"},
		ManifestHash:        "abc123",
		EmergencyBypassTypes: []string{"health_critical"},
		PID:                 12345,
	}

	resp, err := hm.StartHandshake(req)
	if err != nil {
		t.Fatalf("StartHandshake() failed: %v", err)
	}

	// Verify response fields
	if resp.PluginID != "test-plugin" {
		t.Fatalf("expected plugin_id test-plugin, got %s", resp.PluginID)
	}
	if resp.PluginIDRuntime == "" {
		t.Fatal("expected non-empty plugin_id_runtime")
	}
	if resp.State != "REGISTERED" {
		t.Fatalf("expected state REGISTERED, got %s", resp.State)
	}
	if resp.EventBusURL != "nats://127.0.0.1:4222" {
		t.Fatalf("expected event_bus_url nats://127.0.0.1:4222, got %s", resp.EventBusURL)
	}
	if resp.EventBusToken == "" {
		t.Fatal("expected non-empty event_bus_token")
	}
	if resp.ControlPlane != "/tmp/kognis-test.sock" {
		t.Fatalf("expected control_plane /tmp/kognis-test.sock, got %s", resp.ControlPlane)
	}
	if resp.ConfigBundle == nil {
		t.Fatal("expected non-nil config_bundle")
	}
	if resp.Error != "" {
		t.Fatalf("expected no error in response, got %s", resp.Error)
	}

	// Verify plugin is in registry at step 2
	entry, ok := hm.registry.Get("test-plugin")
	if !ok {
		t.Fatal("plugin not found in registry after StartHandshake")
	}
	if entry.Name != "TestPlugin" {
		t.Fatalf("expected name TestPlugin, got %s", entry.Name)
	}
	if entry.State != registry.StateRegistered {
		t.Fatalf("expected state REGISTERED, got %s", entry.State)
	}
	if entry.EventBusToken == "" {
		t.Fatal("expected EventBusToken to be set in registry entry")
	}
	if entry.ManifestHash != "abc123" {
		t.Fatalf("expected manifest_hash abc123, got %s", entry.ManifestHash)
	}
	if entry.PID != 12345 {
		t.Fatalf("expected pid 12345, got %d", entry.PID)
	}
	if len(entry.EmergencyBypassTypes) != 1 || entry.EmergencyBypassTypes[0] != "health_critical" {
		t.Fatalf("expected emergency_bypass_types [health_critical], got %v", entry.EmergencyBypassTypes)
	}

	// Verify handshake step is 2 (Ack)
	step, err := hm.registry.GetHandshakeStep("test-plugin")
	if err != nil {
		t.Fatalf("GetHandshakeStep failed: %v", err)
	}
	if step != int(StepAck) {
		t.Fatalf("expected handshake step %d (Ack), got %d", StepAck, step)
	}

	// Verify pending handshake tracked
	hm.mu.RLock()
	if _, ok := hm.pending["test-plugin"]; !ok {
		t.Fatal("expected pending handshake entry for test-plugin")
	}
	hm.mu.RUnlock()
}

func TestStartHandshakeMissingPluginID(t *testing.T) {
	hm := newTestManager()
	req := &HandshakeRequest{Name: "N", Version: "1.0.0"}

	_, err := hm.StartHandshake(req)
	if err == nil {
		t.Fatal("expected error for missing plugin_id, got nil")
	}
}

func TestStartHandshakeMissingName(t *testing.T) {
	hm := newTestManager()
	req := &HandshakeRequest{PluginID: "id", Version: "1.0.0"}

	_, err := hm.StartHandshake(req)
	if err == nil {
		t.Fatal("expected error for missing name, got nil")
	}
}

func TestStartHandshakeMissingVersion(t *testing.T) {
	hm := newTestManager()
	req := &HandshakeRequest{PluginID: "id", Name: "N"}

	_, err := hm.StartHandshake(req)
	if err == nil {
		t.Fatal("expected error for missing version, got nil")
	}
}

func TestStartHandshakeDuplicateRegistration(t *testing.T) {
	hm := newTestManager()
	req := &HandshakeRequest{
		PluginID: "dup-plugin",
		Name:    "Dup",
		Version: "1.0.0",
		ManifestHash: "abc",
	}

	_, _ = hm.StartHandshake(req)
	_, err := hm.StartHandshake(req)
	if err != nil {
		t.Fatalf("expected success for idempotent handshake, got error: %v", err)
	}
}

func TestStartHandshakeInvalidEmergencyBypass(t *testing.T) {
	hm := newTestManager()
	req := &HandshakeRequest{
		PluginID:             "bad-bypass",
		Name:                 "BadBypass",
		Version:              "1.0.0",
		EmergencyBypassTypes: []string{"unauthorized_type"},
	}

	resp, err := hm.StartHandshake(req)
	if err == nil {
		t.Fatal("expected error for invalid emergency bypass type, got nil")
	}
	if resp != nil && resp.Error == "" {
		t.Fatal("expected error message in response")
	}
}

func TestStartHandshakeCapabilityConflict(t *testing.T) {
	hm := newTestManager()

	req1 := &HandshakeRequest{
		PluginID:     "plugin-a",
		Name:        "PluginA",
		Version:     "1.0.0",
		Capabilities: []string{"COGNITION"},
	}
	if _, err := hm.StartHandshake(req1); err != nil {
		t.Fatalf("first registration should succeed: %v", err)
	}

	// Complete the first plugin's handshake so it's HEALTHY_ACTIVE
	// and its capability is "available"
	_ = hm.registry.UpdateState("plugin-a", registry.StateStarting)
	_ = hm.registry.UpdateState("plugin-a", registry.StateHealthyActive)

	req2 := &HandshakeRequest{
		PluginID:     "plugin-b",
		Name:        "PluginB",
		Version:     "1.0.0",
		Capabilities: []string{"COGNITION"},
	}
	_, err := hm.StartHandshake(req2)
	if err == nil {
		t.Fatal("expected error for capability conflict, got nil")
	}
}

func TestStartHandshakeRuntimeIDFormat(t *testing.T) {
	hm := newTestManager()
	req := &HandshakeRequest{
		PluginID: "my-plugin",
		Name:    "MyPlugin",
		Version: "2.0.0",
	}

	resp, err := hm.StartHandshake(req)
	if err != nil {
		t.Fatalf("StartHandshake() failed: %v", err)
	}

	// Runtime ID should be "my-plugin-<8-char-hex>"
	expected := "my-plugin-"
	if len(resp.PluginIDRuntime) < len(expected)+8 {
		t.Fatalf("runtime ID %q too short, expected format %s<8hex>", resp.PluginIDRuntime, expected)
	}
	if resp.PluginIDRuntime[:len(expected)] != expected {
		t.Fatalf("runtime ID %q should start with %q", resp.PluginIDRuntime, expected)
	}
}

func TestStartHandshakeTokenUniqueness(t *testing.T) {
	hm := newTestManager()

	req1 := &HandshakeRequest{PluginID: "p1", Name: "P1", Version: "1.0.0"}
	req2 := &HandshakeRequest{PluginID: "p2", Name: "P2", Version: "1.0.0"}

	// Register p1 first, then complete its handshake so p2 can register
	resp1, err := hm.StartHandshake(req1)
	if err != nil {
		t.Fatalf("StartHandshake p1 failed: %v", err)
	}

	// Complete p1 so we can register p2 without capability conflicts
	_ = hm.CompleteHandshake("p1", &ReadyMessage{
		PluginID:         "p1",
		SubscribedTopics: []string{"test"},
		HealthEndpoint:   "kognis.health.p1",
	})

	resp2, err := hm.StartHandshake(req2)
	if err != nil {
		t.Fatalf("StartHandshake p2 failed: %v", err)
	}

	if resp1.EventBusToken == resp2.EventBusToken {
		t.Fatal("event bus tokens should be unique across registrations")
	}
}

func TestStartHandshakeCustomEventBusURL(t *testing.T) {
	hm := newTestManager()
	hm.SetEventBusURL("nats://custom:5222")

	req := &HandshakeRequest{PluginID: "url-test", Name: "URLTest", Version: "1.0.0"}
	resp, err := hm.StartHandshake(req)
	if err != nil {
		t.Fatalf("StartHandshake() failed: %v", err)
	}

	if resp.EventBusURL != "nats://custom:5222" {
		t.Fatalf("expected custom EventBusURL, got %s", resp.EventBusURL)
	}
}

// --- Step 3->4: CompleteHandshake ---

func TestCompleteHandshakeSuccess(t *testing.T) {
	hm := newTestManager()

	// Step 1->2: Register plugin
	req := &HandshakeRequest{
		PluginID:     "complete-test",
		Name:        "CompleteTest",
		Version:     "1.0.0",
		Capabilities: []string{"COGNITION"},
	}
	_, err := hm.StartHandshake(req)
	if err != nil {
		t.Fatalf("StartHandshake failed: %v", err)
	}

	// Step 3->4: Complete handshake
	readyMsg := &ReadyMessage{
		PluginID:         "complete-test",
		SubscribedTopics: []string{"kognis.pipeline.cognition", "kognis.health.complete-test"},
		HealthEndpoint:   "kognis.health.complete-test",
	}
	if err := hm.CompleteHandshake("complete-test", readyMsg); err != nil {
		t.Fatalf("CompleteHandshake failed: %v", err)
	}

	// Verify plugin is HEALTHY_ACTIVE
	entry, ok := hm.registry.Get("complete-test")
	if !ok {
		t.Fatal("plugin not found in registry")
	}
	if entry.State != registry.StateHealthyActive {
		t.Fatalf("expected HEALTHY_ACTIVE, got %s", entry.State)
	}

	// Verify handshake step is 4
	step, err := hm.registry.GetHandshakeStep("complete-test")
	if err != nil {
		t.Fatalf("GetHandshakeStep failed: %v", err)
	}
	if step != int(StepActive) {
		t.Fatalf("expected handshake step %d (Active), got %d", StepActive, step)
	}

	// Verify subscribed topics stored
	if len(entry.SubscribedTopics) != 2 {
		t.Fatalf("expected 2 subscribed topics, got %d", len(entry.SubscribedTopics))
	}

	// Verify heartbeat was recorded
	if entry.LastHeartbeat.IsZero() {
		t.Fatal("expected LastHeartbeat to be set")
	}
	if entry.MissedHeartbeats != 0 {
		t.Fatalf("expected MissedHeartbeats 0, got %d", entry.MissedHeartbeats)
	}

	// Verify removed from pending
	hm.mu.RLock()
	if _, ok := hm.pending["complete-test"]; ok {
		t.Fatal("expected pending entry to be removed after completion")
	}
	hm.mu.RUnlock()
}

func TestCompleteHandshakePluginNotFound(t *testing.T) {
	hm := newTestManager()

	readyMsg := &ReadyMessage{
		PluginID:         "nonexistent",
		SubscribedTopics: []string{"test"},
	}
	err := hm.CompleteHandshake("nonexistent", readyMsg)
	if err == nil {
		t.Fatal("expected error for nonexistent plugin, got nil")
	}
}

func TestCompleteHandshakeWrongStep(t *testing.T) {
	hm := newTestManager()

	// Register and complete handshake for plugin
	req := &HandshakeRequest{
		PluginID:     "wrong-step",
		Name:        "WrongStep",
		Version:     "1.0.0",
		Capabilities: []string{"MEMORY"},
	}
	_, err := hm.StartHandshake(req)
	if err != nil {
		t.Fatalf("StartHandshake failed: %v", err)
	}

	readyMsg := &ReadyMessage{
		PluginID:         "wrong-step",
		SubscribedTopics: []string{"test"},
	}
	if err := hm.CompleteHandshake("wrong-step", readyMsg); err != nil {
		t.Fatalf("first CompleteHandshake failed: %v", err)
	}

	// Try to complete again — plugin is now at step 4, this should be idempotent and succeed
	err = hm.CompleteHandshake("wrong-step", readyMsg)
	if err != nil {
		t.Fatalf("expected success for idempotent CompleteHandshake call, got error: %v", err)
	}
}

func TestCompleteHandshakePluginIDMismatch(t *testing.T) {
	hm := newTestManager()

	req := &HandshakeRequest{
		PluginID:     "real-id",
		Name:        "RealID",
		Version:     "1.0.0",
		Capabilities: []string{"COGNITION"},
	}
	_, err := hm.StartHandshake(req)
	if err != nil {
		t.Fatalf("StartHandshake failed: %v", err)
	}

	readyMsg := &ReadyMessage{
		PluginID:         "wrong-id",
		SubscribedTopics: []string{"test"},
	}
	err = hm.CompleteHandshake("real-id", readyMsg)
	if err == nil {
		t.Fatal("expected error for plugin ID mismatch, got nil")
	}
}

func TestCompleteHandshakeRecordsHeartbeat(t *testing.T) {
	hm := newTestManager()

	req := &HandshakeRequest{
		PluginID:     "hb-test",
		Name:        "HBTest",
		Version:     "1.0.0",
		Capabilities: []string{"PERCEPTION"},
	}
	_, _ = hm.StartHandshake(req)

	before := time.Now()
	_ = hm.CompleteHandshake("hb-test", &ReadyMessage{
		PluginID:         "hb-test",
		SubscribedTopics: []string{"kognis.perception"},
	})
	after := time.Now()

	entry, _ := hm.registry.Get("hb-test")
	if entry.LastHeartbeat.Before(before) || entry.LastHeartbeat.After(after) {
		t.Fatalf("LastHeartbeat %v not between %v and %v", entry.LastHeartbeat, before, after)
	}
}

// --- Full 4-step handshake flow ---

func TestFullHandshakeFlow(t *testing.T) {
	hm := newTestManager()

	// Step 1: Plugin sends REGISTER_REQUEST
	req := &HandshakeRequest{
		PluginID:             "full-flow",
		Name:                 "FullFlow",
		Version:              "1.0.0",
		Capabilities:         []string{"COGNITION"},
		ManifestHash:         "sha256:abcdef",
		EmergencyBypassTypes: []string{"creator_emergency"},
		PID:                  9999,
	}

	// Step 1->2: Core processes registration
	resp, err := hm.StartHandshake(req)
	if err != nil {
		t.Fatalf("StartHandshake failed: %v", err)
	}

	// Verify step 2 response
	if resp.PluginIDRuntime == "" {
		t.Fatal("expected runtime ID in step 2 response")
	}
	if resp.EventBusToken == "" {
		t.Fatal("expected event bus token in step 2 response")
	}
	if resp.EventBusURL == "" {
		t.Fatal("expected event bus URL in step 2 response")
	}
	if resp.ControlPlane == "" {
		t.Fatal("expected control plane in step 2 response")
	}

	// Verify plugin at step 2
	step, _ := hm.registry.GetHandshakeStep("full-flow")
	if step != int(StepAck) {
		t.Fatalf("expected step %d after StartHandshake, got %d", StepAck, step)
	}

	// Step 3: Plugin connects to NATS and sends READY
	readyMsg := &ReadyMessage{
		PluginID:         "full-flow",
		SubscribedTopics: []string{"kognis.pipeline.cognition", "kognis.health.full-flow"},
		HealthEndpoint:   "kognis.health.full-flow",
	}

	// Step 3->4: Core marks HEALTHY_ACTIVE
	if err := hm.CompleteHandshake("full-flow", readyMsg); err != nil {
		t.Fatalf("CompleteHandshake failed: %v", err)
	}

	// Verify final state
	entry, _ := hm.registry.Get("full-flow")
	if entry.State != registry.StateHealthyActive {
		t.Fatalf("expected HEALTHY_ACTIVE, got %s", entry.State)
	}
	if entry.HandshakeStep != int(StepActive) {
		t.Fatalf("expected handshake step %d, got %d", StepActive, entry.HandshakeStep)
	}
	if entry.PID != 9999 {
		t.Fatalf("expected PID 9999, got %d", entry.PID)
	}
	if entry.ManifestHash != "sha256:abcdef" {
		t.Fatalf("expected ManifestHash sha256:abcdef, got %s", entry.ManifestHash)
	}
}

// --- CheckTimeouts ---

func TestCheckTimeoutsNoTimeout(t *testing.T) {
	hm := newTestManager()

	req := &HandshakeRequest{PluginID: "no-timeout", Name: "NoTimeout", Version: "1.0.0"}
	_, _ = hm.StartHandshake(req)

	// Immediately check — should not time out
	hm.CheckTimeouts()

	// Plugin should still be in pending
	hm.mu.RLock()
	if _, ok := hm.pending["no-timeout"]; !ok {
		t.Fatal("plugin should still be pending (not timed out yet)")
	}
	hm.mu.RUnlock()

	// Plugin should still be registered, not UNRESPONSIVE
	entry, _ := hm.registry.Get("no-timeout")
	if entry.State == registry.StateUnresponsive {
		t.Fatal("plugin should not be UNRESPONSIVE (timeout not reached)")
	}
}

func TestCheckTimeoutsStep2to3Expired(t *testing.T) {
	hm := newTestManager()

	req := &HandshakeRequest{PluginID: "timeout-2-3", Name: "Timeout23", Version: "1.0.0"}
	_, _ = hm.StartHandshake(req)

	// Artificially age the pending handshake past the Step2to3Timeout (10s)
	hm.mu.Lock()
	if p, ok := hm.pending["timeout-2-3"]; ok {
		p.startedAt = time.Now().Add(-Step2to3Timeout - time.Second)
	}
	hm.mu.Unlock()

	hm.CheckTimeouts()

	// Plugin should be UNRESPONSIVE
	entry, _ := hm.registry.Get("timeout-2-3")
	if entry.State != registry.StateUnresponsive {
		t.Fatalf("expected UNRESPONSIVE after step 2->3 timeout, got %s", entry.State)
	}

	// Plugin should be removed from pending
	hm.mu.RLock()
	if _, ok := hm.pending["timeout-2-3"]; ok {
		t.Fatal("plugin should be removed from pending after timeout")
	}
	hm.mu.RUnlock()
}

func TestCheckTimeoutsStep1to2Expired(t *testing.T) {
	hm := newTestManager()

	req := &HandshakeRequest{PluginID: "timeout-1-2", Name: "Timeout12", Version: "1.0.0"}
	_, _ = hm.StartHandshake(req)

	// Simulate step 1 timeout (plugin stuck at REGISTER step, not yet ACK)
	// This simulates a scenario where the core hasn't even processed the request yet
	hm.mu.Lock()
	if p, ok := hm.pending["timeout-1-2"]; ok {
		p.step = StepRegister
		p.startedAt = time.Now().Add(-Step1to2Timeout - time.Second)
	}
	hm.mu.Unlock()

	hm.CheckTimeouts()

	entry, _ := hm.registry.Get("timeout-1-2")
	if entry.State != registry.StateUnresponsive {
		t.Fatalf("expected UNRESPONSIVE after step 1->2 timeout, got %s", entry.State)
	}
}

func TestCheckTimeoutsMultiplePlugins(t *testing.T) {
	hm := newTestManager()

	// Register two plugins
	req1 := &HandshakeRequest{PluginID: "fast-plugin", Name: "Fast", Version: "1.0.0"}
	req2 := &HandshakeRequest{PluginID: "slow-plugin", Name: "Slow", Version: "1.0.0"}
	_, _ = hm.StartHandshake(req1)
	_, _ = hm.StartHandshake(req2)

	// Age only slow-plugin past timeout
	hm.mu.Lock()
	if p, ok := hm.pending["slow-plugin"]; ok {
		p.startedAt = time.Now().Add(-Step2to3Timeout - time.Second)
	}
	hm.mu.Unlock()

	hm.CheckTimeouts()

	// fast-plugin should still be pending
	hm.mu.RLock()
	if _, ok := hm.pending["fast-plugin"]; !ok {
		t.Fatal("fast-plugin should still be pending")
	}
	// slow-plugin should be timed out
	if _, ok := hm.pending["slow-plugin"]; ok {
		t.Fatal("slow-plugin should be removed from pending after timeout")
	}
	hm.mu.RUnlock()

	// Verify states
	fastEntry, _ := hm.registry.Get("fast-plugin")
	if fastEntry.State == registry.StateUnresponsive {
		t.Fatal("fast-plugin should not be UNRESPONSIVE")
	}
	slowEntry, _ := hm.registry.Get("slow-plugin")
	if slowEntry.State != registry.StateUnresponsive {
		t.Fatalf("slow-plugin should be UNRESPONSIVE, got %s", slowEntry.State)
	}
}

func TestCheckTimeoutsClearedAfterCompletion(t *testing.T) {
	hm := newTestManager()

	req := &HandshakeRequest{PluginID: "completed", Name: "Completed", Version: "1.0.0"}
	_, _ = hm.StartHandshake(req)

	// Complete the handshake
	_ = hm.CompleteHandshake("completed", &ReadyMessage{
		PluginID:         "completed",
		SubscribedTopics: []string{"test"},
	})

	// Age the time beyond timeout
	hm.CheckTimeouts()

	// Plugin should still be HEALTHY_ACTIVE (not affected by timeout check)
	entry, _ := hm.registry.Get("completed")
	if entry.State != registry.StateHealthyActive {
		t.Fatalf("completed plugin should remain HEALTHY_ACTIVE, got %s", entry.State)
	}
}

// --- Graceful Shutdown (SPEC 04 Section 4.3) ---

func TestInitiateShutdownSuccess(t *testing.T) {
	hm := newTestManager()

	// Register and complete handshake
	req := &HandshakeRequest{PluginID: "shutdown-test", Name: "Shutdown", Version: "1.0.0"}
	_, _ = hm.StartHandshake(req)
	_ = hm.CompleteHandshake("shutdown-test", &ReadyMessage{
		PluginID:         "shutdown-test",
		SubscribedTopics: []string{"test"},
	})

	// Initiate shutdown
	err := hm.InitiateShutdown("shutdown-test", 30*time.Second)
	if err != nil {
		t.Fatalf("InitiateShutdown failed: %v", err)
	}

	// Verify plugin is in SHUTTING_DOWN state
	entry, _ := hm.registry.Get("shutdown-test")
	if entry.State != registry.StateShuttingDown {
		t.Fatalf("expected SHUTTING_DOWN, got %s", entry.State)
	}
	if entry.ShutdownRequestedAt.IsZero() {
		t.Fatal("expected ShutdownRequestedAt to be set")
	}
}

func TestInitiateShutdownPluginNotFound(t *testing.T) {
	hm := newTestManager()

	err := hm.InitiateShutdown("nonexistent", 10*time.Second)
	if err == nil {
		t.Fatal("expected error for nonexistent plugin, got nil")
	}
}

func TestInitiateShutdownInvalidState(t *testing.T) {
	hm := newTestManager()

	// Register plugin but don't complete handshake — stays at REGISTERED state
	req := &HandshakeRequest{PluginID: "not-ready", Name: "NotReady", Version: "1.0.0"}
	_, _ = hm.StartHandshake(req)

	// Try to shut down a plugin that's only REGISTERED (not HEALTHY_ACTIVE)
	err := hm.InitiateShutdown("not-ready", 10*time.Second)
	if err == nil {
		t.Fatal("expected error for shutting down REGISTERED plugin, got nil")
	}
}

func TestConfirmShutdownSuccess(t *testing.T) {
	hm := newTestManager()

	// Full lifecycle: register -> complete -> shutdown
	req := &HandshakeRequest{PluginID: "confirm-shutdown", Name: "ConfirmShutdown", Version: "1.0.0"}
	_, _ = hm.StartHandshake(req)
	_ = hm.CompleteHandshake("confirm-shutdown", &ReadyMessage{
		PluginID:         "confirm-shutdown",
		SubscribedTopics: []string{"test"},
	})
	_ = hm.InitiateShutdown("confirm-shutdown", 30*time.Second)

	// Confirm shutdown
	err := hm.ConfirmShutdown("confirm-shutdown")
	if err != nil {
		t.Fatalf("ConfirmShutdown failed: %v", err)
	}

	// Verify plugin is SHUT_DOWN
	entry, _ := hm.registry.Get("confirm-shutdown")
	if entry.State != registry.StateShutDown {
		t.Fatalf("expected SHUT_DOWN, got %s", entry.State)
	}
}

func TestConfirmShutdownNotShuttingDown(t *testing.T) {
	hm := newTestManager()

	// Register and complete, but don't initiate shutdown
	req := &HandshakeRequest{PluginID: "not-shutting", Name: "NotShutting", Version: "1.0.0"}
	_, _ = hm.StartHandshake(req)
	_ = hm.CompleteHandshake("not-shutting", &ReadyMessage{
		PluginID:         "not-shutting",
		SubscribedTopics: []string{"test"},
	})

	err := hm.ConfirmShutdown("not-shutting")
	if err == nil {
		t.Fatal("expected error for confirming shutdown on HEALTHY_ACTIVE plugin, got nil")
	}
}

func TestConfirmShutdownPluginNotFound(t *testing.T) {
	hm := newTestManager()

	err := hm.ConfirmShutdown("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent plugin, got nil")
	}
}

func TestFullShutdownFlow(t *testing.T) {
	hm := newTestManager()

	// Full lifecycle: register -> complete -> shutdown -> confirm
	req := &HandshakeRequest{PluginID: "full-shutdown", Name: "FullShutdown", Version: "1.0.0"}
	_, _ = hm.StartHandshake(req)
	_ = hm.CompleteHandshake("full-shutdown", &ReadyMessage{
		PluginID:         "full-shutdown",
		SubscribedTopics: []string{"test"},
	})

	// Verify at HEALTHY_ACTIVE
	entry, _ := hm.registry.Get("full-shutdown")
	if entry.State != registry.StateHealthyActive {
		t.Fatalf("expected HEALTHY_ACTIVE, got %s", entry.State)
	}

	// Initiate shutdown (Step 1: SHUTDOWN_REQUEST)
	_ = hm.InitiateShutdown("full-shutdown", 15*time.Second)
	entry, _ = hm.registry.Get("full-shutdown")
	if entry.State != registry.StateShuttingDown {
		t.Fatalf("expected SHUTTING_DOWN after InitiateShutdown, got %s", entry.State)
	}

	// Confirm shutdown (Step 3: CONFIRMED, Step 4: plugin.left)
	_ = hm.ConfirmShutdown("full-shutdown")
	entry, _ = hm.registry.Get("full-shutdown")
	if entry.State != registry.StateShutDown {
		t.Fatalf("expected SHUT_DOWN after ConfirmShutdown, got %s", entry.State)
	}
}

// --- HandshakeStep constants ---

func TestHandshakeStepConstants(t *testing.T) {
	if StepRegister != 1 {
		t.Fatalf("expected StepRegister=1, got %d", StepRegister)
	}
	if StepAck != 2 {
		t.Fatalf("expected StepAck=2, got %d", StepAck)
	}
	if StepReady != 3 {
		t.Fatalf("expected StepReady=3, got %d", StepReady)
	}
	if StepActive != 4 {
		t.Fatalf("expected StepActive=4, got %d", StepActive)
	}
}

func TestTimeoutConstants(t *testing.T) {
	if Step1to2Timeout != 5*time.Second {
		t.Fatalf("expected Step1to2Timeout=5s, got %v", Step1to2Timeout)
	}
	if Step2to3Timeout != 10*time.Second {
		t.Fatalf("expected Step2to3Timeout=10s, got %v", Step2to3Timeout)
	}
	if Step3to4Timeout != 2*time.Second {
		t.Fatalf("expected Step3to4Timeout=2s, got %v", Step3to4Timeout)
	}
}

// --- Registry AdvanceHandshake edge cases ---

func TestAdvanceHandshakeBeyondStep4(t *testing.T) {
	hm := newTestManager()

	req := &HandshakeRequest{PluginID: "advance-beyond", Name: "Advance", Version: "1.0.0"}
	_, _ = hm.StartHandshake(req)   // step 1->2
	_ = hm.CompleteHandshake("advance-beyond", &ReadyMessage{
		PluginID:         "advance-beyond",
		SubscribedTopics: []string{"test"},
	}) // step 2->3->4

	// Try to advance beyond step 4
	err := hm.registry.AdvanceHandshake("advance-beyond")
	if err == nil {
		t.Fatal("expected error advancing beyond step 4, got nil")
	}
}

// --- EmergencyBypass validation during handshake ---

func TestStartHandshakeValidEmergencyBypass(t *testing.T) {
	hm := newTestManager()

	req := &HandshakeRequest{
		PluginID:             "valid-bypass",
		Name:                 "ValidBypass",
		Version:              "1.0.0",
		EmergencyBypassTypes: []string{"safety_sound_detected", "physical_hazard"},
	}
	_, err := hm.StartHandshake(req)
	if err != nil {
		t.Fatalf("StartHandshake with valid bypass types should succeed: %v", err)
	}
}

func TestStartHandshakeAllValidBypassTypes(t *testing.T) {
	validTypes := []string{"safety_sound_detected", "health_critical", "creator_emergency", "physical_hazard"}

	for _, bt := range validTypes {
		hm := newTestManager()
		req := &HandshakeRequest{
			PluginID:             "bypass-" + bt,
			Name:                 "Bypass" + bt,
			Version:              "1.0.0",
			EmergencyBypassTypes: []string{bt},
		}
		_, err := hm.StartHandshake(req)
		if err != nil {
			t.Fatalf("StartHandshake with valid bypass type %q should succeed: %v", bt, err)
		}
	}
}

// --- Multiple plugins handshake flow ---

func TestMultiplePluginsFullFlow(t *testing.T) {
	hm := newTestManager()

	plugins := []struct {
		id   string
		name string
		cap  string
	}{
		{"cog-plugin", "Cognition", "COGNITION"},
		{"per-plugin", "Perception", "PERCEPTION"},
		{"mem-plugin", "Memory", "MEMORY"},
	}

	// Register all plugins
	for _, p := range plugins {
		req := &HandshakeRequest{
			PluginID:     p.id,
			Name:         p.name,
			Version:      "1.0.0",
			Capabilities: []string{p.cap},
		}
		if _, err := hm.StartHandshake(req); err != nil {
			t.Fatalf("StartHandshake %s failed: %v", p.id, err)
		}
	}

	// Complete all handshakes
	for _, p := range plugins {
		readyMsg := &ReadyMessage{
			PluginID:         p.id,
			SubscribedTopics: []string{"kognis.pipeline." + p.cap},
		}
		if err := hm.CompleteHandshake(p.id, readyMsg); err != nil {
			t.Fatalf("CompleteHandshake %s failed: %v", p.id, err)
		}
	}

	// Verify all are HEALTHY_ACTIVE
	for _, p := range plugins {
		entry, ok := hm.registry.Get(p.id)
		if !ok {
			t.Fatalf("plugin %s not found", p.id)
		}
		if entry.State != registry.StateHealthyActive {
			t.Fatalf("plugin %s expected HEALTHY_ACTIVE, got %s", p.id, entry.State)
		}
		if entry.HandshakeStep != int(StepActive) {
			t.Fatalf("plugin %s expected step 4, got %d", p.id, entry.HandshakeStep)
		}
	}

	// Verify all capabilities are available
	available := hm.registry.FindAvailableCapabilities()
	if len(available) != 3 {
		t.Fatalf("expected 3 available capabilities, got %d: %v", len(available), available)
	}
}