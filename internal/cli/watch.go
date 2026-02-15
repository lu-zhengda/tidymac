package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/lu-zhengda/macbroom/internal/config"
	"github.com/lu-zhengda/macbroom/internal/utils"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
)

var (
	watchFree     string
	watchInterval int
)

// diskFree returns the available disk space in bytes for the given path.
func diskFree(path string) (int64, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, fmt.Errorf("failed to stat filesystem: %w", err)
	}
	return int64(stat.Bavail) * int64(stat.Bsize), nil
}

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Monitor disk free space",
	Long:  "Poll disk free space at a configurable interval.\nWhen free space drops below --free threshold, print a warning and exit with code 1.\nWith --json, output structured alert data.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if watchFree == "" {
			return fmt.Errorf("--free is required (e.g., --free 10G)")
		}

		threshold, err := config.ParseSize(watchFree)
		if err != nil {
			return fmt.Errorf("invalid --free value %q: %w", watchFree, err)
		}

		interval := time.Duration(watchInterval) * time.Second

		if !jsonFlag {
			fmt.Printf("Watching disk free space (threshold: %s, interval: %ds)\n",
				utils.FormatSize(threshold), watchInterval)
		}

		for {
			free, err := diskFree("/")
			if err != nil {
				return fmt.Errorf("failed to check disk space: %w", err)
			}

			if free < threshold {
				msg := fmt.Sprintf("WARNING: free space %s is below threshold %s",
					utils.FormatSize(free), utils.FormatSize(threshold))

				if jsonFlag {
					if err := printJSON(watchAlertJSON{
						Version:   version,
						Timestamp: time.Now().UTC(),
						FreeBytes: free,
						Threshold: threshold,
						Alert:     true,
						Message:   msg,
					}); err != nil {
						return err
					}
				} else {
					fmt.Println(msg)
				}

				os.Exit(1)
			}

			if !jsonFlag {
				fmt.Printf("  %s  free: %s (OK)\n",
					time.Now().Format("15:04:05"),
					utils.FormatSize(free))
			}

			time.Sleep(interval)
		}
	},
}

func init() {
	watchCmd.Flags().StringVar(&watchFree, "free", "", "Minimum free space threshold (e.g., 10G, 500M)")
	watchCmd.Flags().IntVar(&watchInterval, "interval", 30, "Poll interval in seconds")
}
