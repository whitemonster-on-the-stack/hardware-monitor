package ui

import (
	"math/rand"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/omnitop/internal/metrics"
)

type TickMsg time.Time

var tickRand = rand.New(rand.NewSource(time.Now().UnixNano()))

func tick() tea.Cmd {
	// Add jitter: 900-1100ms
	jitter := time.Duration(tickRand.Intn(200)-100) * time.Millisecond
	return tea.Tick(time.Second+jitter, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

type RootModel struct {
	provider metrics.Provider

	// Sub-models
	gpu     GPUModel
	process ProcessModel
	cpu     CPUModel
	footer  FooterModel

	// Layout state
	width, height int
	col1Pct       float64 // Percentage of width for Left Column (GPU)
	col2Pct       float64 // Percentage of width for Middle Column (Process)
	// Right column takes remaining

	// Tooltip state
	mouseX, mouseY int
	showTooltip    bool
}

func NewRootModel(provider metrics.Provider) RootModel {
	return RootModel{
		provider: provider,
		gpu:      NewGPUModel(),
		process:  NewProcessModel(),
		cpu:      NewCPUModel(),
		footer:   NewFooterModel(),
		col1Pct:  0.30, // 30%
		col2Pct:  0.40, // 40%
	}
}

func (m RootModel) Init() tea.Cmd {
	return tick()
}

func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "[": // Shrink Left Col
			m.col1Pct -= 0.05
			if m.col1Pct < 0.1 { m.col1Pct = 0.1 }
			m.resizeModules()
		case "]": // Expand Left Col
			m.col1Pct += 0.05
			if m.col1Pct+m.col2Pct > 0.9 { m.col1Pct = 0.9 - m.col2Pct }
			m.resizeModules()
		case "{": // Shrink Middle Col (effectively expands Right)
			m.col2Pct -= 0.05
			if m.col2Pct < 0.1 { m.col2Pct = 0.1 }
			m.resizeModules()
		case "}": // Expand Middle Col
			m.col2Pct += 0.05
			if m.col1Pct+m.col2Pct > 0.9 { m.col2Pct = 0.9 - m.col1Pct }
			m.resizeModules()
		}

		// Pass keys to sub-models if needed (e.g. process list scrolling)
		m.process, cmd = m.process.Update(msg)
		cmds = append(cmds, cmd)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeModules()

	case TickMsg:
		// Fetch metrics
		stats, err := m.provider.GetStats()
		if err == nil {
			m.gpu.SetStats(stats.GPU)
			m.process.SetStats(stats.Processes)
			m.cpu.SetStats(*stats)
		}
		// Continue tick
		cmds = append(cmds, tick())

	case tea.MouseMsg:
		m.mouseX = msg.X
		m.mouseY = msg.Y
		// TODO: Hit testing for tooltips

		// Pass mouse to sub-models
		m.process, cmd = m.process.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update other models if they have logic
	// gpu and cpu currently have no internal update logic needing msgs

	return m, tea.Batch(cmds...)
}

func (m *RootModel) resizeModules() {
	if m.width == 0 || m.height == 0 {
		return
	}

	// Calculate widths
	w1 := int(float64(m.width) * m.col1Pct)
	w2 := int(float64(m.width) * m.col2Pct)
	w3 := m.width - w1 - w2

	// Height available for columns (minus footer)
	h := m.height - 1
	if h < 1 { h = 1 }

	m.gpu.SetSize(w1, h)
	m.process.SetSize(w2, h)
	m.cpu.SetSize(w3, h)
	m.footer.SetSize(m.width)
}

func (m RootModel) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	// Render columns
	cols := lipgloss.JoinHorizontal(lipgloss.Top,
		m.gpu.View(),
		m.process.View(),
		m.cpu.View(),
	)

	// Add Footer
	return lipgloss.JoinVertical(lipgloss.Left,
		cols,
		m.footer.View(),
	)
}
