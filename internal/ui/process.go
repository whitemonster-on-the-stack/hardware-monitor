package ui

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"syscall"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/omnitop/internal/metrics"
	"github.com/shirou/gopsutil/v3/process"
)

type SortBy int

const (
	SortCPU SortBy = iota
	SortMem
	SortPID
)

type ProcessModel struct {
	table     table.Model
	width     int
	height    int
	stats     metrics.SystemStats
	sortBy    SortBy
	filter    string
	filtering bool
	textInput textinput.Model
	Alert     bool
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

	ti := textinput.New()
	ti.Placeholder = "Filter..."
	ti.Prompt = "/"
	ti.CharLimit = 30
	ti.Width = 20

	return ProcessModel{
		table:     t,
		sortBy:    SortCPU,
		textInput: ti,
	}
}

func (m ProcessModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m ProcessModel) Update(msg tea.Msg) (ProcessModel, tea.Cmd) {
	var cmd tea.Cmd

	if m.filtering {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter", "esc":
				m.filtering = false
				m.filter = m.textInput.Value()
				m.table.Focus()
				return m, nil
			}
		}
		m.textInput, cmd = m.textInput.Update(msg)
		m.filter = m.textInput.Value() // Live filter
		// Re-apply filter immediately
		m.SetStats(m.stats)
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "/":
			m.filtering = true
			m.textInput.Focus()
			m.table.Blur()
			return m, textinput.Blink
		case "s":
			m.sortBy = (m.sortBy + 1) % 3
			// Re-sort
			m.SetStats(m.stats)
		case "k", "f9":
			if len(m.table.SelectedRow()) > 0 {
				pidStr := m.table.SelectedRow()[0]
				var pid int
				fmt.Sscanf(pidStr, "%d", &pid)
				// Kill
				p, err := os.FindProcess(pid)
				if err == nil {
					_ = p.Signal(syscall.SIGTERM) // or Kill
				}
			}
		case "[": // Renice + (Lower priority, higher value)
			if len(m.table.SelectedRow()) > 0 {
				pidStr := m.table.SelectedRow()[0]
				var pid int
				fmt.Sscanf(pidStr, "%d", &pid)
				proc, err := process.NewProcess(int32(pid))
				if err == nil {
					nice, err := proc.Nice()
					if err == nil {
						_ = syscall.Setpriority(syscall.PRIO_PROCESS, pid, int(nice+1))
					}
				}
			}
		case "]": // Renice - (Higher priority, lower value)
			if len(m.table.SelectedRow()) > 0 {
				pidStr := m.table.SelectedRow()[0]
				var pid int
				fmt.Sscanf(pidStr, "%d", &pid)
				proc, err := process.NewProcess(int32(pid))
				if err == nil {
					nice, err := proc.Nice()
					if err == nil {
						_ = syscall.Setpriority(syscall.PRIO_PROCESS, pid, int(nice-1))
					}
				}
			}
		}
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *ProcessModel) SetStats(stats metrics.SystemStats) {
	m.stats = stats
	procs := stats.Processes

	// Filter
	var filtered []metrics.ProcessInfo
	if m.filter != "" {
		lowerFilter := strings.ToLower(m.filter)
		for _, p := range procs {
			if strings.Contains(strings.ToLower(p.Command), lowerFilter) ||
				strings.Contains(strings.ToLower(p.User), lowerFilter) ||
				fmt.Sprintf("%d", p.PID) == lowerFilter {
				filtered = append(filtered, p)
			}
		}
	} else {
		filtered = make([]metrics.ProcessInfo, len(procs))
		copy(filtered, procs)
	}

	// Sort
	switch m.sortBy {
	case SortCPU:
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].CPUPercent > filtered[j].CPUPercent
		})
	case SortMem:
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].MemPercent > filtered[j].MemPercent
		})
	case SortPID:
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].PID < filtered[j].PID
		})
	}

	rows := make([]table.Row, len(filtered))
	for i, p := range filtered {
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
	// Header: 1, Newline: 1, Mem: 1, Swap: 1, Net: 1, Disk: 1
	// Total reserved: 6 lines.
	tableHeight := h - 6
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

	style := PanelStyle
	if m.Alert {
		style = AlertPanelStyle
	}
	style = style.Copy().Width(m.width).Height(m.height)

	title := "Processes"
	if m.filtering {
		title = m.textInput.View()
	} else if m.filter != "" {
		title = fmt.Sprintf("Filter: %s", m.filter)
	}

	sortStr := "CPU"
	switch m.sortBy {
	case SortMem:
		sortStr = "MEM"
	case SortPID:
		sortStr = "PID"
	}

	header := lipgloss.JoinHorizontal(lipgloss.Left,
		TitleStyle.Render(title),
		lipgloss.PlaceHorizontal(m.width-lipgloss.Width(title)-lipgloss.Width(sortStr)-5, lipgloss.Right, " "),
		MetricLabelStyle.Render(fmt.Sprintf("[%s]", sortStr)),
	)

	// Render Memory/Net/Disk bars at bottom
	memBar := renderBar(int(m.stats.Memory.UsedPercent), 100, m.width-4, fmt.Sprintf("Mem %.1f%%", m.stats.Memory.UsedPercent))
	swapBar := renderBar(int(m.stats.Memory.SwapPercent), 100, m.width-4, fmt.Sprintf("Swap %.1f%%", m.stats.Memory.SwapPercent))

	// Net/Disk (simple bars for speed/activity)
	// Use 100MB/s as arbitrary max for visualization for now
	const maxIO = 100 * 1024 * 1024 // 100MB

	halfWidth := (m.width - 6) / 2
	if halfWidth < 10 {
		halfWidth = 10
	}

	netDownBar := renderBar(int(m.stats.Net.DownloadSpeed), maxIO, halfWidth, fmt.Sprintf("Net ↓ %s/s", formatBytes(m.stats.Net.DownloadSpeed)))
	netUpBar := renderBar(int(m.stats.Net.UploadSpeed), maxIO, halfWidth, fmt.Sprintf("Net ↑ %s/s", formatBytes(m.stats.Net.UploadSpeed)))

	diskReadBar := renderBar(int(m.stats.Disk.ReadSpeed), maxIO, halfWidth, fmt.Sprintf("Disk R %s/s", formatBytes(m.stats.Disk.ReadSpeed)))
	diskWriteBar := renderBar(int(m.stats.Disk.WriteSpeed), maxIO, halfWidth, fmt.Sprintf("Disk W %s/s", formatBytes(m.stats.Disk.WriteSpeed)))

	ioRow1 := lipgloss.JoinHorizontal(lipgloss.Top, netDownBar, "  ", netUpBar)
	ioRow2 := lipgloss.JoinHorizontal(lipgloss.Top, diskReadBar, "  ", diskWriteBar)

	return style.Render(lipgloss.JoinVertical(lipgloss.Left,
		header,
		m.table.View(),
		"\n",
		memBar,
		swapBar,
		ioRow1,
		ioRow2,
	))
}

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
