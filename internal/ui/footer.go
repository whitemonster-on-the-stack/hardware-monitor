package ui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type FooterModel struct {
	width int
	help  string
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

func (m *FooterModel) SetHelp(h string) {
	m.help = h
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

	// If help is set (tooltip), show it prominently
	if m.help != "" {
		// Tooltip might be multi-line, but footer is usually single line.
		// We'll just take the first line or join them with spaces.
		// Actually, prompt says "explain each metric".
		// We can render a multi-line footer if needed, but height is calculated in resizeModules.
		// resizeModules sets height: `h := m.height - 1`.
		// So footer is 1 line effectively? Wait, `View` logic in RootModel:
		// `lipgloss.JoinVertical(lipgloss.Left, cols, m.footer.View())`
		// If footer is multiple lines, `cols` height needs to shrink.
		// For MVP, keep footer single line and replace content.
		content := fmt.Sprintf("INFO: %s", m.help)
		return style.Render(content)
	}

	// Left: Hostname/Uptime (Mocked for now or use os)
	left := fmt.Sprintf("OmniTop | %s", time.Now().Format("15:04:05"))

	// Right: Hotkeys
	right := "q: Quit | Arrows: Select | [ ] { }: Resize | /: Filter | k: Kill"

	// Spacer
	spacerWidth := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 4
	if spacerWidth < 0 {
		spacerWidth = 0
	}
	spacer := lipgloss.NewStyle().Width(spacerWidth).Render("")

	return style.Render(left + spacer + right)
}
