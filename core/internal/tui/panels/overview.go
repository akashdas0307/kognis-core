package panels

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/akashdas0307/kognis-core/core/internal/health"
	"github.com/akashdas0307/kognis-core/core/internal/registry"
)

// RenderOverview renders the overview panel showing system-level metrics.
func RenderOverview(reg *registry.Registry, agg *health.Aggregator, startTime time.Time, width int) string {
	plugins := reg.List()
	capabilities := reg.FindAvailableCapabilities()
	pulses := agg.AllPulses()

	activeCount := 0
	for _, p := range plugins {
		if p.State == registry.StateHealthyActive {
			activeCount++
		}
	}

	healthyPulseCount := 0
	for _, pulse := range pulses {
		if pulse.Status == "HEALTHY" {
			healthyPulseCount++
		}
	}

	uptime := time.Since(startTime)

	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	valueStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF"))

	row := func(label, value string) string {
		return fmt.Sprintf("  %s  %s", labelStyle.Render(label), valueStyle.Render(value))
	}

	s := ""
	s += row("Total Plugins:", fmt.Sprintf("%d", len(plugins))) + "\n"
	s += row("Active Plugins:", fmt.Sprintf("%d", activeCount)) + "\n"
	s += row("Available Capabilities:", fmt.Sprintf("%d", len(capabilities))) + "\n"
	s += row("Health Pulses:", fmt.Sprintf("%d (healthy: %d)", len(pulses), healthyPulseCount)) + "\n"
	s += row("Uptime:", formatUptime(uptime)) + "\n"

	if len(capabilities) > 0 {
		s += "\n"
		s += labelStyle.Render("  Capabilities:") + "\n"
		for _, capID := range capabilities {
			s += fmt.Sprintf("    - %s\n", capID)
		}
	}

	return s
}

func formatUptime(d time.Duration) string {
	d = d.Truncate(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dh%dm%ds", h, m, s)
}