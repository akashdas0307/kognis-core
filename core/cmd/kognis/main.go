package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/akashdas0307/kognis-core/core/internal/config"
	"github.com/akashdas0307/kognis-core/core/internal/controlplane"
	"github.com/akashdas0307/kognis-core/core/internal/eventbus"
	"github.com/akashdas0307/kognis-core/core/internal/health"
	"github.com/akashdas0307/kognis-core/core/internal/registry"
	"github.com/akashdas0307/kognis-core/core/internal/router"
	"github.com/akashdas0307/kognis-core/core/internal/supervisor"
	"github.com/akashdas0307/kognis-core/core/internal/tui"
)

const version = "0.1.0"
const controlPlaneSocket = "/tmp/kognis.sock"

func main() {
	
	tuiEnabled := flag.Bool("tui", false, "Enable TUI dashboard")
	flag.Parse()

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
	msgRouter := router.NewMessageRouter(Router, reg, bus)
	capRouter := router.NewCapabilityRouter(bus, reg)

	// Start timeout checker loops for routers
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				msgRouter.CheckTimeouts()
				capRouter.CheckTimeouts()
			}
		}
	}()

	// Initialize plugin supervisor
	sup := supervisor.New(reg, Router, bus, cfg.Supervisor)

	// Discover and spawn initial plugins
	if err := sup.DiscoverAndSpawn(cfg.PluginsDir); err != nil {
		fmt.Fprintf(os.Stderr, "kognis: failed to discover plugins: %v\n", err)
	}

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

	// Start control plane serving in background
	go func() {
		if err := cpServer.Serve(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "kognis: control plane serve error: %v\n", err)
		}
	}()

	// Wait for socket to exist before starting supervisor
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(controlPlaneSocket); err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Printf("kognis v%s — core daemon starting\n", version)
	fmt.Printf("  config: %s\n", cfg.Path)
	fmt.Printf("  plugins dir: %s\n", cfg.PluginsDir)
	fmt.Printf("  control plane: %s\n", controlPlaneSocket)

	// Start the supervisor loop
	// If TUI is enabled, run it alongside
	if *tuiEnabled {
		dash := tui.NewDashboard(reg, healthAgg, bus)
		go func() {
			if err := dash.Run(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "kognis: tui error: %v\n", err)
			}
		}()
	}

	// This blocks until context is cancelled
	if err := sup.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "kognis: supervisor error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("kognis: shutdown complete")
}
