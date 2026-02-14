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
	width         int
	height        int
	stats         metrics.GPUStats
	Alert         bool
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
	headerHeight := 7 // Header + 5 bars + padding
	availHeight := m.height - headerHeight
	if availHeight < 5 {
		availHeight = 5
	}

	var graphHeight, procHeight int
	if m.showProcesses {
		// Split view
		graphHeight = availHeight / 2
		if graphHeight < 5 {
			graphHeight = 5
		}
		procHeight = availHeight - graphHeight
	} else {
		// Full graph
		graphHeight = availHeight
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
	if height <= 0 {
		return ""
	}
	if len(m.stats.HistoricalUtil) == 0 {
		return "Waiting for data..."
	}

	// Use only last N points that fit width
	data := m.stats.HistoricalUtil
	maxPoints := m.width - 4
	if maxPoints < 1 {
		maxPoints = 1
	}

	var sb strings.Builder
	sb.WriteString(TitleStyle.Render("Utilization History"))
	sb.WriteString("\n")

	realHeight := height - 2 // Minus title
	if realHeight < 1 {
		realHeight = 1
	}

	// Symbols for graph:   ▂▃▄▅▆▇█
	symbols := []rune{' ', ' ', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	// Create grid
	grid := make([][]rune, realHeight)
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
		normH := (val / 100.0) * float64(realHeight)
		fullBlocks := int(math.Floor(normH))
		remainder := normH - float64(fullBlocks)

		gridIdx := startIdx + x
		if gridIdx >= maxPoints {
			continue
		}
		if gridIdx < 0 {
			continue // Should not happen if startIdx >= 0
		}

		// Draw full blocks from bottom
		for y := 0; y < fullBlocks; y++ {
			if realHeight-1-y >= 0 {
				grid[realHeight-1-y][gridIdx] = '█'
			}
		}

		// Draw partial block at top
		if fullBlocks < realHeight {
			symIdx := int(remainder * 8)
			if symIdx > 8 {
				symIdx = 8
			}
			if symIdx > 0 {
				grid[realHeight-1-fullBlocks][gridIdx] = symbols[symIdx]
			}
		}
	}

	for _, row := range grid {
		sb.WriteString(BarStyle.Render(string(row)) + "\n")
	}

	return sb.String()
}

func (m GPUModel) renderProcessTable(height int) string {
	var sb strings.Builder
	sb.WriteString(TitleStyle.Render("GPU Processes"))
	sb.WriteString("\n")

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
