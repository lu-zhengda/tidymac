package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/lu-zhengda/macbroom/internal/dupes"
	"github.com/lu-zhengda/macbroom/internal/trash"
	"github.com/lu-zhengda/macbroom/internal/utils"
	"github.com/spf13/cobra"
)

var (
	dupesMinSize int64
	dupesYes     bool
	dupesDryRun  bool
)

var dupesCmd = &cobra.Command{
	Use:   "dupes [dirs...]",
	Short: "Find duplicate files",
	Long:  "Scan directories for duplicate files using a three-pass algorithm:\n1. Group files by size\n2. Partial hash (first 4KB) for same-size files\n3. Full SHA256 only when partial hashes match\n\nDefaults to ~/Downloads, ~/Desktop, ~/Documents if no dirs given.",
	RunE: func(cmd *cobra.Command, args []string) error {
		dirs := args
		if len(dirs) == 0 {
			home := utils.HomeDir()
			if home == "" {
				return fmt.Errorf("failed to determine home directory")
			}
			dirs = []string{
				filepath.Join(home, "Downloads"),
				filepath.Join(home, "Desktop"),
				filepath.Join(home, "Documents"),
			}
		}

		fmt.Printf("Scanning for duplicates in: %s\n", strings.Join(dirs, ", "))

		var fileCount int
		groups, err := dupes.FindWithProgress(context.Background(), dirs, dupesMinSize, func(path string) {
			fileCount++
			if fileCount%500 == 0 {
				fmt.Printf("\r  Scanned %d files...", fileCount)
			}
		})
		if err != nil {
			return fmt.Errorf("failed to scan for duplicates: %w", err)
		}

		if fileCount >= 500 {
			fmt.Println() // newline after progress
		}

		if len(groups) == 0 {
			fmt.Println("No duplicates found!")
			return nil
		}

		var totalWasted int64
		var totalFiles int
		for _, g := range groups {
			wasted := g.Size * int64(len(g.Files)-1)
			totalWasted += wasted
			totalFiles += len(g.Files)
		}

		fmt.Printf("\nFound %d duplicate groups (%d files, %s wasted)\n",
			len(groups), totalFiles, utils.FormatSize(totalWasted))
		fmt.Println(strings.Repeat("-", 60))

		for i, g := range groups {
			wasted := g.Size * int64(len(g.Files)-1)
			fmt.Printf("\nGroup %d: %s each, %d files (%s wasted)\n",
				i+1, utils.FormatSize(g.Size), len(g.Files), utils.FormatSize(wasted))
			fmt.Printf("  Hash: %s\n", g.Hash[:16]+"...")
			for j, f := range g.Files {
				label := "  "
				if j == 0 {
					label = "  [keep] "
				} else {
					label = "  [copy] "
				}
				fmt.Printf("%s%s\n", label, f)
			}
		}

		if dupesDryRun {
			fmt.Printf("\n[DRY RUN] Would delete %d duplicate copies (%s).\n",
				totalFiles-len(groups), utils.FormatSize(totalWasted))
			fmt.Println("[DRY RUN] No files were deleted.")
			return nil
		}

		printYoloWarning()

		deleteCount := totalFiles - len(groups)
		if !shouldSkipConfirm(dupesYes) {
			if !confirmAction(fmt.Sprintf("\nMove %d duplicate copies (%s) to Trash? (keeps 1 per group)",
				deleteCount, utils.FormatSize(totalWasted))) {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		var deleted, failed int
		var freedSize int64
		for _, g := range groups {
			// Keep the first file, delete the rest.
			for _, f := range g.Files[1:] {
				if err := trash.MoveToTrash(f); err != nil {
					fmt.Printf("  Failed: %s (%v)\n", f, err)
					failed++
				} else {
					deleted++
					freedSize += g.Size
				}
			}
		}

		fmt.Printf("\nDeleted %d duplicates (%s freed)", deleted, utils.FormatSize(freedSize))
		if failed > 0 {
			fmt.Printf(", %d failed", failed)
		}
		fmt.Println()

		return nil
	},
}

func init() {
	dupesCmd.Flags().Int64Var(&dupesMinSize, "min-size", 0, "Minimum file size in bytes (0 = no minimum)")
	dupesCmd.Flags().BoolVarP(&dupesYes, "yes", "y", false, "Skip confirmation prompt")
	dupesCmd.Flags().BoolVar(&dupesDryRun, "dry-run", false, "Show duplicates without deleting")
}
