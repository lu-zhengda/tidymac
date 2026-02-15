package cli

import (
	"fmt"
	"time"

	"github.com/lu-zhengda/macbroom/internal/trends"
	"github.com/lu-zhengda/macbroom/internal/utils"
	"github.com/spf13/cobra"
)

var (
	trendsLast     string
	trendsForecast bool
)

var trendsCmd = &cobra.Command{
	Use:   "trends",
	Short: "Show storage usage trends and forecasting",
	Long:  "Display storage usage trends from collected snapshots.\nUse --forecast to predict when disk will fill up based on growth rate.\nUse 'macbroom trends record' to take a storage snapshot.",
	RunE: func(cmd *cobra.Command, args []string) error {
		store := trends.NewStore(trends.DefaultStorePath())

		snapshots, err := store.GetTrends(trendsLast)
		if err != nil {
			return fmt.Errorf("failed to get trends: %w", err)
		}

		current, err := trends.TakeSnapshot()
		if err != nil {
			return fmt.Errorf("failed to take current snapshot: %w", err)
		}

		report := trends.TrendReport{
			Snapshots: snapshots,
			Current:   *current,
		}

		if trendsForecast && len(snapshots) >= 2 {
			report.Forecast = trends.Forecast(snapshots)
		}

		if jsonFlag {
			return printJSON(trendsJSON{
				Version:   version,
				Timestamp: time.Now().UTC(),
				Report:    report,
			})
		}

		fmt.Println("macbroom -- Storage Trends")
		fmt.Println()

		// Current snapshot.
		fmt.Println("  Current Disk Usage:")
		fmt.Printf("    Total:     %s\n", utils.FormatSize(current.Total))
		fmt.Printf("    Used:      %s (%.1f%%)\n", utils.FormatSize(current.Used), current.UsedPct)
		fmt.Printf("    Available: %s\n", utils.FormatSize(current.Available))

		// Historical snapshots.
		if len(snapshots) > 0 {
			fmt.Println()
			fmt.Printf("  Snapshot History (last %s):\n", trendsLast)
			fmt.Println()
			fmt.Printf("    %-22s %12s %12s %12s %8s\n", "TIMESTAMP", "TOTAL", "USED", "AVAILABLE", "USED%")
			fmt.Printf("    %-22s %12s %12s %12s %8s\n", "---------------------", "----------", "----------", "----------", "------")

			for _, snap := range snapshots {
				ts, err := time.Parse(time.RFC3339, snap.Timestamp)
				if err != nil {
					continue
				}
				fmt.Printf("    %-22s %12s %12s %12s %7.1f%%\n",
					ts.Format("2006-01-02 15:04:05"),
					utils.FormatSize(snap.Total),
					utils.FormatSize(snap.Used),
					utils.FormatSize(snap.Available),
					snap.UsedPct,
				)
			}
		} else {
			fmt.Println()
			fmt.Println("  No snapshot history found.")
			fmt.Println("  Run 'macbroom trends record' to take a snapshot.")
		}

		// Forecast.
		if trendsForecast {
			fmt.Println()
			if len(snapshots) < 2 {
				fmt.Println("  Forecast: insufficient data (need at least 2 snapshots)")
			} else {
				fc := report.Forecast
				if fc == nil {
					fc = trends.Forecast(snapshots)
				}
				fmt.Println("  Disk Full Forecast:")
				fmt.Printf("    Growth Rate:    %s/day\n", formatGrowthRate(fc.GrowthRatePerDay))
				if fc.DaysUntilFull < 0 {
					fmt.Println("    Days Until Full: N/A (disk usage is stable or shrinking)")
					fmt.Println("    Projected Date:  no fill date projected")
				} else {
					fmt.Printf("    Days Until Full: %d\n", fc.DaysUntilFull)
					fmt.Printf("    Projected Date:  %s\n", fc.ProjectedDate)
				}
				fmt.Printf("    Confidence:      %s\n", fc.Confidence)
			}
		}

		fmt.Println()
		return nil
	},
}

var trendsRecordCmd = &cobra.Command{
	Use:   "record",
	Short: "Take a storage snapshot now",
	Long:  "Capture the current disk usage and store it for trend analysis.\nDesigned to be called periodically by a scheduler or agent.",
	RunE: func(cmd *cobra.Command, args []string) error {
		store := trends.NewStore(trends.DefaultStorePath())

		snap, err := store.Record()
		if err != nil {
			return fmt.Errorf("failed to record snapshot: %w", err)
		}

		if jsonFlag {
			return printJSON(trendsRecordJSON{
				Version:   version,
				Timestamp: time.Now().UTC(),
				Snapshot:  *snap,
			})
		}

		fmt.Println("Storage snapshot recorded:")
		fmt.Printf("  Total:     %s\n", utils.FormatSize(snap.Total))
		fmt.Printf("  Used:      %s (%.1f%%)\n", utils.FormatSize(snap.Used), snap.UsedPct)
		fmt.Printf("  Available: %s\n", utils.FormatSize(snap.Available))

		return nil
	},
}

// formatGrowthRate returns a human-readable growth rate string.
func formatGrowthRate(bytesPerDay int64) string {
	if bytesPerDay == 0 {
		return "0 B"
	}
	if bytesPerDay < 0 {
		return "-" + utils.FormatSize(-bytesPerDay)
	}
	return utils.FormatSize(bytesPerDay)
}

func init() {
	trendsCmd.Flags().StringVar(&trendsLast, "last", "30d", "Filter snapshot history (e.g., 7d, 30d, 90d)")
	trendsCmd.Flags().BoolVar(&trendsForecast, "forecast", false, "Predict when disk will fill up")
	trendsCmd.AddCommand(trendsRecordCmd)
}
