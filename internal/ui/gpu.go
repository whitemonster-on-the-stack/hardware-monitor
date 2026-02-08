package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/omnitop/internal/metrics"
)

type GPUModel struct {
	width         int
	height        int
	stats         metrics.GPUStats
	showProcesses bool
}

func NewGPUModel() GPUModel {
	return GPUModel{
		showProcesses: false, // Default to graph view
	}
}

func (m GPUModel) Init() tea.Cmd {
	return nil
}

func (m GPUModel) Update(msg tea.Msg) (GPUModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "g":
			m.showProcesses = !m.showProcesses
		}
	}
	return m, nil
}

func (m *GPUModel) SetStats(stats metrics.GPUStats) {
	m.stats = stats
}

func (m *GPUModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m GPUModel) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	style := PanelStyle.Copy().Width(m.width).Height(m.height)

	if !m.stats.Available {
		content := lipgloss.Place(m.width-2, m.height-2, lipgloss.Center, lipgloss.Center, "GPU Unavailable\n(Run with --mock to see demo)")
		return style.Render(content)
	}

	// Header
	header := TitleStyle.Render(fmt.Sprintf("GPU: %s", m.stats.Name))

	// Telemetry Bars
	// Calculate bar width dynamically
	barWidth := m.width - 4
	if barWidth < 10 {
		barWidth = 10
	}

	utilBar := renderBar(int(m.stats.Utilization), 100, barWidth, "Util")

	memUtilPercent := int(m.stats.MemoryUtil)
	if memUtilPercent == 0 && m.stats.MemoryTotal > 0 {
		memUtilPercent = int(float64(m.stats.MemoryUsed) / float64(m.stats.MemoryTotal) * 100.0)
	}
	memBar := renderBar(memUtilPercent, 100, barWidth, fmt.Sprintf("VRAM %d/%d MB", m.stats.MemoryUsed/1024/1024, m.stats.MemoryTotal/1024/1024))

	tempBar := renderBar(int(m.stats.Temperature), 100, barWidth, fmt.Sprintf("Temp %d°C", m.stats.Temperature))
	fanBar := renderBar(int(m.stats.FanSpeed), 100, barWidth, fmt.Sprintf("Fan %d%%", m.stats.FanSpeed))

	powerVal := int(m.stats.PowerUsage / 1000) // mW -> W
	powerLimit := int(m.stats.PowerLimit / 1000)
	powerLabel := fmt.Sprintf("Pwr %dW", powerVal)
	if powerLimit > 0 {
		powerLabel += fmt.Sprintf("/%dW", powerLimit)
	}
	powerBar := renderBar(powerVal, powerLimit, barWidth, powerLabel)

	// Content selection (Graph vs Processes)
	var mainContent string
	availableHeight := m.height - 8 // Header + 5 bars + padding
	if availableHeight < 5 {
		availableHeight = 5
	}

	if m.showProcesses {
		mainContent = m.renderProcessTable(availableHeight)
	} else {
		mainContent = m.renderGraph(availableHeight)
	}

	footer := MetricLabelStyle.Render("Press 'g' to toggle View")

	// Combine
	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		utilBar,
		memBar,
		tempBar,
		fanBar,
		powerBar,
		"\n",
		mainContent,
		footer,
	)

	return style.Render(content)
}

func renderBar(value, max, width int, label string) string {
	if max <= 0 {
		max = 100
	} // Avoid divide by zero
	if width < 10 {
		return label
	}
	barWidth := width - lipgloss.Width(label) - 2
	if barWidth < 0 {
		barWidth = 0
	}

	ratio := float64(value) / float64(max)
	if ratio > 1.0 {
		ratio = 1.0
	}
	filled := int(ratio * float64(barWidth))
	empty := barWidth - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	style := BarStyle
	if ratio > 0.8 {
		style = AlertBarStyle
	}

	return fmt.Sprintf("%s %s", label, style.Render(bar))
}

func (m GPUModel) renderGraph(height int) string {
	if len(m.stats.HistoricalUtil) == 0 {
		return "Waiting for data..."
	}

	// Use only last N points that fit width
	data := m.stats.HistoricalUtil
	maxPoints := m.width - 4
	if maxPoints < 1 {
		maxPoints = 1
	}

	// Create a window of data
	window := data
	if len(window) > maxPoints {
		window = window[len(window)-maxPoints:]
	}

	var sb strings.Builder
	sb.WriteString(TitleStyle.Render("Utilization History"))
	sb.WriteString("\n")

	// Braille-like blocks:  ▂▃▄▅▆▇█
	// blocks := []rune{' ', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	// Single line graph for now to save space, repeated for height?
	// Or multiple lines.
	// Let's do a simple multi-line block graph.

	// Create grid
	grid := make([][]rune, height)
	for i := range grid {
		grid[i] = make([]rune, maxPoints)
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}

	for x, val := range window {
		// val is 0-100
		// Height is 'height'
		// Calculate how many blocks high
		h := int((val / 100.0) * float64(height))
		if h > height {
			h = height
		}

		// Fill from bottom
		for y := 0; y < h; y++ {
			grid[height-1-y][x] = '█'
		}
		// Add a "cap" block if we want more precision, but full block is fine for MVP
	}

	for _, row := range grid {
		// Trim right side if strictly needed, but maxPoints handles it
		sb.WriteString(BarStyle.Render(string(row)) + "\n")
	}

	return sb.String()
}

func (m GPUModel) renderProcessTable(height int) string {
	var sb strings.Builder
	sb.WriteString(TitleStyle.Render("GPU Processes"))
	sb.WriteString("\n")

	if len(m.stats.Processes) == 0 {
		sb.WriteString("No GPU processes found.")
		return sb.String()
	}

	// Header
	sb.WriteString(fmt.Sprintf("%-8s %-15s %s\n", "PID", "Mem", "Name"))

	count := 0
	for _, p := range m.stats.Processes {
		if count >= height-2 {
			break
		}
		memStr := fmt.Sprintf("%dMiB", p.MemoryUsed/1024/1024)
		sb.WriteString(fmt.Sprintf("%-8d %-15s %s\n", p.PID, memStr, p.Name))
		count++
	}

	return sb.String()
}
