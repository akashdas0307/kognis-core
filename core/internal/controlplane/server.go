package controlplane

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
)

// Server is the gRPC control plane that plugins connect to.
type Server struct {
	listener net.Listener
	server   *grpc.Server
	socket   string
}

// New creates a new control plane gRPC server.
func New(socketPath string) (*Server, error) {
	// Remove existing socket file
	if _, err := os.Stat(socketPath); err == nil {
		os.Remove(socketPath)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("listen on unix socket %s: %w", socketPath, err)
	}

	server := grpc.NewServer()

	return &Server{
		listener: listener,
		server:   server,
		socket:   socketPath,
	}, nil
}

// Server returns the underlying gRPC server for registering services.
func (s *Server) GRPCServer() *grpc.Server {
	return s.server
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
		os.Remove(s.socket)
	}
}