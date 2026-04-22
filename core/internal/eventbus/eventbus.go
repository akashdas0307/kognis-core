package eventbus

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/nats-io/nats.go"
	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/kognis-framework/kognis-core/core/internal/config"
)

// Bus wraps an embedded NATS server and its client connection.
type Bus struct {
	conn   *nats.Conn
	server *natsserver.Server
	cfg    config.NATSConfig
}

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
	nc, err := nats.Connect(url)
	if err != nil {
		server.Shutdown()
		return nil, fmt.Errorf("connect to NATS server: %w", err)
	}

	log.Printf("eventbus: NATS server %s started on port %d", cfg.ServerName, cfg.Port)

	return &Bus{
		conn:   nc,
		server: server,
		cfg:    cfg,
	}, nil
}

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