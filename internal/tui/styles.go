package tui

import "github.com/charmbracelet/lipgloss"

// ---------------------------------------------------------------------------
// Existing styles -- now referencing theme color variables.
// ---------------------------------------------------------------------------

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorSecondary)

	dimStyle = lipgloss.NewStyle().
			Foreground(colorDim)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorDim).
			MarginTop(1)

	statusBarStyle = lipgloss.NewStyle().
			Background(colorSubtle).
			Foreground(colorText).
			Padding(0, 1)
)

// ---------------------------------------------------------------------------
// Shared styles -- reusable across views (replaces inline definitions).
// ---------------------------------------------------------------------------

var (
	successStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorSuccess)

	failStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorDanger)

	warnStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWarning)

	dangerBannerStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorDanger).
				Background(colorDangerBg).
				Padding(0, 1)

	headerBarStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWhite).
			Background(colorPrimary).
			Padding(0, 1)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorSubtle).
			Padding(1, 2)

	footerStyle = lipgloss.NewStyle().
			Foreground(colorDim).
			MarginTop(1)
)
