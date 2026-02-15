package tui

import "github.com/charmbracelet/lipgloss"

// ---------------------------------------------------------------------------
// Color palette -- single source of truth for all TUI colors.
// Values are ANSI-256 color codes passed to lipgloss.Color().
// ---------------------------------------------------------------------------

var (
	colorPrimary   = lipgloss.Color("170")
	colorSecondary = lipgloss.Color("212")
	colorSuccess   = lipgloss.Color("82")
	colorWarning   = lipgloss.Color("214")
	colorDanger    = lipgloss.Color("196")
	colorDim       = lipgloss.Color("241")
	colorSubtle    = lipgloss.Color("236")
	colorText      = lipgloss.Color("252")
	colorWhite     = lipgloss.Color("255")
	colorDangerBg  = lipgloss.Color("52")
)

// ---------------------------------------------------------------------------
// Category colors -- used in the dashboard and treemap views.
// ---------------------------------------------------------------------------

var categoryColors = map[string]lipgloss.Color{
	"System Junk":       lipgloss.Color("75"),
	"Browser Cache":     lipgloss.Color("214"),
	"Xcode Junk":        lipgloss.Color("141"),
	"Large & Old Files": lipgloss.Color("223"),
	"Docker":            lipgloss.Color("39"),
	"Node.js":           lipgloss.Color("119"),
	"Homebrew":          lipgloss.Color("208"),
	"iOS Simulators":    lipgloss.Color("183"),
	"Python":            lipgloss.Color("220"),
	"Rust":              lipgloss.Color("173"),
	"Go":                lipgloss.Color("74"),
	"JetBrains":         lipgloss.Color("171"),
	"Maven":             lipgloss.Color("167"),
	"Gradle":            lipgloss.Color("108"),
	"Ruby":              lipgloss.Color("161"),
}

// CategoryColor returns the theme color for a scan category.
// Unknown categories fall back to colorPrimary.
func CategoryColor(name string) lipgloss.Color {
	if c, ok := categoryColors[name]; ok {
		return c
	}
	return colorPrimary
}

// ---------------------------------------------------------------------------
// Bar colors -- used for usage-ratio bars (e.g. Space Lens).
// ---------------------------------------------------------------------------

var (
	barColorHigh   = lipgloss.Color("196")
	barColorMedium = lipgloss.Color("214")
	barColorLow    = lipgloss.Color("82")
)

// barColor returns a color based on a 0.0-1.0 ratio.
//   - >= 0.75 -> high (red)
//   - >= 0.40 -> medium (orange/yellow)
//   - < 0.40  -> low (green)
func barColor(ratio float64) lipgloss.Color {
	switch {
	case ratio >= 0.75:
		return barColorHigh
	case ratio >= 0.40:
		return barColorMedium
	default:
		return barColorLow
	}
}

// ---------------------------------------------------------------------------
// Treemap colors -- used for adjacent block coloring in treemap views.
// ---------------------------------------------------------------------------

var treemapColors = []lipgloss.Color{
	lipgloss.Color("75"),
	lipgloss.Color("214"),
	lipgloss.Color("141"),
	lipgloss.Color("223"),
	lipgloss.Color("39"),
	lipgloss.Color("119"),
	lipgloss.Color("208"),
	lipgloss.Color("183"),
	lipgloss.Color("220"),
	lipgloss.Color("171"),
	lipgloss.Color("82"),
	lipgloss.Color("212"),
}
