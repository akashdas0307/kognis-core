package panels

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/akashdas0307/kognis-core/core/internal/registry"
)

// RenderPlugins renders the plugins panel showing a table of all registered plugins.
func RenderPlugins(reg *registry.Registry, width int) string {
	plugins := reg.List()

	if len(plugins) == 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("  No plugins registered.")
	}

	// Column widths
	idW := 14
	nameW := 20
	verW := 8
	stateW := 16
	heartbeatW := 12
	restartW := 8

	// Header
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	header := headerStyle.Render(fmt.Sprintf("  %s  %s  %s  %s  %s  %s",
		padCol("ID", idW),
		padCol("Name", nameW),
		padCol("Version", verW),
		padCol("State", stateW),
		padCol("Heartbeat", heartbeatW),
		padCol("Restarts", restartW),
	))

	// Separator
	sep := "  " + strings.Repeat("-", min(width-4, idW+nameW+verW+stateW+heartbeatW+restartW+10))

	// Rows
	var rows []string
	for _, p := range plugins {
		stateStyle := stateColor(p.State)
		row := fmt.Sprintf("  %s  %s  %s  %s  %s  %s",
			padCol(p.ID, idW),
			padCol(p.Name, nameW),
			padCol(p.Version, verW),
			stateStyle.Render(padCol(string(p.State), stateW)),
			padCol(formatTimeAgo(p.LastHeartbeat), heartbeatW),
			padCol(fmt.Sprintf("%d", p.RestartCount), restartW),
		)
		rows = append(rows, row)
	}

	return header + "\n" + sep + "\n" + strings.Join(rows, "\n")
}

func stateColor(state registry.PluginState) lipgloss.Style {
	switch state {
	case registry.StateHealthyActive:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	case registry.StateUnhealthy:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B"))
	case registry.StateUnresponsive:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8C00"))
	case registry.StateCircuitOpen:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4444"))
	case registry.StateDead:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
	case registry.StateStarting:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700"))
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	}
}

func formatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	d := time.Since(t)
	d = d.Truncate(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

func padCol(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}