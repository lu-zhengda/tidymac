package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderHeader draws a header bar with breadcrumb navigation.
func renderHeader(parts ...string) string {
	breadcrumb := "macbroom"
	for _, p := range parts {
		breadcrumb += " > " + p
	}
	return headerBarStyle.Render(breadcrumb) + "\n"
}

// renderFooter draws a footer with keybind hints.
func renderFooter(hints string) string {
	return footerStyle.Render(hints)
}

// renderProgressBar draws a progress bar of the given width.
// ratio should be between 0.0 and 1.0.
func renderProgressBar(ratio float64, width int) string {
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	filled := int(ratio * float64(width))
	if filled == 0 && ratio > 0 {
		filled = 1
	}
	empty := width - filled
	color := barColor(ratio)
	fillStyle := lipgloss.NewStyle().Foreground(color)
	return "[" + fillStyle.Render(strings.Repeat("\u2588", filled)) + strings.Repeat("\u2591", empty) + "]"
}
