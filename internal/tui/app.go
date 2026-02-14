package tui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lu-zhengda/macbroom/internal/engine"
	"github.com/lu-zhengda/macbroom/internal/maintain"
	"github.com/lu-zhengda/macbroom/internal/scanner"
	"github.com/lu-zhengda/macbroom/internal/trash"
	"github.com/lu-zhengda/macbroom/internal/utils"
)

type viewState int

const (
	viewMenu viewState = iota
	viewDashboard
	viewCategory
	viewConfirm
	viewResult
	viewSpaceLens
	viewMaintain
	viewMaintainResult
)

type scanDoneMsg struct {
	results []engine.ScanResult
}

type cleanDoneMsg struct {
	cleaned int
	failed  int
	size    int64
}

type spaceLensDoneMsg struct {
	nodes []scanner.SpaceLensNode
	path  string
}

type maintainDoneMsg struct {
	results []maintain.Result
}

// menuItem represents a main menu option.
type menuItem struct {
	label       string
	description string
}

var menuItems = []menuItem{
	{"Clean", "Scan and remove junk files"},
	{"Space Lens", "Visualize disk space usage"},
	{"Maintenance", "Run system maintenance tasks"},
}

type Model struct {
	engine      *engine.Engine
	currentView viewState
	results     []engine.ScanResult
	scanning    bool
	cursor      int
	selected    map[int]bool

	categoryIdx    int
	categoryCursor int
	scrollOffset   int

	// Result view state
	lastCleaned int
	lastFailed  int
	lastSize    int64

	// Space Lens state
	slPath    string
	slNodes   []scanner.SpaceLensNode
	slCursor  int
	slLoading bool

	// Maintenance state
	maintainResults []maintain.Result

	spinner spinner.Model

	width  int
	height int
}

func New(e *engine.Engine) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("170"))

	return Model{
		engine:   e,
		selected: make(map[int]bool),
		spinner:  sp,
		slPath:   "/",
	}
}

func (m Model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m Model) doScan() tea.Cmd {
	return func() tea.Msg {
		results := m.engine.ScanGrouped(context.Background())
		return scanDoneMsg{results: results}
	}
}

func (m Model) doSpaceLens() tea.Cmd {
	path := m.slPath
	return func() tea.Msg {
		sl := scanner.NewSpaceLens(path, 1)
		nodes, _ := sl.Analyze(context.Background())
		return spaceLensDoneMsg{nodes: nodes, path: path}
	}
}

func (m Model) doMaintain() tea.Cmd {
	return func() tea.Msg {
		results := maintain.RunSafe()
		return maintainDoneMsg{results: results}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case spinner.TickMsg:
		if m.scanning || m.slLoading || m.currentView == viewMaintain {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case scanDoneMsg:
		m.scanning = false
		m.results = msg.results
		m.currentView = viewDashboard
		return m, nil

	case cleanDoneMsg:
		m.lastCleaned = msg.cleaned
		m.lastFailed = msg.failed
		m.lastSize = msg.size
		m.currentView = viewResult
		return m, nil

	case spaceLensDoneMsg:
		m.slLoading = false
		m.slNodes = msg.nodes
		m.slPath = msg.path
		return m, nil

	case maintainDoneMsg:
		m.maintainResults = msg.results
		m.currentView = viewMaintainResult
		return m, nil

	case tea.KeyMsg:
		// Global quit
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}

		switch m.currentView {
		case viewMenu:
			return m.updateMenu(msg)
		case viewDashboard:
			return m.updateDashboard(msg)
		case viewCategory:
			return m.updateCategory(msg)
		case viewConfirm:
			return m.updateConfirm(msg)
		case viewResult:
			return m.updateResult(msg)
		case viewSpaceLens:
			return m.updateSpaceLens(msg)
		case viewMaintain:
			// waiting for results, no input
		case viewMaintainResult:
			return m.updateMaintainResult(msg)
		}
	}

	return m, nil
}

func (m Model) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(menuItems)-1 {
			m.cursor++
		}
	case "enter":
		switch m.cursor {
		case 0: // Clean
			m.scanning = true
			m.currentView = viewDashboard
			return m, tea.Batch(m.doScan(), m.spinner.Tick)
		case 1: // Space Lens
			m.slLoading = true
			m.slCursor = 0
			m.slPath = "/"
			m.currentView = viewSpaceLens
			return m, tea.Batch(m.doSpaceLens(), m.spinner.Tick)
		case 2: // Maintenance
			m.currentView = viewMaintain
			return m, tea.Batch(m.doMaintain(), m.spinner.Tick)
		}
	}
	return m, nil
}

func (m Model) updateDashboard(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.results)-1 {
			m.cursor++
		}
	case "enter":
		if len(m.results) > 0 {
			m.categoryIdx = m.cursor
			m.categoryCursor = 0
			m.scrollOffset = 0
			m.selected = make(map[int]bool)
			if m.categoryIdx < len(m.results) {
				for i := range m.results[m.categoryIdx].Targets {
					m.selected[i] = true
				}
			}
			m.currentView = viewCategory
		}
	case "esc", "backspace":
		m.currentView = viewMenu
		m.cursor = 0
	}
	return m, nil
}

func (m Model) updateCategory(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.categoryCursor > 0 {
			m.categoryCursor--
			m.ensureCursorVisible()
		}
	case "down", "j":
		if m.categoryIdx < len(m.results) {
			max := len(m.results[m.categoryIdx].Targets) - 1
			if m.categoryCursor < max {
				m.categoryCursor++
				m.ensureCursorVisible()
			}
		}
	case " ":
		if m.selected[m.categoryCursor] {
			delete(m.selected, m.categoryCursor)
		} else {
			m.selected[m.categoryCursor] = true
		}
	case "a":
		// Toggle all
		if len(m.selected) == len(m.results[m.categoryIdx].Targets) {
			m.selected = make(map[int]bool)
		} else {
			for i := range m.results[m.categoryIdx].Targets {
				m.selected[i] = true
			}
		}
	case "d", "enter":
		if len(m.selected) > 0 {
			m.currentView = viewConfirm
		}
	case "esc", "backspace":
		m.currentView = viewDashboard
		m.cursor = m.categoryIdx
	}
	return m, nil
}

func (m *Model) ensureCursorVisible() {
	visible := m.visibleItemCount()
	if m.categoryCursor < m.scrollOffset {
		m.scrollOffset = m.categoryCursor
	}
	if m.categoryCursor >= m.scrollOffset+visible {
		m.scrollOffset = m.categoryCursor - visible + 1
	}
}

func (m Model) visibleItemCount() int {
	// Reserve lines for: title(2) + status bar(1) + help(2) + padding(1) = 6
	available := m.height - 6
	if available < 5 {
		available = 5
	}
	return available
}

func (m Model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		return m, m.doClean()
	case "n", "esc", "backspace":
		m.currentView = viewCategory
	}
	return m, nil
}

func (m Model) updateResult(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "r":
		m.scanning = true
		m.currentView = viewDashboard
		m.cursor = 0
		return m, tea.Batch(m.doScan(), m.spinner.Tick)
	case "esc", "backspace":
		m.currentView = viewMenu
		m.cursor = 0
	}
	return m, nil
}

func (m Model) updateSpaceLens(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.slLoading {
		return m, nil
	}
	switch msg.String() {
	case "up", "k":
		if m.slCursor > 0 {
			m.slCursor--
		}
	case "down", "j":
		max := len(m.slNodes) - 1
		if max > 29 {
			max = 29
		}
		if m.slCursor < max {
			m.slCursor++
		}
	case "enter":
		if m.slCursor < len(m.slNodes) && m.slNodes[m.slCursor].IsDir {
			m.slPath = m.slNodes[m.slCursor].Path
			m.slLoading = true
			m.slCursor = 0
			return m, tea.Batch(m.doSpaceLens(), m.spinner.Tick)
		}
	case "h":
		// Go up one directory
		if idx := lastSlash(m.slPath); idx > 0 {
			m.slPath = m.slPath[:idx]
			m.slLoading = true
			m.slCursor = 0
			return m, tea.Batch(m.doSpaceLens(), m.spinner.Tick)
		}
	case "esc", "backspace":
		m.currentView = viewMenu
		m.cursor = 1
	}
	return m, nil
}

func (m Model) updateMaintainResult(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "backspace", "enter":
		m.currentView = viewMenu
		m.cursor = 2
	}
	return m, nil
}

func (m Model) doClean() tea.Cmd {
	return func() tea.Msg {
		if m.categoryIdx >= len(m.results) {
			return cleanDoneMsg{}
		}
		targets := m.results[m.categoryIdx].Targets
		var cleaned, failed int
		var totalSize int64
		for i, t := range targets {
			if !m.selected[i] {
				continue
			}
			if err := trash.MoveToTrash(t.Path); err != nil {
				failed++
			} else {
				cleaned++
				totalSize += t.Size
			}
		}
		return cleanDoneMsg{cleaned: cleaned, failed: failed, size: totalSize}
	}
}

// --- Views ---

func (m Model) View() string {
	if m.scanning {
		return titleStyle.Render("macbroom") + "\n\n" + m.spinner.View() + " Scanning your Mac...\n"
	}

	switch m.currentView {
	case viewMenu:
		return m.viewMenu()
	case viewDashboard:
		return m.viewDashboard()
	case viewCategory:
		return m.viewCategory()
	case viewConfirm:
		return m.viewConfirm()
	case viewResult:
		return m.viewResult()
	case viewSpaceLens:
		return m.viewSpaceLens()
	case viewMaintain:
		return m.viewMaintain()
	case viewMaintainResult:
		return m.viewMaintainResult()
	default:
		return m.viewMenu()
	}
}

func (m Model) viewMenu() string {
	s := titleStyle.Render("macbroom") + "\n\n"

	for i, item := range menuItems {
		if i == m.cursor {
			s += selectedStyle.Render("> "+item.label) + "  " + dimStyle.Render(item.description) + "\n"
		} else {
			s += fmt.Sprintf("  %-15s %s\n", item.label, dimStyle.Render(item.description))
		}
	}

	s += helpStyle.Render("\n\nj/k navigate | enter select | q quit")
	return s
}

func (m Model) viewDashboard() string {
	s := titleStyle.Render("macbroom -- Clean") + "\n\n"

	if len(m.results) == 0 {
		s += "No junk found. Your Mac is clean!\n"
		return s + helpStyle.Render("\nesc back | q quit")
	}

	var totalSize int64
	for i, r := range m.results {
		var catSize int64
		for _, t := range r.Targets {
			catSize += t.Size
		}
		totalSize += catSize

		cursor := "  "
		style := dimStyle
		if i == m.cursor {
			cursor = "> "
			style = selectedStyle
		}

		line := fmt.Sprintf("%s%-25s %10s  (%d items)",
			cursor, r.Category, utils.FormatSize(catSize), len(r.Targets))
		s += style.Render(line) + "\n"
	}

	s += "\n" + statusBarStyle.Render(fmt.Sprintf(" Total reclaimable: %s ", utils.FormatSize(totalSize)))
	s += helpStyle.Render("\n\nj/k navigate | enter view details | esc back | q quit")
	return s
}

func (m Model) viewCategory() string {
	if m.categoryIdx >= len(m.results) {
		return "No category selected"
	}

	r := m.results[m.categoryIdx]
	s := titleStyle.Render("macbroom -- "+r.Category) + "\n\n"

	if r.Error != nil {
		s += fmt.Sprintf("Error scanning: %v\n", r.Error)
		return s
	}

	visible := m.visibleItemCount()
	total := len(r.Targets)
	end := m.scrollOffset + visible
	if end > total {
		end = total
	}

	for i := m.scrollOffset; i < end; i++ {
		t := r.Targets[i]
		cursor := "  "
		if i == m.categoryCursor {
			cursor = "> "
		}

		check := "[ ]"
		if m.selected[i] {
			check = "[x]"
		}

		line := fmt.Sprintf("%s%s %-35s %10s",
			cursor, check, truncPath(t.Path, 35), utils.FormatSize(t.Size))

		if i == m.categoryCursor {
			s += selectedStyle.Render(line) + "\n"
		} else {
			s += line + "\n"
		}
	}

	// Scroll indicator
	if total > visible {
		s += dimStyle.Render(fmt.Sprintf("  [%d-%d of %d]", m.scrollOffset+1, end, total)) + "\n"
	}

	var selectedSize int64
	var selectedCount int
	for i, t := range r.Targets {
		if m.selected[i] {
			selectedSize += t.Size
			selectedCount++
		}
	}

	s += "\n" + statusBarStyle.Render(fmt.Sprintf(" Selected: %d items (%s) ", selectedCount, utils.FormatSize(selectedSize)))
	s += helpStyle.Render("\n\nj/k navigate | space toggle | a toggle all | d delete | esc back | q quit")
	return s
}

func (m Model) viewConfirm() string {
	if m.categoryIdx >= len(m.results) {
		return "Nothing to confirm"
	}

	r := m.results[m.categoryIdx]

	dangerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("196")).
		Background(lipgloss.Color("52")).
		Padding(0, 1)

	s := dangerStyle.Render(" CONFIRM DELETION ") + "\n\n"

	var selectedSize int64
	var selectedCount int
	var hasRisky bool
	for i, t := range r.Targets {
		if m.selected[i] {
			selectedSize += t.Size
			selectedCount++
			if t.Risk >= scanner.Risky {
				hasRisky = true
			}

			riskLabel := ""
			if t.Risk >= scanner.Moderate {
				riskLabel = fmt.Sprintf(" [%s]", t.Risk)
			}
			s += fmt.Sprintf("  %s (%s)%s\n", truncPath(t.Path, 45), utils.FormatSize(t.Size), riskLabel)
		}
	}

	s += "\n"
	if hasRisky {
		warnStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
		s += warnStyle.Render("  WARNING: Selection includes risky items that may contain user data!") + "\n\n"
	}

	s += fmt.Sprintf("  %d items | %s | will be moved to Trash (recoverable)\n", selectedCount, utils.FormatSize(selectedSize))
	s += helpStyle.Render("\n  y confirm | n cancel | q quit")
	return s
}

func (m Model) viewResult() string {
	s := titleStyle.Render("macbroom -- Cleanup Complete") + "\n\n"

	successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("82"))
	failStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))

	s += successStyle.Render(fmt.Sprintf("  Cleaned: %d items (%s freed)", m.lastCleaned, utils.FormatSize(m.lastSize))) + "\n"
	if m.lastFailed > 0 {
		s += failStyle.Render(fmt.Sprintf("  Failed:  %d items", m.lastFailed)) + "\n"
	}

	s += helpStyle.Render("\n  r re-scan | esc menu | q quit")
	return s
}

func (m Model) viewSpaceLens() string {
	s := titleStyle.Render("macbroom -- Space Lens") + "\n"
	s += dimStyle.Render(m.slPath) + "\n\n"

	if m.slLoading {
		return s + m.spinner.View() + " Analyzing...\n"
	}

	if len(m.slNodes) == 0 {
		s += "Empty directory.\n"
		return s + helpStyle.Render("\nesc back | q quit")
	}

	maxSize := m.slNodes[0].Size
	visible := m.slNodes
	if len(visible) > 30 {
		visible = visible[:30]
	}

	for i, node := range visible {
		cursor := "  "
		if i == m.slCursor {
			cursor = "> "
		}

		icon := "  "
		if node.IsDir {
			icon = "D "
		}

		name := node.Name
		if len(name) > 30 {
			name = name[:27] + "..."
		}

		bar := renderBar(node.Size, maxSize, 25)
		line := fmt.Sprintf("%s%s %-30s %10s %s",
			cursor, icon, name, utils.FormatSize(node.Size), bar)

		if i == m.slCursor {
			s += selectedStyle.Render(line) + "\n"
		} else {
			s += line + "\n"
		}
	}

	if len(m.slNodes) > 30 {
		s += dimStyle.Render(fmt.Sprintf("\n  ... and %d more items", len(m.slNodes)-30)) + "\n"
	}

	s += helpStyle.Render("\nj/k navigate | enter drill in | h go up | esc back | q quit")
	return s
}

func (m Model) viewMaintain() string {
	s := titleStyle.Render("macbroom -- Maintenance") + "\n\n"
	s += m.spinner.View() + " Running maintenance tasks...\n"
	return s
}

func (m Model) viewMaintainResult() string {
	s := titleStyle.Render("macbroom -- Maintenance Complete") + "\n\n"

	successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("82"))
	failStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	skipStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))

	for _, r := range m.maintainResults {
		if r.Success {
			s += successStyle.Render(fmt.Sprintf("  [OK]     %s", r.Task.Name)) + "\n"
		} else if r.Task.NeedsSudo {
			s += skipStyle.Render(fmt.Sprintf("  [SKIP]   %s (requires sudo)", r.Task.Name)) + "\n"
		} else {
			s += failStyle.Render(fmt.Sprintf("  [FAIL]   %s: %v", r.Task.Name, r.Error)) + "\n"
		}
	}

	hasSudo := false
	for _, r := range m.maintainResults {
		if r.Task.NeedsSudo {
			hasSudo = true
			break
		}
	}
	if hasSudo {
		s += dimStyle.Render("\n  Tip: run 'macbroom maintain' in terminal for sudo tasks") + "\n"
	}

	s += helpStyle.Render("\nesc back | q quit")
	return s
}

func truncPath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}
	return "..." + path[len(path)-maxLen+3:]
}

func lastSlash(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '/' {
			return i
		}
	}
	return -1
}
