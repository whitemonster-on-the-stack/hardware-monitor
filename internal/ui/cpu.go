package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/omnitop/internal/metrics"
)

type CPUModel struct {
	width  int
	height int
	stats  metrics.SystemStats // Holds all for summary
}

func NewCPUModel() CPUModel {
	return CPUModel{}
}

func (m CPUModel) Init() tea.Cmd {
	return nil
}

func (m CPUModel) Update(msg tea.Msg) (CPUModel, tea.Cmd) {
	return m, nil
}

func (m *CPUModel) SetStats(stats metrics.SystemStats) {
	m.stats = stats
}

func (m *CPUModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m CPUModel) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	style := PanelStyle.Copy().Width(m.width).Height(m.height)

	// CPU Header
	cpuHeader := TitleStyle.Render(fmt.Sprintf("CPU: %.1f%%", m.stats.CPU.GlobalUsagePercent))

	// Load Average
	loadStr := fmt.Sprintf("Load: %.2f %.2f %.2f", m.stats.CPU.LoadAvg[0], m.stats.CPU.LoadAvg[1], m.stats.CPU.LoadAvg[2])
	load := MetricLabelStyle.Render(loadStr)

	// Cores
	cores := renderCores(m.stats.CPU.PerCoreUsage, m.stats.CPU.PerCoreTemp, m.width-4)

	// Memory Summary
	mem := renderBar(int(m.stats.Memory.UsedPercent), 100, m.width-4, fmt.Sprintf("Mem %.1f%%", m.stats.Memory.UsedPercent))
	swap := renderBar(int(m.stats.Memory.SwapPercent), 100, m.width-4, fmt.Sprintf("Swap %.1f%%", m.stats.Memory.SwapPercent))

	// GPU Summary
	gpu := ""
	if m.stats.GPU.Available {
		gpu = renderBar(int(m.stats.GPU.Utilization), 100, m.width-4, fmt.Sprintf("GPU %d%%", m.stats.GPU.Utilization))
	} else {
		gpu = MetricLabelStyle.Render("GPU: N/A")
	}

	// Combine
	content := lipgloss.JoinVertical(lipgloss.Left,
		cpuHeader,
		load,
		cores,
		"\n",
		TitleStyle.Render("Memory"),
		mem,
		swap,
		"\n",
		TitleStyle.Render("GPU Summary"),
		gpu,
	)

	return style.Render(content)
}

func renderCores(usage []float64, temps []float64, width int) string {
	var sb strings.Builder
	colWidth := (width / 2) - 2
	if colWidth < 10 {
		colWidth = width // single column
	}

	if len(usage) == 0 {
		return "No CPU Data"
	}

	for i := 0; i < len(usage); i += 2 {
		idx1 := i
		tempStr1 := ""
		if len(temps) > idx1 && temps[idx1] > 0 {
			tempStr1 = fmt.Sprintf(" %d°C", int(temps[idx1]))
		}
		label1 := fmt.Sprintf("%d%s", idx1, tempStr1)
		bar1 := renderBarCompact(int(usage[idx1]), 100, colWidth, label1)

		if i+1 < len(usage) {
			idx2 := i + 1
			tempStr2 := ""
			if len(temps) > idx2 && temps[idx2] > 0 {
				tempStr2 = fmt.Sprintf(" %d°C", int(temps[idx2]))
			}
			label2 := fmt.Sprintf("%d%s", idx2, tempStr2)
			bar2 := renderBarCompact(int(usage[idx2]), 100, colWidth, label2)

			// Pad to align
			padding := width - lipgloss.Width(bar1) - lipgloss.Width(bar2)
			if padding < 0 { padding = 0 }
			sb.WriteString(bar1 + strings.Repeat(" ", padding) + bar2 + "\n")
		} else {
			sb.WriteString(bar1 + "\n")
		}
	}

	return sb.String()
}

func renderBarCompact(value, max, width int, label string) string {
	// [Label  |||||     ]
	// Label takes some space.
	labelLen := lipgloss.Width(label)
	barLen := width - labelLen - 3 // [ ] and space
	if barLen < 5 {
		return fmt.Sprintf("%s %d%%", label, value)
	}

	filled := int(float64(value) / float64(max) * float64(barLen))
	if filled > barLen {
		filled = barLen
	}
	empty := barLen - filled

	bar := strings.Repeat("|", filled) + strings.Repeat(" ", empty)

	style := BarStyle
	if value > 80 {
		style = AlertBarStyle
	}

	return fmt.Sprintf("%s [%s]", label, style.Render(bar))
}
