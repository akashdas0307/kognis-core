package eventbus

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/akashdas0307/kognis-core/core/internal/config"
)

// newTestBus creates a Bus with a unique port for each test.
func newTestBus(t *testing.T, port int) *Bus {
	t.Helper()
	cfg := config.NATSConfig{
		ServerName: "test",
		Port:       port,
		DataDir:    t.TempDir(),
	}
	bus, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	t.Cleanup(func() { bus.Close() })
	return bus
}

func TestNew_StartsServerAndConnects(t *testing.T) {
	bus := newTestBus(t, 14222)

	if !bus.IsConnected() {
		t.Error("IsConnected() = false after New(); want true")
	}
	if bus.Conn() == nil {
		t.Error("Conn() = nil; want non-nil")
	}
}

func TestPublishSubscribe_BasicMessageFlow(t *testing.T) {
	bus := newTestBus(t, 14223)

	ch := make(chan []byte, 1)

	_, err := bus.Subscribe("test.basic", func(msg *nats.Msg) {
		ch <- msg.Data
	})
	if err != nil {
		t.Fatalf("Subscribe() error: %v", err)
	}

	payload := []byte("hello eventbus")
	if err := bus.Publish("test.basic", payload); err != nil {
		t.Fatalf("Publish() error: %v", err)
	}

	select {
	case got := <-ch:
		if string(got) != string(payload) {
			t.Errorf("received %q; want %q", got, payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestPublishState_CorrectTopicAndJSON(t *testing.T) {
	bus := newTestBus(t, 14224)

	ch := make(chan []byte, 1)

	// Subscribe to the state topic: state.myplugin.mystate
	_, err := bus.SubscribeState("myplugin", "mystate", func(msg *nats.Msg) {
		ch <- msg.Data
	})
	if err != nil {
		t.Fatalf("SubscribeState() error: %v", err)
	}

	if err := bus.PublishState("myplugin", "mystate", "idle", "reasoning"); err != nil {
		t.Fatalf("PublishState() error: %v", err)
	}

	select {
	case data := <-ch:
		var msg stateMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			t.Fatalf("unmarshal state message: %v", err)
		}
		if msg.Source != "myplugin" {
			t.Errorf("Source = %q; want %q", msg.Source, "myplugin")
		}
		if msg.Previous != "idle" {
			t.Errorf("Previous = %v; want %q", msg.Previous, "idle")
		}
		if msg.Current != "reasoning" {
			t.Errorf("Current = %v; want %q", msg.Current, "reasoning")
		}
		if msg.Timestamp == "" {
			t.Error("Timestamp is empty; want RFC3339 timestamp")
		}
		// Verify timestamp is valid RFC3339
		if _, err := time.Parse(time.RFC3339, msg.Timestamp); err != nil {
			t.Errorf("Timestamp %q is not valid RFC3339: %v", msg.Timestamp, err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for state message")
	}
}

func TestPublishState_SkipsWhenValuesEqual(t *testing.T) {
	bus := newTestBus(t, 14225)

	received := make(chan []byte, 1)

	_, err := bus.SubscribeState("eqplugin", "eqstate", func(msg *nats.Msg) {
		received <- msg.Data
	})
	if err != nil {
		t.Fatalf("SubscribeState() error: %v", err)
	}

	// Publish with oldValue == newValue (string comparison) — should be skipped
	if err := bus.PublishState("eqplugin", "eqstate", "same", "same"); err != nil {
		t.Fatalf("PublishState() error: %v", err)
	}

	select {
	case <-received:
		t.Error("received message when oldValue == newValue; should have been skipped")
	case <-time.After(500 * time.Millisecond):
		// Expected: no message received
	}

	// Now publish with different values — should arrive
	if err := bus.PublishState("eqplugin", "eqstate", "old", "new"); err != nil {
		t.Fatalf("PublishState() error: %v", err)
	}

	select {
	case data := <-received:
		var msg stateMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			t.Fatalf("unmarshal state message: %v", err)
		}
		if msg.Current != "new" {
			t.Errorf("Current = %v; want %q", msg.Current, "new")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for state message with different values")
	}
}

func TestSubscribeState_ReceivesStateChanges(t *testing.T) {
	bus := newTestBus(t, 14226)

	ch := make(chan string, 2)

	_, err := bus.SubscribeState("core", "activity_state", func(msg *nats.Msg) {
		var sm stateMessage
		if err := json.Unmarshal(msg.Data, &sm); err == nil {
			ch <- sm.Current.(string)
		}
	})
	if err != nil {
		t.Fatalf("SubscribeState() error: %v", err)
	}

	if err := bus.PublishState("core", "activity_state", "idle", "reasoning"); err != nil {
		t.Fatalf("PublishState() error: %v", err)
	}

	select {
	case val := <-ch:
		if val != "reasoning" {
			t.Errorf("got %q; want %q", val, "reasoning")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for state change")
	}
}

func TestTopicHelperFunctions(t *testing.T) {
	tests := []struct {
		name  string
		got   string
		want  string
	}{
		{"PipelineSubject", PipelineSubject("cognition"), "kognis.pipeline.cognition"},
		{"SlotSubject", SlotSubject("cognition", "perception"), "kognis.pipeline.cognition.slot.perception"},
		{"HealthSubject", HealthSubject("cognitive_core"), "kognis.health.cognitive_core"},
		{"StateSubject", StateSubject("persona", "emotional_state"), "state.persona.emotional_state"},
		{"CapabilitySubject", CapabilitySubject(), "kognis.capability.changed"},
		{"PluginLifecycleSubject", PluginLifecycleSubject(), "kognis.plugin.lifecycle"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %q; want %q", tt.got, tt.want)
			}
		})
	}
}

func TestIsConnected(t *testing.T) {
	bus := newTestBus(t, 14227)

	if !bus.IsConnected() {
		t.Error("IsConnected() = false; want true")
	}

	bus.Close()

	if bus.IsConnected() {
		t.Error("IsConnected() = true after Close(); want false")
	}
}

func TestConnectionStats(t *testing.T) {
	bus := newTestBus(t, 14228)

	stats := bus.ConnectionStats()
	if stats.Uptime <= 0 {
		t.Error("Uptime <= 0; want positive duration")
	}
	// A freshly created bus may have zero published/received messages; that's fine.
	// Just ensure the struct is populated.
	_ = stats.TotalMessagesPublished
	_ = stats.TotalMessagesReceived
}

func TestPublishJSON(t *testing.T) {
	bus := newTestBus(t, 14229)

	ch := make(chan []byte, 1)

	_, err := bus.Subscribe("test.json", func(msg *nats.Msg) {
		ch <- msg.Data
	})
	if err != nil {
		t.Fatalf("Subscribe() error: %v", err)
	}

	type testPayload struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}
	payload := testPayload{Name: "test", Value: 42}

	if err := bus.PublishJSON("test.json", payload); err != nil {
		t.Fatalf("PublishJSON() error: %v", err)
	}

	select {
	case data := <-ch:
		var got testPayload
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("unmarshal JSON: %v", err)
		}
		if got.Name != "test" || got.Value != 42 {
			t.Errorf("got {%q, %d}; want {%q, %d}", got.Name, got.Value, "test", 42)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for JSON message")
	}
}

func TestDrain(t *testing.T) {
	cfg := config.NATSConfig{
		ServerName: "test",
		Port:       14230,
		DataDir:    t.TempDir(),
	}
	bus, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	if err := bus.Drain(); err != nil {
		t.Errorf("Drain() error: %v", err)
	}

	// Clean up after drain
	bus.Close()
}