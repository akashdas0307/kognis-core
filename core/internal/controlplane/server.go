package controlplane

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"

	"github.com/akashdas0307/kognis-core/core/internal/eventbus"
	"github.com/akashdas0307/kognis-core/core/internal/health"
	"github.com/akashdas0307/kognis-core/core/internal/registry"
)

// Server is the gRPC control plane that plugins connect to.
type Server struct {
	listener net.Listener
	server   *grpc.Server
	socket   string
	service  *ControlPlaneService
}

// ServerOption configures a Server during construction.
type ServerOption func(*serverConfig)

// serverConfig holds optional configuration for the Server.
type serverConfig struct {
	registry  *registry.Registry
	bus       *eventbus.Bus
	healthAgg *health.Aggregator
	handshake *HandshakeManager
}

// WithRegistry sets the plugin registry for the control plane.
func WithRegistry(reg *registry.Registry) ServerOption {
	return func(c *serverConfig) { c.registry = reg }
}

// WithBus sets the event bus for the control plane.
func WithBus(bus *eventbus.Bus) ServerOption {
	return func(c *serverConfig) { c.bus = bus }
}

// WithHealthAggregator sets the health aggregator for the control plane.
func WithHealthAggregator(agg *health.Aggregator) ServerOption {
	return func(c *serverConfig) { c.healthAgg = agg }
}

// WithHandshakeManager sets the handshake manager for the control plane.
func WithHandshakeManager(hm *HandshakeManager) ServerOption {
	return func(c *serverConfig) { c.handshake = hm }
}

// New creates a new control plane gRPC server.
// The functional options pattern allows backward compatibility: callers who
// only pass socketPath get the same behavior as before. When registry, bus,
// and handshake manager are provided via options, the ControlPlaneService is
// automatically registered.
func New(socketPath string, opts ...ServerOption) (*Server, error) {
	// Apply options
	cfg := &serverConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	// Remove existing socket file
	if _, err := os.Stat(socketPath); err == nil {
		_ = os.Remove(socketPath)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("listen on unix socket %s: %w", socketPath, err)
	}

	server := grpc.NewServer()

	s := &Server{
		listener: listener,
		server:   server,
		socket:   socketPath,
	}

	// Register the ControlPlaneService if the required dependencies are provided.
	if cfg.registry != nil {
		svc := NewControlPlaneService(cfg.registry, cfg.bus, cfg.healthAgg, cfg.handshake)
		s.service = svc
		RegisterControlPlaneServer(server, svc)
		log.Printf("controlplane: ControlPlaneService registered on %s", socketPath)
	}

	return s, nil
}

// GRPCServer returns the underlying gRPC server for registering services.
func (s *Server) GRPCServer() *grpc.Server {
	return s.server
}

// Service returns the ControlPlaneService, or nil if not registered.
func (s *Server) Service() *ControlPlaneService {
	return s.service
}

// Serve starts accepting gRPC connections. Blocks until context is cancelled.
func (s *Server) Serve(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		log.Println("controlplane: stopping gRPC server")
		s.server.GracefulStop()
	}()

	log.Printf("controlplane: listening on %s", s.socket)
	if err := s.server.Serve(s.listener); err != nil {
		return fmt.Errorf("gRPC serve: %w", err)
	}
	return nil
}

// Close cleans up the control plane.
func (s *Server) Close() {
	if s.server != nil {
		s.server.GracefulStop()
	}
	if s.socket != "" {
		_ = os.Remove(s.socket)
	}
}