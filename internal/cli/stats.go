package cli

import (
	"fmt"
	"sort"

	"github.com/lu-zhengda/macbroom/internal/history"
	"github.com/lu-zhengda/macbroom/internal/utils"
	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show cleanup history and statistics",
	RunE: func(cmd *cobra.Command, args []string) error {
		h := history.New(history.DefaultPath())
		stats := h.Stats()

		if jsonFlag {
			return printJSON(buildStatsJSON(stats))
		}

		fmt.Println("macbroom -- Cleanup Stats")
		fmt.Println()

		fmt.Printf("  Total freed all-time:  %s\n", utils.FormatSize(stats.TotalFreed))
		fmt.Printf("  Total cleanups:        %d\n", stats.TotalCleanups)

		if len(stats.ByCategory) > 0 {
			fmt.Println()
			fmt.Println("  By Category:")

			// Sort categories by bytes freed descending for stable output.
			type catEntry struct {
				name  string
				stats history.CategoryStats
			}
			cats := make([]catEntry, 0, len(stats.ByCategory))
			for name, cs := range stats.ByCategory {
				cats = append(cats, catEntry{name: name, stats: cs})
			}
			sort.Slice(cats, func(i, j int) bool {
				return cats[i].stats.BytesFreed > cats[j].stats.BytesFreed
			})

			for _, c := range cats {
				label := "cleanups"
				if c.stats.Cleanups == 1 {
					label = "cleanup"
				}
				fmt.Printf("    %-22s %10s  (%d %s)\n",
					c.name, utils.FormatSize(c.stats.BytesFreed), c.stats.Cleanups, label)
			}
		}

		if len(stats.Recent) > 0 {
			fmt.Println()
			fmt.Println("  Recent:")

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

		if stats.TotalCleanups == 0 {
			fmt.Println("  No cleanup history yet. Run 'macbroom clean' to get started.")
		}

		fmt.Println()
		return nil
	},
}
