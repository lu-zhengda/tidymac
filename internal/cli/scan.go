package cli

import (
	"fmt"
	"time"

	"github.com/lu-zhengda/macbroom/internal/config"
	"github.com/lu-zhengda/macbroom/internal/scancache"
	"github.com/lu-zhengda/macbroom/internal/scanner"
	"github.com/spf13/cobra"
)

var (
	scanFilter    CategoryFilter
	scanExclude   []string
	scanThreshold string
)

// filterByThreshold returns only targets above the given size threshold.
func filterByThreshold(targets []scanner.Target, threshold int64) []scanner.Target {
	filtered := make([]scanner.Target, 0, len(targets))
	for _, t := range targets {
		if t.Size >= threshold {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

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

		if !jsonFlag {
			fmt.Println("Scanning...")
		}
		targets, err := scanWithCategories(e, cats)
		if err != nil {
			return fmt.Errorf("scan failed: %w", err)
		}

		// Apply --threshold filter if set.
		if scanThreshold != "" {
			thresholdBytes, err := config.ParseSize(scanThreshold)
			if err != nil {
				return fmt.Errorf("invalid threshold %q: %w", scanThreshold, err)
			}
			targets = filterByThreshold(targets, thresholdBytes)
		}

		prev, prevErr := scancache.Load(scancache.DefaultPath())

		grouped := make(map[string]*scancache.CategorySnapshot)
		for _, t := range targets {
			cs := grouped[t.Category]
			if cs == nil {
				cs = &scancache.CategorySnapshot{Name: t.Category}
				grouped[t.Category] = cs
			}
			cs.Size += t.Size
			cs.Items++
		}
		var cats2 []scancache.CategorySnapshot
		for _, cs := range grouped {
			cats2 = append(cats2, *cs)
		}
		var totalSize int64
		for _, t := range targets {
			totalSize += t.Size
		}
		curr := scancache.Snapshot{Timestamp: time.Now().UTC(), Categories: cats2, TotalSize: totalSize}

		var diff *scancache.DiffResult
		if prevErr == nil {
			d := scancache.Diff(prev, curr)
			diff = &d
		}

		_ = scancache.Save(scancache.DefaultPath(), curr)

		if jsonFlag {
			return printJSON(buildScanJSON(targets, diff))
		}

		printScanResults(targets, diff)
		return nil
	},
}

func init() {
	f := scanCmd.Flags()
	f.StringVar(&scanThreshold, "threshold", "", "Only show items above this size (e.g., 100M, 1G)")
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
