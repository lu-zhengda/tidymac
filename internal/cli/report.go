package cli

import (
	"fmt"
	"sort"
	"time"

	"github.com/lu-zhengda/macbroom/internal/history"
	"github.com/lu-zhengda/macbroom/internal/utils"
	"github.com/spf13/cobra"
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate a cleanup history report",
	Long:  "Reads macbroom's cleanup history and generates a summary report.\nWith --json, outputs structured report data.",
	RunE: func(cmd *cobra.Command, args []string) error {
		h := history.New(history.DefaultPath())
		stats := h.Stats()

		if jsonFlag {
			byCategory := make(map[string]int64, len(stats.ByCategory))
			for name, cs := range stats.ByCategory {
				byCategory[name] = cs.BytesFreed
			}
			return printJSON(reportJSON{
				Version:       version,
				Timestamp:     time.Now().UTC(),
				TotalFreed:    stats.TotalFreed,
				TotalCleanups: stats.TotalCleanups,
				ByCategory:    byCategory,
				Recent:        stats.Recent,
			})
		}

		fmt.Println("macbroom -- Cleanup Report")
		fmt.Println()

		if stats.TotalCleanups == 0 {
			fmt.Println("  No cleanup history found.")
			fmt.Println("  Run 'macbroom clean' to get started, then check back here.")
			fmt.Println()
			return nil
		}

		fmt.Printf("  Total space freed:  %s\n", utils.FormatSize(stats.TotalFreed))
		fmt.Printf("  Total cleanups:     %d\n", stats.TotalCleanups)

		if len(stats.ByCategory) > 0 {
			fmt.Println()
			fmt.Println("  Breakdown by category:")

			type catEntry struct {
				name  string
				freed int64
				count int
			}
			cats := make([]catEntry, 0, len(stats.ByCategory))
			for name, cs := range stats.ByCategory {
				cats = append(cats, catEntry{name: name, freed: cs.BytesFreed, count: cs.Cleanups})
			}
			sort.Slice(cats, func(i, j int) bool {
				return cats[i].freed > cats[j].freed
			})

			for _, c := range cats {
				label := "cleanups"
				if c.count == 1 {
					label = "cleanup"
				}
				fmt.Printf("    %-22s %10s  (%d %s)\n", c.name, utils.FormatSize(c.freed), c.count, label)
			}
		}

		if len(stats.Recent) > 0 {
			fmt.Println()
			fmt.Println("  Recent activity:")

			for _, e := range stats.Recent {
				label := "items"
				if e.Items == 1 {
					label = "item"
				}
				fmt.Printf("    %s  %-22s %3d %-5s  %10s  (%s)\n",
					e.Timestamp.Format("2006-01-02 15:04"),
					e.Category,
					e.Items,
					label,
					utils.FormatSize(e.BytesFreed),
					e.Method)
			}
		}

		fmt.Println()
		return nil
	},
}
