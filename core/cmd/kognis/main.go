package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/kognis-framework/kognis-core/core/internal/config"
	"github.com/kognis-framework/kognis-core/core/internal/eventbus"
	"github.com/kognis-framework/kognis-core/core/internal/registry"
	"github.com/kognis-framework/kognis-core/core/internal/router"
	"github.com/kognis-framework/kognis-core/core/internal/supervisor"
)

const version = "0.1.0"

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

	// Initialize pipeline router
	Router := router.New(reg, bus)

	// Initialize plugin supervisor
	sup := supervisor.New(reg, Router, bus, cfg.Supervisor)

	fmt.Printf("kognis v%s — core daemon starting\n", version)
	fmt.Printf("  config: %s\n", cfg.Path)
	fmt.Printf("  plugins dir: %s\n", cfg.PluginsDir)

	// Start the control plane (gRPC server for plugins)
	// This blocks until context is cancelled
	if err := sup.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "kognis: supervisor error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("kognis: shutdown complete")
}