package cli

import (
	"fmt"

	"github.com/spf13/cobra"
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
		fmt.Println("Cleaning... (not yet implemented)")
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
