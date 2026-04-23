package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nats-io/nats.go"

	"github.com/akashdas0307/kognis-core/core/internal/eventbus"
	"github.com/akashdas0307/kognis-core/core/internal/health"
	"github.com/akashdas0307/kognis-core/core/internal/registry"
	"github.com/akashdas0307/kognis-core/core/internal/tui/panels"
)

const refreshInterval = 2 * time.Second

// tickMsg is sent every refreshInterval to trigger data refresh.
type tickMsg time.Time

// Dashboard is the terminal UI for monitoring the Kognis core daemon.
// It wraps a bubbletea Model for backward compatibility.
type Dashboard struct {
	Model model
	bus   *eventbus.Bus
}

// NewDashboard creates a new TUI dashboard.
func NewDashboard(reg *registry.Registry, agg *health.Aggregator, bus *eventbus.Bus) *Dashboard {
	return &Dashboard{
		Model: newModel(reg, agg),
		bus:   bus,
	}
}

// Run starts the dashboard. Blocks until the user quits or context is cancelled.
func (d *Dashboard) Run(ctx context.Context) error {
	p := tea.NewProgram(
		d.Model,
		tea.WithAltScreen(),
		tea.WithContext(ctx),
	)

	// Subscribe to health pulses to trigger real-time updates in the UI
	if d.bus != nil {
		sub, err := d.bus.Subscribe("kognis.health.>", func(msg *nats.Msg) {
			// Sending a tickMsg triggers a re-render in the BubbleTea loop
			p.Send(tickMsg(time.Now()))
		})
		if err == nil {
			defer sub.Unsubscribe()
		}
	}

	_, err := p.Run()
	return err
}

// --- bubbletea Model ---

type model struct {
	registry   *registry.Registry
	aggregator *health.Aggregator
	width      int
	height     int
	activePanel int // 0=overview, 1=plugins, 2=health
	ready      bool
	startTime  time.Time
}

func newModel(reg *registry.Registry, agg *health.Aggregator) model {
	return model{
		registry:    reg,
		aggregator:  agg,
		activePanel: 0,
		startTime:   time.Now(),
	}
}

// NewModel creates a bubbletea model for the dashboard (exported for tests).
func NewModel(reg *registry.Registry, agg *health.Aggregator) tea.Model {
	return newModel(reg, agg)
}

// Init implements tea.Model.
func (m model) Init() tea.Cmd {
	return tickCmd()
}

// Update implements tea.Model.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case tickMsg:
		// Refresh data — just re-render by returning the model unchanged.
		// The View() method reads live from registry/aggregator each time.
		return m, tickCmd()

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.activePanel = (m.activePanel + 1) % 3
			return m, nil
		case "1":
			m.activePanel = 0
			return m, nil
		case "2":
			m.activePanel = 1
			return m, nil
		case "3":
			m.activePanel = 2
			return m, nil
		}
	}

	return m, nil
}

// View implements tea.Model.
func (m model) View() string {
	if !m.ready {
		return "\n  Initializing Kognis Dashboard..."
	}

	// Styles
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		MarginBottom(1)

	tabActiveStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1)

	tabInactiveStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Padding(0, 1)

	// Header
	header := headerStyle.Render("Kognis Core Dashboard")

	// Tab bar
	tabNames := []string{"Overview", "Plugins", "Health"}
	var tabs []string
	for i, name := range tabNames {
		if i == m.activePanel {
			tabs = append(tabs, tabActiveStyle.Render(name))
		} else {
			tabs = append(tabs, tabInactiveStyle.Render(name))
		}
	}
	tabBar := lipgloss.JoinHorizontal(lipgloss.Bottom, tabs...)

	// Active panel content
	var content string
	switch m.activePanel {
	case 0:
		content = panels.RenderOverview(m.registry, m.aggregator, m.startTime, m.width)
	case 1:
		content = panels.RenderPlugins(m.registry, m.width)
	case 2:
		content = panels.RenderHealth(m.aggregator, m.width)
	}

	// Key hints
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
	hints := hintStyle.Render("[1/2/3] Switch Panel  [Tab] Next  [q] Quit")

	// Layout
	return fmt.Sprintf("%s\n%s\n\n%s\n\n%s", header, tabBar, content, hints)
}

func tickCmd() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Ensure model implements tea.Model at compile time.
var _ tea.Model = model{}

// FormatDuration returns a human-readable duration string.
func FormatDuration(d time.Duration) string {
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