package ui

import (
	"fmt"
	"log"
	"math/rand"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/omnitop/internal/config"
	"github.com/google/omnitop/internal/metrics"
)

type TickMsg time.Time

var tickRand = rand.New(rand.NewSource(time.Now().UnixNano()))

func tick(interval int) tea.Cmd {
	// Add jitter: 90-110% of interval
	base := time.Duration(interval) * time.Millisecond
	jitter := time.Duration(tickRand.Intn(int(base/5)) - int(base/10))
	return tea.Tick(base+jitter, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

type RootModel struct {
	provider metrics.Provider
	config   *config.ProfileConfiguration

	// Sub-models
	gpu     GPUModel
	process ProcessModel
	cpu     CPUModel
	footer  FooterModel

	// Layout state
	width, height int
	col1Pct       float64 // Percentage of width for Left Column (GPU)
	col2Pct       float64 // Percentage of width for Middle Column (Process)
	// Right column takes remaining

	// Tooltip state
	mouseX, mouseY int
	showTooltip    bool

	// Alert state
	lastAlert time.Time
}

func NewRootModel(provider metrics.Provider, cfg *config.ProfileConfiguration) RootModel {
	return RootModel{
		provider:    provider,
		config:      cfg,
		gpu:         NewGPUModel(),
		process:     NewProcessModel(),
		cpu:         NewCPUModel(),
		footer:      NewFooterModel(),
		col1Pct:     cfg.ColumnWidths["gpu"],
		col2Pct:     cfg.ColumnWidths["process"],
		showTooltip: cfg.ShowTooltips,
	}
}

func (m RootModel) Init() tea.Cmd {
	return tick(m.config.RefreshInterval)
}

func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			// Save config on exit
			m.config.ColumnWidths["gpu"] = m.col1Pct
			m.config.ColumnWidths["process"] = m.col2Pct
			m.config.ColumnWidths["cpu"] = 1.0 - m.col1Pct - m.col2Pct
			// Best effort save to profiles.json
			if err := config.SaveConfig("profiles.json", m.config); err != nil {
				log.Printf("Failed to save config: %v", err)
			}
			return m, tea.Quit
		case "[": // Shrink Left Col
			m.col1Pct -= 0.05
			if m.col1Pct < 0.1 {
				m.col1Pct = 0.1
			}
			m.resizeModules()
		case "]": // Expand Left Col
			m.col1Pct += 0.05
			if m.col1Pct+m.col2Pct > 0.9 {
				m.col1Pct = 0.9 - m.col2Pct
			}
			m.resizeModules()
		case "{": // Shrink Middle Col (effectively expands Right)
			m.col2Pct -= 0.05
			if m.col2Pct < 0.1 {
				m.col2Pct = 0.1
			}
			m.resizeModules()
		case "}": // Expand Middle Col
			m.col2Pct += 0.05
			if m.col1Pct+m.col2Pct > 0.9 {
				m.col2Pct = 0.9 - m.col1Pct
			}
			m.resizeModules()
		case "t": // Toggle Tooltips
			m.showTooltip = !m.showTooltip
		}

		// Pass keys to sub-models
		m.process, cmd = m.process.Update(msg)
		cmds = append(cmds, cmd)
		m.gpu, cmd = m.gpu.Update(msg) // GPU toggle process list
		cmds = append(cmds, cmd)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeModules()

	case TickMsg:
		// Fetch metrics
		stats, err := m.provider.GetStats()
		if err == nil {
			m.gpu.SetStats(stats.GPU)
			m.process.SetStats(stats.Processes)
			m.cpu.SetStats(*stats)
			m.footer.SetStats(stats)

			// Check Alerts
			m.checkAlerts(stats)
		}
		// Continue tick
		cmds = append(cmds, tick(m.config.RefreshInterval))

	case tea.MouseMsg:
		m.mouseX = msg.X
		m.mouseY = msg.Y
		// Pass mouse to sub-models
		m.process, cmd = m.process.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *RootModel) checkAlerts(stats *metrics.SystemStats) {
	// Simple rate limiting: 1 alert per minute
	if time.Since(m.lastAlert) < time.Minute {
		return
	}

	var alerts []string

	if stats.CPU.GlobalUsagePercent > 90 {
		alerts = append(alerts, fmt.Sprintf("High CPU Usage: %.1f%%", stats.CPU.GlobalUsagePercent))
	}
	if stats.Memory.UsedPercent > 90 {
		alerts = append(alerts, fmt.Sprintf("High Memory Usage: %.1f%%", stats.Memory.UsedPercent))
	}
	if stats.GPU.Available {
		if stats.GPU.Utilization > 95 {
			alerts = append(alerts, fmt.Sprintf("High GPU Usage: %d%%", stats.GPU.Utilization))
		}
		if stats.GPU.Temperature > 85 {
			alerts = append(alerts, fmt.Sprintf("High GPU Temp: %d°C", stats.GPU.Temperature))
		}
	}

	if len(alerts) > 0 {
		m.lastAlert = time.Now()
		// Send notification
		msg := "System Alert: " + alerts[0]
		if len(alerts) > 1 {
			msg += fmt.Sprintf(" (+%d more)", len(alerts)-1)
		}
		// Fire and forget
		go func() {
			_ = exec.Command("notify-send", "OmniTop Alert", msg, "-u", "critical").Run()
		}()
	}
}

func (m *RootModel) resizeModules() {
	if m.width == 0 || m.height == 0 {
		return
	}

	// Calculate widths
	w1 := int(float64(m.width) * m.col1Pct)
	w2 := int(float64(m.width) * m.col2Pct)
	w3 := m.width - w1 - w2

	// Height available for columns (minus footer)
	h := m.height - 1
	if h < 1 {
		h = 1
	}

	m.gpu.SetSize(w1, h)
	m.process.SetSize(w2, h)
	m.cpu.SetSize(w3, h)
	m.footer.SetSize(m.width)
}

func (m RootModel) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	// Render columns
	cols := lipgloss.JoinHorizontal(lipgloss.Top,
		m.gpu.View(),
		m.process.View(),
		m.cpu.View(),
	)

	// Add Footer
	footerView := m.footer.View()

	// Tooltip (Appended to footer for MVP)
	if m.showTooltip {
		tooltip := m.getTooltipText()
		if tooltip != "" {
			tStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorPaleBlue)).
				Background(lipgloss.Color(ColorMidnightBlack)).
				Padding(0, 1).
				Border(lipgloss.NormalBorder(), true, false, false, false).
				BorderForeground(lipgloss.Color(ColorSteelGray))

			footerView = lipgloss.JoinVertical(lipgloss.Left, footerView, tStyle.Render("INFO: "+tooltip))
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		cols,
		footerView,
	)
}

func (m RootModel) getTooltipText() string {
	// Determine column based on mouseX
	w1 := int(float64(m.width) * m.col1Pct)
	w2 := int(float64(m.width) * m.col2Pct)

	if m.mouseX < w1 {
		return "GPU Panel: Shows NVIDIA GPU utilization, VRAM usage, and temps. Press 'g' to toggle process view."
	} else if m.mouseX < w1+w2 {
		return "Process Panel: Sortable list of running processes. Use 'k' to kill, 'c/m/p' to sort."
	} else {
		return "CPU Panel: Per-core usage bars and system load averages."
	}
}
