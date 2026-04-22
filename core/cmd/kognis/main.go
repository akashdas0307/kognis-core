package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/akashdas0307/kognis-core/core/internal/config"
	"github.com/akashdas0307/kognis-core/core/internal/controlplane"
	"github.com/akashdas0307/kognis-core/core/internal/eventbus"
	"github.com/akashdas0307/kognis-core/core/internal/health"
	"github.com/akashdas0307/kognis-core/core/internal/registry"
	"github.com/akashdas0307/kognis-core/core/internal/router"
	"github.com/akashdas0307/kognis-core/core/internal/supervisor"
)

const version = "0.1.0"
const controlPlaneSocket = "/tmp/kognis.sock"

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "kognis: config error: %v\n", err)
		os.Exit(1)
	}

	// Start embedded NATS event bus
	bus, err := eventbus.New(cfg.NATS)
	if err != nil {
		fmt.Fprintf(os.Stderr, "kognis: event bus error: %v\n", err)
		os.Exit(1)
	}
	defer bus.Close()

	// Initialize plugin registry
	reg := registry.New()

	// Initialize health pulse aggregator
	healthAgg := health.NewAggregator(reg, bus)

	// Initialize handshake manager for control plane
	hm := controlplane.NewHandshakeManager(reg, bus, controlPlaneSocket)

	// Initialize pipeline router
	Router := router.New(reg, bus)

	// Initialize plugin supervisor
	sup := supervisor.New(reg, Router, bus, cfg.Supervisor)

	// Initialize and start the control plane (gRPC server for plugins)
	cpServer, err := controlplane.New(
		controlPlaneSocket,
		controlplane.WithRegistry(reg),
		controlplane.WithBus(bus),
		controlplane.WithHandshakeManager(hm),
		controlplane.WithHealthAggregator(healthAgg),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "kognis: control plane error: %v\n", err)
		os.Exit(1)
	}
	defer cpServer.Close()

	go func() {
		if err := cpServer.Serve(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "kognis: control plane serve error: %v\n", err)
		}
	}()

	fmt.Printf("kognis v%s — core daemon starting\n", version)
	fmt.Printf("  config: %s\n", cfg.Path)
	fmt.Printf("  plugins dir: %s\n", cfg.PluginsDir)
	fmt.Printf("  control plane: %s\n", controlPlaneSocket)

	// Start the supervisor loop
	// This blocks until context is cancelled
	if err := sup.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "kognis: supervisor error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("kognis: shutdown complete")
}
