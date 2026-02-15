package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lu-zhengda/macbroom/internal/scanner"
	"github.com/lu-zhengda/macbroom/internal/utils"
)

// RiskBreakdown holds aggregated byte sizes grouped by risk level.
type RiskBreakdown struct {
	Safe     int64 `json:"safe"`
	Moderate int64 `json:"moderate"`
	Risky    int64 `json:"risky"`
	Total    int64 `json:"total"`
}

// riskSummary aggregates targets by their risk level.
func riskSummary(targets []scanner.Target) RiskBreakdown {
	var rb RiskBreakdown
	for _, t := range targets {
		switch t.Risk {
		case scanner.Safe:
			rb.Safe += t.Size
		case scanner.Moderate:
			rb.Moderate += t.Size
		case scanner.Risky:
			rb.Risky += t.Size
		}
		rb.Total += t.Size
	}
	return rb
}

// riskSummaryLine renders a colored risk summary string.
// Returns empty string if Total == 0.
func riskSummaryLine(rb RiskBreakdown) string {
	if rb.Total == 0 {
		return ""
	}

	safeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	moderateStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	riskyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	pct := func(n int64) int {
		return int(float64(n) / float64(rb.Total) * 100)
	}

	var parts []string

	parts = append(parts, safeStyle.Render(
		fmt.Sprintf("Safe: %s (%d%%)", utils.FormatSize(rb.Safe), pct(rb.Safe)),
	))

	if rb.Moderate > 0 {
		parts = append(parts, moderateStyle.Render(
			fmt.Sprintf("Moderate: %s (%d%%)", utils.FormatSize(rb.Moderate), pct(rb.Moderate)),
		))
	}

	if rb.Risky > 0 {
		parts = append(parts, riskyStyle.Render(
			fmt.Sprintf("Risky: %s (%d%%)", utils.FormatSize(rb.Risky), pct(rb.Risky)),
		))
	}

	return strings.Join(parts, "  ")
}
