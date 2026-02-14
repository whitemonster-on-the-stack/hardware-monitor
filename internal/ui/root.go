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
	// Add jitter
	jitter := time.Duration(tickRand.Intn(200)-100) * time.Millisecond
	base := time.Duration(interval) * time.Millisecond
	if base <= 0 {
		base = time.Second
	}
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
	tooltipContent string
	lastAlertTime  time.Time
}

func NewRootModel(provider metrics.Provider, cfg *config.ProfileConfiguration) RootModel {
	// Defaults if config is missing values
	col1 := 0.30
	col2 := 0.40
	if cfg != nil {
		if v, ok := cfg.ColumnWidths["gpu"]; ok {
			col1 = v
		}
		if v, ok := cfg.ColumnWidths["process"]; ok {
			col2 = v
		}
	}

	return RootModel{
		provider: provider,
		config:   cfg,
		gpu:      NewGPUModel(),
		process:  NewProcessModel(),
		cpu:      NewCPUModel(),
		footer:   NewFooterModel(),
		col1Pct:  col1,
		col2Pct:  col2,
	}
}

func (m RootModel) Init() tea.Cmd {
	interval := 1000
	if m.config != nil {
		interval = m.config.RefreshInterval
	}
	return tick(interval)
}

func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			// Save config on exit
			if m.config != nil {
				m.config.ColumnWidths["gpu"] = m.col1Pct
				m.config.ColumnWidths["process"] = m.col2Pct
				m.config.ColumnWidths["cpu"] = 1.0 - m.col1Pct - m.col2Pct
				// Best effort save to profiles.json
				if err := config.SaveConfig("profiles.json", m.config); err != nil {
					log.Printf("Failed to save config: %v", err)
				}
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
			m.process.SetStats(*stats)
			m.cpu.SetStats(*stats)
			m.checkAlerts(stats)
		}
		// Continue tick
		interval := 1000
		if m.config != nil {
			interval = m.config.RefreshInterval
		}
		cmds = append(cmds, tick(interval))

	case tea.MouseMsg:
		m.mouseX = msg.X
		m.mouseY = msg.Y

		// Tooltip Logic
		if m.config != nil && m.config.ShowTooltips {
			m.updateTooltip()
		}

		// Pass mouse to sub-models
		m.process, cmd = m.process.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *RootModel) checkAlerts(stats *metrics.SystemStats) {
	if m.config == nil {
		return
	}

	alerts := []string{}
	cpuAlert := false
	gpuAlert := false
	memAlert := false

	// Check CPU Global
	if stats.CPU.GlobalUsagePercent > m.config.AlertThresholds.CPUUsagePercent {
		cpuAlert = true
		alerts = append(alerts, fmt.Sprintf("CPU Load %.0f%%", stats.CPU.GlobalUsagePercent))
	}
	// Check CPU Individual Cores (Sample check: if any core is > threshold)
	for _, usage := range stats.CPU.PerCoreUsage {
		if usage > m.config.AlertThresholds.CPUUsagePercent {
			cpuAlert = true
			// Avoid spamming alert message, just generic CPU High
			break
		}
	}
	// Check CPU Temps
	for _, temp := range stats.CPU.PerCoreTemp {
		if temp > m.config.AlertThresholds.CPUTempCelsius {
			cpuAlert = true
			alerts = append(alerts, fmt.Sprintf("CPU Temp %.0fC", temp))
			break
		}
	}

	// Check GPU
	if stats.GPU.Available {
		if float64(stats.GPU.Utilization) > m.config.AlertThresholds.GPUUsagePercent {
			gpuAlert = true
			alerts = append(alerts, fmt.Sprintf("GPU Util %d%%", stats.GPU.Utilization))
		}
		if float64(stats.GPU.Temperature) > m.config.AlertThresholds.GPUTempCelsius {
			gpuAlert = true
			alerts = append(alerts, fmt.Sprintf("GPU Temp %dC", stats.GPU.Temperature))
		}
	}

	// Check Memory (in Process module)
	if stats.Memory.UsedPercent > m.config.AlertThresholds.MemoryUsagePercent {
		memAlert = true
		alerts = append(alerts, fmt.Sprintf("Mem %.0f%%", stats.Memory.UsedPercent))
	}

	m.cpu.Alert = cpuAlert
	m.gpu.Alert = gpuAlert
	m.process.Alert = memAlert

	// Notify
	if len(alerts) > 0 && time.Since(m.lastAlertTime) > 10*time.Second {
		m.lastAlertTime = time.Now()
		msg := "Alert: " + alerts[0]
		if len(alerts) > 1 {
			msg += fmt.Sprintf(" (+%d more)", len(alerts)-1)
		}

		// Run in background
		go exec.Command("notify-send", "-u", "critical", "OmniTop Alert", msg).Run()
	}
}

func (m *RootModel) updateTooltip() {
	m.showTooltip = false
	if m.width == 0 {
		return
	}

	// Determine column
	w1 := int(float64(m.width) * m.col1Pct)
	w2 := int(float64(m.width) * m.col2Pct)

	if m.mouseX < w1 {
		// GPU
		m.showTooltip = true
		m.tooltipContent = "GPU Stats:\nUtilization of graphics core\nand VRAM usage."
	} else if m.mouseX < w1+w2 {
		// Process
		m.showTooltip = true
		m.tooltipContent = "Processes:\nList of active tasks.\nSort by CPU/MEM.\nKill: k, Renice: []"
	} else {
		// CPU
		m.showTooltip = true
		m.tooltipContent = "CPU Stats:\nPer-core usage bars.\nLoad Avg: 1/5/15m."
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

	// Update footer help text based on tooltip state
	if m.showTooltip && m.tooltipContent != "" {
		m.footer.SetHelp(m.tooltipContent)
	} else {
		m.footer.SetHelp("")
	}

	// Render Footer
	footer := m.footer.View()

	// Combine
	view := lipgloss.JoinVertical(lipgloss.Left,
		cols,
		footer,
	)

	return view
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
