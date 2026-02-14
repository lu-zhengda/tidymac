package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lu-zhengda/macbroom/internal/dupes"
	"github.com/lu-zhengda/macbroom/internal/engine"
	"github.com/lu-zhengda/macbroom/internal/history"
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
	viewSpaceLensConfirm
	viewMaintain
	viewMaintainResult
	viewDupes
	viewDupesResult
	viewDupesConfirm
	viewUninstallInput
	viewUninstallResults
	viewUninstallConfirm
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

type spaceLensProgressMsg struct {
	name string
}

type maintainDoneMsg struct {
	results []maintain.Result
}

type dupesDoneMsg struct {
	groups []dupes.Group
}

type dupesProgressMsg struct {
	path string
}

type dupesCleanDoneMsg struct {
	deleted int
	failed  int
	freed   int64
}

type uninstallScanDoneMsg struct {
	targets []scanner.Target
}

type uninstallCleanDoneMsg struct {
	deleted int
	failed  int
	freed   int64
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
	{"Duplicates", "Find and remove duplicate files"},
	{"Uninstall", "Remove apps and all related files"},
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
	slPath         string
	slNodes        []scanner.SpaceLensNode
	slCursor       int
	slScrollOffset int
	slLoading      bool
	slScanning     string // current item being scanned
	slCancel       context.CancelFunc
	slProgressCh   chan string
	slDeleteTarget *scanner.SpaceLensNode

	// Maintenance state
	maintainResults []maintain.Result

	// Duplicates state
	dupGroups       []dupes.Group
	dupLoading      bool
	dupScanning     string // current file being scanned
	dupCancel       context.CancelFunc
	dupProgressCh   chan string
	dupCursor       int
	dupScrollOffset int
	dupSelected     map[string]bool // key: "groupIdx:fileIdx", tracks copies to delete
	dupDeleted      int
	dupFailed       int
	dupFreed        int64

	// Uninstall state
	uiTextInput    textinput.Model
	uiTargets      []scanner.Target
	uiLoading      bool
	uiCursor       int
	uiScrollOffset int
	uiSelected     map[int]bool
	uiDeleted      int
	uiFailed       int
	uiFreed        int64

	spinner spinner.Model

	width  int
	height int
}

func New(e *engine.Engine) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("170"))

	ti := textinput.New()
	ti.Placeholder = "Enter app name..."
	ti.CharLimit = 100
	ti.Width = 40

	return Model{
		engine:      e,
		selected:    make(map[int]bool),
		spinner:     sp,
		slPath:      "/",
		uiTextInput: ti,
		uiSelected:  make(map[int]bool),
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

func startSpaceLens(path string) (context.CancelFunc, chan string, tea.Cmd) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan string, 1)

	analyzeCmd := func() tea.Msg {
		sl := scanner.NewSpaceLens(path, 1)
		sl.SetProgress(func(name string) {
			select {
			case ch <- name:
			default:
			}
		})
		nodes, _ := sl.Analyze(ctx)
		close(ch)
		return spaceLensDoneMsg{nodes: nodes, path: path}
	}

	return cancel, ch, tea.Batch(analyzeCmd, listenSpaceLensProgress(ch))
}

func listenSpaceLensProgress(ch chan string) tea.Cmd {
	return func() tea.Msg {
		name, ok := <-ch
		if !ok {
			return nil
		}
		return spaceLensProgressMsg{name: name}
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
		if m.scanning || m.slLoading || m.dupLoading || m.uiLoading || m.currentView == viewMaintain {
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

		// Record cleanup history.
		if msg.cleaned > 0 && m.categoryIdx < len(m.results) {
			h := history.New(history.DefaultPath())
			_ = h.Record(history.Entry{
				Timestamp:  time.Now(),
				Category:   m.results[m.categoryIdx].Category,
				Items:      msg.cleaned,
				BytesFreed: msg.size,
				Method:     "trash",
			})
		}

		return m, nil

	case spaceLensProgressMsg:
		m.slScanning = msg.name
		if m.slProgressCh != nil {
			return m, listenSpaceLensProgress(m.slProgressCh)
		}
		return m, nil

	case spaceLensDoneMsg:
		m.slLoading = false
		m.slNodes = msg.nodes
		m.slPath = msg.path
		m.slScanning = ""
		m.slCancel = nil
		m.slProgressCh = nil
		m.slScrollOffset = 0
		return m, nil

	case maintainDoneMsg:
		m.maintainResults = msg.results
		m.currentView = viewMaintainResult
		return m, nil

	case dupesProgressMsg:
		m.dupScanning = msg.path
		if m.dupProgressCh != nil {
			return m, listenDupesProgress(m.dupProgressCh)
		}
		return m, nil

	case dupesDoneMsg:
		m.dupLoading = false
		m.dupGroups = msg.groups
		m.dupScanning = ""
		m.dupCancel = nil
		m.dupProgressCh = nil
		m.dupCursor = 0
		m.dupScrollOffset = 0
		// Pre-select all copies (skip index 0 in each group = the "keep" file).
		m.dupSelected = make(map[string]bool)
		for gi, g := range m.dupGroups {
			for fi := 1; fi < len(g.Files); fi++ {
				m.dupSelected[fmt.Sprintf("%d:%d", gi, fi)] = true
			}
		}
		return m, nil

	case dupesCleanDoneMsg:
		m.dupDeleted = msg.deleted
		m.dupFailed = msg.failed
		m.dupFreed = msg.freed
		m.currentView = viewDupesResult
		return m, nil

	case uninstallScanDoneMsg:
		m.uiLoading = false
		m.uiTargets = msg.targets
		m.uiCursor = 0
		m.uiScrollOffset = 0
		m.uiSelected = make(map[int]bool)
		for i := range msg.targets {
			m.uiSelected[i] = true
		}
		m.currentView = viewUninstallResults
		return m, nil

	case uninstallCleanDoneMsg:
		m.uiDeleted = msg.deleted
		m.uiFailed = msg.failed
		m.uiFreed = msg.freed
		// Re-use viewResult for the uninstall result display.
		m.lastCleaned = msg.deleted
		m.lastFailed = msg.failed
		m.lastSize = msg.freed
		m.currentView = viewResult
		return m, nil

	case tea.KeyMsg:
		// Global quit (skip "q" when user is typing in text input).
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if msg.String() == "q" && !(m.currentView == viewUninstallInput && !m.uiLoading) {
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
		case viewSpaceLensConfirm:
			return m.updateSpaceLensConfirm(msg)
		case viewMaintain:
			// waiting for results, no input
		case viewMaintainResult:
			return m.updateMaintainResult(msg)
		case viewDupes:
			return m.updateDupes(msg)
		case viewDupesConfirm:
			return m.updateDupesConfirm(msg)
		case viewDupesResult:
			return m.updateDupesResult(msg)
		case viewUninstallInput:
			return m.updateUninstallInput(msg)
		case viewUninstallResults:
			return m.updateUninstallResults(msg)
		case viewUninstallConfirm:
			return m.updateUninstallConfirm(msg)
		}

	default:
		// Forward cursor blink and other messages to the text input when active.
		if m.currentView == viewUninstallInput && !m.uiLoading {
			var cmd tea.Cmd
			m.uiTextInput, cmd = m.uiTextInput.Update(msg)
			return m, cmd
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
			m.slScrollOffset = 0
			m.slPath = "/"
			m.currentView = viewSpaceLens
			cancel, ch, cmd := startSpaceLens(m.slPath)
			m.slCancel = cancel
			m.slProgressCh = ch
			return m, tea.Batch(cmd, m.spinner.Tick)
		case 2: // Maintenance
			m.currentView = viewMaintain
			return m, tea.Batch(m.doMaintain(), m.spinner.Tick)
		case 3: // Duplicates
			m.dupLoading = true
			m.dupCursor = 0
			m.dupScrollOffset = 0
			m.currentView = viewDupes
			cancel, ch, cmd := startDupesScan()
			m.dupCancel = cancel
			m.dupProgressCh = ch
			return m, tea.Batch(cmd, m.spinner.Tick)
		case 4: // Uninstall
			m.uiTextInput.Reset()
			m.uiTextInput.Focus()
			m.uiTargets = nil
			m.uiSelected = make(map[int]bool)
			m.currentView = viewUninstallInput
			return m, m.uiTextInput.Cursor.BlinkCmd()
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
		if msg.String() == "esc" || msg.String() == "backspace" {
			if m.slCancel != nil {
				m.slCancel()
			}
			m.slLoading = false
			m.slScanning = ""
			m.slCancel = nil
			m.slProgressCh = nil
			m.currentView = viewMenu
			m.cursor = 1
		}
		return m, nil
	}
	switch msg.String() {
	case "up", "k":
		if m.slCursor > 0 {
			m.slCursor--
			m.slEnsureCursorVisible()
		}
	case "down", "j":
		max := len(m.slNodes) - 1
		if m.slCursor < max {
			m.slCursor++
			m.slEnsureCursorVisible()
		}
	case "enter":
		if m.slCursor < len(m.slNodes) && m.slNodes[m.slCursor].IsDir {
			m.slPath = m.slNodes[m.slCursor].Path
			m.slLoading = true
			m.slCursor = 0
			m.slScrollOffset = 0
			cancel, ch, cmd := startSpaceLens(m.slPath)
			m.slCancel = cancel
			m.slProgressCh = ch
			return m, tea.Batch(cmd, m.spinner.Tick)
		}
	case "d":
		if m.slCursor < len(m.slNodes) {
			node := m.slNodes[m.slCursor]
			m.slDeleteTarget = &node
			m.currentView = viewSpaceLensConfirm
		}
	case "h":
		// Go up one directory
		if idx := lastSlash(m.slPath); idx > 0 {
			m.slPath = m.slPath[:idx]
			m.slLoading = true
			m.slCursor = 0
			m.slScrollOffset = 0
			cancel, ch, cmd := startSpaceLens(m.slPath)
			m.slCancel = cancel
			m.slProgressCh = ch
			return m, tea.Batch(cmd, m.spinner.Tick)
		}
	case "esc", "backspace":
		m.currentView = viewMenu
		m.cursor = 1
	}
	return m, nil
}

func (m *Model) slEnsureCursorVisible() {
	visible := m.visibleItemCount()
	if m.slCursor < m.slScrollOffset {
		m.slScrollOffset = m.slCursor
	}
	if m.slCursor >= m.slScrollOffset+visible {
		m.slScrollOffset = m.slCursor - visible + 1
	}
}

func (m Model) updateSpaceLensConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		if m.slDeleteTarget != nil {
			_ = trash.MoveToTrash(m.slDeleteTarget.Path)
			m.slDeleteTarget = nil
			m.slLoading = true
			m.slCursor = 0
			m.slScrollOffset = 0
			m.currentView = viewSpaceLens
			cancel, ch, cmd := startSpaceLens(m.slPath)
			m.slCancel = cancel
			m.slProgressCh = ch
			return m, tea.Batch(cmd, m.spinner.Tick)
		}
		m.currentView = viewSpaceLens
	case "n", "esc":
		m.slDeleteTarget = nil
		m.currentView = viewSpaceLens
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

func startDupesScan() (context.CancelFunc, chan string, tea.Cmd) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan string, 1)

	home := utils.HomeDir()
	dirs := []string{
		filepath.Join(home, "Downloads"),
		filepath.Join(home, "Desktop"),
		filepath.Join(home, "Documents"),
	}

	scanCmd := func() tea.Msg {
		groups, _ := dupes.FindWithProgress(ctx, dirs, 0, func(path string) {
			select {
			case ch <- path:
			default:
			}
		})
		close(ch)
		return dupesDoneMsg{groups: groups}
	}

	return cancel, ch, tea.Batch(scanCmd, listenDupesProgress(ch))
}

func listenDupesProgress(ch chan string) tea.Cmd {
	return func() tea.Msg {
		path, ok := <-ch
		if !ok {
			return nil
		}
		return dupesProgressMsg{path: path}
	}
}

// dupesFileList returns a flat list of (groupIdx, fileIdx, path, size, isKeep) for display.
type dupesEntry struct {
	groupIdx int
	fileIdx  int
	path     string
	size     int64
	hash     string
	isKeep   bool
	isHeader bool
}

func (m Model) dupesFlatList() []dupesEntry {
	var entries []dupesEntry
	for gi, g := range m.dupGroups {
		wasted := g.Size * int64(len(g.Files)-1)
		entries = append(entries, dupesEntry{
			groupIdx: gi,
			isHeader: true,
			size:     wasted,
			hash:     g.Hash,
		})
		for fi, f := range g.Files {
			entries = append(entries, dupesEntry{
				groupIdx: gi,
				fileIdx:  fi,
				path:     f,
				size:     g.Size,
				isKeep:   fi == 0,
			})
		}
	}
	return entries
}

func (m Model) updateDupes(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.dupLoading {
		if msg.String() == "esc" || msg.String() == "backspace" {
			if m.dupCancel != nil {
				m.dupCancel()
			}
			m.dupLoading = false
			m.dupScanning = ""
			m.dupCancel = nil
			m.dupProgressCh = nil
			m.currentView = viewMenu
			m.cursor = 3
		}
		return m, nil
	}

	entries := m.dupesFlatList()

	switch msg.String() {
	case "up", "k":
		if m.dupCursor > 0 {
			m.dupCursor--
			// Skip headers.
			if m.dupCursor >= 0 && m.dupCursor < len(entries) && entries[m.dupCursor].isHeader {
				if m.dupCursor > 0 {
					m.dupCursor--
				}
			}
			m.dupEnsureCursorVisible(entries)
		}
	case "down", "j":
		if m.dupCursor < len(entries)-1 {
			m.dupCursor++
			// Skip headers.
			if m.dupCursor < len(entries) && entries[m.dupCursor].isHeader {
				if m.dupCursor < len(entries)-1 {
					m.dupCursor++
				}
			}
			m.dupEnsureCursorVisible(entries)
		}
	case " ":
		if m.dupCursor < len(entries) {
			e := entries[m.dupCursor]
			if !e.isHeader && !e.isKeep {
				key := fmt.Sprintf("%d:%d", e.groupIdx, e.fileIdx)
				if m.dupSelected[key] {
					delete(m.dupSelected, key)
				} else {
					m.dupSelected[key] = true
				}
			}
		}
	case "d", "enter":
		if len(m.dupSelected) > 0 {
			m.currentView = viewDupesConfirm
		}
	case "esc", "backspace":
		m.currentView = viewMenu
		m.cursor = 3
	}
	return m, nil
}

func (m *Model) dupEnsureCursorVisible(entries []dupesEntry) {
	visible := m.visibleItemCount()
	if m.dupCursor < m.dupScrollOffset {
		m.dupScrollOffset = m.dupCursor
	}
	if m.dupCursor >= m.dupScrollOffset+visible {
		m.dupScrollOffset = m.dupCursor - visible + 1
	}
}

func (m Model) updateDupesConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		return m, m.doDupesClean()
	case "n", "esc", "backspace":
		m.currentView = viewDupes
	}
	return m, nil
}

func (m Model) updateDupesResult(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "r":
		// Re-scan.
		m.dupLoading = true
		m.dupCursor = 0
		m.dupScrollOffset = 0
		m.currentView = viewDupes
		cancel, ch, cmd := startDupesScan()
		m.dupCancel = cancel
		m.dupProgressCh = ch
		return m, tea.Batch(cmd, m.spinner.Tick)
	case "esc", "backspace", "enter":
		m.currentView = viewMenu
		m.cursor = 3
	}
	return m, nil
}

func (m Model) doDupesClean() tea.Cmd {
	return func() tea.Msg {
		var deleted, failed int
		var freed int64
		for gi, g := range m.dupGroups {
			for fi, f := range g.Files {
				key := fmt.Sprintf("%d:%d", gi, fi)
				if !m.dupSelected[key] {
					continue
				}
				if err := trash.MoveToTrash(f); err != nil {
					failed++
				} else {
					deleted++
					freed += g.Size
				}
			}
		}
		return dupesCleanDoneMsg{deleted: deleted, failed: failed, freed: freed}
	}
}

func (m Model) updateUninstallInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		appName := m.uiTextInput.Value()
		if appName == "" {
			return m, nil
		}
		m.uiLoading = true
		return m, tea.Batch(m.doUninstallScan(appName), m.spinner.Tick)
	case "esc":
		m.currentView = viewMenu
		m.cursor = 4
		return m, nil
	}

	var cmd tea.Cmd
	m.uiTextInput, cmd = m.uiTextInput.Update(msg)
	return m, cmd
}

func (m Model) doUninstallScan(appName string) tea.Cmd {
	return func() tea.Msg {
		s := scanner.NewAppScanner("", "")
		targets, _ := s.FindRelatedFiles(context.Background(), appName)
		return uninstallScanDoneMsg{targets: targets}
	}
}

func (m Model) updateUninstallResults(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.uiCursor > 0 {
			m.uiCursor--
			m.uiEnsureCursorVisible()
		}
	case "down", "j":
		if m.uiCursor < len(m.uiTargets)-1 {
			m.uiCursor++
			m.uiEnsureCursorVisible()
		}
	case " ":
		if m.uiSelected[m.uiCursor] {
			delete(m.uiSelected, m.uiCursor)
		} else {
			m.uiSelected[m.uiCursor] = true
		}
	case "a":
		if len(m.uiSelected) == len(m.uiTargets) {
			m.uiSelected = make(map[int]bool)
		} else {
			for i := range m.uiTargets {
				m.uiSelected[i] = true
			}
		}
	case "d", "enter":
		if len(m.uiSelected) > 0 {
			m.currentView = viewUninstallConfirm
		}
	case "esc", "backspace":
		m.uiTextInput.Focus()
		m.currentView = viewUninstallInput
		return m, m.uiTextInput.Cursor.BlinkCmd()
	}
	return m, nil
}

func (m *Model) uiEnsureCursorVisible() {
	visible := m.visibleItemCount()
	if m.uiCursor < m.uiScrollOffset {
		m.uiScrollOffset = m.uiCursor
	}
	if m.uiCursor >= m.uiScrollOffset+visible {
		m.uiScrollOffset = m.uiCursor - visible + 1
	}
}

func (m Model) updateUninstallConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		return m, m.doUninstallClean()
	case "n", "esc", "backspace":
		m.currentView = viewUninstallResults
	}
	return m, nil
}

func (m Model) doUninstallClean() tea.Cmd {
	return func() tea.Msg {
		var deleted, failed int
		var freed int64
		for i, t := range m.uiTargets {
			if !m.uiSelected[i] {
				continue
			}
			if err := trash.MoveToTrash(t.Path); err != nil {
				failed++
			} else {
				deleted++
				freed += t.Size
			}
		}
		return uninstallCleanDoneMsg{deleted: deleted, failed: failed, freed: freed}
	}
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
	case viewSpaceLensConfirm:
		return m.viewSpaceLensConfirm()
	case viewMaintain:
		return m.viewMaintain()
	case viewMaintainResult:
		return m.viewMaintainResult()
	case viewDupes:
		return m.viewDupes()
	case viewDupesConfirm:
		return m.viewDupesConfirm()
	case viewDupesResult:
		return m.viewDupesResult()
	case viewUninstallInput:
		return m.viewUninstallInput()
	case viewUninstallResults:
		return m.viewUninstallResults()
	case viewUninstallConfirm:
		return m.viewUninstallConfirm()
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

	if m.slLoading {
		s += dimStyle.Render(m.slPath) + "\n\n"
		s += m.spinner.View() + " Analyzing...\n"
		if m.slScanning != "" {
			name := m.slScanning
			if len(name) > 40 {
				name = name[:37] + "..."
			}
			s += dimStyle.Render("  "+name) + "\n"
		}
		s += helpStyle.Render("\nesc cancel")
		return s
	}

	// Calculate total size for header and percentages.
	var totalSize int64
	for _, node := range m.slNodes {
		totalSize += node.Size
	}

	s += dimStyle.Render(fmt.Sprintf("%s (%s)", m.slPath, utils.FormatSize(totalSize))) + "\n\n"

	if len(m.slNodes) == 0 {
		s += "Empty directory.\n"
		return s + helpStyle.Render("\nesc back | q quit")
	}

	maxSize := m.slNodes[0].Size
	visible := m.visibleItemCount()
	total := len(m.slNodes)
	end := m.slScrollOffset + visible
	if end > total {
		end = total
	}

	for i := m.slScrollOffset; i < end; i++ {
		node := m.slNodes[i]
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

		pct := 0
		if totalSize > 0 {
			pct = int(float64(node.Size) / float64(totalSize) * 100)
		}

		bar := renderBar(node.Size, maxSize, 25)
		line := fmt.Sprintf("%s%s %-30s %10s  %3d%%  %s",
			cursor, icon, name, utils.FormatSize(node.Size), pct, bar)

		if i == m.slCursor {
			s += selectedStyle.Render(line) + "\n"
		} else {
			s += line + "\n"
		}
	}

	// Scroll indicator
	if total > visible {
		s += dimStyle.Render(fmt.Sprintf("  [%d-%d of %d]", m.slScrollOffset+1, end, total)) + "\n"
	}

	s += helpStyle.Render("\nj/k navigate | enter drill in | d delete | h go up | esc back | q quit")
	return s
}

func (m Model) viewSpaceLensConfirm() string {
	if m.slDeleteTarget == nil {
		return "Nothing to confirm"
	}

	dangerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("196")).
		Background(lipgloss.Color("52")).
		Padding(0, 1)

	s := dangerStyle.Render(" DELETE ITEM ") + "\n\n"
	s += fmt.Sprintf("  Delete %s (%s)? (y/n)\n", m.slDeleteTarget.Name, utils.FormatSize(m.slDeleteTarget.Size))
	s += dimStyle.Render(fmt.Sprintf("\n  Path: %s", m.slDeleteTarget.Path)) + "\n"
	s += helpStyle.Render("\n  y confirm | n cancel | q quit")
	return s
}

func (m Model) viewDupes() string {
	s := titleStyle.Render("macbroom -- Duplicates") + "\n"

	if m.dupLoading {
		s += "\n" + m.spinner.View() + " Scanning for duplicates...\n"
		if m.dupScanning != "" {
			name := m.dupScanning
			if len(name) > 50 {
				name = "..." + name[len(name)-47:]
			}
			s += dimStyle.Render("  "+name) + "\n"
		}
		s += helpStyle.Render("\nesc cancel")
		return s
	}

	if len(m.dupGroups) == 0 {
		s += "\nNo duplicates found!\n"
		return s + helpStyle.Render("\nesc back | q quit")
	}

	var totalWasted int64
	var totalCopies int
	for _, g := range m.dupGroups {
		totalWasted += g.Size * int64(len(g.Files)-1)
		totalCopies += len(g.Files) - 1
	}
	s += dimStyle.Render(fmt.Sprintf("%d groups, %d copies, %s wasted",
		len(m.dupGroups), totalCopies, utils.FormatSize(totalWasted))) + "\n\n"

	entries := m.dupesFlatList()
	visible := m.visibleItemCount()
	total := len(entries)
	end := m.dupScrollOffset + visible
	if end > total {
		end = total
	}

	for i := m.dupScrollOffset; i < end; i++ {
		e := entries[i]
		if e.isHeader {
			hashShort := e.hash
			if len(hashShort) > 12 {
				hashShort = hashShort[:12]
			}
			s += dimStyle.Render(fmt.Sprintf("  --- Group %d: %s wasted (hash: %s...) ---",
				e.groupIdx+1, utils.FormatSize(e.size), hashShort)) + "\n"
			continue
		}

		cursor := "  "
		if i == m.dupCursor {
			cursor = "> "
		}

		label := "[keep]"
		check := ""
		if !e.isKeep {
			key := fmt.Sprintf("%d:%d", e.groupIdx, e.fileIdx)
			if m.dupSelected[key] {
				check = "[x] "
			} else {
				check = "[ ] "
			}
			label = ""
		} else {
			label = "[keep] "
		}

		path := e.path
		if len(path) > 45 {
			path = "..." + path[len(path)-42:]
		}

		line := fmt.Sprintf("%s%s%s%s (%s)", cursor, label, check, path, utils.FormatSize(e.size))

		if i == m.dupCursor {
			s += selectedStyle.Render(line) + "\n"
		} else {
			s += line + "\n"
		}
	}

	if total > visible {
		s += dimStyle.Render(fmt.Sprintf("  [%d-%d of %d]", m.dupScrollOffset+1, end, total)) + "\n"
	}

	var selectedSize int64
	var selectedCount int
	for key := range m.dupSelected {
		selectedCount++
		var gi, fi int
		fmt.Sscanf(key, "%d:%d", &gi, &fi)
		if gi < len(m.dupGroups) {
			selectedSize += m.dupGroups[gi].Size
		}
	}

	s += "\n" + statusBarStyle.Render(fmt.Sprintf(" Selected: %d copies (%s) ", selectedCount, utils.FormatSize(selectedSize)))
	s += helpStyle.Render("\n\nj/k navigate | space toggle | d delete selected | esc back | q quit")
	return s
}

func (m Model) viewDupesConfirm() string {
	dangerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("196")).
		Background(lipgloss.Color("52")).
		Padding(0, 1)

	s := dangerStyle.Render(" CONFIRM DUPLICATE DELETION ") + "\n\n"

	var selectedSize int64
	var selectedCount int
	for key := range m.dupSelected {
		selectedCount++
		var gi, fi int
		fmt.Sscanf(key, "%d:%d", &gi, &fi)
		if gi < len(m.dupGroups) {
			selectedSize += m.dupGroups[gi].Size
		}
	}

	s += fmt.Sprintf("  %d duplicate copies | %s | will be moved to Trash (recoverable)\n",
		selectedCount, utils.FormatSize(selectedSize))
	s += dimStyle.Render("\n  One copy per group will be kept.") + "\n"
	s += helpStyle.Render("\n  y confirm | n cancel | q quit")
	return s
}

func (m Model) viewDupesResult() string {
	s := titleStyle.Render("macbroom -- Duplicates Cleaned") + "\n\n"

	successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("82"))
	failStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))

	s += successStyle.Render(fmt.Sprintf("  Deleted: %d copies (%s freed)", m.dupDeleted, utils.FormatSize(m.dupFreed))) + "\n"
	if m.dupFailed > 0 {
		s += failStyle.Render(fmt.Sprintf("  Failed:  %d items", m.dupFailed)) + "\n"
	}

	s += helpStyle.Render("\n  r re-scan | esc menu | q quit")
	return s
}

func (m Model) viewUninstallInput() string {
	s := titleStyle.Render("macbroom -- Uninstall") + "\n\n"

	if m.uiLoading {
		s += m.spinner.View() + " Searching for app files...\n"
		return s
	}

	s += "  Enter the name of the application to uninstall:\n\n"
	s += "  " + m.uiTextInput.View() + "\n"
	s += helpStyle.Render("\n\nenter search | esc back | q quit")
	return s
}

func (m Model) viewUninstallResults() string {
	s := titleStyle.Render("macbroom -- Uninstall") + "\n\n"

	if len(m.uiTargets) == 0 {
		s += fmt.Sprintf("  No files found for %q.\n", m.uiTextInput.Value())
		return s + helpStyle.Render("\nesc search again | q quit")
	}

	s += dimStyle.Render(fmt.Sprintf("  Found %d items for %q", len(m.uiTargets), m.uiTextInput.Value())) + "\n\n"

	visible := m.visibleItemCount()
	total := len(m.uiTargets)
	end := m.uiScrollOffset + visible
	if end > total {
		end = total
	}

	for i := m.uiScrollOffset; i < end; i++ {
		t := m.uiTargets[i]
		cursor := "  "
		if i == m.uiCursor {
			cursor = "> "
		}

		check := "[ ]"
		if m.uiSelected[i] {
			check = "[x]"
		}

		line := fmt.Sprintf("%s%s %-35s %10s",
			cursor, check, truncPath(t.Path, 35), utils.FormatSize(t.Size))

		if i == m.uiCursor {
			s += selectedStyle.Render(line) + "\n"
		} else {
			s += line + "\n"
		}
	}

	if total > visible {
		s += dimStyle.Render(fmt.Sprintf("  [%d-%d of %d]", m.uiScrollOffset+1, end, total)) + "\n"
	}

	var selectedSize int64
	var selectedCount int
	for i, t := range m.uiTargets {
		if m.uiSelected[i] {
			selectedSize += t.Size
			selectedCount++
		}
	}

	s += "\n" + statusBarStyle.Render(fmt.Sprintf(" Selected: %d items (%s) ", selectedCount, utils.FormatSize(selectedSize)))
	s += helpStyle.Render("\n\nj/k navigate | space toggle | a toggle all | d delete | esc back | q quit")
	return s
}

func (m Model) viewUninstallConfirm() string {
	dangerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("196")).
		Background(lipgloss.Color("52")).
		Padding(0, 1)

	s := dangerStyle.Render(" CONFIRM UNINSTALL ") + "\n\n"

	var selectedSize int64
	var selectedCount int
	for i, t := range m.uiTargets {
		if m.uiSelected[i] {
			selectedSize += t.Size
			selectedCount++
			s += fmt.Sprintf("  %s (%s)\n", truncPath(t.Path, 50), utils.FormatSize(t.Size))
		}
	}

	s += fmt.Sprintf("\n  %d items | %s | will be moved to Trash (recoverable)\n", selectedCount, utils.FormatSize(selectedSize))
	s += helpStyle.Render("\n  y confirm | n cancel | q quit")
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
