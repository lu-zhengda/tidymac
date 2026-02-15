package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	scanSystem    bool
	scanBrowser   bool
	scanXcode     bool
	scanLarge     bool
	scanDocker    bool
	scanNode      bool
	scanHomebrew  bool
	scanSimulator bool
	scanPython    bool
	scanRust      bool
	scanGo        bool
	scanJetBrains bool
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan for junk files and reclaimable space",
	RunE: func(cmd *cobra.Command, args []string) error {
		e := buildEngine()
		cats := selectedCategories(scanSystem, scanBrowser, scanXcode, scanLarge, scanDocker, scanNode, scanHomebrew, scanSimulator, scanPython, scanRust, scanGo, scanJetBrains)

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
	scanCmd.Flags().BoolVar(&scanDocker, "docker", false, "Scan Docker junk only")
	scanCmd.Flags().BoolVar(&scanNode, "node", false, "Scan Node.js cache only")
	scanCmd.Flags().BoolVar(&scanHomebrew, "homebrew", false, "Scan Homebrew cache only")
	scanCmd.Flags().BoolVar(&scanSimulator, "simulator", false, "Scan iOS Simulator data only")
	scanCmd.Flags().BoolVar(&scanPython, "python", false, "Scan Python cache only")
	scanCmd.Flags().BoolVar(&scanRust, "rust", false, "Scan Rust cache only")
	scanCmd.Flags().BoolVar(&scanGo, "go", false, "Scan Go cache only")
	scanCmd.Flags().BoolVar(&scanJetBrains, "jetbrains", false, "Scan JetBrains cache only")
}
