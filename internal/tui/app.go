package tui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zhengda-lu/tidymac/internal/engine"
	"github.com/zhengda-lu/tidymac/internal/trash"
	"github.com/zhengda-lu/tidymac/internal/utils"
)

type viewState int

const (
	viewDashboard viewState = iota
	viewCategory
	viewConfirm
)

type scanDoneMsg struct {
	results []engine.ScanResult
}

type cleanDoneMsg struct {
	cleaned int
	failed  int
	size    int64
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
		scanning: true,
		spinner:  sp,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.doScan(), m.spinner.Tick)
}

func (m Model) doScan() tea.Cmd {
	return func() tea.Msg {
		results := m.engine.ScanGrouped(context.Background())
		return scanDoneMsg{results: results}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case spinner.TickMsg:
		if m.scanning {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case scanDoneMsg:
		m.scanning = false
		m.results = msg.results
		return m, nil

	case cleanDoneMsg:
		m.scanning = true
		return m, tea.Batch(m.doScan(), m.spinner.Tick)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.currentView == viewDashboard {
				return m, tea.Quit
			}
			m.currentView = viewDashboard
			m.cursor = 0
			return m, nil

		case "up", "k":
			if m.currentView == viewDashboard && m.cursor > 0 {
				m.cursor--
			} else if m.currentView == viewCategory && m.categoryCursor > 0 {
				m.categoryCursor--
			}

		case "down", "j":
			if m.currentView == viewDashboard && m.cursor < len(m.results)-1 {
				m.cursor++
			} else if m.currentView == viewCategory {
				if m.categoryIdx < len(m.results) {
					max := len(m.results[m.categoryIdx].Targets) - 1
					if m.categoryCursor < max {
						m.categoryCursor++
					}
				}
			}

		case "enter":
			if m.currentView == viewDashboard && len(m.results) > 0 {
				m.categoryIdx = m.cursor
				m.categoryCursor = 0
				m.selected = make(map[int]bool)
				if m.categoryIdx < len(m.results) {
					for i := range m.results[m.categoryIdx].Targets {
						m.selected[i] = true
					}
				}
				m.currentView = viewCategory
			} else if m.currentView == viewCategory {
				m.currentView = viewConfirm
			}

		case " ":
			if m.currentView == viewCategory {
				if m.selected[m.categoryCursor] {
					delete(m.selected, m.categoryCursor)
				} else {
					m.selected[m.categoryCursor] = true
				}
			}

		case "d":
			if m.currentView == viewCategory && len(m.selected) > 0 {
				m.currentView = viewConfirm
			}

		case "y":
			if m.currentView == viewConfirm {
				return m, m.doClean()
			}

		case "n":
			if m.currentView == viewConfirm {
				m.currentView = viewCategory
			}

		case "escape":
			if m.currentView != viewDashboard {
				m.currentView = viewDashboard
			}
		}
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

func (m Model) View() string {
	if m.scanning {
		return titleStyle.Render("tidymac") + "\n\n" + m.spinner.View() + " Scanning your Mac...\n"
	}

	switch m.currentView {
	case viewCategory:
		return m.viewCategory()
	case viewConfirm:
		return m.viewConfirm()
	default:
		return m.viewDashboard()
	}
}

func (m Model) viewDashboard() string {
	s := titleStyle.Render("tidymac -- Dashboard") + "\n\n"

	if len(m.results) == 0 {
		s += "No junk found. Your Mac is clean!\n"
		return s + helpStyle.Render("\nq quit")
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
	s += helpStyle.Render("\n\nj/k navigate | enter view details | q quit")
	return s
}

func (m Model) viewCategory() string {
	if m.categoryIdx >= len(m.results) {
		return "No category selected"
	}

	r := m.results[m.categoryIdx]
	s := titleStyle.Render("tidymac -- "+r.Category) + "\n\n"

	if r.Error != nil {
		s += fmt.Sprintf("Error scanning: %v\n", r.Error)
		return s
	}

	for i, t := range r.Targets {
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

	var selectedSize int64
	var selectedCount int
	for i, t := range r.Targets {
		if m.selected[i] {
			selectedSize += t.Size
			selectedCount++
		}
	}

	s += "\n" + statusBarStyle.Render(fmt.Sprintf(" Selected: %d items (%s) ", selectedCount, utils.FormatSize(selectedSize)))
	s += helpStyle.Render("\n\nj/k navigate | space toggle | d delete selected | esc back | q quit")
	return s
}

func (m Model) viewConfirm() string {
	if m.categoryIdx >= len(m.results) {
		return "Nothing to confirm"
	}

	r := m.results[m.categoryIdx]
	s := titleStyle.Render("tidymac -- Confirm Deletion") + "\n\n"

	var selectedSize int64
	var selectedCount int
	for i, t := range r.Targets {
		if m.selected[i] {
			selectedSize += t.Size
			selectedCount++
			s += fmt.Sprintf("  %s (%s)\n", truncPath(t.Path, 50), utils.FormatSize(t.Size))
		}
	}

	s += fmt.Sprintf("\n%d items, %s will be moved to Trash.\n", selectedCount, utils.FormatSize(selectedSize))
	s += helpStyle.Render("\ny confirm | n cancel")
	return s
}

func truncPath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}
	return "..." + path[len(path)-maxLen+3:]
}
