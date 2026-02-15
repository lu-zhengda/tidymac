package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lu-zhengda/macbroom/internal/scancache"
	"github.com/lu-zhengda/macbroom/internal/scanner"
	"github.com/lu-zhengda/macbroom/internal/tui"
	"github.com/lu-zhengda/macbroom/internal/utils"
)

// printJSON encodes v as indented JSON to stdout.
func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// CLI output styles
// ---------------------------------------------------------------------------

var (
	boldStyle    = lipgloss.NewStyle().Bold(true)
	dimStyle     = lipgloss.NewStyle().Faint(true)
	riskModerate = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	riskRisky    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	totalStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("82"))
)

// categoryGroup holds a category name and its targets for sorted output.
type categoryGroup struct {
	Name      string
	Targets   []scanner.Target
	TotalSize int64
}

func printScanResults(targets []scanner.Target, diff *scancache.DiffResult) {
	if len(targets) == 0 {
		fmt.Println("No junk files found.")
		return
	}

	// Group targets by category.
	grouped := make(map[string][]scanner.Target)
	for _, t := range targets {
		grouped[t.Category] = append(grouped[t.Category], t)
	}

	// Build sorted category groups.
	groups := make([]categoryGroup, 0, len(grouped))
	for name, items := range grouped {
		var catSize int64
		for _, item := range items {
			catSize += item.Size
		}
		// Sort items within category by size descending.
		sort.Slice(items, func(i, j int) bool {
			return items[i].Size > items[j].Size
		})
		groups = append(groups, categoryGroup{Name: name, Targets: items, TotalSize: catSize})
	}

	// Sort categories by total size descending.
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].TotalSize > groups[j].TotalSize
	})

	var totalSize int64
	for _, g := range groups {
		totalSize += g.TotalSize

		// Category header with themed color + bold.
		catColor := tui.CategoryColor(g.Name)
		header := lipgloss.NewStyle().Bold(true).Foreground(catColor).
			Render(fmt.Sprintf("%s (%s, %d items)", g.Name, utils.FormatSize(g.TotalSize), len(g.Targets)))
		diffStr := diffIndicator(g.Name, diff)
		if diffStr != "" {
			fmt.Printf("\n%s %s\n", header, diffStr)
		} else {
			fmt.Printf("\n%s\n", header)
		}
		fmt.Println(strings.Repeat("-", 60))

		for _, item := range g.Targets {
			risk := ""
			if item.Risk == scanner.Risky {
				risk = " " + riskRisky.Render("[Risky]")
			} else if item.Risk == scanner.Moderate {
				risk = " " + riskModerate.Render("[Moderate]")
			}
			padded := fmt.Sprintf("%10s", utils.FormatSize(item.Size))
			sizeStr := boldStyle.Render(padded)
			fmt.Printf("  %-40s %s%s\n", truncatePath(item.Path, 40), sizeStr, risk)
		}
	}

	fmt.Printf("\n%s\n", totalStyle.Render(fmt.Sprintf("Total reclaimable: %s", utils.FormatSize(totalSize))))

	if line := riskSummaryLine(riskSummary(targets)); line != "" {
		fmt.Printf("%s\n", line)
	}

	if diff != nil {
		var deltaStr string
		if diff.TotalDelta >= 0 {
			deltaStr = "+" + utils.FormatSize(diff.TotalDelta)
		} else {
			deltaStr = "-" + utils.FormatSize(-diff.TotalDelta)
		}
		fmt.Printf("  %s\n", dimStyle.Render(fmt.Sprintf("%s since %s", deltaStr, diff.PreviousTimestamp.Format("Jan 2"))))
	}
}

// diffIndicator returns a styled string showing how a category changed since the last scan.
func diffIndicator(name string, diff *scancache.DiffResult) string {
	if diff == nil {
		return ""
	}

	cd, ok := diff.Categories[name]
	if !ok {
		return ""
	}

	if cd.IsNew {
		return dimStyle.Render("(new)")
	}

	if cd.Delta == 0 {
		return dimStyle.Render("unchanged")
	}

	if cd.Delta > 0 {
		return riskModerate.Render("+" + utils.FormatSize(cd.Delta))
	}

	greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	return greenStyle.Render("-" + utils.FormatSize(-cd.Delta))
}

func truncatePath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}
	return "..." + path[len(path)-maxLen+3:]
}

func confirmAction(prompt string) bool {
	fmt.Printf("%s [y/N]: ", prompt)
	var response string
	fmt.Scanln(&response)
	return strings.ToLower(strings.TrimSpace(response)) == "y"
}

// confirmDangerous requires typing "yes" (not just "y") for permanent deletions.
func confirmDangerous(prompt string) bool {
	fmt.Printf("\n  *** DANGER ***\n")
	fmt.Printf("  %s\n", prompt)
	fmt.Printf("  This action is IRREVERSIBLE. Files will be permanently deleted.\n")
	fmt.Printf("\n  Type 'yes' to confirm: ")
	var response string
	fmt.Scanln(&response)
	return strings.ToLower(strings.TrimSpace(response)) == "yes"
}
