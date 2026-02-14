package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/lu-zhengda/macbroom/internal/trash"
	"github.com/lu-zhengda/macbroom/internal/utils"
)

var (
	cleanPermanent  bool
	cleanYes        bool
	cleanDryRun     bool
	cleanSystem     bool
	cleanBrowser    bool
	cleanXcode      bool
	cleanLarge      bool
	cleanDocker     bool
	cleanNode       bool
	cleanHomebrew   bool
	cleanSimulator  bool
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean selected junk files",
	RunE: func(cmd *cobra.Command, args []string) error {
		e := buildEngine()
		cats := selectedCategories(cleanSystem, cleanBrowser, cleanXcode, cleanLarge, cleanDocker, cleanNode, cleanHomebrew, cleanSimulator)

		fmt.Println("Scanning...")
		targets, err := scanWithCategories(e, cats)
		if err != nil {
			return fmt.Errorf("scan failed: %w", err)
		}

		if len(targets) == 0 {
			fmt.Println("Nothing to clean!")
			return nil
		}

		printScanResults(targets)

		var totalSize int64
		for _, t := range targets {
			totalSize += t.Size
		}

		if cleanDryRun {
			action := "move"
			if cleanPermanent {
				action = "permanently delete"
			}
			fmt.Printf("\n[DRY RUN] Would %s %d items (%s).\n", action, len(targets), utils.FormatSize(totalSize))
			fmt.Println("[DRY RUN] No files were deleted.")
			return nil
		}

		printYoloWarning()

		if !shouldSkipConfirm(cleanYes) {
			if cleanPermanent {
				if !confirmDangerous(fmt.Sprintf("Permanently delete %d items (%s)?", len(targets), utils.FormatSize(totalSize))) {
					fmt.Println("Cancelled.")
					return nil
				}
			} else {
				if !confirmAction(fmt.Sprintf("\nMove %d items (%s) to Trash?", len(targets), utils.FormatSize(totalSize))) {
					fmt.Println("Cancelled.")
					return nil
				}
			}
		}

		var cleaned, failed int
		for _, t := range targets {
			var err error
			if cleanPermanent {
				err = trash.PermanentDelete(t.Path)
			} else {
				err = trash.MoveToTrash(t.Path)
			}
			if err != nil {
				fmt.Printf("  Failed: %s (%v)\n", t.Path, err)
				failed++
			} else {
				cleaned++
			}
		}

		fmt.Printf("\nCleaned %d items (%s freed)", cleaned, utils.FormatSize(totalSize))
		if failed > 0 {
			fmt.Printf(", %d failed", failed)
		}
		fmt.Println()
		return nil
	},
}

func init() {
	cleanCmd.Flags().BoolVar(&cleanPermanent, "permanent", false, "Permanently delete instead of moving to Trash")
	cleanCmd.Flags().BoolVarP(&cleanYes, "yes", "y", false, "Skip confirmation prompt")
	cleanCmd.Flags().BoolVar(&cleanDryRun, "dry-run", false, "Show what would be deleted without actually deleting")
	cleanCmd.Flags().BoolVar(&cleanSystem, "system", false, "Clean system junk only")
	cleanCmd.Flags().BoolVar(&cleanBrowser, "browser", false, "Clean browser caches only")
	cleanCmd.Flags().BoolVar(&cleanXcode, "xcode", false, "Clean Xcode junk only")
	cleanCmd.Flags().BoolVar(&cleanLarge, "large", false, "Clean large/old files only")
	cleanCmd.Flags().BoolVar(&cleanDocker, "docker", false, "Clean Docker junk only")
	cleanCmd.Flags().BoolVar(&cleanNode, "node", false, "Clean Node.js cache only")
	cleanCmd.Flags().BoolVar(&cleanHomebrew, "homebrew", false, "Clean Homebrew cache only")
	cleanCmd.Flags().BoolVar(&cleanSimulator, "simulator", false, "Clean iOS Simulator data only")
}
