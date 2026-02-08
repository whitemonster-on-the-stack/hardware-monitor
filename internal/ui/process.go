package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/omnitop/internal/metrics"
	"github.com/shirou/gopsutil/v3/process"
)

type ProcessModel struct {
	table    table.Model
	width    int
	height   int
	stats    []metrics.ProcessInfo
	sortBy   string // "cpu", "mem", "pid"
	sortDesc bool
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
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color(ColorMidnightBlack)).
		Background(lipgloss.Color(ColorIceBlue)).
		Bold(false)
	t.SetStyles(s)

	return ProcessModel{
		table:    t,
		sortBy:   "cpu",
		sortDesc: true,
	}
}

func (m ProcessModel) Init() tea.Cmd {
	return nil
}

func (m ProcessModel) Update(msg tea.Msg) (ProcessModel, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "c":
			m.sortBy = "cpu"
			m.sortDesc = true
			m.sortStats()
		case "m":
			m.sortBy = "mem"
			m.sortDesc = true
			m.sortStats()
		case "p":
			m.sortBy = "pid"
			m.sortDesc = false
			m.sortStats()
		case "k", "f9": // Kill / Signal
			selected := m.table.SelectedRow()
			if len(selected) > 0 {
				pidStr := selected[0]
				var pid int32
				fmt.Sscanf(pidStr, "%d", &pid)
				// Actual kill logic
				if p, err := process.NewProcess(pid); err == nil {
					// MVP: Terminate signal
					_ = p.Terminate()
				}
			}
		}
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *ProcessModel) SetStats(procs []metrics.ProcessInfo) {
	m.stats = procs
	m.sortStats()
}

func (m *ProcessModel) sortStats() {
	// Sort m.stats based on sortBy
	sort.Slice(m.stats, func(i, j int) bool {
		var less bool
		switch m.sortBy {
		case "cpu":
			less = m.stats[i].CPUPercent < m.stats[j].CPUPercent
		case "mem":
			less = m.stats[i].MemPercent < m.stats[j].MemPercent
		case "pid":
			less = m.stats[i].PID < m.stats[j].PID
		default:
			less = m.stats[i].CPUPercent < m.stats[j].CPUPercent
		}

		if m.sortDesc {
			return !less
		}
		return less
	})

	// Update table rows
	// Try to keep selection stable?
	// For MVP, just update rows.
	rows := make([]table.Row, len(m.stats))
	for i, p := range m.stats {
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

	// Fixed widths for numeric columns
	cols[0].Width = 6  // PID
	cols[1].Width = 10 // User
	cols[2].Width = 6  // CPU
	cols[3].Width = 6  // Mem

	usedWidth := 6 + 10 + 6 + 6 + 10 // + padding
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

	headerText := fmt.Sprintf("Processes (Sort: %s) [c:CPU m:MEM p:PID k:Kill]", strings.ToUpper(m.sortBy))

	return style.Render(lipgloss.JoinVertical(lipgloss.Left,
		TitleStyle.Render(headerText),
		m.table.View(),
	))
}
