package panels

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/akashdas0307/kognis-core/core/internal/health"
)

// RenderHealth renders the health panel showing pulse status for all plugins.
func RenderHealth(agg *health.Aggregator, width int) string {
	pulses := agg.AllPulses()

	if len(pulses) == 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("  No health pulses received.")
	}

	// Sort by plugin ID for deterministic output
	ids := make([]string, 0, len(pulses))
	for id := range pulses {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	// Column widths
	idW := 16
	statusW := 14
	latencyW := 12
	memW := 12
	activityW := 20

	// Header
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	header := headerStyle.Render(fmt.Sprintf("  %s  %s  %s  %s  %s",
		padCol("Plugin ID", idW),
		padCol("Status", statusW),
		padCol("Latency", latencyW),
		padCol("Memory", memW),
		padCol("Activity", activityW),
	))

	// Separator
	sep := "  " + strings.Repeat("-", min(width-4, idW+statusW+latencyW+memW+activityW+8))

	// Rows
	var rows []string
	for _, id := range ids {
		pulse := pulses[id]
		statusStyle := pulseStatusColor(pulse.Status)

		latencyStr := extractMetric(pulse.Metrics, "latency_p50", "ms")
		memStr := extractMetric(pulse.Metrics, "memory", "MB")
		activity := pulse.CurrentActivity
		if activity == "" {
			activity = "-"
		}

		row := fmt.Sprintf("  %s  %s  %s  %s  %s",
			padCol(pulse.PluginID, idW),
			statusStyle.Render(padCol(pulse.Status, statusW)),
			padCol(latencyStr, latencyW),
			padCol(memStr, memW),
			padCol(activity, activityW),
		)
		rows = append(rows, row)

		// Show alerts if present
		for _, alert := range pulse.Alerts {
			alertStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B"))
			rows = append(rows, alertStyle.Render(fmt.Sprintf("    [%s] %s: %s", alert.Severity, alert.Code, alert.Message)))
		}
	}

	return header + "\n" + sep + "\n" + strings.Join(rows, "\n")
}

// extractMetric extracts a metric value from the metrics map and formats it.
func extractMetric(metrics map[string]interface{}, key, unit string) string {
	if metrics == nil {
		return "-"
	}
	v, ok := metrics[key]
	if !ok {
		return "-"
	}
	switch n := v.(type) {
	case float64:
		if unit == "MB" {
			return fmt.Sprintf("%.0f %s", n, unit)
		}
		return fmt.Sprintf("%.0f %s", n, unit)
	case int:
		return fmt.Sprintf("%d %s", n, unit)
	case int64:
		return fmt.Sprintf("%d %s", n, unit)
	default:
		return "-"
	}
}

func pulseStatusColor(status string) lipgloss.Style {
	switch status {
	case "HEALTHY":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	case "DEGRADED":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700"))
	case "ERROR":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B"))
	case "CRITICAL":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
	case "UNRESPONSIVE":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8C00"))
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	}
}