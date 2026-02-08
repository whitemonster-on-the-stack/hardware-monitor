package ui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/omnitop/internal/metrics"
)

type FooterModel struct {
	width  int
	uptime uint64
}

func NewFooterModel() FooterModel {
	return FooterModel{}
}

func (m FooterModel) Init() tea.Cmd {
	return nil
}

func (m FooterModel) Update(msg tea.Msg) (FooterModel, tea.Cmd) {
	return m, nil
}

func (m *FooterModel) SetSize(w int) {
	m.width = w
}

func (m *FooterModel) SetStats(stats *metrics.SystemStats) {
	m.uptime = stats.Uptime
}

func (m FooterModel) View() string {
	if m.width == 0 {
		return ""
	}

	style := lipgloss.NewStyle().
		Width(m.width).
		Background(lipgloss.Color(ColorSteelGray)).
		Foreground(lipgloss.Color(ColorMidnightBlack)).
		Padding(0, 1)

	// Format uptime
	d := time.Duration(m.uptime) * time.Second
	uptimeStr := fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)

	// Left: Hostname/Uptime
	left := fmt.Sprintf("OmniTop | Uptime: %s | %s", uptimeStr, time.Now().Format("15:04:05"))

	// Right: Hotkeys
	right := "q: Quit | Arrows: Select | [ ] { }: Resize"

	// Spacer
	spacerWidth := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if spacerWidth < 0 {
		spacerWidth = 0
	}
	spacer := lipgloss.NewStyle().Width(spacerWidth).Render("")

	return style.Render(left + spacer + right)
}
