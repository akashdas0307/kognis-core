package main

import (
	"fmt"
	"os"

	"github.com/akashdas0307/kognis-core/core/internal/config"
	"github.com/akashdas0307/kognis-core/core/internal/eventbus"
	"github.com/akashdas0307/kognis-core/core/internal/health"
	"github.com/akashdas0307/kognis-core/core/internal/registry"
	"github.com/akashdas0307/kognis-core/core/internal/tui"
)

func main() {
	fmt.Println("--- TUI Manual Verification Pass ---")

	// 1. Initialize components
	reg := registry.New()
	
	// Create NATS config
	natsCfg := config.NATSConfig{
		Port:       4223, // Use different port
		DataDir:    "/tmp/kognis-nats-test",
		ServerName: "verify-tui",
	}

	bus, err := eventbus.New(natsCfg)
	if err != nil {
		fmt.Printf("Failed to init eventbus: %v\n", err)
		os.Exit(1)
	}
	agg := health.NewAggregator(reg, bus)

	// 2. Instantiate TUI Dashboard
	dash := tui.NewDashboard(reg, agg, bus)

	if dash == nil {
		fmt.Println("❌ TUI Dashboard failed to initialize")
		os.Exit(1)
	}
	fmt.Println("✅ TUI Dashboard initialized successfully")

	// 3. Verify Render capability
	view := dash.Model.View() // Model is a field

	if view == "" {
		fmt.Println("❌ TUI View() is empty")
		os.Exit(1)
	}
	fmt.Println("✅ TUI rendering verified (non-empty view)")
	fmt.Println("--- Verification Complete ---")
}
