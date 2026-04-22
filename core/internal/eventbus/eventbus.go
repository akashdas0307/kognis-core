package eventbus

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/akashdas0307/kognis-core/core/internal/config"
)

// Bus wraps an embedded NATS server and its client connection.
type Bus struct {
	conn   *nats.Conn
	server *natsserver.Server
	cfg    config.NATSConfig

	mu              sync.RWMutex
	onReconnect     []func()
	onDisconnect    []func(error)
	startedAt       time.Time
}

// Stats holds connection statistics.
type Stats struct {
	TotalMessagesPublished uint64
	TotalMessagesReceived  uint64
	Uptime                 time.Duration
}

// stateMessage is the JSON payload for state broadcast messages (SPEC 06).
type stateMessage struct {
	Timestamp string      `json:"timestamp"`
	Previous  interface{} `json:"previous"`
	Current   interface{} `json:"current"`
	Source    string      `json:"source"`
}

// --- Topic Helper Functions (package-level) ---

// PipelineSubject returns the NATS subject for a pipeline: kognis.pipeline.<name>.
func PipelineSubject(pipelineName string) string {
	return fmt.Sprintf("kognis.pipeline.%s", pipelineName)
}

// SlotSubject returns the NATS subject for a pipeline slot: kognis.pipeline.<name>.slot.<slot>.
func SlotSubject(pipelineName, slotName string) string {
	return fmt.Sprintf("kognis.pipeline.%s.slot.%s", pipelineName, slotName)
}

// HealthSubject returns the NATS subject for a plugin's health: kognis.health.<plugin_id>.
func HealthSubject(pluginID string) string {
	return fmt.Sprintf("kognis.health.%s", pluginID)
}

// StateSubject returns the NATS subject for a plugin state broadcast: state.<plugin_id>.<state_name>.
func StateSubject(pluginID, stateName string) string {
	return fmt.Sprintf("state.%s.%s", pluginID, stateName)
}

// CapabilitySubject returns the NATS subject for capability changes: kognis.capability.changed.
func CapabilitySubject() string {
	return "kognis.capability.changed"
}

// PluginLifecycleSubject returns the NATS subject for plugin lifecycle events: kognis.plugin.lifecycle.
func PluginLifecycleSubject() string {
	return "kognis.plugin.lifecycle"
}

// --- Bus Constructor ---

// New starts an embedded NATS server and connects a client to it.
func New(cfg config.NATSConfig) (*Bus, error) {
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create NATS data directory: %w", err)
	}

	opts := &natsserver.Options{
		ServerName: cfg.ServerName,
		Port:       cfg.Port,
		StoreDir:   cfg.DataDir,
		NoLog:      true,
	}

	server, err := natsserver.NewServer(opts)
	if err != nil {
		return nil, fmt.Errorf("create NATS server: %w", err)
	}

	go server.Start()

	// Wait for server to be ready
	if !server.ReadyForConnections(5 * time.Second) {
		server.Shutdown()
		return nil, fmt.Errorf("NATS server did not start within timeout")
	}

	url := fmt.Sprintf("nats://127.0.0.1:%d", cfg.Port)

	b := &Bus{
		cfg:       cfg,
		startedAt: time.Now(),
	}

	nc, err := nats.Connect(url,
		nats.ReconnectHandler(func(c *nats.Conn) {
			log.Printf("eventbus: reconnected to NATS server %s", c.ConnectedUrl())
			b.mu.RLock()
			handlers := make([]func(), len(b.onReconnect))
			copy(handlers, b.onReconnect)
			b.mu.RUnlock()
			for _, fn := range handlers {
				fn()
			}
		}),
		nats.DisconnectErrHandler(func(c *nats.Conn, err error) {
			if err != nil {
				log.Printf("eventbus: disconnected from NATS server: %v", err)
			}
			b.mu.RLock()
			handlers := make([]func(error), len(b.onDisconnect))
			copy(handlers, b.onDisconnect)
			b.mu.RUnlock()
			for _, fn := range handlers {
				fn(err)
			}
		}),
	)
	if err != nil {
		server.Shutdown()
		return nil, fmt.Errorf("connect to NATS server: %w", err)
	}

	b.conn = nc
	b.server = server

	log.Printf("eventbus: NATS server %s started on port %d", cfg.ServerName, cfg.Port)

	return b, nil
}

// --- Existing Methods (unchanged) ---

// Conn returns the client connection for publishing and subscribing.
func (b *Bus) Conn() *nats.Conn {
	return b.conn
}

// Publish publishes data to a subject.
func (b *Bus) Publish(subject string, data []byte) error {
	return b.conn.Publish(subject, data)
}

// Subscribe subscribes to a subject with a callback.
func (b *Bus) Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	return b.conn.Subscribe(subject, handler)
}

// Close shuts down the client connection and embedded server.
func (b *Bus) Close() {
	if b.conn != nil {
		b.conn.Close()
	}
	if b.server != nil {
		b.server.Shutdown()
		_ = os.RemoveAll(filepath.Join(b.cfg.DataDir, "server.pid"))
	}
}

// --- State Broadcast Helpers (SPEC 06) ---

// PublishState publishes a state change to the state.<plugin_id>.<state_name> topic.
// It skips publishing when oldValue and newValue are equal (string comparison).
// The message payload follows SPEC 06 JSON format: timestamp, previous, current, source.
func (b *Bus) PublishState(pluginID, stateName string, oldValue, newValue interface{}) error {
	// Skip if value has not changed (string comparison)
	oldStr := fmt.Sprintf("%v", oldValue)
	newStr := fmt.Sprintf("%v", newValue)
	if oldStr == newStr {
		return nil
	}

	msg := stateMessage{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Previous:  oldValue,
		Current:   newValue,
		Source:    pluginID,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal state message: %w", err)
	}

	subject := StateSubject(pluginID, stateName)
	return b.conn.Publish(subject, data)
}

// SubscribeState subscribes to state changes on state.<plugin_id>.<state_name>.
func (b *Bus) SubscribeState(pluginID, stateName string, handler nats.MsgHandler) (*nats.Subscription, error) {
	subject := StateSubject(pluginID, stateName)
	return b.conn.Subscribe(subject, handler)
}

// --- Connection Health Tracking ---

// IsConnected returns whether the NATS connection is alive.
func (b *Bus) IsConnected() bool {
	if b.conn == nil {
		return false
	}
	return b.conn.IsConnected()
}

// ConnectionStats returns connection statistics.
func (b *Bus) ConnectionStats() Stats {
	s := Stats{
		Uptime: time.Since(b.startedAt),
	}
	if b.conn != nil {
		cs := b.conn.Stats()
		s.TotalMessagesPublished = cs.OutMsgs
		s.TotalMessagesReceived = cs.InMsgs
	}
	return s
}

// LastError returns the last connection error, or nil if none.
func (b *Bus) LastError() error {
	if b.conn == nil {
		return nil
	}
	return b.conn.LastError()
}

// --- Reconnection Callbacks ---

// OnReconnect registers a callback invoked when the NATS connection reconnects.
func (b *Bus) OnReconnect(fn func()) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.onReconnect = append(b.onReconnect, fn)
}

// OnDisconnect registers a callback invoked when the NATS connection disconnects.
func (b *Bus) OnDisconnect(fn func(error)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.onDisconnect = append(b.onDisconnect, fn)
}

// --- Typed Message Helpers ---

// PublishJSON marshals v to JSON and publishes it on the given subject.
func (b *Bus) PublishJSON(subject string, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal JSON for subject %s: %w", subject, err)
	}
	return b.conn.Publish(subject, data)
}

// --- Graceful Drain ---

// Drain performs a graceful drain of the NATS connection.
func (b *Bus) Drain() error {
	if b.conn == nil {
		return nil
	}
	return b.conn.Drain()
}