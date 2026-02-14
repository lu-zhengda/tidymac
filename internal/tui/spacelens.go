package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lu-zhengda/tidymac/internal/scanner"
	"github.com/lu-zhengda/tidymac/internal/utils"
)

// SpaceLensModel is the standalone Space Lens TUI (used by `spacelens -i`).
type SpaceLensModel struct {
	path    string
	nodes   []scanner.SpaceLensNode
	cursor  int
	loading bool
	width   int
	height  int
}

func NewSpaceLensModel(path string) SpaceLensModel {
	return SpaceLensModel{path: path, loading: true}
}

func (m SpaceLensModel) Init() tea.Cmd {
	return m.doAnalyze()
}

func (m SpaceLensModel) doAnalyze() tea.Cmd {
	path := m.path
	return func() tea.Msg {
		sl := scanner.NewSpaceLens(path, 1)
		nodes, _ := sl.Analyze(context.Background())
		return spaceLensDoneMsg{nodes: nodes, path: path}
	}
}

func (m SpaceLensModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case spaceLensDoneMsg:
		m.loading = false
		m.nodes = msg.nodes
		m.path = msg.path

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.nodes)-1 {
				m.cursor++
			}
		case "enter":
			if m.cursor < len(m.nodes) && m.nodes[m.cursor].IsDir {
				m.path = m.nodes[m.cursor].Path
				m.loading = true
				m.cursor = 0
				return m, m.doAnalyze()
			}
		case "backspace", "h":
			if idx := lastSlash(m.path); idx > 0 {
				m.path = m.path[:idx]
				m.loading = true
				m.cursor = 0
				return m, m.doAnalyze()
			}
		}
	}
	return m, nil
}

func (m SpaceLensModel) View() string {
	s := titleStyle.Render("tidymac -- Space Lens") + "\n"
	s += dimStyle.Render(m.path) + "\n\n"

	if m.loading {
		return s + "Analyzing...\n"
	}

	if len(m.nodes) == 0 {
		return s + "Empty directory.\n"
	}

	maxSize := m.nodes[0].Size
	visible := m.nodes
	if len(visible) > 30 {
		visible = visible[:30]
	}

	for i, node := range visible {
		cursor := "  "
		if i == m.cursor {
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

		if i == m.cursor {
			s += selectedStyle.Render(line) + "\n"
		} else {
			s += line + "\n"
		}
	}

	if len(m.nodes) > 30 {
		s += dimStyle.Render(fmt.Sprintf("\n  ... and %d more items", len(m.nodes)-30)) + "\n"
	}

	s += helpStyle.Render("\nj/k navigate | enter drill into folder | h/backspace go up | q quit")
	return s
}

func renderBar(size, maxSize int64, width int) string {
	if maxSize == 0 {
		return ""
	}
	ratio := float64(size) / float64(maxSize)
	filled := int(ratio * float64(width))
	if filled == 0 && size > 0 {
		filled = 1
	}
	return "|" + strings.Repeat("#", filled) + strings.Repeat(".", width-filled) + "|"
}
