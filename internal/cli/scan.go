package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	scanSystem  bool
	scanBrowser bool
	scanXcode   bool
	scanLarge   bool
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan for junk files and reclaimable space",
	RunE: func(cmd *cobra.Command, args []string) error {
		e := buildEngine()
		cats := selectedCategories(scanSystem, scanBrowser, scanXcode, scanLarge)

		fmt.Println("Scanning...")
		targets, err := scanWithCategories(e, cats)
		if err != nil {
			return fmt.Errorf("scan failed: %w", err)
		}

		printScanResults(targets)
		return nil
	},
}

func init() {
	scanCmd.Flags().BoolVar(&scanSystem, "system", false, "Scan system junk only")
	scanCmd.Flags().BoolVar(&scanBrowser, "browser", false, "Scan browser caches only")
	scanCmd.Flags().BoolVar(&scanXcode, "xcode", false, "Scan Xcode junk only")
	scanCmd.Flags().BoolVar(&scanLarge, "large", false, "Scan large/old files only")
}
