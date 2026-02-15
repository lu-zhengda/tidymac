package cli

import (
	"fmt"
	"time"

	"github.com/lu-zhengda/macbroom/internal/history"
	"github.com/lu-zhengda/macbroom/internal/schedule"
	"github.com/lu-zhengda/macbroom/internal/trash"
	"github.com/lu-zhengda/macbroom/internal/utils"
	"github.com/spf13/cobra"
)

var (
	cleanPermanent bool
	cleanYes       bool
	cleanDryRun    bool
	cleanQuiet     bool
	cleanFilter    CategoryFilter
	cleanExclude   []string
)

// cleanPrint prints to stdout only when --quiet is not set.
func cleanPrint(format string, a ...any) {
	if !cleanQuiet {
		fmt.Printf(format, a...)
	}
}

// cleanPrintln prints a line to stdout only when --quiet is not set.
func cleanPrintln(a ...any) {
	if !cleanQuiet {
		fmt.Println(a...)
	}
}

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean selected junk files",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(cleanExclude) > 0 {
			combined := make([]string, 0, len(appConfig.Exclude)+len(cleanExclude))
			combined = append(combined, appConfig.Exclude...)
			combined = append(combined, cleanExclude...)
			appConfig.Exclude = combined
		}
		e := buildEngine()
		cats := selectedCategories(cleanFilter)

		cleanPrintln("Scanning...")
		targets, err := scanWithCategories(e, cats)
		if err != nil {
			return fmt.Errorf("scan failed: %w", err)
		}

		if len(targets) == 0 {
			cleanPrintln("Nothing to clean!")
			return nil
		}

		if !cleanQuiet {
			printScanResults(targets)
		}

		var totalSize int64
		for _, t := range targets {
			totalSize += t.Size
		}

		if cleanDryRun {
			action := "move"
			if cleanPermanent {
				action = "permanently delete"
			}
			cleanPrint("\n[DRY RUN] Would %s %d items (%s).\n", action, len(targets), utils.FormatSize(totalSize))
			cleanPrintln("[DRY RUN] No files were deleted.")
			return nil
		}

		if !cleanQuiet {
			printYoloWarning()
		}

		if !shouldSkipConfirm(cleanYes) {
			if cleanPermanent {
				if !confirmDangerous(fmt.Sprintf("Permanently delete %d items (%s)?", len(targets), utils.FormatSize(totalSize))) {
					cleanPrintln("Cancelled.")
					return nil
				}
			} else {
				if !confirmAction(fmt.Sprintf("\nMove %d items (%s) to Trash?", len(targets), utils.FormatSize(totalSize))) {
					cleanPrintln("Cancelled.")
					return nil
				}
			}
		}

		type catResult struct {
			items int
			bytes int64
		}
		byCategory := make(map[string]*catResult)

		var cleaned, failed int
		for _, t := range targets {
			var err error
			if cleanPermanent {
				err = trash.PermanentDelete(t.Path)
			} else {
				err = trash.MoveToTrash(t.Path)
			}
			if err != nil {
				cleanPrint("  Failed: %s (%v)\n", t.Path, err)
				failed++
			} else {
				cleaned++
				cr := byCategory[t.Category]
				if cr == nil {
					cr = &catResult{}
					byCategory[t.Category] = cr
				}
				cr.items++
				cr.bytes += t.Size
			}
		}

		cleanPrint("\nCleaned %d items (%s freed)", cleaned, utils.FormatSize(totalSize))
		if failed > 0 {
			cleanPrint(", %d failed", failed)
		}
		cleanPrintln()

		// Record cleanup history per category.
		method := "trash"
		if cleanPermanent {
			method = "permanent"
		}
		h := history.New(history.DefaultPath())
		now := time.Now()
		for cat, cr := range byCategory {
			_ = h.Record(history.Entry{
				Timestamp:  now,
				Category:   cat,
				Items:      cr.items,
				BytesFreed: cr.bytes,
				Method:     method,
			})
		}

		// Send macOS notification when running in quiet mode with notify enabled.
		if cleanQuiet && appConfig != nil && appConfig.Schedule.Notify && cleaned > 0 {
			msg := fmt.Sprintf("Cleaned %d items, freed %s", cleaned, utils.FormatSize(totalSize))
			_ = schedule.Notify("macbroom", msg)
		}

		return nil
	},
}

func init() {
	cleanCmd.Flags().BoolVar(&cleanPermanent, "permanent", false, "Permanently delete instead of moving to Trash")
	cleanCmd.Flags().BoolVarP(&cleanYes, "yes", "y", false, "Skip confirmation prompt")
	cleanCmd.Flags().BoolVar(&cleanDryRun, "dry-run", false, "Show what would be deleted without actually deleting")
	cleanCmd.Flags().BoolVarP(&cleanQuiet, "quiet", "q", false, "Suppress all output (for scheduled runs)")
	f := cleanCmd.Flags()
	f.BoolVar(&cleanFilter.System, "system", false, "Clean system junk only")
	f.BoolVar(&cleanFilter.Browser, "browser", false, "Clean browser caches only")
	f.BoolVar(&cleanFilter.Xcode, "xcode", false, "Clean Xcode junk only")
	f.BoolVar(&cleanFilter.Large, "large", false, "Clean large/old files only")
	f.BoolVar(&cleanFilter.Docker, "docker", false, "Clean Docker junk only")
	f.BoolVar(&cleanFilter.Node, "node", false, "Clean Node.js cache only")
	f.BoolVar(&cleanFilter.Homebrew, "homebrew", false, "Clean Homebrew cache only")
	f.BoolVar(&cleanFilter.Simulator, "simulator", false, "Clean iOS Simulator data only")
	f.BoolVar(&cleanFilter.Python, "python", false, "Clean Python cache only")
	f.BoolVar(&cleanFilter.Rust, "rust", false, "Clean Rust cache only")
	f.BoolVar(&cleanFilter.Go, "go", false, "Clean Go cache only")
	f.BoolVar(&cleanFilter.JetBrains, "jetbrains", false, "Clean JetBrains cache only")
	f.BoolVar(&cleanFilter.Maven, "maven", false, "Clean Maven cache only")
	f.BoolVar(&cleanFilter.Gradle, "gradle", false, "Clean Gradle cache only")
	f.BoolVar(&cleanFilter.Ruby, "ruby", false, "Clean Ruby cache only")
	f.BoolVar(&cleanFilter.Dev, "dev", false, "Clean all dev-tool caches")
	f.BoolVar(&cleanFilter.Caches, "caches", false, "Clean all general caches")
	f.BoolVar(&cleanFilter.All, "all", false, "Clean everything")
	f.StringSliceVar(&cleanExclude, "exclude", nil, "Exclude paths matching pattern (glob or dir/**)")
}
