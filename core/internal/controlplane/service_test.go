package controlplane

import (
	"context"
	"testing"
	"time"

	"github.com/akashdas0307/kognis-core/core/internal/registry"
)

// newTestService creates a ControlPlaneService with a real registry and handshake manager.
// The event bus and health aggregator are nil (not needed for most tests).
func newTestService() (*ControlPlaneService, *registry.Registry, *HandshakeManager) {
	reg := registry.New()
	hm := NewHandshakeManager(reg, nil, "/tmp/kognis-test.sock")
	svc := NewControlPlaneService(reg, nil, nil, hm)
	return svc, reg, hm
}

// newTestServiceNoHandshake creates a ControlPlaneService without a handshake manager.
func newTestServiceNoHandshake() (*ControlPlaneService, *registry.Registry) {
	reg := registry.New()
	svc := NewControlPlaneService(reg, nil, nil, nil)
	return svc, reg
}

// registerTestPlugin is a helper that registers a plugin and completes its handshake
// so it reaches HEALTHY_ACTIVE state.
func registerTestPlugin(hm *HandshakeManager, pluginID, name, version string, caps []string) error {
	_, err := hm.StartHandshake(&HandshakeRequest{
		PluginID:     pluginID,
		Name:        name,
		Version:     version,
		Capabilities: caps,
	})
	if err != nil {
		return err
	}
	return hm.CompleteHandshake(pluginID, &ReadyMessage{
		PluginID:         pluginID,
		SubscribedTopics: []string{"test"},
	})
}

// --- Constructor Tests ---

func TestNewControlPlaneService(t *testing.T) {
	svc, _, _ := newTestService()
	if svc == nil {
		t.Fatal("NewControlPlaneService returned nil")
	}
	if svc.registry == nil {
		t.Fatal("expected registry to be set")
	}
	if svc.handshake == nil {
		t.Fatal("expected handshake manager to be set")
	}
}

func TestNewControlPlaneServiceNoHandshake(t *testing.T) {
	svc, _ := newTestServiceNoHandshake()
	if svc == nil {
		t.Fatal("NewControlPlaneService returned nil")
	}
	if svc.handshake != nil {
		t.Fatal("expected handshake manager to be nil")
	}
}

// --- Register Tests ---

func TestServiceRegisterSuccess(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()

	resp, err := svc.Register(ctx, &RegisterRequest{
		PluginId:    "test-plugin",
		Name:        "TestPlugin",
		Version:     "1.0.0",
		Capabilities: []string{"COGNITION"},
	})
	if err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	if resp.PluginId != "test-plugin" {
		t.Fatalf("expected plugin_id test-plugin, got %s", resp.PluginId)
	}
	if resp.PluginIdRuntime == "" {
		t.Fatal("expected non-empty plugin_id_runtime")
	}
	if resp.State != "REGISTERED" {
		t.Fatalf("expected state REGISTERED, got %s", resp.State)
	}
	if resp.EventBusUrl == "" {
		t.Fatal("expected non-empty event_bus_url")
	}
	if resp.EventBusToken == "" {
		t.Fatal("expected non-empty event_bus_token")
	}
	if resp.ControlPlane == "" {
		t.Fatal("expected non-empty control_plane")
	}
	if resp.Error != "" {
		t.Fatalf("expected no error, got %s", resp.Error)
	}
}

func TestServiceRegisterMissingFields(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()

	// Missing PluginId — StartHandshake returns nil response for validation errors,
	// so the gRPC service returns a gRPC error.
	_, err := svc.Register(ctx, &RegisterRequest{
		Name:    "Test",
		Version: "1.0.0",
	})
	if err == nil {
		t.Fatal("expected error for missing plugin_id")
	}
}

func TestServiceRegisterDuplicatePlugin(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()

	// First registration
	_, _ = svc.Register(ctx, &RegisterRequest{
		PluginId: "dup-plugin",
		Name:     "Dup",
		Version:  "1.0.0",
	})

	// Duplicate registration
	resp, err := svc.Register(ctx, &RegisterRequest{
		PluginId: "dup-plugin",
		Name:     "Dup",
		Version:  "1.0.0",
	})
	if err != nil {
		t.Fatalf("expected response with error field, not gRPC error: %v", err)
	}
	if resp.Error == "" {
		t.Fatal("expected error field in response for duplicate registration")
	}
}

func TestServiceRegisterNoHandshakeManager(t *testing.T) {
	svc, _ := newTestServiceNoHandshake()
	ctx := context.Background()

	_, err := svc.Register(ctx, &RegisterRequest{
		PluginId: "no-hm",
		Name:    "NoHM",
		Version: "1.0.0",
	})
	if err == nil {
		t.Fatal("expected error when handshake manager is nil")
	}
}

// --- HealthCheck Tests ---

func TestServiceHealthCheckSuccess(t *testing.T) {
	svc, _, hm := newTestService()
	ctx := context.Background()

	// Register and complete handshake for a plugin
	_ = registerTestPlugin(hm, "health-test", "HealthTest", "1.0.0", []string{"COGNITION"})

	resp, err := svc.HealthCheck(ctx, &HealthCheckRequest{PluginId: "health-test"})
	if err != nil {
		t.Fatalf("HealthCheck() failed: %v", err)
	}

	if resp.State != "HEALTHY_ACTIVE" {
		t.Fatalf("expected state HEALTHY_ACTIVE, got %s", resp.State)
	}
	if resp.Status != "HEALTHY" {
		t.Fatalf("expected status HEALTHY (derived from registry state), got %s", resp.Status)
	}
}

func TestServiceHealthCheckPluginNotFound(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()

	_, err := svc.HealthCheck(ctx, &HealthCheckRequest{PluginId: "nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent plugin")
	}
}

func TestServiceHealthCheckEmptyPluginID(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()

	_, err := svc.HealthCheck(ctx, &HealthCheckRequest{PluginId: ""})
	if err == nil {
		t.Fatal("expected error for empty plugin_id")
	}
}

func TestServiceHealthCheckUnhealthyPlugin(t *testing.T) {
	svc, reg, hm := newTestService()
	ctx := context.Background()

	// Register and complete handshake
	_ = registerTestPlugin(hm, "unhealthy-test", "Unhealthy", "1.0.0", []string{"PERCEPTION"})

	// Transition to UNHEALTHY state
	_ = reg.UpdateState("unhealthy-test", registry.StateUnhealthy)

	resp, err := svc.HealthCheck(ctx, &HealthCheckRequest{PluginId: "unhealthy-test"})
	if err != nil {
		t.Fatalf("HealthCheck() failed: %v", err)
	}

	if resp.State != "UNHEALTHY" {
		t.Fatalf("expected state UNHEALTHY, got %s", resp.State)
	}
	if resp.Status != "ERROR" {
		t.Fatalf("expected status ERROR (mapped from UNHEALTHY), got %s", resp.Status)
	}
}

func TestServiceHealthCheckUnresponsivePlugin(t *testing.T) {
	svc, reg, hm := newTestService()
	ctx := context.Background()

	_ = registerTestPlugin(hm, "unresp-test", "Unresponsive", "1.0.0", []string{"MEMORY"})

	// Transition through valid path to UNRESPONSIVE
	_ = reg.UpdateState("unresp-test", registry.StateUnhealthy)
	_ = reg.UpdateState("unresp-test", registry.StateUnresponsive)

	resp, err := svc.HealthCheck(ctx, &HealthCheckRequest{PluginId: "unresp-test"})
	if err != nil {
		t.Fatalf("HealthCheck() failed: %v", err)
	}

	if resp.Status != "UNRESPONSIVE" {
		t.Fatalf("expected status UNRESPONSIVE, got %s", resp.Status)
	}
}

func TestServiceHealthCheckShuttingDownPlugin(t *testing.T) {
	svc, _, hm := newTestService()
	ctx := context.Background()

	_ = registerTestPlugin(hm, "shutting-test", "Shutting", "1.0.0", []string{"COGNITION"})

	// Initiate shutdown
	_ = hm.InitiateShutdown("shutting-test", 30*time.Second)

	resp, err := svc.HealthCheck(ctx, &HealthCheckRequest{PluginId: "shutting-test"})
	if err != nil {
		t.Fatalf("HealthCheck() failed: %v", err)
	}

	if resp.State != "SHUTTING_DOWN" {
		t.Fatalf("expected state SHUTTING_DOWN, got %s", resp.State)
	}
	if resp.Status != "UNRESPONSIVE" {
		t.Fatalf("expected status UNRESPONSIVE for SHUTTING_DOWN, got %s", resp.Status)
	}
}

// --- ListPlugins Tests ---

func TestServiceListPluginsEmpty(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()

	resp, err := svc.ListPlugins(ctx, &ListPluginsRequest{})
	if err != nil {
		t.Fatalf("ListPlugins() failed: %v", err)
	}

	if len(resp.Plugins) != 0 {
		t.Fatalf("expected 0 plugins, got %d", len(resp.Plugins))
	}
}

func TestServiceListPluginsSingle(t *testing.T) {
	svc, _, hm := newTestService()
	ctx := context.Background()

	_ = registerTestPlugin(hm, "list-test", "ListTest", "2.0.0", []string{"COGNITION"})

	resp, err := svc.ListPlugins(ctx, &ListPluginsRequest{})
	if err != nil {
		t.Fatalf("ListPlugins() failed: %v", err)
	}

	if len(resp.Plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(resp.Plugins))
	}

	p := resp.Plugins[0]
	if p.Id != "list-test" {
		t.Fatalf("expected plugin id list-test, got %s", p.Id)
	}
	if p.Name != "ListTest" {
		t.Fatalf("expected plugin name ListTest, got %s", p.Name)
	}
	if p.Version != "2.0.0" {
		t.Fatalf("expected plugin version 2.0.0, got %s", p.Version)
	}
	if p.State != "HEALTHY_ACTIVE" {
		t.Fatalf("expected plugin state HEALTHY_ACTIVE, got %s", p.State)
	}
}

func TestServiceListPluginsMultiple(t *testing.T) {
	svc, _, hm := newTestService()
	ctx := context.Background()

	_ = registerTestPlugin(hm, "plugin-a", "PluginA", "1.0.0", []string{"COGNITION"})
	_ = registerTestPlugin(hm, "plugin-b", "PluginB", "1.0.0", []string{"PERCEPTION"})
	_ = registerTestPlugin(hm, "plugin-c", "PluginC", "1.0.0", []string{"MEMORY"})

	resp, err := svc.ListPlugins(ctx, &ListPluginsRequest{})
	if err != nil {
		t.Fatalf("ListPlugins() failed: %v", err)
	}

	if len(resp.Plugins) != 3 {
		t.Fatalf("expected 3 plugins, got %d", len(resp.Plugins))
	}

	// Verify all plugins are present (order not guaranteed)
	found := make(map[string]bool)
	for _, p := range resp.Plugins {
		found[p.Id] = true
	}
	for _, id := range []string{"plugin-a", "plugin-b", "plugin-c"} {
		if !found[id] {
			t.Fatalf("expected plugin %s in list", id)
		}
	}
}

func TestServiceListPluginsIncludesUnhealthy(t *testing.T) {
	svc, reg, hm := newTestService()
	ctx := context.Background()

	_ = registerTestPlugin(hm, "healthy-plugin", "Healthy", "1.0.0", []string{"COGNITION"})

	// Register second plugin and make it unhealthy
	_ = registerTestPlugin(hm, "unhealthy-plugin", "Unhealthy", "1.0.0", []string{"PERCEPTION"})
	_ = reg.UpdateState("unhealthy-plugin", registry.StateUnhealthy)

	resp, err := svc.ListPlugins(ctx, &ListPluginsRequest{})
	if err != nil {
		t.Fatalf("ListPlugins() failed: %v", err)
	}

	if len(resp.Plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(resp.Plugins))
	}

	for _, p := range resp.Plugins {
		if p.Id == "unhealthy-plugin" && p.State != "UNHEALTHY" {
			t.Fatalf("expected unhealthy-plugin state UNHEALTHY, got %s", p.State)
		}
	}
}

// --- Shutdown Tests ---

func TestServiceShutdownSuccess(t *testing.T) {
	svc, _, hm := newTestService()
	ctx := context.Background()

	_ = registerTestPlugin(hm, "shutdown-test", "Shutdown", "1.0.0", []string{"COGNITION"})

	resp, err := svc.Shutdown(ctx, &ShutdownPluginRequest{
		PluginId:          "shutdown-test",
		GracePeriodSeconds: 30,
	})
	if err != nil {
		t.Fatalf("Shutdown() failed: %v", err)
	}

	if !resp.Accepted {
		t.Fatalf("expected accepted=true, got false: %s", resp.Reason)
	}

	// Verify plugin is in SHUTTING_DOWN state
	entry, ok := svc.registry.Get("shutdown-test")
	if !ok {
		t.Fatal("plugin not found in registry")
	}
	if entry.State != registry.StateShuttingDown {
		t.Fatalf("expected SHUTTING_DOWN, got %s", entry.State)
	}
}

func TestServiceShutdownEmptyPluginID(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()

	_, err := svc.Shutdown(ctx, &ShutdownPluginRequest{PluginId: ""})
	if err == nil {
		t.Fatal("expected error for empty plugin_id")
	}
}

func TestServiceShutdownPluginNotFound(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()

	_, err := svc.Shutdown(ctx, &ShutdownPluginRequest{
		PluginId:          "nonexistent",
		GracePeriodSeconds: 10,
	})
	if err == nil {
		t.Fatal("expected error for nonexistent plugin")
	}
}

func TestServiceShutdownInvalidState(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()

	// Register plugin but don't complete handshake — stays at REGISTERED
	_, _ = svc.Register(ctx, &RegisterRequest{
		PluginId: "not-active",
		Name:     "NotActive",
		Version:  "1.0.0",
	})

	resp, err := svc.Shutdown(ctx, &ShutdownPluginRequest{
		PluginId:          "not-active",
		GracePeriodSeconds: 10,
	})
	if err != nil {
		t.Fatalf("Shutdown() returned gRPC error instead of response: %v", err)
	}
	if resp.Accepted {
		t.Fatal("expected accepted=false for invalid state transition")
	}
}

func TestServiceShutdownDefaultGracePeriod(t *testing.T) {
	svc, _, hm := newTestService()
	ctx := context.Background()

	_ = registerTestPlugin(hm, "default-grace", "DefaultGrace", "1.0.0", []string{"COGNITION"})

	resp, err := svc.Shutdown(ctx, &ShutdownPluginRequest{
		PluginId:          "default-grace",
		GracePeriodSeconds: 0, // should default to 30s
	})
	if err != nil {
		t.Fatalf("Shutdown() failed: %v", err)
	}
	if !resp.Accepted {
		t.Fatalf("expected accepted=true, got false: %s", resp.Reason)
	}
	// Reason should mention 30s default
	if resp.Reason == "" {
		t.Fatal("expected non-empty reason")
	}
}

func TestServiceShutdownNoHandshakeManager(t *testing.T) {
	svc, reg := newTestServiceNoHandshake()
	ctx := context.Background()

	// Manually register a plugin in the registry
	reg.Register(&registry.PluginEntry{
		ID:         "fallback-plugin",
		Name:      "Fallback",
		Version:   "1.0.0",
		State:     registry.StateHealthyActive,
	})

	resp, err := svc.Shutdown(ctx, &ShutdownPluginRequest{
		PluginId:          "fallback-plugin",
		GracePeriodSeconds: 15,
	})
	if err != nil {
		t.Fatalf("Shutdown() failed: %v", err)
	}
	if !resp.Accepted {
		t.Fatalf("expected accepted=true for fallback shutdown, got false: %s", resp.Reason)
	}

	// Verify plugin is in SHUTTING_DOWN state
	entry, _ := reg.Get("fallback-plugin")
	if entry.State != registry.StateShuttingDown {
		t.Fatalf("expected SHUTTING_DOWN, got %s", entry.State)
	}
}

func TestServiceShutdownNoHandshakeManagerInvalidState(t *testing.T) {
	svc, reg := newTestServiceNoHandshake()
	ctx := context.Background()

	// Register a plugin in REGISTERED state (cannot transition to SHUTTING_DOWN)
	reg.Register(&registry.PluginEntry{
		ID:     "bad-state",
		Name:   "BadState",
		Version: "1.0.0",
		State:  registry.StateRegistered,
	})

	resp, err := svc.Shutdown(ctx, &ShutdownPluginRequest{
		PluginId:          "bad-state",
		GracePeriodSeconds: 10,
	})
	if err != nil {
		t.Fatalf("Shutdown() returned gRPC error instead of response: %v", err)
	}
	if resp.Accepted {
		t.Fatal("expected accepted=false for invalid state transition")
	}
}

// --- stateToHealthStatus Tests ---

func TestStateToHealthStatus(t *testing.T) {
	tests := []struct {
		state    registry.PluginState
		expected string
	}{
		{registry.StateHealthyActive, "HEALTHY"},
		{registry.StateUnhealthy, "ERROR"},
		{registry.StateUnresponsive, "UNRESPONSIVE"},
		{registry.StateShuttingDown, "UNRESPONSIVE"},
		{registry.StateShutDown, "UNRESPONSIVE"},
		{registry.StateStarting, "HEALTHY"},
		{registry.StateRegistered, "HEALTHY"},
		{registry.StateCircuitOpen, "CRITICAL"},
		{registry.StateDead, "CRITICAL"},
	}

	for _, tt := range tests {
		result := stateToHealthStatus(tt.state)
		if result != tt.expected {
			t.Errorf("stateToHealthStatus(%s) = %q, want %q", tt.state, result, tt.expected)
		}
	}
}

// --- Server Option Tests ---

func TestServerNewWithOptions(t *testing.T) {
	reg := registry.New()
	hm := NewHandshakeManager(reg, nil, "/tmp/kognis-test-server.sock")

	srv, err := New("/tmp/kognis-test-opt.sock",
		WithRegistry(reg),
		WithHandshakeManager(hm),
	)
	if err != nil {
		t.Fatalf("New() with options failed: %v", err)
	}
	defer srv.Close()

	if srv.Service() == nil {
		t.Fatal("expected service to be registered")
	}
}

func TestServerNewWithoutOptions(t *testing.T) {
	srv, err := New("/tmp/kognis-test-no-opts.sock")
	if err != nil {
		t.Fatalf("New() without options failed: %v", err)
	}
	defer srv.Close()

	if srv.Service() != nil {
		t.Fatal("expected service to be nil without registry option")
	}
}

func TestServerNewBackwardCompatible(t *testing.T) {
	// The old New(socketPath) call should still work
	srv, err := New("/tmp/kognis-test-compat.sock")
	if err != nil {
		t.Fatalf("New() backward compatible call failed: %v", err)
	}
	defer srv.Close()

	// Should still have a working gRPC server
	if srv.GRPCServer() == nil {
		t.Fatal("expected gRPC server to be available")
	}
}

func TestServerNewWithAllOptions(t *testing.T) {
	reg := registry.New()
	hm := NewHandshakeManager(reg, nil, "/tmp/kognis-test-all.sock")

	srv, err := New("/tmp/kognis-test-allopts.sock",
		WithRegistry(reg),
		WithBus(nil),
		WithHealthAggregator(nil),
		WithHandshakeManager(hm),
	)
	if err != nil {
		t.Fatalf("New() with all options failed: %v", err)
	}
	defer srv.Close()

	svc := srv.Service()
	if svc == nil {
		t.Fatal("expected service to be registered")
	}
	if svc.registry == nil {
		t.Fatal("expected registry in service")
	}
	if svc.handshake == nil {
		t.Fatal("expected handshake manager in service")
	}
}

// --- Integration: Full lifecycle via service ---

func TestServiceFullLifecycle(t *testing.T) {
	svc, reg, _ := newTestService()
	ctx := context.Background()

	// Step 1: Register
	regResp, err := svc.Register(ctx, &RegisterRequest{
		PluginId:    "lifecycle-test",
		Name:        "Lifecycle",
		Version:     "1.0.0",
		Capabilities: []string{"COGNITION"},
	})
	if err != nil {
		t.Fatalf("Register() failed: %v", err)
	}
	if regResp.Error != "" {
		t.Fatalf("Register() returned error: %s", regResp.Error)
	}

	// Complete handshake manually (service only handles step 1->2)
	hm := svc.handshake
	_ = hm.CompleteHandshake("lifecycle-test", &ReadyMessage{
		PluginID:         "lifecycle-test",
		SubscribedTopics: []string{"kognis.pipeline.cognition"},
	})

	// Step 2: HealthCheck (should be HEALTHY)
	hcResp, err := svc.HealthCheck(ctx, &HealthCheckRequest{PluginId: "lifecycle-test"})
	if err != nil {
		t.Fatalf("HealthCheck() failed: %v", err)
	}
	if hcResp.Status != "HEALTHY" {
		t.Fatalf("expected HEALTHY, got %s", hcResp.Status)
	}
	if hcResp.State != "HEALTHY_ACTIVE" {
		t.Fatalf("expected HEALTHY_ACTIVE, got %s", hcResp.State)
	}

	// Step 3: ListPlugins (should include our plugin)
	listResp, err := svc.ListPlugins(ctx, &ListPluginsRequest{})
	if err != nil {
		t.Fatalf("ListPlugins() failed: %v", err)
	}
	if len(listResp.Plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(listResp.Plugins))
	}

	// Step 4: Shutdown
	shutdownResp, err := svc.Shutdown(ctx, &ShutdownPluginRequest{
		PluginId:          "lifecycle-test",
		GracePeriodSeconds: 15,
	})
	if err != nil {
		t.Fatalf("Shutdown() failed: %v", err)
	}
	if !shutdownResp.Accepted {
		t.Fatalf("expected accepted=true: %s", shutdownResp.Reason)
	}

	// Verify final state
	entry, _ := reg.Get("lifecycle-test")
	if entry.State != registry.StateShuttingDown {
		t.Fatalf("expected SHUTTING_DOWN, got %s", entry.State)
	}
}