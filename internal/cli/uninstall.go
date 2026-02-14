package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zhengda-lu/macbroom/internal/scanner"
	"github.com/zhengda-lu/macbroom/internal/trash"
	"github.com/zhengda-lu/macbroom/internal/utils"
)

var uninstallPermanent bool

var uninstallCmd = &cobra.Command{
	Use:   "uninstall [app-name]",
	Short: "Completely uninstall an application",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		appName := args[0]
		s := scanner.NewAppScanner("", "")

		fmt.Printf("Searching for files related to %q...\n", appName)
		targets, err := s.FindRelatedFiles(context.Background(), appName)
		if err != nil {
			return fmt.Errorf("failed to find app files: %w", err)
		}

		if len(targets) == 0 {
			fmt.Printf("No files found for %q.\n", appName)
			return nil
		}

		printScanResults(targets)

		var totalSize int64
		for _, t := range targets {
			totalSize += t.Size
		}

		if !confirmAction(fmt.Sprintf("\nRemove all %d items (%s)?", len(targets), utils.FormatSize(totalSize))) {
			fmt.Println("Cancelled.")
			return nil
		}

		for _, t := range targets {
			var err error
			if uninstallPermanent {
				err = trash.PermanentDelete(t.Path)
			} else {
				err = trash.MoveToTrash(t.Path)
			}
			if err != nil {
				fmt.Printf("  Failed: %s (%v)\n", t.Path, err)
			}
		}

		fmt.Printf("Uninstalled %q successfully.\n", appName)
		return nil
	},
}

func init() {
	uninstallCmd.Flags().BoolVar(&uninstallPermanent, "permanent", false, "Permanently delete instead of moving to Trash")
}
