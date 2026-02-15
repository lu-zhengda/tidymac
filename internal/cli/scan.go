package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	scanFilter  CategoryFilter
	scanExclude []string
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan for junk files and reclaimable space",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(scanExclude) > 0 {
			combined := make([]string, 0, len(appConfig.Exclude)+len(scanExclude))
			combined = append(combined, appConfig.Exclude...)
			combined = append(combined, scanExclude...)
			appConfig.Exclude = combined
		}
		e := buildEngine()
		cats := selectedCategories(scanFilter)

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
	f := scanCmd.Flags()
	f.BoolVar(&scanFilter.System, "system", false, "Scan system junk only")
	f.BoolVar(&scanFilter.Browser, "browser", false, "Scan browser caches only")
	f.BoolVar(&scanFilter.Xcode, "xcode", false, "Scan Xcode junk only")
	f.BoolVar(&scanFilter.Large, "large", false, "Scan large/old files only")
	f.BoolVar(&scanFilter.Docker, "docker", false, "Scan Docker junk only")
	f.BoolVar(&scanFilter.Node, "node", false, "Scan Node.js cache only")
	f.BoolVar(&scanFilter.Homebrew, "homebrew", false, "Scan Homebrew cache only")
	f.BoolVar(&scanFilter.Simulator, "simulator", false, "Scan iOS Simulator data only")
	f.BoolVar(&scanFilter.Python, "python", false, "Scan Python cache only")
	f.BoolVar(&scanFilter.Rust, "rust", false, "Scan Rust cache only")
	f.BoolVar(&scanFilter.Go, "go", false, "Scan Go cache only")
	f.BoolVar(&scanFilter.JetBrains, "jetbrains", false, "Scan JetBrains cache only")
	f.BoolVar(&scanFilter.Maven, "maven", false, "Scan Maven cache only")
	f.BoolVar(&scanFilter.Gradle, "gradle", false, "Scan Gradle cache only")
	f.BoolVar(&scanFilter.Ruby, "ruby", false, "Scan Ruby cache only")
	f.BoolVar(&scanFilter.Dev, "dev", false, "Scan all dev-tool caches")
	f.BoolVar(&scanFilter.Caches, "caches", false, "Scan all general caches")
	f.BoolVar(&scanFilter.All, "all", false, "Scan everything")
	f.StringSliceVar(&scanExclude, "exclude", nil, "Exclude paths matching pattern (glob or dir/**)")
}
