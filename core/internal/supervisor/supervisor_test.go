package supervisor

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/akashdas0307/kognis-core/core/internal/config"
	"github.com/akashdas0307/kognis-core/core/internal/eventbus"
	"github.com/akashdas0307/kognis-core/core/internal/registry"
	"github.com/akashdas0307/kognis-core/core/internal/router"
)

// testPortCounter provides unique ports for parallel tests.
var testPortCounter int64 = 16300

// nextTestPort returns a unique port number for test isolation.
func nextTestPort() int {
	return int(atomic.AddInt64(&testPortCounter, 1))
}

// testHarness holds the infrastructure needed for integration-style tests.
type testHarness struct {
	bus      *eventbus.Bus
	registry *registry.Registry
	router   *router.Router
	cfg      config.SupervisorConfig
}

func newTestHarness(t *testing.T) *testHarness {
	t.Helper()

	port := nextTestPort()

	// Start embedded NATS on a unique port
	bus, err := eventbus.New(config.NATSConfig{
		Port:       port,
		DataDir:    t.TempDir(),
		ServerName: fmt.Sprintf("test-nats-%d", port),
	})
	if err != nil {
		t.Fatalf("create event bus (port %d): %v", port, err)
	}
	t.Cleanup(func() { bus.Close() })

	reg := registry.New()

	// Create a minimal router (no pipeline templates needed for supervisor tests)
	rtr := router.New(reg, bus)

	cfg := config.SupervisorConfig{
		HeartbeatIntervalSec:   1,
		RegistrationTimeoutSec:  5,
		ShutdownGracePeriodSec: 1,
		MaxRestartAttempts:      5,
	}

	return &testHarness{
		bus:      bus,
		registry: reg,
		router:   rtr,
		cfg:      cfg,
	}
}

// --- Unit Tests: SPEC 08 Backoff Schedule (via registry.BackoffDuration) ---

func TestSPEC08BackoffSchedule(t *testing.T) {
	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{1, 0},                    // Attempt 1: immediate
		{2, 30 * time.Second},     // Attempt 2: 30s
		{3, 2 * time.Minute},      // Attempt 3: 2m
		{4, 5 * time.Minute},      // Attempt 4: 5m
		{5, 15 * time.Minute},     // Attempt 5: 15m
		{6, 1 * time.Hour},        // Beyond 5: 1h
		{10, 1 * time.Hour},       // Beyond 5: 1h
		{100, 1 * time.Hour},      // Beyond 5: 1h
	}

	for _, tt := range tests {
		got := registry.BackoffDuration(tt.attempt)
		if got != tt.expected {
			t.Errorf("BackoffDuration(%d) = %v, want %v (SPEC 08 Section 8.4)", tt.attempt, got, tt.expected)
		}
	}
}

func TestBackoffNotExponential(t *testing.T) {
	// Verify the schedule is NOT exponential (2^n). The old implementation
	// used 2^n seconds capped at 30s, which is wrong per SPEC 08.
	got := registry.BackoffDuration(2)
	if got == 2*time.Second {
		t.Errorf("BackoffDuration(2) returned 2s (exponential schedule); expected 30s per SPEC 08")
	}
	if got != 30*time.Second {
		t.Errorf("BackoffDuration(2) = %v, want 30s per SPEC 08", got)
	}

	got = registry.BackoffDuration(3)
	if got == 4*time.Second {
		t.Errorf("BackoffDuration(3) returned 4s (exponential schedule); expected 2m per SPEC 08")
	}
	if got != 2*time.Minute {
		t.Errorf("BackoffDuration(3) = %v, want 2m per SPEC 08", got)
	}
}

// --- Unit Tests: Circuit Breaker ---

func TestCircuitBreakerOpensAfterMaxRestarts(t *testing.T) {
	reg := registry.New()
	maxRestarts := 5

	entry := &registry.PluginEntry{
		ID:           "test-plugin",
		Name:         "TestPlugin",
		Version:      "1.0.0",
		State:        registry.StateUnresponsive,
		Capabilities: []string{"test_cap"},
	}
	if err := reg.Register(entry); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	// Simulate 5 restart attempts
	for i := 0; i < maxRestarts; i++ {
		count, err := reg.RecordRestartAttempt("test-plugin")
		if err != nil {
			t.Fatalf("record restart attempt %d: %v", i+1, err)
		}
		t.Logf("Restart attempt %d: count=%d", i+1, count)
	}

	// After exactly maxRestarts, circuit should open
	if !reg.ShouldCircuitOpen("test-plugin", maxRestarts) {
		t.Errorf("ShouldCircuitOpen() = false after %d restarts, want true", maxRestarts)
	}

	// Verify state transition to CIRCUIT_OPEN is valid
	err := reg.UpdateState("test-plugin", registry.StateCircuitOpen)
	if err != nil {
		t.Errorf("transition UNRESPONSIVE -> CIRCUIT_OPEN failed: %v", err)
	}
}

func TestCircuitBreakerDoesNotOpenBeforeMaxRestarts(t *testing.T) {
	reg := registry.New()
	maxRestarts := 5

	entry := &registry.PluginEntry{
		ID:           "test-plugin",
		Name:         "TestPlugin",
		Version:      "1.0.0",
		State:        registry.StateUnresponsive,
		Capabilities: []string{"test_cap"},
	}
	if err := reg.Register(entry); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	// Simulate 4 restart attempts (one below max)
	for i := 0; i < maxRestarts-1; i++ {
		reg.RecordRestartAttempt("test-plugin")
	}

	if reg.ShouldCircuitOpen("test-plugin", maxRestarts) {
		t.Errorf("ShouldCircuitOpen() = true after %d restarts (max=%d), want false", maxRestarts-1, maxRestarts)
	}
}

// --- Unit Tests: Heartbeat Timeout Detection ---

func TestHeartbeatTimeoutThreeMissedIsUnresponsive(t *testing.T) {
	reg := registry.New()

	entry := &registry.PluginEntry{
		ID:           "test-plugin",
		Name:         "TestPlugin",
		Version:      "1.0.0",
		State:        registry.StateHealthyActive,
		Capabilities: []string{"test_cap"},
	}
	if err := reg.Register(entry); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	// Record initial heartbeat
	reg.RecordHeartbeat("test-plugin")

	// Simulate 3 missed heartbeats (SPEC 04 Section 4.7: 3 missed = UNRESPONSIVE)
	for i := 0; i < 3; i++ {
		count, err := reg.IncrementMissedHeartbeats("test-plugin")
		if err != nil {
			t.Fatalf("increment missed heartbeats: %v", err)
		}
		t.Logf("After increment %d: missed=%d", i+1, count)
	}

	// Check heartbeat timeouts
	timeoutIDs := reg.CheckHeartbeatTimeouts(3)
	if len(timeoutIDs) != 1 || timeoutIDs[0] != "test-plugin" {
		t.Errorf("CheckHeartbeatTimeouts(3) = %v, want [test-plugin]", timeoutIDs)
	}

	// Verify transition to UNRESPONSIVE is valid
	err := reg.UpdateState("test-plugin", registry.StateUnresponsive)
	if err != nil {
		t.Errorf("transition HEALTHY_ACTIVE -> UNRESPONSIVE failed: %v", err)
	}
}

func TestHeartbeatTimeoutTwoMissedNotUnresponsive(t *testing.T) {
	reg := registry.New()

	entry := &registry.PluginEntry{
		ID:           "test-plugin",
		Name:         "TestPlugin",
		Version:      "1.0.0",
		State:        registry.StateHealthyActive,
		Capabilities: []string{"test_cap"},
	}
	if err := reg.Register(entry); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	// Only 2 missed heartbeats (below threshold)
	for i := 0; i < 2; i++ {
		reg.IncrementMissedHeartbeats("test-plugin")
	}

	timeoutIDs := reg.CheckHeartbeatTimeouts(3)
	if len(timeoutIDs) != 0 {
		t.Errorf("CheckHeartbeatTimeouts(3) = %v, want empty (only 2 missed)", timeoutIDs)
	}
}

func TestHeartbeatResetsMissedCount(t *testing.T) {
	reg := registry.New()

	entry := &registry.PluginEntry{
		ID:           "test-plugin",
		Name:         "TestPlugin",
		Version:      "1.0.0",
		State:        registry.StateHealthyActive,
		Capabilities: []string{"test_cap"},
	}
	if err := reg.Register(entry); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	// Simulate 2 missed heartbeats
	reg.IncrementMissedHeartbeats("test-plugin")
	reg.IncrementMissedHeartbeats("test-plugin")

	plugin, _ := reg.Get("test-plugin")
	if plugin.MissedHeartbeats != 2 {
		t.Errorf("MissedHeartbeats = %d, want 2", plugin.MissedHeartbeats)
	}

	// Record heartbeat -- should reset missed count to 0
	reg.RecordHeartbeat("test-plugin")

	plugin, _ = reg.Get("test-plugin")
	if plugin.MissedHeartbeats != 0 {
		t.Errorf("MissedHeartbeats = %d after RecordHeartbeat, want 0", plugin.MissedHeartbeats)
	}
}

// --- Unit Tests: Restart Count Reset on Recovery ---

func TestRestartCountResetsOnHealthyRecovery(t *testing.T) {
	reg := registry.New()

	entry := &registry.PluginEntry{
		ID:           "test-plugin",
		Name:         "TestPlugin",
		Version:      "1.0.0",
		State:        registry.StateUnresponsive,
		Capabilities: []string{"test_cap"},
	}
	if err := reg.Register(entry); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	// Simulate 3 restart attempts
	for i := 0; i < 3; i++ {
		reg.RecordRestartAttempt("test-plugin")
	}

	plugin, _ := reg.Get("test-plugin")
	if plugin.RestartCount != 3 {
		t.Errorf("RestartCount = %d, want 3", plugin.RestartCount)
	}

	// Plugin recovers: transition through UNRESPONSIVE -> STARTING -> HEALTHY_ACTIVE
	reg.UpdateState("test-plugin", registry.StateStarting)
	reg.UpdateState("test-plugin", registry.StateHealthyActive)

	// Reset restart count on recovery
	err := reg.ResetRestartCount("test-plugin")
	if err != nil {
		t.Fatalf("ResetRestartCount: %v", err)
	}

	plugin, _ = reg.Get("test-plugin")
	if plugin.RestartCount != 0 {
		t.Errorf("RestartCount = %d after recovery, want 0", plugin.RestartCount)
	}

	// ShouldCircuitOpen should now return false
	if reg.ShouldCircuitOpen("test-plugin", 5) {
		t.Errorf("ShouldCircuitOpen() = true after reset, want false")
	}
}

// --- Integration Tests: Supervisor with Event Bus ---

func TestSupervisorRestartUsesSPEC08Backoff(t *testing.T) {
	h := newTestHarness(t)

	sv := New(h.registry, h.router, h.bus, h.cfg)

	// Register a plugin that will become unresponsive
	entry := &registry.PluginEntry{
		ID:           "plugin-restart-test",
		Name:         "RestartTest",
		Version:      "1.0.0",
		State:        registry.StateUnresponsive,
		Capabilities: []string{"restart_cap"},
	}
	if err := h.registry.Register(entry); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	// Trigger restart via restartPlugin directly
	// (checkPluginHealth would change state to STARTING, preventing further calls)
	sv.restartPlugin("plugin-restart-test")

	// Verify first restart attempt uses 0 backoff (immediate)
	plugin, ok := h.registry.Get("plugin-restart-test")
	if !ok {
		t.Fatal("plugin not found after restart")
	}
	if plugin.RestartCount != 1 {
		t.Errorf("RestartCount = %d, want 1", plugin.RestartCount)
	}

	// Verify backoff for attempt 1 is 0 (immediate per SPEC 08)
	backoff := registry.BackoffDuration(plugin.RestartCount)
	if backoff != 0 {
		t.Errorf("BackoffDuration(1) = %v, want 0 (immediate per SPEC 08)", backoff)
	}

	// Verify the plugin was transitioned to STARTING
	if plugin.State != registry.StateStarting {
		t.Errorf("plugin state = %s, want STARTING after restart", plugin.State)
	}
}

func TestSupervisorCircuitBreakerIntegration(t *testing.T) {
	h := newTestHarness(t)

	sv := New(h.registry, h.router, h.bus, h.cfg)

	// Register a plugin in UNRESPONSIVE state
	entry := &registry.PluginEntry{
		ID:           "plugin-circuit-test",
		Name:         "CircuitTest",
		Version:      "1.0.0",
		State:        registry.StateUnresponsive,
		Capabilities: []string{"circuit_cap"},
	}
	if err := h.registry.Register(entry); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	// Call restartPlugin maxRestartAttempts times.
	// Each call: checks ShouldCircuitOpen (false for first N-1), increments count, sets STARTING.
	// We need to reset state to UNRESPONSIVE between calls so restartPlugin keeps getting called.
	for i := 0; i < h.cfg.MaxRestartAttempts; i++ {
		sv.restartPlugin("plugin-circuit-test")
		// After restartPlugin, state is STARTING. Set back to UNRESPONSIVE for next iteration
		// (simulating the plugin failing to start again).
		h.registry.UpdateState("plugin-circuit-test", registry.StateUnresponsive)
	}

	// Now RestartCount = maxRestartAttempts. The next restartPlugin call should open the circuit.
	sv.restartPlugin("plugin-circuit-test")

	plugin, _ := h.registry.Get("plugin-circuit-test")
	if plugin.State != registry.StateCircuitOpen {
		t.Errorf("plugin state = %s, want CIRCUIT_OPEN after exceeding max restarts", plugin.State)
	}
}

func TestSupervisorHeartbeatTimeoutIntegration(t *testing.T) {
	h := newTestHarness(t)

	sv := New(h.registry, h.router, h.bus, h.cfg)

	entry := &registry.PluginEntry{
		ID:           "plugin-heartbeat-test",
		Name:         "HeartbeatTest",
		Version:      "1.0.0",
		State:        registry.StateHealthyActive,
		Capabilities: []string{"hb_cap"},
	}
	if err := h.registry.Register(entry); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	// Record initial heartbeat so we have a baseline
	h.registry.RecordHeartbeat("plugin-heartbeat-test")

	// Simulate 3 missed heartbeats (without calling RecordHeartbeat)
	for i := 0; i < 3; i++ {
		h.registry.IncrementMissedHeartbeats("plugin-heartbeat-test")
	}

	// Run heartbeat timeout check
	sv.checkHeartbeatTimeouts()

	// Verify plugin is now detected as timed out
	timeoutIDs := h.registry.CheckHeartbeatTimeouts(3)
	found := false
	for _, id := range timeoutIDs {
		if id == "plugin-heartbeat-test" {
			found = true
		}
	}
	if !found {
		t.Error("plugin-heartbeat-test not in timeout list after 3 missed heartbeats")
	}
}

func TestSupervisorGracefulShutdown(t *testing.T) {
	h := newTestHarness(t)

	sv := New(h.registry, h.router, h.bus, h.cfg)

	// Register multiple plugins
	for i := 0; i < 3; i++ {
		id := fmt.Sprintf("plugin-shutdown-%d", i)
		entry := &registry.PluginEntry{
			ID:           id,
			Name:         fmt.Sprintf("ShutdownTest%d", i),
			Version:      "1.0.0",
			State:        registry.StateHealthyActive,
			Capabilities: []string{fmt.Sprintf("shutdown_cap_%d", i)},
		}
		if err := h.registry.Register(entry); err != nil {
			t.Fatalf("register plugin %s: %v", id, err)
		}
	}

	// Subscribe to shutdown commands
	var shutdownCount int32
	for i := 0; i < 3; i++ {
		id := fmt.Sprintf("plugin-shutdown-%d", i)
		subject := fmt.Sprintf("kognis.plugin.%s.shutdown", id)
		_, err := h.bus.Subscribe(subject, func(msg *nats.Msg) {
			atomic.AddInt32(&shutdownCount, 1)
		})
		if err != nil {
			t.Fatalf("subscribe to shutdown subject: %v", err)
		}
	}

	// Run supervisor with immediate cancellation
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		sv.Run(ctx)
		close(done)
	}()

	// Give the supervisor a moment to start
	time.Sleep(100 * time.Millisecond)

	// Cancel context to trigger shutdown
	cancel()

	// Wait for shutdown to complete
	select {
	case <-done:
		// Good, shutdown completed
	case <-time.After(5 * time.Second):
		t.Fatal("supervisor Run() did not return within timeout after cancellation")
	}

	// Verify all plugins are shut down
	for _, plugin := range h.registry.List() {
		if plugin.State != registry.StateShutDown {
			t.Errorf("plugin %s state = %s, want SHUT_DOWN", plugin.ID, plugin.State)
		}
	}
}

func TestSupervisorHealthPulseRecovery(t *testing.T) {
	h := newTestHarness(t)

	sv := New(h.registry, h.router, h.bus, h.cfg)

	// Register a plugin in unresponsive state with some restart history
	entry := &registry.PluginEntry{
		ID:           "plugin-recovery-test",
		Name:         "RecoveryTest",
		Version:      "1.0.0",
		State:        registry.StateUnresponsive,
		Capabilities: []string{"recovery_cap"},
	}
	if err := h.registry.Register(entry); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	// Record 3 restart attempts
	for i := 0; i < 3; i++ {
		h.registry.RecordRestartAttempt("plugin-recovery-test")
	}

	// Transition to HEALTHY_ACTIVE (simulate successful recovery)
	h.registry.UpdateState("plugin-recovery-test", registry.StateStarting)
	h.registry.UpdateState("plugin-recovery-test", registry.StateHealthyActive)

	// Simulate a HEALTHY pulse -- should reset RestartCount since > 0
	pulseData, _ := json.Marshal(map[string]string{
		"plugin_id": "plugin-recovery-test",
		"status":    "HEALTHY",
	})

	msg := &nats.Msg{
		Data: pulseData,
	}

	sv.handleHealthPulse(msg)

	// Verify restart count was reset
	plugin, _ := h.registry.Get("plugin-recovery-test")
	if plugin.RestartCount != 0 {
		t.Errorf("RestartCount = %d after HEALTHY pulse recovery, want 0", plugin.RestartCount)
	}

	if plugin.MissedHeartbeats != 0 {
		t.Errorf("MissedHeartbeats = %d after HEALTHY pulse, want 0", plugin.MissedHeartbeats)
	}
}

func TestSupervisorHealthPulseResetsHeartbeatOnHealthy(t *testing.T) {
	h := newTestHarness(t)

	sv := New(h.registry, h.router, h.bus, h.cfg)

	entry := &registry.PluginEntry{
		ID:           "plugin-hb-reset-test",
		Name:         "HBResetTest",
		Version:      "1.0.0",
		State:        registry.StateHealthyActive,
		Capabilities: []string{"hb_reset_cap"},
	}
	if err := h.registry.Register(entry); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	// Simulate 2 missed heartbeats
	h.registry.IncrementMissedHeartbeats("plugin-hb-reset-test")
	h.registry.IncrementMissedHeartbeats("plugin-hb-reset-test")

	plugin, _ := h.registry.Get("plugin-hb-reset-test")
	if plugin.MissedHeartbeats != 2 {
		t.Errorf("MissedHeartbeats = %d before pulse, want 2", plugin.MissedHeartbeats)
	}

	// Send HEALTHY pulse
	pulseData, _ := json.Marshal(map[string]string{
		"plugin_id": "plugin-hb-reset-test",
		"status":    "HEALTHY",
	})
	msg := &nats.Msg{Data: pulseData}

	sv.handleHealthPulse(msg)

	// Verify heartbeat was recorded (missed count reset)
	plugin, _ = h.registry.Get("plugin-hb-reset-test")
	if plugin.MissedHeartbeats != 0 {
		t.Errorf("MissedHeartbeats = %d after HEALTHY pulse, want 0", plugin.MissedHeartbeats)
	}
}

func TestSupervisorIgnoresUnknownPluginHealthPulse(t *testing.T) {
	h := newTestHarness(t)

	sv := New(h.registry, h.router, h.bus, h.cfg)

	// Send a health pulse for a non-existent plugin -- should not panic
	pulseData, _ := json.Marshal(map[string]string{
		"plugin_id": "nonexistent-plugin",
		"status":    "HEALTHY",
	})
	msg := &nats.Msg{Data: pulseData}

	// Should not panic
	sv.handleHealthPulse(msg)
}

func TestSupervisorInvalidHealthPulseIgnored(t *testing.T) {
	h := newTestHarness(t)

	sv := New(h.registry, h.router, h.bus, h.cfg)

	// Send invalid JSON
	msg := &nats.Msg{Data: []byte("not-json")}
	// Should not panic
	sv.handleHealthPulse(msg)
}

func TestSupervisorRunLifecycle(t *testing.T) {
	h := newTestHarness(t)

	sv := New(h.registry, h.router, h.bus, h.cfg)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- sv.Run(ctx)
	}()

	// Let it run briefly
	time.Sleep(200 * time.Millisecond)

	// Cancel to trigger shutdown
	cancel()

	var err error
	select {
	case err = <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("supervisor Run() did not return within timeout")
	}

	if err != nil {
		t.Errorf("supervisor Run() returned error: %v", err)
	}
}