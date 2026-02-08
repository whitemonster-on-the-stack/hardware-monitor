package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/omnitop/internal/metrics"
)

type GPUModel struct {
	width  int
	height int
	stats  metrics.GPUStats
}

func NewGPUModel() GPUModel {
	return GPUModel{}
}

func (m GPUModel) Init() tea.Cmd {
	return nil
}

func (m GPUModel) Update(msg tea.Msg) (GPUModel, tea.Cmd) {
	// GPUModel is updated by the Root model passing new stats
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
		content := lipgloss.Place(m.width-2, m.height-2, lipgloss.Center, lipgloss.Center, "GPU Unavailable")
		return style.Render(content)
	}

	// Header
	header := TitleStyle.Render(fmt.Sprintf("GPU: %s", m.stats.Name))

	// Metrics
	utilBar := renderBar(int(m.stats.Utilization), 100, m.width-4, "Util")
	
	// Prefer provider-supplied MemoryUtil if available; otherwise compute from used/total.
	memUtilPercent := int(m.stats.MemoryUtil)
	if memUtilPercent == 0 && m.stats.MemoryTotal > 0 {
		// Compute percentage as (used / total) * 100, guarding against integer truncation.
		memUtilPercent = int(float64(m.stats.MemoryUsed) / float64(m.stats.MemoryTotal) * 100.0)
	}
	
	memBar := renderBar(memUtilPercent, 100, m.width-4, fmt.Sprintf("VRAM %d/%d MB", m.stats.MemoryUsed/1024/1024, m.stats.MemoryTotal/1024/1024))
	tempBar := renderBar(int(m.stats.Temperature), 100, m.width-4, fmt.Sprintf("Temp %d°C", m.stats.Temperature))
	fanBar := renderBar(int(m.stats.FanSpeed), 100, m.width-4, fmt.Sprintf("Fan %d%%", m.stats.FanSpeed))

	// Historical Graph (Simple Braille/Block implementation)
	graphHeight := m.height - 10 // Reserve space for bars and header
	if graphHeight < 5 {
		graphHeight = 5
	}
	graph := m.renderGraph(graphHeight)

	// Combine
	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		utilBar,
		memBar,
		tempBar,
		fanBar,
		graph,
	)

	return style.Render(content)
}

func renderBar(value, max, width int, label string) string {
	if width < 10 {
		return label
	}
	barWidth := width - lipgloss.Width(label) - 2
	if barWidth < 0 {
		barWidth = 0
	}

	filled := int(float64(value) / float64(max) * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	empty := barWidth - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	style := BarStyle
	if value > 80 {
		style = AlertBarStyle
	}

	return fmt.Sprintf("%s %s", label, style.Render(bar))
}

func (m GPUModel) renderGraph(height int) string {
	// Simple sparkline-like graph using the history
	if len(m.stats.HistoricalUtil) == 0 {
		return "Waiting for data..."
	}

	// Use only last N points that fit width
	data := m.stats.HistoricalUtil
	maxPoints := m.width - 4
	if maxPoints < 1 {
		maxPoints = 1
	}
	if len(data) > maxPoints {
		data = data[len(data)-maxPoints:]
	}

	var sb strings.Builder
	sb.WriteString(TitleStyle.Render("Utilization History"))
	sb.WriteString("\n")

	// Very simple graph for MVP: just iterate data and draw blocks relative to height
	// A real braille graph requires 2x4 pixel mapping, which is complex to implement from scratch in one go without errors.
	// We'll use vertical bars  ▂▃▄▅▆▇█

	// Normalize to height? Actually simpler to just show a sparkline on one line if height is small,
	// or multiple lines. For MVP, let's do a single line sparkline repeated or scaled?
	// The requirement is "Large historical graph".
	// Let's attempt a basic multi-line graph.

	// Find max (always 100 for util)
	maxVal := 100.0

	// Create a grid
	grid := make([][]rune, height)
	for i := range grid {
		grid[i] = make([]rune, len(data))
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}

	for x, val := range data {
		// Calculate height of bar
		barHeight := int((val / maxVal) * float64(height))
		if barHeight > height {
			barHeight = height
		}

		// Fill from bottom
		for y := 0; y < barHeight; y++ {
			grid[height-1-y][x] = '█' // or specific block
		}
	}

	for _, row := range grid {
		sb.WriteString(string(row) + "\n")
	}

	return sb.String()
}
