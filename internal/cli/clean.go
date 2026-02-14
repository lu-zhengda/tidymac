package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zhengda-lu/macbroom/internal/trash"
	"github.com/zhengda-lu/macbroom/internal/utils"
)

var (
	cleanPermanent bool
	cleanYes       bool
	cleanSystem    bool
	cleanBrowser   bool
	cleanXcode     bool
	cleanLarge     bool
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean selected junk files",
	RunE: func(cmd *cobra.Command, args []string) error {
		e := buildEngine()
		cats := selectedCategories(cleanSystem, cleanBrowser, cleanXcode, cleanLarge)

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

		if !cleanYes {
			action := "Move to Trash"
			if cleanPermanent {
				action = "PERMANENTLY DELETE"
			}
			if !confirmAction(fmt.Sprintf("\n%s %s of files?", action, utils.FormatSize(totalSize))) {
				fmt.Println("Cancelled.")
				return nil
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
	cleanCmd.Flags().BoolVar(&cleanSystem, "system", false, "Clean system junk only")
	cleanCmd.Flags().BoolVar(&cleanBrowser, "browser", false, "Clean browser caches only")
	cleanCmd.Flags().BoolVar(&cleanXcode, "xcode", false, "Clean Xcode junk only")
	cleanCmd.Flags().BoolVar(&cleanLarge, "large", false, "Clean large/old files only")
}
