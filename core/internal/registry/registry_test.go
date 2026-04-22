package registry

import (
	"errors"
	"fmt"
	"testing"
	"time"
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

// --- New tests for M-013 extensions ---

func TestRegisterSetsRegisteredAt(t *testing.T) {
	r := New()
	entry := &PluginEntry{
		ID:    "p1",
		Name:  "Plugin1",
		State: StateRegistered,
	}

	before := time.Now()
	r.Register(entry)
	after := time.Now()

	got, _ := r.Get("p1")
	if got.RegisteredAt.Before(before) || got.RegisteredAt.After(after) {
		t.Fatalf("RegisteredAt = %v, expected between %v and %v", got.RegisteredAt, before, after)
	}
}

func TestRegisterInitializesHandshakeStep(t *testing.T) {
	r := New()
	entry := &PluginEntry{
		ID:    "p1",
		State: StateRegistered,
	}
	r.Register(entry)

	got, _ := r.Get("p1")
	if got.HandshakeStep != 1 {
		t.Fatalf("expected HandshakeStep=1 after registration, got %d", got.HandshakeStep)
	}
}

func TestRegisterPreservesExplicitHandshakeStep(t *testing.T) {
	r := New()
	entry := &PluginEntry{
		ID:            "p1",
		State:         StateRegistered,
		HandshakeStep: 3,
	}
	r.Register(entry)

	got, _ := r.Get("p1")
	if got.HandshakeStep != 3 {
		t.Fatalf("expected HandshakeStep=3 (preserved), got %d", got.HandshakeStep)
	}
}

func TestRegisterNewFieldsZeroInitialized(t *testing.T) {
	r := New()
	entry := &PluginEntry{
		ID:    "p1",
		State: StateRegistered,
	}
	r.Register(entry)

	got, _ := r.Get("p1")
	if got.MissedHeartbeats != 0 {
		t.Fatalf("expected MissedHeartbeats=0, got %d", got.MissedHeartbeats)
	}
	if got.RestartCount != 0 {
		t.Fatalf("expected RestartCount=0, got %d", got.RestartCount)
	}
	if got.LastHeartbeat != (time.Time{}) {
		t.Fatalf("expected LastHeartbeat zero, got %v", got.LastHeartbeat)
	}
	if got.LastRestartAt != (time.Time{}) {
		t.Fatalf("expected LastRestartAt zero, got %v", got.LastRestartAt)
	}
	if got.ShutdownRequestedAt != (time.Time{}) {
		t.Fatalf("expected ShutdownRequestedAt zero, got %v", got.ShutdownRequestedAt)
	}
	if got.ManifestHash != "" {
		t.Fatalf("expected ManifestHash empty, got %s", got.ManifestHash)
	}
	if got.SubscribedTopics != nil {
		t.Fatalf("expected SubscribedTopics nil, got %v", got.SubscribedTopics)
	}
	if got.ConfigBundle != nil {
		t.Fatalf("expected ConfigBundle nil, got %v", got.ConfigBundle)
	}
}

// --- Heartbeat tests (SPEC 04 Section 4.7) ---

func TestRecordHeartbeat(t *testing.T) {
	r := New()
	entry := &PluginEntry{
		ID:    "p1",
		State: StateHealthyActive,
	}
	r.Register(entry)

	// Increment missed heartbeats first
	r.IncrementMissedHeartbeats("p1")
	r.IncrementMissedHeartbeats("p1")

	p, _ := r.Get("p1")
	if p.MissedHeartbeats != 2 {
		t.Fatalf("expected 2 missed heartbeats, got %d", p.MissedHeartbeats)
	}

	// Record heartbeat should reset missed count
	err := r.RecordHeartbeat("p1")
	if err != nil {
		t.Fatalf("RecordHeartbeat failed: %v", err)
	}

	p, _ = r.Get("p1")
	if p.MissedHeartbeats != 0 {
		t.Fatalf("expected MissedHeartbeats=0 after heartbeat, got %d", p.MissedHeartbeats)
	}
	if p.LastHeartbeat.IsZero() {
		t.Fatal("expected LastHeartbeat to be set after RecordHeartbeat")
	}
}

func TestRecordHeartbeatNonexistent(t *testing.T) {
	r := New()
	err := r.RecordHeartbeat("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent plugin, got nil")
	}
}

func TestIncrementMissedHeartbeats(t *testing.T) {
	r := New()
	entry := &PluginEntry{
		ID:    "p1",
		State: StateHealthyActive,
	}
	r.Register(entry)

	count, err := r.IncrementMissedHeartbeats("p1")
	if err != nil {
		t.Fatalf("IncrementMissedHeartbeats failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected count=1, got %d", count)
	}

	count, err = r.IncrementMissedHeartbeats("p1")
	if err != nil {
		t.Fatalf("IncrementMissedHeartbeats failed: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected count=2, got %d", count)
	}
}

func TestIncrementMissedHeartbeatsNonexistent(t *testing.T) {
	r := New()
	count, err := r.IncrementMissedHeartbeats("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent plugin, got nil")
	}
	if count != 0 {
		t.Fatalf("expected count=0 for error, got %d", count)
	}
}

func TestCheckHeartbeatTimeouts(t *testing.T) {
	r := New()
	p1 := &PluginEntry{ID: "p1", State: StateHealthyActive}
	p2 := &PluginEntry{ID: "p2", State: StateHealthyActive}
	p3 := &PluginEntry{ID: "p3", State: StateHealthyActive}

	r.Register(p1)
	r.Register(p2)
	r.Register(p3)

	// p1: 3 missed heartbeats, p2: 2 missed, p3: 0 missed
	r.IncrementMissedHeartbeats("p1")
	r.IncrementMissedHeartbeats("p1")
	r.IncrementMissedHeartbeats("p1")
	r.IncrementMissedHeartbeats("p2")
	r.IncrementMissedHeartbeats("p2")

	timeouts := r.CheckHeartbeatTimeouts(3)
	if len(timeouts) != 1 {
		t.Fatalf("expected 1 timed-out plugin, got %d: %v", len(timeouts), timeouts)
	}
	found := false
	for _, id := range timeouts {
		if id == "p1" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected p1 in timeouts, got %v", timeouts)
	}

	// With threshold 2, both p1 and p2 should be returned
	timeouts = r.CheckHeartbeatTimeouts(2)
	if len(timeouts) != 2 {
		t.Fatalf("expected 2 timed-out plugins with threshold 2, got %d: %v", len(timeouts), timeouts)
	}
}

func TestCheckHeartbeatTimeoutsDefault(t *testing.T) {
	r := New()
	p1 := &PluginEntry{ID: "p1", State: StateHealthyActive}
	r.Register(p1)

	// With < 3 missed, should not timeout
	r.IncrementMissedHeartbeats("p1")
	r.IncrementMissedHeartbeats("p1")
	timeouts := r.CheckHeartbeatTimeouts(3)
	if len(timeouts) != 0 {
		t.Fatalf("expected 0 timeouts with 2 missed and threshold 3, got %v", timeouts)
	}

	// With 3 missed, should timeout
	r.IncrementMissedHeartbeats("p1")
	timeouts = r.CheckHeartbeatTimeouts(3)
	if len(timeouts) != 1 {
		t.Fatalf("expected 1 timeout with 3 missed and threshold 3, got %v", timeouts)
	}
}

// --- Restart/Backoff tests (SPEC 08 Section 8.4) ---

func TestRecordRestartAttempt(t *testing.T) {
	r := New()
	entry := &PluginEntry{
		ID:    "p1",
		State: StateUnresponsive,
	}
	r.Register(entry)

	count, err := r.RecordRestartAttempt("p1")
	if err != nil {
		t.Fatalf("RecordRestartAttempt failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected restart count=1, got %d", count)
	}

	p, _ := r.Get("p1")
	if p.RestartCount != 1 {
		t.Fatalf("expected RestartCount=1, got %d", p.RestartCount)
	}
	if p.LastRestartAt.IsZero() {
		t.Fatal("expected LastRestartAt to be set")
	}

	count, err = r.RecordRestartAttempt("p1")
	if err != nil {
		t.Fatalf("RecordRestartAttempt failed: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected restart count=2, got %d", count)
	}
}

func TestRecordRestartAttemptNonexistent(t *testing.T) {
	r := New()
	count, err := r.RecordRestartAttempt("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent plugin, got nil")
	}
	if count != 0 {
		t.Fatalf("expected count=0 for error, got %d", count)
	}
}

func TestShouldCircuitOpen(t *testing.T) {
	r := New()
	entry := &PluginEntry{
		ID:    "p1",
		State: StateUnresponsive,
	}
	r.Register(entry)

	// After 4 restarts, should not circuit open (5 is the threshold)
	for i := 0; i < 4; i++ {
		r.RecordRestartAttempt("p1")
	}
	if r.ShouldCircuitOpen("p1", 5) {
		t.Fatal("expected ShouldCircuitOpen=false with 4 restarts and max=5")
	}

	// After 5 restarts, should circuit open
	r.RecordRestartAttempt("p1")
	if !r.ShouldCircuitOpen("p1", 5) {
		t.Fatal("expected ShouldCircuitOpen=true with 5 restarts and max=5")
	}
}

func TestShouldCircuitOpenNonexistent(t *testing.T) {
	r := New()
	if r.ShouldCircuitOpen("nonexistent", 5) {
		t.Fatal("expected false for nonexistent plugin")
	}
}

func TestBackoffDuration(t *testing.T) {
	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{1, 0},
		{2, 30 * time.Second},
		{3, 2 * time.Minute},
		{4, 5 * time.Minute},
		{5, 15 * time.Minute},
		{6, 1 * time.Hour},  // beyond 5: 1-hour cooldown
		{10, 1 * time.Hour},  // beyond 5: 1-hour cooldown
	}

	for _, tt := range tests {
		got := BackoffDuration(tt.attempt)
		if got != tt.expected {
			t.Errorf("BackoffDuration(%d) = %v, want %v", tt.attempt, got, tt.expected)
		}
	}
}

func TestResetRestartCount(t *testing.T) {
	r := New()
	entry := &PluginEntry{
		ID:    "p1",
		State: StateHealthyActive,
	}
	r.Register(entry)

	r.RecordRestartAttempt("p1")
	r.RecordRestartAttempt("p1")
	r.RecordRestartAttempt("p1")

	p, _ := r.Get("p1")
	if p.RestartCount != 3 {
		t.Fatalf("expected RestartCount=3, got %d", p.RestartCount)
	}

	err := r.ResetRestartCount("p1")
	if err != nil {
		t.Fatalf("ResetRestartCount failed: %v", err)
	}

	p, _ = r.Get("p1")
	if p.RestartCount != 0 {
		t.Fatalf("expected RestartCount=0 after reset, got %d", p.RestartCount)
	}
	if p.LastRestartAt != (time.Time{}) {
		t.Fatalf("expected LastRestartAt to be zero after reset, got %v", p.LastRestartAt)
	}
}

func TestResetRestartCountNonexistent(t *testing.T) {
	r := New()
	err := r.ResetRestartCount("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent plugin, got nil")
	}
}

// --- Emergency Bypass tests (SPEC 14) ---

func TestValidateEmergencyBypassAuthorized(t *testing.T) {
	r := New()
	entry := &PluginEntry{
		ID:                   "audio-plugin",
		State:                StateHealthyActive,
		EmergencyBypassTypes: []string{"safety_sound_detected"},
	}
	r.Register(entry)

	err := r.ValidateEmergencyBypass("audio-plugin", "safety_sound_detected")
	if err != nil {
		t.Fatalf("expected authorized bypass, got error: %v", err)
	}
}

func TestValidateEmergencyBypassUnauthorizedType(t *testing.T) {
	r := New()
	entry := &PluginEntry{
		ID:                   "audio-plugin",
		State:                StateHealthyActive,
		EmergencyBypassTypes: []string{"safety_sound_detected"},
	}
	r.Register(entry)

	err := r.ValidateEmergencyBypass("audio-plugin", "health_critical")
	if err == nil {
		t.Fatal("expected error for unauthorized bypass type, got nil")
	}
	if !errors.Is(err, ErrBypassUnauthorized) {
		t.Fatalf("expected ErrBypassUnauthorized, got %v", err)
	}
}

func TestValidateEmergencyBypassInvalidType(t *testing.T) {
	r := New()
	entry := &PluginEntry{
		ID:                   "audio-plugin",
		State:                StateHealthyActive,
		EmergencyBypassTypes: []string{"safety_sound_detected"},
	}
	r.Register(entry)

	err := r.ValidateEmergencyBypass("audio-plugin", "invalid_bypass_type")
	if err == nil {
		t.Fatal("expected error for invalid bypass type, got nil")
	}
	if !errors.Is(err, ErrBypassUnauthorized) {
		t.Fatalf("expected ErrBypassUnauthorized, got %v", err)
	}
}

func TestValidateEmergencyBypassNoBypassTypes(t *testing.T) {
	r := New()
	entry := &PluginEntry{
		ID:    "plain-plugin",
		State: StateHealthyActive,
	}
	r.Register(entry)

	err := r.ValidateEmergencyBypass("plain-plugin", "safety_sound_detected")
	if err == nil {
		t.Fatal("expected error for plugin with no bypass types, got nil")
	}
	if !errors.Is(err, ErrBypassUnauthorized) {
		t.Fatalf("expected ErrBypassUnauthorized, got %v", err)
	}
}

func TestValidateEmergencyBypassNonexistentPlugin(t *testing.T) {
	r := New()
	err := r.ValidateEmergencyBypass("nonexistent", "safety_sound_detected")
	if err == nil {
		t.Fatal("expected error for nonexistent plugin, got nil")
	}
}

func TestRegisterRejectsInvalidBypassType(t *testing.T) {
	r := New()
	entry := &PluginEntry{
		ID:                   "bad-plugin",
		State:                StateRegistered,
		EmergencyBypassTypes: []string{"not_a_real_bypass"},
	}

	err := r.Register(entry)
	if err == nil {
		t.Fatal("expected error for invalid bypass type in manifest, got nil")
	}
	if !errors.Is(err, ErrBypassUnauthorized) {
		t.Fatalf("expected ErrBypassUnauthorized, got %v", err)
	}
}

func TestAllValidBypassTypes(t *testing.T) {
	r := New()
	validTypes := []string{"safety_sound_detected", "health_critical", "creator_emergency", "physical_hazard"}

	for i, bt := range validTypes {
		entry := &PluginEntry{
			ID:                   "plugin-" + string(rune('A'+i)),
			State:                StateRegistered,
			EmergencyBypassTypes: []string{bt},
		}
		err := r.Register(entry)
		if err != nil {
			t.Fatalf("failed to register plugin with valid bypass type %s: %v", bt, err)
		}

		err = r.ValidateEmergencyBypass(entry.ID, bt)
		if err != nil {
			t.Fatalf("expected validation to pass for bypass type %s, got: %v", bt, err)
		}
	}
}

// --- Capability Status Tracking tests (SPEC 05) ---

func TestSetCapabilityStatus(t *testing.T) {
	r := New()
	entry := &PluginEntry{
		ID:           "p1",
		State:        StateHealthyActive,
		Capabilities: []string{"vision"},
	}
	r.Register(entry)

	// Initially "available" because HEALTHY_ACTIVE
	err := r.SetCapabilityStatus("vision", "unavailable")
	if err != nil {
		t.Fatalf("SetCapabilityStatus failed: %v", err)
	}

	err = r.SetCapabilityStatus("vision", "available")
	if err != nil {
		t.Fatalf("SetCapabilityStatus failed: %v", err)
	}
}

func TestSetCapabilityStatusInvalidStatus(t *testing.T) {
	r := New()
	entry := &PluginEntry{
		ID:           "p1",
		State:        StateRegistered,
		Capabilities: []string{"vision"},
	}
	r.Register(entry)

	err := r.SetCapabilityStatus("vision", "broken")
	if err == nil {
		t.Fatal("expected error for invalid status, got nil")
	}
}

func TestSetCapabilityStatusNotFound(t *testing.T) {
	r := New()

	err := r.SetCapabilityStatus("nonexistent", "available")
	if err == nil {
		t.Fatal("expected error for nonexistent capability, got nil")
	}
	if !errors.Is(err, ErrCapabilityNotFound) {
		t.Fatalf("expected ErrCapabilityNotFound, got %v", err)
	}
}

func TestFindAvailableCapabilities(t *testing.T) {
	r := New()
	p1 := &PluginEntry{
		ID:           "p1",
		State:        StateHealthyActive,
		Capabilities: []string{"vision", "hearing"},
	}
	r.Register(p1)

	available := r.FindAvailableCapabilities()
	if len(available) != 2 {
		t.Fatalf("expected 2 available capabilities, got %d: %v", len(available), available)
	}

	// Make one unavailable
	r.SetCapabilityStatus("vision", "unavailable")

	available = r.FindAvailableCapabilities()
	if len(available) != 1 {
		t.Fatalf("expected 1 available capability after setting vision unavailable, got %d: %v", len(available), available)
	}
	if available[0] != "hearing" {
		t.Fatalf("expected hearing to be available, got %v", available)
	}
}

func TestCapabilityStatusTransitionsWithPluginState(t *testing.T) {
	r := New()
	entry := &PluginEntry{
		ID:           "p1",
		State:        StateRegistered,
		Capabilities: []string{"test.cap"},
	}
	r.Register(entry)

	// REGISTERED state — capabilities should be unavailable
	capEntry := r.capabilities["test.cap"]
	if capEntry.Status != "unavailable" {
		t.Fatalf("expected unavailable for REGISTERED state, got %s", capEntry.Status)
	}

	// Transition to STARTING — still unavailable
	r.UpdateState("p1", StateStarting)
	if capEntry.Status != "unavailable" {
		t.Fatalf("expected unavailable for STARTING state, got %s", capEntry.Status)
	}

	// Transition to HEALTHY_ACTIVE — capabilities become available
	r.UpdateState("p1", StateHealthyActive)
	if capEntry.Status != "available" {
		t.Fatalf("expected available for HEALTHY_ACTIVE state, got %s", capEntry.Status)
	}

	// Transition to UNHEALTHY — capabilities become unavailable
	r.UpdateState("p1", StateUnhealthy)
	if capEntry.Status != "unavailable" {
		t.Fatalf("expected unavailable for UNHEALTHY state, got %s", capEntry.Status)
	}

	// Transition back to HEALTHY_ACTIVE — available again
	r.UpdateState("p1", StateHealthyActive)
	if capEntry.Status != "available" {
		t.Fatalf("expected available for HEALTHY_ACTIVE state, got %s", capEntry.Status)
	}

	// Transition to UNRESPONSIVE — unavailable
	r.UpdateState("p1", StateUnresponsive)
	if capEntry.Status != "unavailable" {
		t.Fatalf("expected unavailable for UNRESPONSIVE state, got %s", capEntry.Status)
	}

	// Transition to CIRCUIT_OPEN — unavailable
	r.UpdateState("p1", StateCircuitOpen)
	if capEntry.Status != "unavailable" {
		t.Fatalf("expected unavailable for CIRCUIT_OPEN state, got %s", capEntry.Status)
	}
}

// --- Handshake Step tests (SPEC 04 Section 4.2) ---

func TestAdvanceHandshake(t *testing.T) {
	r := New()
	entry := &PluginEntry{
		ID:    "p1",
		State: StateRegistered,
	}
	r.Register(entry)

	// After registration, handshake step should be 1
	step, err := r.GetHandshakeStep("p1")
	if err != nil {
		t.Fatalf("GetHandshakeStep failed: %v", err)
	}
	if step != 1 {
		t.Fatalf("expected initial handshake step=1, got %d", step)
	}

	// Advance to step 2
	err = r.AdvanceHandshake("p1")
	if err != nil {
		t.Fatalf("AdvanceHandshake to step 2 failed: %v", err)
	}
	step, _ = r.GetHandshakeStep("p1")
	if step != 2 {
		t.Fatalf("expected step=2, got %d", step)
	}

	// Advance to step 3
	err = r.AdvanceHandshake("p1")
	if err != nil {
		t.Fatalf("AdvanceHandshake to step 3 failed: %v", err)
	}
	step, _ = r.GetHandshakeStep("p1")
	if step != 3 {
		t.Fatalf("expected step=3, got %d", step)
	}

	// Advance to step 4
	err = r.AdvanceHandshake("p1")
	if err != nil {
		t.Fatalf("AdvanceHandshake to step 4 failed: %v", err)
	}
	step, _ = r.GetHandshakeStep("p1")
	if step != 4 {
		t.Fatalf("expected step=4, got %d", step)
	}

	// Advance past step 4 should error
	err = r.AdvanceHandshake("p1")
	if err == nil {
		t.Fatal("expected error when advancing past step 4, got nil")
	}
}

func TestAdvanceHandshakeNonexistent(t *testing.T) {
	r := New()
	err := r.AdvanceHandshake("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent plugin, got nil")
	}
}

func TestGetHandshakeStepNonexistent(t *testing.T) {
	r := New()
	step, err := r.GetHandshakeStep("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent plugin, got nil")
	}
	if step != 0 {
		t.Fatalf("expected step=0 for error, got %d", step)
	}
}

// --- Graceful Shutdown tests (SPEC 04 Section 4.3) ---

func TestRequestShutdown(t *testing.T) {
	r := New()
	entry := &PluginEntry{
		ID:           "p1",
		State:        StateHealthyActive,
		Capabilities: []string{"vision"},
	}
	r.Register(entry)

	err := r.RequestShutdown("p1")
	if err != nil {
		t.Fatalf("RequestShutdown failed: %v", err)
	}

	p, _ := r.Get("p1")
	if p.State != StateShuttingDown {
		t.Fatalf("expected state SHUTTING_DOWN, got %s", p.State)
	}
	if p.ShutdownRequestedAt.IsZero() {
		t.Fatal("expected ShutdownRequestedAt to be set")
	}

	// Capability should be unavailable
	capEntry := r.capabilities["vision"]
	if capEntry.Status != "unavailable" {
		t.Fatalf("expected capability unavailable after shutdown request, got %s", capEntry.Status)
	}
}

func TestRequestShutdownInvalidTransition(t *testing.T) {
	r := New()
	entry := &PluginEntry{
		ID:    "p1",
		State: StateRegistered,
	}
	r.Register(entry)

	// Cannot go from REGISTERED to SHUTTING_DOWN
	err := r.RequestShutdown("p1")
	if err == nil {
		t.Fatal("expected error for invalid state transition, got nil")
	}
	if !errors.Is(err, ErrInvalidStateTransition) {
		t.Fatalf("expected ErrInvalidStateTransition, got %v", err)
	}
}

func TestRequestShutdownNonexistent(t *testing.T) {
	r := New()
	err := r.RequestShutdown("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent plugin, got nil")
	}
}

// --- KGN-* Error Code tests (SPEC 07) ---

func TestKGNErrorCodes(t *testing.T) {
	// Verify all new error codes exist with correct messages
	errorTests := []struct {
		err     error
		message string
	}{
		{ErrRegistrationTimeout, "KGN-LIFECYCLE-REGISTRATION_TIMEOUT-ERROR"},
		{ErrUnresponsive, "KGN-LIFECYCLE-UNRESPONSIVE-ERROR"},
		{ErrStartupFailed, "KGN-LIFECYCLE-STARTUP_FAILED-ERROR"},
		{ErrShutdownTimeout, "KGN-LIFECYCLE-SHUTDOWN_TIMEOUT-WARNING"},
		{ErrMaxRestartsExceeded, "KGN-LIFECYCLE-MAX_RESTARTS_EXCEEDED-CRITICAL"},
		{ErrCapabilityNotFound, "KGN-CAPABILITY-NOT_FOUND-ERROR"},
		{ErrCapabilityUnavailable, "KGN-CAPABILITY-UNAVAILABLE-ERROR"},
		{ErrPermissionDenied, "KGN-PERMISSION-DENIED-ERROR"},
		{ErrBypassUnauthorized, "KGN-PERMISSION-BYPASS_UNAUTHORIZED-ERROR"},
	}

	for _, tt := range errorTests {
		if tt.err.Error() != tt.message {
			t.Errorf("expected error message %q, got %q", tt.message, tt.err.Error())
		}
	}

	// Verify existing errors are preserved
	if ErrCapabilityConflict.Error() != "CAPABILITY_CONFLICT" {
		t.Errorf("expected ErrCapabilityConflict message CAPABILITY_CONFLICT, got %s", ErrCapabilityConflict.Error())
	}
	if ErrInvalidStateTransition.Error() != "INVALID_STATE_TRANSITION" {
		t.Errorf("expected ErrInvalidStateTransition message INVALID_STATE_TRANSITION, got %s", ErrInvalidStateTransition.Error())
	}
}

func TestKGNErrorCodesWithErrorsIs(t *testing.T) {
	// Verify errors.Is works with wrapped errors
	wrapped := fmt.Errorf("plugin failed: %w", ErrMaxRestartsExceeded)
	if !errors.Is(wrapped, ErrMaxRestartsExceeded) {
		t.Fatal("errors.Is should recognize wrapped ErrMaxRestartsExceeded")
	}

	wrapped2 := fmt.Errorf("bypass denied: %w", ErrBypassUnauthorized)
	if !errors.Is(wrapped2, ErrBypassUnauthorized) {
		t.Fatal("errors.Is should recognize wrapped ErrBypassUnauthorized")
	}
}

// --- Edge case tests ---

func TestDoubleRegistration(t *testing.T) {
	r := New()
	entry := &PluginEntry{
		ID:    "p1",
		State: StateRegistered,
	}

	err := r.Register(entry)
	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	err = r.Register(entry)
	if err == nil {
		t.Fatal("expected error for double registration, got nil")
	}
}

func TestOperationsOnNonexistentPlugin(t *testing.T) {
	r := New()

	// All operations on nonexistent plugins should return errors
	if _, err := r.IncrementMissedHeartbeats("nonexistent"); err == nil {
		t.Fatal("IncrementMissedHeartbeats should error on nonexistent plugin")
	}
	if err := r.RecordHeartbeat("nonexistent"); err == nil {
		t.Fatal("RecordHeartbeat should error on nonexistent plugin")
	}
	if err := r.ResetRestartCount("nonexistent"); err == nil {
		t.Fatal("ResetRestartCount should error on nonexistent plugin")
	}
	if _, err := r.GetHandshakeStep("nonexistent"); err == nil {
		t.Fatal("GetHandshakeStep should error on nonexistent plugin")
	}
	if err := r.AdvanceHandshake("nonexistent"); err == nil {
		t.Fatal("AdvanceHandshake should error on nonexistent plugin")
	}
	if err := r.RequestShutdown("nonexistent"); err == nil {
		t.Fatal("RequestShutdown should error on nonexistent plugin")
	}
	if _, err := r.RecordRestartAttempt("nonexistent"); err == nil {
		t.Fatal("RecordRestartAttempt should error on nonexistent plugin")
	}
}

func TestFullLifecycleWithHandshake(t *testing.T) {
	r := New()

	// Register
	entry := &PluginEntry{
		ID:                "lifecycle-plugin",
		Name:              "LifecyclePlugin",
		Version:           "1.0.0",
		State:             StateRegistered,
		Capabilities:      []string{"perception.vision", "perception.audio"},
		EmergencyBypassTypes: []string{"safety_sound_detected"},
		ManifestHash:      "abc123",
		ConfigBundle:      map[string]string{"log_level": "info"},
	}
	err := r.Register(entry)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Verify initial state
	p, ok := r.Get("lifecycle-plugin")
	if !ok {
		t.Fatal("plugin not found after registration")
	}
	if p.HandshakeStep != 1 {
		t.Fatalf("expected handshake step 1, got %d", p.HandshakeStep)
	}
	if p.RegisteredAt.IsZero() {
		t.Fatal("expected RegisteredAt to be set")
	}
	if p.ManifestHash != "abc123" {
		t.Fatalf("expected manifest hash abc123, got %s", p.ManifestHash)
	}

	// Advance through handshake steps
	r.AdvanceHandshake("lifecycle-plugin") // step 2
	r.AdvanceHandshake("lifecycle-plugin") // step 3
	r.AdvanceHandshake("lifecycle-plugin") // step 4

	step, _ := r.GetHandshakeStep("lifecycle-plugin")
	if step != 4 {
		t.Fatalf("expected handshake step 4, got %d", step)
	}

	// Transition to STARTING then HEALTHY_ACTIVE
	r.UpdateState("lifecycle-plugin", StateStarting)
	r.UpdateState("lifecycle-plugin", StateHealthyActive)

	// Verify capabilities are available
	avail := r.FindAvailableCapabilities()
	if len(avail) != 2 {
		t.Fatalf("expected 2 available capabilities, got %d", len(avail))
	}

	// Record heartbeats
	r.RecordHeartbeat("lifecycle-plugin")
	p, _ = r.Get("lifecycle-plugin")
	if p.MissedHeartbeats != 0 {
		t.Fatalf("expected 0 missed heartbeats, got %d", p.MissedHeartbeats)
	}

	// Simulate missed heartbeats
	r.IncrementMissedHeartbeats("lifecycle-plugin")
	r.IncrementMissedHeartbeats("lifecycle-plugin")
	r.IncrementMissedHeartbeats("lifecycle-plugin")
	timeouts := r.CheckHeartbeatTimeouts(3)
	if len(timeouts) != 1 {
		t.Fatalf("expected 1 timed-out plugin, got %d", len(timeouts))
	}

	// Emergency bypass validation
	err = r.ValidateEmergencyBypass("lifecycle-plugin", "safety_sound_detected")
	if err != nil {
		t.Fatalf("expected bypass authorized, got: %v", err)
	}

	// Unauthorized bypass
	err = r.ValidateEmergencyBypass("lifecycle-plugin", "health_critical")
	if err == nil {
		t.Fatal("expected bypass unauthorized error, got nil")
	}

	// Graceful shutdown
	err = r.RequestShutdown("lifecycle-plugin")
	if err != nil {
		t.Fatalf("RequestShutdown failed: %v", err)
	}

	p, _ = r.Get("lifecycle-plugin")
	if p.State != StateShuttingDown {
		t.Fatalf("expected SHUTTING_DOWN, got %s", p.State)
	}
	if p.ShutdownRequestedAt.IsZero() {
		t.Fatal("expected ShutdownRequestedAt to be set")
	}

	// Complete shutdown
	r.UpdateState("lifecycle-plugin", StateShutDown)

	// After shutdown, capabilities should be unavailable
	avail = r.FindAvailableCapabilities()
	if len(avail) != 0 {
		t.Fatalf("expected 0 available capabilities after shutdown, got %d", len(avail))
	}
}

func TestRestartBackoffFullCycle(t *testing.T) {
	r := New()
	entry := &PluginEntry{
		ID:           "unstable-plugin",
		State:        StateUnresponsive,
		Capabilities: []string{"test.cap"},
	}
	r.Register(entry)

	// Simulate 5 restart attempts
	for i := 1; i <= 5; i++ {
		count, _ := r.RecordRestartAttempt("unstable-plugin")
		if count != i {
			t.Fatalf("expected restart count %d, got %d", i, count)
		}
	}

	// Should circuit open at 5
	if !r.ShouldCircuitOpen("unstable-plugin", 5) {
		t.Fatal("expected ShouldCircuitOpen=true after 5 restarts")
	}

	// Verify backoff durations per SPEC 08
	if BackoffDuration(1) != 0 {
		t.Fatalf("BackoffDuration(1) = %v, want 0", BackoffDuration(1))
	}
	if BackoffDuration(2) != 30*time.Second {
		t.Fatalf("BackoffDuration(2) = %v, want 30s", BackoffDuration(2))
	}
	if BackoffDuration(3) != 2*time.Minute {
		t.Fatalf("BackoffDuration(3) = %v, want 2m", BackoffDuration(3))
	}
	if BackoffDuration(4) != 5*time.Minute {
		t.Fatalf("BackoffDuration(4) = %v, want 5m", BackoffDuration(4))
	}
	if BackoffDuration(5) != 15*time.Minute {
		t.Fatalf("BackoffDuration(5) = %v, want 15m", BackoffDuration(5))
	}

	// Reset after recovery
	err := r.ResetRestartCount("unstable-plugin")
	if err != nil {
		t.Fatalf("ResetRestartCount failed: %v", err)
	}

	p, _ := r.Get("unstable-plugin")
	if p.RestartCount != 0 {
		t.Fatalf("expected RestartCount=0 after reset, got %d", p.RestartCount)
	}

	// Should no longer circuit open
	if r.ShouldCircuitOpen("unstable-plugin", 5) {
		t.Fatal("expected ShouldCircuitOpen=false after reset")
	}
}