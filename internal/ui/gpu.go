package ui

import (
	"fmt"
	"math"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/omnitop/internal/metrics"
)

type GPUModel struct {
	width  int
	height int
	stats  metrics.GPUStats
	Alert  bool
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

	style := PanelStyle
	if m.Alert {
		style = AlertPanelStyle
	}
	style = style.Copy().Width(m.width).Height(m.height)

	if !m.stats.Available {
		content := lipgloss.Place(m.width-2, m.height-2, lipgloss.Center, lipgloss.Center, "GPU Unavailable\n(Run with --mock to see demo)")
		return style.Render(content)
	}

	// Header
	header := TitleStyle.Render(fmt.Sprintf("GPU: %s", m.stats.Name))

	// Metrics Bars
	// Calculate available width for bars
	barLabelWidth := 20 // Approx width for labels
	barWidth := m.width - 4 - barLabelWidth
	if barWidth < 10 {
		barWidth = 10
	}

	utilBar := renderBar(int(m.stats.Utilization), 100, m.width-4, "Util")

	memUtilPercent := int(m.stats.MemoryUtil)
	if memUtilPercent == 0 && m.stats.MemoryTotal > 0 {
		memUtilPercent = int(float64(m.stats.MemoryUsed) / float64(m.stats.MemoryTotal) * 100.0)
	}
	memBar := renderBar(memUtilPercent, 100, m.width-4, fmt.Sprintf("VRAM %d/%d MB", m.stats.MemoryUsed/1024/1024, m.stats.MemoryTotal/1024/1024))

	tempBar := renderBar(int(m.stats.Temperature), 100, m.width-4, fmt.Sprintf("Temp %d°C", m.stats.Temperature))
	fanBar := renderBar(int(m.stats.FanSpeed), 100, m.width-4, fmt.Sprintf("Fan %d%%", m.stats.FanSpeed))

	// Power Bar
	powerW := m.stats.PowerUsage / 1000
	powerLimitW := m.stats.PowerLimit / 1000
	if powerLimitW == 0 {
		powerLimitW = 300
	} // Default fallback if 0
	powerPct := int(float64(powerW) / float64(powerLimitW) * 100)
	powerBar := renderBar(powerPct, 100, m.width-4, fmt.Sprintf("Pwr %dW", powerW))

	// Calculate space for graph vs process list
	// We want roughly 50% for graph, remaining for processes if height allows
	availHeight := m.height - 7 // Header + 5 bars + padding
	if availHeight < 5 {
		availHeight = 5 // Minimum fallback
	}

	graphHeight := availHeight / 2
	if graphHeight < 5 {
		graphHeight = 5
	}

	// Process list gets remaining space
	procHeight := availHeight - graphHeight - 2 // -2 for headers/padding
	if procHeight < 0 {
		procHeight = 0
	}

	// Render Graph
	graph := m.renderGraph(graphHeight)

	// Render Process List
	procList := ""
	if procHeight > 2 {
		procList = m.renderProcessTable(procHeight)
	}

	// Combine
	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		utilBar,
		memBar,
		tempBar,
		fanBar,
		powerBar,
		"\n",
		graph,
		"\n",
		procList,
	)

	return style.Render(content)
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

	// Symbols for graph:   ▂▃▄▅▆▇█
	symbols := []rune{' ', ' ', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	// Create grid
	grid := make([][]rune, height)
	for i := range grid {
		grid[i] = make([]rune, maxPoints) // Use maxPoints instead of len(data) to ensure full width
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}

	// Determine start index for data in the grid (right-aligned)
	startIdx := maxPoints - len(data)

	for x, val := range data {
		// Calculate height relative to max 100
		// val is 0-100
		// height is e.g. 10
		// normalized height = val / 100 * height

		normH := (val / 100.0) * float64(height)
		fullBlocks := int(math.Floor(normH))
		remainder := normH - float64(fullBlocks)

		gridIdx := startIdx + x
		if gridIdx >= maxPoints {
			continue
		}

		// Draw full blocks from bottom
		for y := 0; y < fullBlocks; y++ {
			if height-1-y >= 0 {
				grid[height-1-y][gridIdx] = '█'
			}
		}

		// Draw partial block at top
		if fullBlocks < height {
			symIdx := int(remainder * 8)
			if symIdx > 8 {
				symIdx = 8
			}
			if symIdx > 0 {
				grid[height-1-fullBlocks][gridIdx] = symbols[symIdx]
			}
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

func (m GPUModel) renderProcessTable(height int) string {
	var sb strings.Builder
	sb.WriteString(TitleStyle.Render("GPU Processes"))
	sb.WriteString("\n")

	// Filter GPU processes
	// Assuming stats.Processes contains all system processes, we need to filter
	// wait, stats.Processes is missing in GPUStats struct in types.go?
	// Let's check types.go. Yes, GPUStats has `Processes []GPUProcess`.

	if len(m.stats.Processes) == 0 {
		sb.WriteString(MetricLabelStyle.Render("No GPU processes"))
		return sb.String()
	}

	// Columns: PID, Name, VRAM
	// PID (6), Name (15), VRAM (10)
	header := fmt.Sprintf("%-6s %-15s %-10s", "PID", "Name", "VRAM")
	sb.WriteString(MetricLabelStyle.Render(header) + "\n")

	remainingHeight := height - 2 // Header + Title
	if remainingHeight < 0 {
		remainingHeight = 0
	}

	for i, p := range m.stats.Processes {
		if i >= remainingHeight {
			break
		}

		vramStr := fmt.Sprintf("%d MB", p.MemoryUsed/1024/1024)
		name := p.Name
		if len(name) > 15 {
			name = name[:12] + "..."
		}

		line := fmt.Sprintf("%-6d %-15s %-10s", p.PID, name, vramStr)
		sb.WriteString(MetricValueStyle.Render(line) + "\n")
	}

	return sb.String()
}
