package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Eyepatch5263/gopulse/internal/system"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("42"))

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("42")).
			Padding(1, 2)

	panelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	valueStyle = lipgloss.NewStyle().
			Bold(true)

	barFilledStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	barEmptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	processSelectedStyle = lipgloss.NewStyle().
				Bold(true).
				Background(lipgloss.Color("237")).
				Foreground(lipgloss.Color("42"))
)

func renderProcessTable(
	processes []system.Process,
	selected int,
	width int,
	byMemory bool,
	processErr error,
) string {
	panelWidth := max(30, width-2)
	nameWidth := max(10, min(22, panelWidth-40))

	sortLabel := "PID"
	if byMemory {
		sortLabel = "RSS memory"
	}

	rows := []string{
		panelTitleStyle.Render("PROCESSES"),
		subtitleStyle.Render(
			fmt.Sprintf("Sorted by %s", sortLabel),
		),
		"",
		fmt.Sprintf(
			"%-7s %-7s %-*s %-8s %10s",
			"PID", "PPID", nameWidth, "NAME", "STATE", "RSS",
		),
	}

	// Show at most eight rows at a time.
	visible := min(8, len(processes))
	start := 0

	if selected >= visible && visible > 0 {
		start = selected - visible + 1
	}

	end := min(start+visible, len(processes))

	for i := start; i < end; i++ {
		p := processes[i]

		rss := "-"
		if p.RSSAvailable {
			rss = fmt.Sprintf("%d MB", p.RSSKB/1024)
		}

		row := fmt.Sprintf(
			"%-7d %-7d %-*.*s %-8s %10s",
			p.PID,
			p.PPid,
			nameWidth, nameWidth, p.Name,
			p.State,
			rss,
		)

		if i == selected {
			row = processSelectedStyle.Render(row)
		}

		rows = append(rows, row)
	}

	if processErr != nil {
		rows = append(rows, errorStyle.Render(
			"Process scan error: "+processErr.Error(),
		))
	} else if len(processes) == 0 {
		rows = append(rows, subtitleStyle.Render("No processes found."))
	}

	rows = append(rows, "")
	rows = append(rows, subtitleStyle.Render(
		"↑/↓ Navigate  •  s: change sort  •  q: quit",
	))

	return panelStyle.Width(panelWidth).Render(
		lipgloss.JoinVertical(lipgloss.Left, rows...),
	)
}

func renderProcessDetails(
	processes []system.Process,
	selected int,
	width int,
) string {
	if selected < 0 || selected >= len(processes) {
		return panelStyle.Width(max(30, width-2)).Render(
			subtitleStyle.Render("Select a process to inspect."),
		)
	}

	p := processes[selected]

	rss := "Unavailable"
	if p.RSSAvailable {
		rss = fmt.Sprintf("%d MB", p.RSSKB/1024)
	}

	cmdline := p.Cmdline
	if cmdline == "" {
		cmdline = "Unavailable or empty"
	}

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		panelTitleStyle.Render("PROCESS DETAILS"),
		"",
		fmt.Sprintf("PID:      %d", p.PID),
		fmt.Sprintf("PPID:     %d", p.PPid),
		fmt.Sprintf("Name:     %s", p.Name),
		fmt.Sprintf("State:    %s", p.State),
		fmt.Sprintf("RSS:      %s", rss),
		"",
		labelStyle.Render("Command line"),
		cmdline,
	)

	return panelStyle.Width(max(30, width-2)).Render(content)
}

func renderProcessTree(
	processes []system.Process,
	selected int,
	width int,
) string {
	rows := buildProcessTree(processes)

	panelWidth := max(30, width-2)

	if len(rows) == 0 {
		return panelStyle.Width(panelWidth).Render(
			subtitleStyle.Render("No processes found."),
		)
	}

	selectedPID := 0
	if selected >= 0 && selected < len(processes) {
		selectedPID = processes[selected].PID
	}

	selectedRow := 0
	for i, row := range rows {
		if row.process.PID == selectedPID {
			selectedRow = i
			break
		}
	}

	const visibleRows = 8

	start := 0
	if selectedRow >= visibleRows {
		start = selectedRow - visibleRows + 1
	}

	end := min(start+visibleRows, len(rows))

	lines := []string{
		panelTitleStyle.Render("PROCESS TREE"),
		subtitleStyle.Render(
			fmt.Sprintf("%d processes", len(rows)),
		),
		"",
	}

	for i := start; i < end; i++ {
		row := rows[i]
		p := row.process

		prefix := strings.Repeat("  ", row.depth)
		if row.depth > 0 {
			prefix += "└─ "
		}

		rss := "-"
		if p.RSSAvailable {
			rss = fmt.Sprintf("%d MB", p.RSSKB/1024)
		}

		line := fmt.Sprintf(
			"%s%s (PID %d, PPID %d) [%s] RSS %s",
			prefix,
			p.Name,
			p.PID,
			p.PPid,
			p.State,
			rss,
		)

		// Highlight the selected process.
		if p.PID == selectedPID {
			line = processSelectedStyle.Render(line)
		}

		lines = append(lines, line)
	}

	lines = append(lines, "")
	lines = append(lines, subtitleStyle.Render(
		"↑/↓ Navigate  •  t: flat/tree  •  s: sort  •  q: quit",
	))

	return panelStyle.Width(panelWidth).Render(
		lipgloss.JoinVertical(lipgloss.Left, lines...),
	)
}

func progressBar(percent float64, width int) string {
	if width < 1 {
		return ""
	}

	// Clamp the percentage to the range 0–100.
	percent = max(0, min(100, percent))

	filled := int(percent * float64(width) / 100)
	empty := width - filled

	return barFilledStyle.Render(strings.Repeat("█", filled)) +
		barEmptyStyle.Render(strings.Repeat("░", empty))
}

type model struct {
	processes    []system.Process
	selected     int
	sortByMemory bool
	processErr   error
	width        int
	height       int

	previousCPU system.CPUStats
	hasPrevious bool

	cpuUsage float64
	memory   system.Memory
	err      error

	showTree bool
}

type tickMsg time.Time

type sampleMsg struct {
	processes    []system.Process
	hasProcesses bool
	processErr   error
	current      system.CPUStats
	hasCurrent   bool
	cpuUsage     float64
	hasUsage     bool
	memory       system.Memory
	err          error
}

type treeRow struct {
	process system.Process
	depth   int
}

func buildProcessTree(processes []system.Process) []treeRow {
	byPID := make(map[int]system.Process, len(processes))
	children := make(map[int][]system.Process)

	for _, p := range processes {
		byPID[p.PID] = p
	}

	var roots []system.Process

	for _, p := range processes {
		// Treat processes with missing parents as roots.
		// This also handles processes whose parents aren't visible.
		if p.PPid == p.PID {
			roots = append(roots, p)
		} else if _, exists := byPID[p.PPid]; !exists {
			roots = append(roots, p)
		} else {
			children[p.PPid] = append(children[p.PPid], p)
		}
	}

	sort.Slice(roots, func(i, j int) bool {
		return roots[i].PID < roots[j].PID
	})

	for pid := range children {
		sort.Slice(children[pid], func(i, j int) bool {
			return children[pid][i].PID < children[pid][j].PID
		})
	}

	var rows []treeRow
	visited := make(map[int]bool, len(processes))

	var walk func(system.Process, int)

	walk = func(p system.Process, depth int) {
		if visited[p.PID] {
			return
		}

		visited[p.PID] = true
		rows = append(rows, treeRow{
			process: p,
			depth:   depth,
		})

		for _, child := range children[p.PID] {
			walk(child, depth+1)
		}
	}

	for _, root := range roots {
		walk(root, 0)
	}

	// Include any remaining processes defensively, for example if
	// malformed input contains a parent-child cycle.
	for _, p := range processes {
		if !visited[p.PID] {
			walk(p, 0)
		}
	}

	return rows
}

func initialModel() model {
	return model{
		sortByMemory: true,
	}
}

func sortProcess(processes []system.Process, byMemory bool) {
	sort.Slice(processes, func(i, j int) bool {
		a, b := processes[i], processes[j]

		if byMemory {
			// processes with known RSS appears before unknown RSS
			if a.RSSAvailable != b.RSSAvailable {
				return a.RSSAvailable
			}

			// large RSS appear first
			if a.RSSAvailable && a.RSSKB != b.RSSKB {
				return a.RSSKB > b.RSSKB
			}
		}

		// default case

		return a.PID < b.PID
	})
}

func (m model) Init() tea.Cmd {
	return sampleCmd(system.CPUStats{}, false)
}

func moveSelection(
	processes []system.Process,
	selected int,
	direction int,
	treeMode bool,
) int {
	if len(processes) == 0 {
		return 0
	}

	if !treeMode {
		return max(0, min(len(processes)-1, selected+direction))
	}

	rows := buildProcessTree(processes)

	if selected < 0 || selected >= len(processes) {
		return 0
	}

	selectedPID := processes[selected].PID
	position := 0

	for i, row := range rows {
		if row.process.PID == selectedPID {
			position = i
			break
		}
	}

	position = max(0, min(len(rows)-1, position+direction))
	targetPID := rows[position].process.PID

	for i, p := range processes {
		if p.PID == targetPID {
			return i
		}
	}

	return selected
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tickMsg:
		return m, sampleCmd(m.previousCPU, m.hasPrevious)
	case sampleMsg:
		m.err = msg.err
		m.processErr = msg.processErr

		selectedPID := 0
		if m.selected >= 0 && m.selected < len(m.processes) {
			selectedPID = m.processes[m.selected].PID
		}

		m.processes = msg.processes
		sortProcess(m.processes, m.sortByMemory)

		m.selected = 0
		for i, p := range m.processes {
			if p.PID == selectedPID && selectedPID != 0 {
				m.selected = i
				break
			}
		}

		if m.selected >= len(m.processes) {
			m.selected = max(0, len(m.processes)-1)
		}

		if msg.hasCurrent {
			m.previousCPU = msg.current
			m.hasPrevious = true
		}

		if msg.hasUsage {
			m.cpuUsage = msg.cpuUsage
		}

		if msg.err == nil {
			m.memory = msg.memory
		}

		return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return tickMsg(t)
		})

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			m.selected = moveSelection(
				m.processes, m.selected, -1, m.showTree,
			)

		case "down", "j":
			m.selected = moveSelection(
				m.processes, m.selected, 1, m.showTree,
			)

		case "s":
			selectedPID := 0
			if m.selected >= 0 && m.selected < len(m.processes) {
				selectedPID = m.processes[m.selected].PID
			}
			m.sortByMemory = !m.sortByMemory
			sortProcess(m.processes, m.sortByMemory)

			for i, p := range m.processes {
				if p.PID == selectedPID && selectedPID != 0 {
					m.selected = i
					break
				}
			}
		case "t":
			m.showTree = !m.showTree
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func sampleCmd(previous system.CPUStats, hasPrevious bool) tea.Cmd {
	return func() tea.Msg {
		curr, err := system.ReadCPUStats()
		if err != nil {
			return sampleMsg{err: err}
		}

		processes, processErr := system.ReadProcesses()

		res := sampleMsg{
			current:    curr,
			hasCurrent: true,
		}

		if hasPrevious {
			usage, err := system.CPUUsage(previous, curr)

			if err != nil {
				res.err = err
				return res
			}
			res.cpuUsage = usage
			res.hasUsage = true
		}

		memory, err := system.ReadMemory()
		if err != nil {
			res.err = err
			return res
		}

		res.memory = memory
		res.processes = processes
		res.hasProcesses = processErr == nil
		res.processErr = processErr
		return res
	}
}

func (m model) View() string {
	width := m.width
	if width < 30 {
		width = 80
	}

	header := titleStyle.Render("GoPulse") +
		"  " +
		subtitleStyle.Render("SYSTEM MONITOR")

	if m.err != nil {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			header,
			"",
			errorStyle.Render("Error: "+m.err.Error()),
			"",
			subtitleStyle.Render("Press q to quit"),
		)
	}

	totalMB := float64(m.memory.TotalKB) / 1024
	availableMB := float64(m.memory.AvailableKB) / 1024
	usedMB := totalMB - availableMB

	memoryPercent := 0.0
	if totalMB > 0 {
		memoryPercent = usedMB / totalMB * 100
	}

	panelWidth := width - 2
	if panelWidth < 20 {
		panelWidth = 20
	}

	barWidth := max(5, panelWidth-12)

	cpuPanel := panelStyle.Width(panelWidth).Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			panelTitleStyle.Render("CPU"),
			"",
			fmt.Sprintf("Usage   %.1f%%", m.cpuUsage),
			progressBar(m.cpuUsage, barWidth),
		),
	)

	memoryPanel := panelStyle.Width(panelWidth).Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			panelTitleStyle.Render("MEMORY"),
			"",
			fmt.Sprintf("Used       %.0f MB", usedMB),
			progressBar(memoryPercent, barWidth),
			"",
			fmt.Sprintf("Total      %.0f MB", totalMB),
			fmt.Sprintf("Available  %.0f MB", availableMB),
		),
	)

	var panels string

	if width >= 76 {
		panelWidth = (width - 5) / 2
		barWidth = max(5, panelWidth-12)

		cpuPanel = panelStyle.Width(panelWidth).Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				panelTitleStyle.Render("CPU"),
				"",
				fmt.Sprintf("Usage   %.1f%%", m.cpuUsage),
				progressBar(m.cpuUsage, barWidth),
			),
		)

		memoryPanel = panelStyle.Width(panelWidth).Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				panelTitleStyle.Render("MEMORY"),
				"",
				fmt.Sprintf("Used       %.0f MB", usedMB),
				progressBar(memoryPercent, barWidth),
				"",
				fmt.Sprintf("Total      %.0f MB", totalMB),
				fmt.Sprintf("Available  %.0f MB", availableMB),
			),
		)

		panels = lipgloss.JoinHorizontal(
			lipgloss.Top,
			cpuPanel,
			" ",
			memoryPanel,
		)
	} else {
		panels = lipgloss.JoinVertical(
			lipgloss.Left,
			cpuPanel,
			memoryPanel,
		)
	}

	footer := subtitleStyle.Render(
		fmt.Sprintf("Terminal: %d × %d  •  Refresh: 1s  •  q: quit",
			m.width, m.height),
	)

	var processPanel string

	if m.showTree {
		processPanel = renderProcessTree(
			m.processes,
			m.selected,
			width,
		)
	} else {
		processPanel = renderProcessTable(
			m.processes,
			m.selected,
			width,
			m.sortByMemory,
			m.processErr,
		)
	}

	detailsPanel := renderProcessDetails(
		m.processes,
		m.selected,
		width,
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		panels,
		"",
		processPanel,
		"",
		detailsPanel,
		"",
		footer,
	)
}

func Run() error {
	program := tea.NewProgram(initialModel())
	_, err := program.Run()
	return err
}
