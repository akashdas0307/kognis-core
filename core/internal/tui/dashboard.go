package tui

import (
	"context"
	"fmt"
	"log"

	"github.com/akashdas0307/kognis-core/core/internal/health"
	"github.com/akashdas0307/kognis-core/core/internal/registry"
)

// Dashboard is the terminal UI for monitoring the Kognis core daemon.
type Dashboard struct {
	registry   *registry.Registry
	aggregator *health.Aggregator
}

// NewDashboard creates a new TUI dashboard.
func NewDashboard(reg *registry.Registry, agg *health.Aggregator) *Dashboard {
	return &Dashboard{
		registry:   reg,
		aggregator: agg,
	}
}

// Run starts the dashboard. Blocks until context is cancelled.
func (d *Dashboard) Run(ctx context.Context) error {
	log.Println("tui: dashboard starting")

	for {
		select {
		case <-ctx.Done():
			log.Println("tui: dashboard shutting down")
			return nil
		default:
			// TODO: implement bubbletea program
			// For now, log status summary
			plugins := d.registry.List()
			pulses := d.aggregator.AllPulses()

			fmt.Printf("\n--- Kognis Dashboard ---\n")
			fmt.Printf("Plugins: %d\n", len(plugins))
			for _, p := range plugins {
				fmt.Printf("  %s (%s) - %s\n", p.Name, p.Version, p.State)
			}
			fmt.Printf("Health Pulses: %d\n", len(pulses))
			fmt.Printf("------------------------\n")

			return nil
		}
	}
}