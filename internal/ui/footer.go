package ui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type FooterModel struct {
	width int
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

func (m FooterModel) View() string {
	if m.width == 0 {
		return ""
	}

	style := lipgloss.NewStyle().
		Width(m.width).
		Background(lipgloss.Color(ColorSteelGray)).
		Foreground(lipgloss.Color(ColorMidnightBlack)).
		Padding(0, 1)

	// Left: Hostname/Uptime (Mocked for now or use os)
	left := fmt.Sprintf("OmniTop | %s", time.Now().Format("15:04:05"))

	// Right: Hotkeys
	right := "q: Quit | [/]: Resize GPU | {/}: Resize Process | Up/Down: Nav"

	// Spacer
	spacerWidth := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if spacerWidth < 0 {
		spacerWidth = 0
	}
	spacer := lipgloss.NewStyle().Width(spacerWidth).Render("")

	return style.Render(left + spacer + right)
}
