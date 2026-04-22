package health

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/akashdas0307/kognis-core/core/internal/config"
	"github.com/akashdas0307/kognis-core/core/internal/eventbus"
	"github.com/akashdas0307/kognis-core/core/internal/registry"
)

// newTestBus creates a Bus with a unique port for each test.
func newTestBus(t *testing.T, port int) *eventbus.Bus {
	t.Helper()
	cfg := config.NATSConfig{
		ServerName: "test-health",
		Port:       port,
		DataDir:    t.TempDir(),
	}
	bus, err := eventbus.New(cfg)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	t.Cleanup(func() { bus.Close() })
	return bus
}

// setupPluginForTest registers a plugin and transitions it to HEALTHY_ACTIVE.
func setupPluginForTest(t *testing.T, reg *registry.Registry, pluginID string) {
	t.Helper()
	entry := &registry.PluginEntry{
		ID:    pluginID,
		Name:  pluginID,
		State: registry.StateRegistered,
	}
	if err := reg.Register(entry); err != nil {
		t.Fatalf("Register() error: %v", err)
	}
	if err := reg.UpdateState(pluginID, registry.StateStarting); err != nil {
		t.Fatalf("UpdateState to STARTING error: %v", err)
	}
	if err := reg.UpdateState(pluginID, registry.StateHealthyActive); err != nil {
		t.Fatalf("UpdateState to HEALTHY_ACTIVE error: %v", err)
	}
}

// newTestAggregator creates a full test fixture: registry, bus, and aggregator.
func newTestAggregator(t *testing.T, port int) (*Aggregator, *registry.Registry, *eventbus.Bus) {
	t.Helper()
	reg := registry.New()
	bus := newTestBus(t, port)
	agg := NewAggregator(reg, bus)
	return agg, reg, bus
}

// --- mapStatusToState tests ---

func TestMapStatusToState(t *testing.T) {
	tests := []struct {
		status string
		want   registry.PluginState
	}{
		{"HEALTHY", registry.StateHealthyActive},
		{"DEGRADED", registry.StateUnhealthy},
		{"ERROR", registry.StateUnhealthy},
		{"CRITICAL", registry.StateUnresponsive},
		{"UNRESPONSIVE", registry.StateUnresponsive},
		{"", ""},
		{"UNKNOWN", ""},
	}

	for _, tt := range tests {
		got := mapStatusToState(tt.status)
		if got != tt.want {
			t.Errorf("mapStatusToState(%q) = %q, want %q", tt.status, got, tt.want)
		}
	}
}

// --- Pulse struct tests ---

func TestPulseJSONRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	dispatchAt := now.Add(-5 * time.Second)

	pulse := Pulse{
		PluginID:        "test-plugin",
		Timestamp:       now,
		Status:          "HEALTHY",
		Metrics:         map[string]interface{}{"queue_depth": float64(3), "latency_p50": float64(12.5)},
		CurrentActivity: "processing",
		LastDispatchAt:  &dispatchAt,
		Alerts: []Alert{
			{Severity: "warning", Code: "HIGH_QUEUE", Message: "Queue depth is elevated"},
		},
	}

	data, err := json.Marshal(pulse)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}

	var decoded Pulse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}

	if decoded.PluginID != pulse.PluginID {
		t.Errorf("PluginID = %q, want %q", decoded.PluginID, pulse.PluginID)
	}
	if decoded.Status != pulse.Status {
		t.Errorf("Status = %q, want %q", decoded.Status, pulse.Status)
	}
	if decoded.CurrentActivity != pulse.CurrentActivity {
		t.Errorf("CurrentActivity = %q, want %q", decoded.CurrentActivity, pulse.CurrentActivity)
	}
	if len(decoded.Alerts) != 1 {
		t.Fatalf("Alerts length = %d, want 1", len(decoded.Alerts))
	}
	if decoded.Alerts[0].Severity != "warning" {
		t.Errorf("Alert Severity = %q, want %q", decoded.Alerts[0].Severity, "warning")
	}
	if decoded.Alerts[0].Code != "HIGH_QUEUE" {
		t.Errorf("Alert Code = %q, want %q", decoded.Alerts[0].Code, "HIGH_QUEUE")
	}
	if decoded.LastDispatchAt == nil {
		t.Fatal("LastDispatchAt is nil, want non-nil")
	}
}

func TestPulseJSONWithNilFields(t *testing.T) {
	pulse := Pulse{
		PluginID:  "minimal-plugin",
		Timestamp: time.Now(),
		Status:    "DEGRADED",
	}

	data, err := json.Marshal(pulse)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}

	var decoded Pulse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}

	if decoded.Metrics != nil {
		t.Errorf("Metrics = %v, want nil", decoded.Metrics)
	}
	if decoded.LastDispatchAt != nil {
		t.Errorf("LastDispatchAt = %v, want nil", decoded.LastDispatchAt)
	}
	if decoded.Alerts != nil {
		t.Errorf("Alerts = %v, want nil", decoded.Alerts)
	}
}

// --- Alert struct tests ---

func TestAlertFields(t *testing.T) {
	alert := Alert{Severity: "critical", Code: "OOM", Message: "Out of memory"}
	if alert.Severity != "critical" {
		t.Errorf("Severity = %q, want %q", alert.Severity, "critical")
	}
	if alert.Code != "OOM" {
		t.Errorf("Code = %q, want %q", alert.Code, "OOM")
	}
}

// --- RecordPulse tests ---

func TestRecordPulse_StoresLatestPulse(t *testing.T) {
	agg, reg, _ := newTestAggregator(t, 14300)
	setupPluginForTest(t, reg, "p1")

	pulse := &Pulse{
		PluginID: "p1",
		Status:   "HEALTHY",
		Metrics:  map[string]interface{}{"queue_depth": float64(1)},
	}
	agg.RecordPulse(pulse)

	got, ok := agg.GetPulse("p1")
	if !ok {
		t.Fatal("GetPulse returned false after RecordPulse")
	}
	if got.Status != "HEALTHY" {
		t.Errorf("Status = %q, want %q", got.Status, "HEALTHY")
	}
	if got.PluginID != "p1" {
		t.Errorf("PluginID = %q, want %q", got.PluginID, "p1")
	}
}

func TestRecordPulse_SetsTimestampIfZero(t *testing.T) {
	agg, reg, _ := newTestAggregator(t, 14301)
	setupPluginForTest(t, reg, "p1")

	pulse := &Pulse{PluginID: "p1", Status: "HEALTHY"}
	agg.RecordPulse(pulse)

	got, _ := agg.GetPulse("p1")
	if got.Timestamp.IsZero() {
		t.Error("Timestamp is zero; want auto-set to now")
	}
}

func TestRecordPulse_PreservesExistingTimestamp(t *testing.T) {
	agg, reg, _ := newTestAggregator(t, 14302)
	setupPluginForTest(t, reg, "p1")

	customTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	pulse := &Pulse{PluginID: "p1", Status: "HEALTHY", Timestamp: customTime}
	agg.RecordPulse(pulse)

	got, _ := agg.GetPulse("p1")
	if !got.Timestamp.Equal(customTime) {
		t.Errorf("Timestamp = %v, want %v", got.Timestamp, customTime)
	}
}

func TestRecordPulse_UpdatesRegistryState(t *testing.T) {
	agg, reg, _ := newTestAggregator(t, 14303)
	setupPluginForTest(t, reg, "p1")

	// Record HEALTHY pulse
	agg.RecordPulse(&Pulse{PluginID: "p1", Status: "HEALTHY"})
	entry, _ := reg.Get("p1")
	if entry.State != registry.StateHealthyActive {
		t.Errorf("State = %s, want %s", entry.State, registry.StateHealthyActive)
	}

	// Record DEGRADED pulse — should transition to UNHEALTHY
	agg.RecordPulse(&Pulse{PluginID: "p1", Status: "DEGRADED"})
	entry, _ = reg.Get("p1")
	if entry.State != registry.StateUnhealthy {
		t.Errorf("State = %s after DEGRADED, want %s", entry.State, registry.StateUnhealthy)
	}

	// Record ERROR pulse — should stay UNHEALTHY
	agg.RecordPulse(&Pulse{PluginID: "p1", Status: "ERROR"})
	entry, _ = reg.Get("p1")
	if entry.State != registry.StateUnhealthy {
		t.Errorf("State = %s after ERROR, want %s", entry.State, registry.StateUnhealthy)
	}

	// Record HEALTHY pulse — should recover to HEALTHY_ACTIVE
	agg.RecordPulse(&Pulse{PluginID: "p1", Status: "HEALTHY"})
	entry, _ = reg.Get("p1")
	if entry.State != registry.StateHealthyActive {
		t.Errorf("State = %s after HEALTHY recovery, want %s", entry.State, registry.StateHealthyActive)
	}
}

func TestRecordPulse_CriticalTransitionsToUnresponsive(t *testing.T) {
	agg, reg, _ := newTestAggregator(t, 14304)
	setupPluginForTest(t, reg, "p1")

	agg.RecordPulse(&Pulse{PluginID: "p1", Status: "CRITICAL"})
	entry, _ := reg.Get("p1")
	if entry.State != registry.StateUnresponsive {
		t.Errorf("State = %s after CRITICAL, want %s", entry.State, registry.StateUnresponsive)
	}
}

func TestRecordPulse_UnresponsiveStatus(t *testing.T) {
	agg, reg, _ := newTestAggregator(t, 14305)
	setupPluginForTest(t, reg, "p1")

	agg.RecordPulse(&Pulse{PluginID: "p1", Status: "UNRESPONSIVE"})
	entry, _ := reg.Get("p1")
	if entry.State != registry.StateUnresponsive {
		t.Errorf("State = %s after UNRESPONSIVE, want %s", entry.State, registry.StateUnresponsive)
	}
}

func TestRecordPulse_OverwritesLatestPulse(t *testing.T) {
	agg, reg, _ := newTestAggregator(t, 14306)
	setupPluginForTest(t, reg, "p1")

	agg.RecordPulse(&Pulse{PluginID: "p1", Status: "HEALTHY", CurrentActivity: "idle"})
	agg.RecordPulse(&Pulse{PluginID: "p1", Status: "DEGRADED", CurrentActivity: "recovering"})

	got, _ := agg.GetPulse("p1")
	if got.Status != "DEGRADED" {
		t.Errorf("Status = %q, want %q", got.Status, "DEGRADED")
	}
	if got.CurrentActivity != "recovering" {
		t.Errorf("CurrentActivity = %q, want %q", got.CurrentActivity, "recovering")
	}
}

func TestRecordPulse_MultiplePlugins(t *testing.T) {
	agg, reg, _ := newTestAggregator(t, 14307)
	setupPluginForTest(t, reg, "p1")
	setupPluginForTest(t, reg, "p2")

	agg.RecordPulse(&Pulse{PluginID: "p1", Status: "HEALTHY"})
	agg.RecordPulse(&Pulse{PluginID: "p2", Status: "ERROR"})

	p1, _ := agg.GetPulse("p1")
	p2, _ := agg.GetPulse("p2")

	if p1.Status != "HEALTHY" {
		t.Errorf("p1 Status = %q, want %q", p1.Status, "HEALTHY")
	}
	if p2.Status != "ERROR" {
		t.Errorf("p2 Status = %q, want %q", p2.Status, "ERROR")
	}
}

// --- History tests ---

func TestRecordPulse_MaintainsHistory(t *testing.T) {
	agg, reg, _ := newTestAggregator(t, 14308)
	setupPluginForTest(t, reg, "p1")

	for i := 0; i < 5; i++ {
		agg.RecordPulse(&Pulse{
			PluginID:  "p1",
			Status:    "HEALTHY",
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
		})
	}

	hist := agg.GetHistory("p1")
	if len(hist) != 5 {
		t.Fatalf("history length = %d, want 5", len(hist))
	}
}

func TestGetHistory_ReturnsCopy(t *testing.T) {
	agg, reg, _ := newTestAggregator(t, 14309)
	setupPluginForTest(t, reg, "p1")

	agg.RecordPulse(&Pulse{PluginID: "p1", Status: "HEALTHY"})

	hist := agg.GetHistory("p1")
	hist[0].Status = "TAMPERED"

	// Original should not be affected
	got, _ := agg.GetPulse("p1")
	if got.Status != "HEALTHY" {
		t.Error("GetHistory returned a reference, not a copy")
	}
}

func TestGetHistory_NonexistentPlugin(t *testing.T) {
	agg, _, _ := newTestAggregator(t, 14310)

	hist := agg.GetHistory("nonexistent")
	if hist != nil {
		t.Errorf("GetHistory for nonexistent plugin = %v, want nil", hist)
	}
}

func TestRecordPulse_AutoPruneHistory(t *testing.T) {
	agg, reg, _ := newTestAggregator(t, 14311)
	setupPluginForTest(t, reg, "p1")
	agg.SetMaxHistory(3)

	for i := 0; i < 5; i++ {
		agg.RecordPulse(&Pulse{
			PluginID:  "p1",
			Status:    "HEALTHY",
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
		})
	}

	hist := agg.GetHistory("p1")
	if len(hist) != 3 {
		t.Fatalf("history length = %d after auto-prune with maxHistory=3, want 3", len(hist))
	}
}

// --- PruneHistory tests ---

func TestPruneHistory(t *testing.T) {
	agg, reg, _ := newTestAggregator(t, 14312)
	setupPluginForTest(t, reg, "p1")
	agg.SetMaxHistory(5)

	// Record 10 pulses
	for i := 0; i < 10; i++ {
		agg.RecordPulse(&Pulse{
			PluginID:  "p1",
			Status:    "HEALTHY",
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
		})
	}

	// Auto-pruning already happened during RecordPulse, but let's
	// also test explicit PruneHistory by manually extending history
	// We need to directly manipulate the history map via a workaround.
	// Since auto-prune already ran, history should be at 5 already.
	hist := agg.GetHistory("p1")
	if len(hist) != 5 {
		t.Fatalf("history length = %d, want 5", len(hist))
	}

	// Call PruneHistory explicitly — should be idempotent
	agg.PruneHistory()
	hist = agg.GetHistory("p1")
	if len(hist) != 5 {
		t.Fatalf("history length = %d after PruneHistory, want 5", len(hist))
	}
}

func TestPruneHistory_KeepsNewestEntries(t *testing.T) {
	agg, reg, _ := newTestAggregator(t, 14313)
	setupPluginForTest(t, reg, "p1")
	agg.SetMaxHistory(3)

	// Record pulses with distinct timestamps
	for i := 0; i < 5; i++ {
		agg.RecordPulse(&Pulse{
			PluginID:  "p1",
			Status:    "HEALTHY",
			Timestamp: time.Date(2025, 1, 1, 0, i, 0, 0, time.UTC),
		})
	}

	hist := agg.GetHistory("p1")
	if len(hist) != 3 {
		t.Fatalf("history length = %d, want 3", len(hist))
	}

	// The newest entries should be kept: minute 2, 3, 4
	first := hist[0].Timestamp.Minute()
	if first != 2 {
		t.Errorf("oldest kept entry timestamp minute = %d, want 2", first)
	}
	last := hist[2].Timestamp.Minute()
	if last != 4 {
		t.Errorf("newest kept entry timestamp minute = %d, want 4", last)
	}
}

// --- SetMaxHistory tests ---

func TestSetMaxHistory(t *testing.T) {
	agg, _, _ := newTestAggregator(t, 14314)

	if agg.maxHistory != DefaultMaxHistory {
		t.Errorf("default maxHistory = %d, want %d", agg.maxHistory, DefaultMaxHistory)
	}

	agg.SetMaxHistory(50)
	if agg.maxHistory != 50 {
		t.Errorf("maxHistory = %d after SetMaxHistory(50), want 50", agg.maxHistory)
	}
}

func TestSetMaxHistory_IgnoresZero(t *testing.T) {
	agg, _, _ := newTestAggregator(t, 14315)

	agg.SetMaxHistory(50)
	agg.SetMaxHistory(0) // should be ignored
	if agg.maxHistory != 50 {
		t.Errorf("maxHistory = %d after SetMaxHistory(0), want 50 (unchanged)", agg.maxHistory)
	}
}

func TestSetMaxHistory_IgnoresNegative(t *testing.T) {
	agg, _, _ := newTestAggregator(t, 14316)

	agg.SetMaxHistory(50)
	agg.SetMaxHistory(-1) // should be ignored
	if agg.maxHistory != 50 {
		t.Errorf("maxHistory = %d after SetMaxHistory(-1), want 50 (unchanged)", agg.maxHistory)
	}
}

// --- GetDerivedMetrics tests ---

func TestGetDerivedMetrics_UptimePercent(t *testing.T) {
	agg, reg, _ := newTestAggregator(t, 14317)
	setupPluginForTest(t, reg, "p1")

	// 3 HEALTHY + 2 ERROR = 60% uptime
	statuses := []string{"HEALTHY", "HEALTHY", "ERROR", "HEALTHY", "ERROR"}
	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	for i, status := range statuses {
		agg.RecordPulse(&Pulse{
			PluginID:  "p1",
			Status:    status,
			Timestamp: baseTime.Add(time.Duration(i) * time.Minute),
		})
	}

	dm := agg.GetDerivedMetrics("p1")
	if dm == nil {
		t.Fatal("GetDerivedMetrics returned nil")
	}
	if dm.UptimePercent != 60.0 {
		t.Errorf("UptimePercent = %v, want 60.0", dm.UptimePercent)
	}
}

func TestGetDerivedMetrics_ErrorRate(t *testing.T) {
	agg, reg, _ := newTestAggregator(t, 14318)
	setupPluginForTest(t, reg, "p1")

	// 2 ERROR pulses over 4 minutes = 0.5 errors/min
	statuses := []string{"HEALTHY", "ERROR", "HEALTHY", "ERROR", "HEALTHY"}
	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	for i, status := range statuses {
		agg.RecordPulse(&Pulse{
			PluginID:  "p1",
			Status:    status,
			Timestamp: baseTime.Add(time.Duration(i) * time.Minute),
		})
	}

	dm := agg.GetDerivedMetrics("p1")
	if dm == nil {
		t.Fatal("GetDerivedMetrics returned nil")
	}
	if dm.ErrorRate != 0.5 {
		t.Errorf("ErrorRate = %v, want 0.5", dm.ErrorRate)
	}
}

func TestGetDerivedMetrics_LatencyAverages(t *testing.T) {
	agg, reg, _ := newTestAggregator(t, 14319)
	setupPluginForTest(t, reg, "p1")

	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 4; i++ {
		agg.RecordPulse(&Pulse{
			PluginID:  "p1",
			Status:    "HEALTHY",
			Timestamp: baseTime.Add(time.Duration(i) * time.Minute),
			Metrics: map[string]interface{}{
				"latency_p50": float64(10 + i*2), // 10, 12, 14, 16 => avg 13
				"latency_p99": float64(50 + i*5), // 50, 55, 60, 65 => avg 57.5
			},
		})
	}

	dm := agg.GetDerivedMetrics("p1")
	if dm == nil {
		t.Fatal("GetDerivedMetrics returned nil")
	}
	if dm.AvgLatencyP50 != 13.0 {
		t.Errorf("AvgLatencyP50 = %v, want 13.0", dm.AvgLatencyP50)
	}
	if dm.AvgLatencyP99 != 57.5 {
		t.Errorf("AvgLatencyP99 = %v, want 57.5", dm.AvgLatencyP99)
	}
}

func TestGetDerivedMetrics_CriticalCountsAsError(t *testing.T) {
	agg, reg, _ := newTestAggregator(t, 14320)
	setupPluginForTest(t, reg, "p1")

	// 1 HEALTHY + 1 CRITICAL over 1 minute = 50% uptime, 1 error/min
	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	agg.RecordPulse(&Pulse{PluginID: "p1", Status: "HEALTHY", Timestamp: baseTime})
	agg.RecordPulse(&Pulse{PluginID: "p1", Status: "CRITICAL", Timestamp: baseTime.Add(time.Minute)})

	dm := agg.GetDerivedMetrics("p1")
	if dm == nil {
		t.Fatal("GetDerivedMetrics returned nil")
	}
	if dm.UptimePercent != 50.0 {
		t.Errorf("UptimePercent = %v, want 50.0", dm.UptimePercent)
	}
	if dm.ErrorRate != 1.0 {
		t.Errorf("ErrorRate = %v, want 1.0", dm.ErrorRate)
	}
}

func TestGetDerivedMetrics_NoLatencyMetrics(t *testing.T) {
	agg, reg, _ := newTestAggregator(t, 14321)
	setupPluginForTest(t, reg, "p1")

	agg.RecordPulse(&Pulse{PluginID: "p1", Status: "HEALTHY"})
	agg.RecordPulse(&Pulse{
		PluginID:  "p1",
		Status:    "HEALTHY",
		Timestamp: time.Now().Add(time.Minute),
	})

	dm := agg.GetDerivedMetrics("p1")
	if dm == nil {
		t.Fatal("GetDerivedMetrics returned nil")
	}
	if dm.AvgLatencyP50 != 0 {
		t.Errorf("AvgLatencyP50 = %v, want 0 (no metrics)", dm.AvgLatencyP50)
	}
	if dm.AvgLatencyP99 != 0 {
		t.Errorf("AvgLatencyP99 = %v, want 0 (no metrics)", dm.AvgLatencyP99)
	}
}

func TestGetDerivedMetrics_NonexistentPlugin(t *testing.T) {
	agg, _, _ := newTestAggregator(t, 14322)

	dm := agg.GetDerivedMetrics("nonexistent")
	if dm != nil {
		t.Errorf("GetDerivedMetrics for nonexistent plugin = %v, want nil", dm)
	}
}

func TestGetDerivedMetrics_SinglePulse(t *testing.T) {
	agg, reg, _ := newTestAggregator(t, 14323)
	setupPluginForTest(t, reg, "p1")

	agg.RecordPulse(&Pulse{PluginID: "p1", Status: "HEALTHY"})

	dm := agg.GetDerivedMetrics("p1")
	if dm == nil {
		t.Fatal("GetDerivedMetrics returned nil for single pulse")
	}
	if dm.UptimePercent != 100.0 {
		t.Errorf("UptimePercent = %v, want 100.0", dm.UptimePercent)
	}
	// Error rate is 0 with only 1 pulse (can't compute rate from a single point)
	if dm.ErrorRate != 0 {
		t.Errorf("ErrorRate = %v, want 0 (single pulse)", dm.ErrorRate)
	}
}

// --- GetAlerts tests ---

func TestGetAlerts(t *testing.T) {
	agg, reg, _ := newTestAggregator(t, 14324)
	setupPluginForTest(t, reg, "p1")

	alerts := []Alert{
		{Severity: "warning", Code: "HIGH_QUEUE", Message: "Queue depth elevated"},
		{Severity: "error", Code: "SLOW_RESPONSE", Message: "P99 latency above threshold"},
	}
	agg.RecordPulse(&Pulse{PluginID: "p1", Status: "DEGRADED", Alerts: alerts})

	got := agg.GetAlerts("p1")
	if len(got) != 2 {
		t.Fatalf("alerts length = %d, want 2", len(got))
	}
	if got[0].Code != "HIGH_QUEUE" {
		t.Errorf("alert[0].Code = %q, want %q", got[0].Code, "HIGH_QUEUE")
	}
	if got[1].Severity != "error" {
		t.Errorf("alert[1].Severity = %q, want %q", got[1].Severity, "error")
	}
}

func TestGetAlerts_NoAlerts(t *testing.T) {
	agg, reg, _ := newTestAggregator(t, 14325)
	setupPluginForTest(t, reg, "p1")

	agg.RecordPulse(&Pulse{PluginID: "p1", Status: "HEALTHY"})

	got := agg.GetAlerts("p1")
	if got != nil {
		t.Errorf("GetAlerts = %v, want nil for pulse with no alerts", got)
	}
}

func TestGetAlerts_NonexistentPlugin(t *testing.T) {
	agg, _, _ := newTestAggregator(t, 14326)

	got := agg.GetAlerts("nonexistent")
	if got != nil {
		t.Errorf("GetAlerts for nonexistent plugin = %v, want nil", got)
	}
}

func TestGetAlerts_ReturnsCopy(t *testing.T) {
	agg, reg, _ := newTestAggregator(t, 14327)
	setupPluginForTest(t, reg, "p1")

	agg.RecordPulse(&Pulse{
		PluginID: "p1",
		Status:   "DEGRADED",
		Alerts:   []Alert{{Severity: "warning", Code: "TEST", Message: "test alert"}},
	})

	got := agg.GetAlerts("p1")
	got[0].Code = "TAMPERED"

	// Original should not be affected
	original := agg.GetAlerts("p1")
	if original[0].Code != "TEST" {
		t.Error("GetAlerts returned a reference, not a copy")
	}
}

// --- SystemHealthStatus tests ---

func TestSystemHealthStatus(t *testing.T) {
	agg, reg, _ := newTestAggregator(t, 14328)
	setupPluginForTest(t, reg, "p1")
	setupPluginForTest(t, reg, "p2")
	setupPluginForTest(t, reg, "p3")

	agg.RecordPulse(&Pulse{PluginID: "p1", Status: "HEALTHY"})
	agg.RecordPulse(&Pulse{PluginID: "p2", Status: "DEGRADED"})
	agg.RecordPulse(&Pulse{PluginID: "p3", Status: "ERROR"})

	status := agg.SystemHealthStatus()
	if len(status) != 3 {
		t.Fatalf("SystemHealthStatus length = %d, want 3", len(status))
	}
	if status["p1"] != "HEALTHY" {
		t.Errorf("p1 status = %q, want %q", status["p1"], "HEALTHY")
	}
	if status["p2"] != "DEGRADED" {
		t.Errorf("p2 status = %q, want %q", status["p2"], "DEGRADED")
	}
	if status["p3"] != "ERROR" {
		t.Errorf("p3 status = %q, want %q", status["p3"], "ERROR")
	}
}

func TestSystemHealthStatus_Empty(t *testing.T) {
	agg, _, _ := newTestAggregator(t, 14329)

	status := agg.SystemHealthStatus()
	if len(status) != 0 {
		t.Errorf("SystemHealthStatus length = %d for empty aggregator, want 0", len(status))
	}
}

// --- AllPulses tests ---

func TestAllPulses(t *testing.T) {
	agg, reg, _ := newTestAggregator(t, 14330)
	setupPluginForTest(t, reg, "p1")
	setupPluginForTest(t, reg, "p2")

	agg.RecordPulse(&Pulse{PluginID: "p1", Status: "HEALTHY"})
	agg.RecordPulse(&Pulse{PluginID: "p2", Status: "ERROR"})

	all := agg.AllPulses()
	if len(all) != 2 {
		t.Fatalf("AllPulses length = %d, want 2", len(all))
	}
	if all["p1"].Status != "HEALTHY" {
		t.Errorf("p1 Status = %q, want %q", all["p1"].Status, "HEALTHY")
	}
	if all["p2"].Status != "ERROR" {
		t.Errorf("p2 Status = %q, want %q", all["p2"].Status, "ERROR")
	}
}

// --- NATS subscription integration test ---

func TestNATSSubscription_ReceivesPulse(t *testing.T) {
	agg, reg, bus := newTestAggregator(t, 14331)
	_ = agg
	setupPluginForTest(t, reg, "nats-plugin")

	ch := make(chan struct{})

	// Subscribe to state changes to know when the pulse was processed
	bus.SubscribeState("nats-plugin", "health_status", func(msg *nats.Msg) {
		ch <- struct{}{}
	})

	// Publish a health pulse on the correct NATS subject
	pulse := Pulse{
		PluginID: "nats-plugin",
		Status:   "HEALTHY",
		Metrics:  map[string]interface{}{"queue_depth": float64(1)},
	}
	data, err := json.Marshal(pulse)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}
	bus.Publish(eventbus.HealthSubject("nats-plugin"), data)

	// Wait for the pulse to be processed
	select {
	case <-ch:
		// State change received — pulse was processed
	case <-time.After(3 * time.Second):
		// State change might not fire on first pulse; verify via GetPulse directly
	}

	// Verify pulse was recorded
	got, ok := agg.GetPulse("nats-plugin")
	if !ok {
		t.Fatal("GetPulse returned false; NATS pulse not received")
	}
	if got.Status != "HEALTHY" {
		t.Errorf("Status = %q, want %q", got.Status, "HEALTHY")
	}
}

func TestNATSSubscription_IgnoresInvalidJSON(t *testing.T) {
	agg, reg, bus := newTestAggregator(t, 14332)
	_ = agg
	setupPluginForTest(t, reg, "p1")

	// Publish invalid JSON — should be logged and ignored, not crash
	bus.Publish(eventbus.HealthSubject("p1"), []byte("not json"))

	// The aggregator should still be functional
	agg.RecordPulse(&Pulse{PluginID: "p1", Status: "HEALTHY"})
	got, ok := agg.GetPulse("p1")
	if !ok {
		t.Fatal("GetPulse returned false after invalid JSON; aggregator may have crashed")
	}
	if got.Status != "HEALTHY" {
		t.Errorf("Status = %q, want %q", got.Status, "HEALTHY")
	}
}

func TestNATSSubscription_SkipsNonPulseMessages(t *testing.T) {
	agg, reg, bus := newTestAggregator(t, 14333)
	setupPluginForTest(t, reg, "p1")

	// First record a real pulse
	agg.RecordPulse(&Pulse{PluginID: "p1", Status: "HEALTHY"})

	// Publish a supervisor-style notification (has no Status field)
	bus.Publish(eventbus.HealthSubject("p1"), []byte(`{"event":"recovered","plugin_id":"p1","timestamp":"2025-01-01T00:00:00Z"}`))

	// Wait briefly for processing
	time.Sleep(100 * time.Millisecond)

	// The original pulse should still be the latest
	got, ok := agg.GetPulse("p1")
	if !ok {
		t.Fatal("GetPulse returned false")
	}
	if got.Status != "HEALTHY" {
		t.Errorf("Status = %q after non-pulse message, want %q", got.Status, "HEALTHY")
	}
}

// --- toFloat helper tests ---

func TestToFloat(t *testing.T) {
	tests := []struct {
		input interface{}
		want  float64
		ok    bool
	}{
		{float64(3.14), 3.14, true},
		{float32(2.5), 2.5, true},
		{int(42), 42.0, true},
		{int32(7), 7.0, true},
		{int64(100), 100.0, true},
		{"not a number", 0, false},
		{nil, 0, false},
		{true, 0, false},
	}

	for _, tt := range tests {
		got, ok := toFloat(tt.input)
		if ok != tt.ok {
			t.Errorf("toFloat(%v) ok = %v, want %v", tt.input, ok, tt.ok)
		}
		if ok && got != tt.want {
			t.Errorf("toFloat(%v) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

// --- Pulse with LastDispatchAt test ---

func TestRecordPulse_WithLastDispatchAt(t *testing.T) {
	agg, reg, _ := newTestAggregator(t, 14334)
	setupPluginForTest(t, reg, "p1")

	dispatchAt := time.Now().Add(-10 * time.Second)
	agg.RecordPulse(&Pulse{
		PluginID:       "p1",
		Status:         "HEALTHY",
		LastDispatchAt: &dispatchAt,
		CurrentActivity: "reasoning",
	})

	got, ok := agg.GetPulse("p1")
	if !ok {
		t.Fatal("GetPulse returned false")
	}
	if got.CurrentActivity != "reasoning" {
		t.Errorf("CurrentActivity = %q, want %q", got.CurrentActivity, "reasoning")
	}
	if got.LastDispatchAt == nil {
		t.Fatal("LastDispatchAt is nil, want non-nil")
	}
	if !got.LastDispatchAt.Equal(dispatchAt) {
		t.Errorf("LastDispatchAt = %v, want %v", got.LastDispatchAt, dispatchAt)
	}
}

// --- Registry state transition edge case ---

func TestRecordPulse_InvalidStateTransitionHandled(t *testing.T) {
	agg, reg, _ := newTestAggregator(t, 14335)
	setupPluginForTest(t, reg, "p1")

	// Record CRITICAL pulse — transitions HEALTHY_ACTIVE -> UNRESPONSIVE
	agg.RecordPulse(&Pulse{PluginID: "p1", Status: "CRITICAL"})
	entry, _ := reg.Get("p1")
	if entry.State != registry.StateUnresponsive {
		t.Fatalf("State = %s after CRITICAL, want %s", entry.State, registry.StateUnresponsive)
	}

	// Now try to record HEALTHY pulse — UNRESPONSIVE -> HEALTHY_ACTIVE is NOT a valid transition
	// The aggregator should log the error but not crash
	agg.RecordPulse(&Pulse{PluginID: "p1", Status: "HEALTHY"})

	// The registry state should remain UNRESPONSIVE (invalid transition was rejected)
	entry, _ = reg.Get("p1")
	if entry.State != registry.StateUnresponsive {
		t.Errorf("State = %s after invalid transition attempt, want %s", entry.State, registry.StateUnresponsive)
	}

	// But the aggregator should still have recorded the pulse
	got, ok := agg.GetPulse("p1")
	if !ok {
		t.Fatal("GetPulse returned false")
	}
	if got.Status != "HEALTHY" {
		t.Errorf("Pulse Status = %q, want %q (pulse should be recorded even if state update fails)", got.Status, "HEALTHY")
	}
}

// --- Full lifecycle integration test ---

func TestFullHealthLifecycle(t *testing.T) {
	agg, reg, _ := newTestAggregator(t, 14336)
	setupPluginForTest(t, reg, "lifecycle-plugin")

	baseTime := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

	// Phase 1: Plugin starts healthy
	agg.RecordPulse(&Pulse{
		PluginID:  "lifecycle-plugin",
		Status:    "HEALTHY",
		Timestamp: baseTime,
		Metrics:   map[string]interface{}{"latency_p50": float64(5), "latency_p99": float64(20)},
	})

	// Phase 2: Degrades
	agg.RecordPulse(&Pulse{
		PluginID:  "lifecycle-plugin",
		Status:    "DEGRADED",
		Timestamp: baseTime.Add(time.Minute),
		Metrics:   map[string]interface{}{"latency_p50": float64(50), "latency_p99": float64(200)},
		Alerts:    []Alert{{Severity: "warning", Code: "HIGH_LATENCY", Message: "Latency elevated"}},
	})

	// Phase 3: Error
	agg.RecordPulse(&Pulse{
		PluginID:  "lifecycle-plugin",
		Status:    "ERROR",
		Timestamp: baseTime.Add(2 * time.Minute),
		Metrics:   map[string]interface{}{"latency_p50": float64(100), "latency_p99": float64(500)},
		Alerts: []Alert{
			{Severity: "warning", Code: "HIGH_LATENCY", Message: "Latency elevated"},
			{Severity: "error", Code: "ERR_RATE_HIGH", Message: "Error rate above threshold"},
		},
	})

	// Phase 4: Recovery
	agg.RecordPulse(&Pulse{
		PluginID:  "lifecycle-plugin",
		Status:    "HEALTHY",
		Timestamp: baseTime.Add(3 * time.Minute),
		Metrics:   map[string]interface{}{"latency_p50": float64(8), "latency_p99": float64(30)},
	})

	// Verify history
	hist := agg.GetHistory("lifecycle-plugin")
	if len(hist) != 4 {
		t.Fatalf("history length = %d, want 4", len(hist))
	}

	// Verify latest pulse
	got, ok := agg.GetPulse("lifecycle-plugin")
	if !ok {
		t.Fatal("GetPulse returned false")
	}
	if got.Status != "HEALTHY" {
		t.Errorf("latest Status = %q, want %q", got.Status, "HEALTHY")
	}

	// Verify derived metrics
	dm := agg.GetDerivedMetrics("lifecycle-plugin")
	if dm == nil {
		t.Fatal("GetDerivedMetrics returned nil")
	}
	if dm.UptimePercent != 50.0 {
		t.Errorf("UptimePercent = %v, want 50.0 (2 HEALTHY out of 4)", dm.UptimePercent)
	}
	// 1 ERROR over 3 minutes = ~0.333 errors/min
	expectedRate := 1.0 / 3.0
	if dm.ErrorRate != expectedRate {
		t.Errorf("ErrorRate = %v, want %v (1 ERROR over 3 minutes)", dm.ErrorRate, expectedRate)
	}

	// Verify alerts from latest pulse (HEALTHY, no alerts)
	alerts := agg.GetAlerts("lifecycle-plugin")
	if alerts != nil {
		t.Errorf("GetAlerts = %v after recovery, want nil", alerts)
	}

	// Verify registry state
	entry, _ := reg.Get("lifecycle-plugin")
	if entry.State != registry.StateHealthyActive {
		t.Errorf("State = %s, want %s", entry.State, registry.StateHealthyActive)
	}

	// Verify system health status
	shs := agg.SystemHealthStatus()
	if shs["lifecycle-plugin"] != "HEALTHY" {
		t.Errorf("SystemHealthStatus = %q, want %q", shs["lifecycle-plugin"], "HEALTHY")
	}
}