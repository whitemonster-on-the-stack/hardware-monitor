package ui

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/omnitop/internal/metrics"
)

type ProcessModel struct {
	table  table.Model
	width  int
	height int
	stats  []metrics.ProcessInfo
}

func NewProcessModel() ProcessModel {
	columns := []table.Column{
		{Title: "PID", Width: 6},
		{Title: "User", Width: 10},
		{Title: "CPU%", Width: 6},
		{Title: "Mem%", Width: 6},
		{Title: "Command", Width: 20},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(lipgloss.Color(ColorSteelGray)).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color(ColorIceBlue)).
		Background(lipgloss.Color(ColorSteelGray)).
		Bold(false)
	t.SetStyles(s)

	return ProcessModel{
		table: t,
	}
}

func (m ProcessModel) Init() tea.Cmd {
	return nil
}

func (m ProcessModel) Update(msg tea.Msg) (ProcessModel, tea.Cmd) {
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *ProcessModel) SetStats(procs []metrics.ProcessInfo) {
	// Create a copy to sort
	sorted := make([]metrics.ProcessInfo, len(procs))
	copy(sorted, procs)

	// Sort by CPU usage descending
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].CPUPercent > sorted[j].CPUPercent
	})
	m.stats = sorted

	rows := make([]table.Row, len(sorted))
	for i, p := range sorted {
		rows[i] = table.Row{
			fmt.Sprintf("%d", p.PID),
			p.User,
			fmt.Sprintf("%.1f", p.CPUPercent),
			fmt.Sprintf("%.1f", p.MemPercent),
			p.Command,
		}
	}
	m.table.SetRows(rows)
}

func (m *ProcessModel) SetSize(w, h int) {
	m.width = w
	m.height = h

	// Calculate available height for table
	tableHeight := h - 4
	if tableHeight < 1 {
		tableHeight = 1
	}
	m.table.SetHeight(tableHeight)

	// Adjust columns
	cols := m.table.Columns()
	usedWidth := 0
	for i, c := range cols {
		if i != 4 { // Command is last
			usedWidth += c.Width
		}
	}
	// Add padding/borders estimation
	usedWidth += 10

	remaining := w - usedWidth
	if remaining < 10 {
		remaining = 10
	}
	cols[4].Width = remaining
	m.table.SetColumns(cols)
}

func (m ProcessModel) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	style := PanelStyle.Copy().Width(m.width).Height(m.height)

	return style.Render(lipgloss.JoinVertical(lipgloss.Left,
		TitleStyle.Render("Processes"),
		m.table.View(),
	))
}
