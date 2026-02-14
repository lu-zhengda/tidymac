package scanner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lu-zhengda/macbroom/internal/utils"
)

type AppScanner struct {
	appsDir     string
	libraryBase string
}

func NewAppScanner(appsDir, libraryBase string) *AppScanner {
	return &AppScanner{appsDir: appsDir, libraryBase: libraryBase}
}

func (a *AppScanner) Name() string        { return "App Uninstaller" }
func (a *AppScanner) Description() string { return "Find and remove applications with all related files" }
func (a *AppScanner) Risk() RiskLevel     { return Moderate }

func (a *AppScanner) apps() string {
	if a.appsDir != "" {
		return a.appsDir
	}
	return "/Applications"
}

func (a *AppScanner) library() string {
	if a.libraryBase != "" {
		return a.libraryBase
	}
	return utils.LibraryPath("")
}

func (a *AppScanner) Scan(_ context.Context) ([]Target, error) {
	return nil, nil
}

func (a *AppScanner) ListApps() []string {
	entries, err := os.ReadDir(a.apps())
	if err != nil {
		return nil
	}

	var apps []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".app") {
			name := strings.TrimSuffix(entry.Name(), ".app")
			apps = append(apps, name)
		}
	}
	return apps
}

func (a *AppScanner) FindRelatedFiles(ctx context.Context, appName string) ([]Target, error) {
	var targets []Target
	lib := a.library()

	searchDirs := []string{
		"Application Support",
		"Caches",
		"Preferences",
		"Logs",
		"Saved Application State",
		"Containers",
		"HTTPStorages",
		"WebKit",
		"LaunchAgents",
		"Application Scripts",
		"Group Containers",
		"Cookies",
	}

	appNameLower := strings.ToLower(appName)

	for _, dir := range searchDirs {
		select {
		case <-ctx.Done():
			return targets, ctx.Err()
		default:
		}

		searchPath := filepath.Join(lib, dir)
		if !utils.DirExists(searchPath) {
			continue
		}

		entries, err := os.ReadDir(searchPath)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			entryNameLower := strings.ToLower(entry.Name())
			if !strings.Contains(entryNameLower, appNameLower) {
				continue
			}

			entryPath := filepath.Join(searchPath, entry.Name())
			info, err := entry.Info()
			if err != nil {
				continue
			}

			var size int64
			if info.IsDir() {
				size, _ = utils.DirSize(entryPath)
			} else {
				size = info.Size()
			}

			targets = append(targets, Target{
				Path:        entryPath,
				Size:        size,
				Category:    "App Uninstaller",
				Description: dir + " (" + appName + ")",
				Risk:        Moderate,
				ModTime:     info.ModTime(),
				IsDir:       info.IsDir(),
			})
		}
	}

	appBundle := filepath.Join(a.apps(), appName+".app")
	if info, err := os.Stat(appBundle); err == nil {
		size, _ := utils.DirSize(appBundle)
		targets = append(targets, Target{
			Path:        appBundle,
			Size:        size,
			Category:    "App Uninstaller",
			Description: "Application bundle",
			Risk:        Moderate,
			ModTime:     info.ModTime(),
			IsDir:       true,
		})
	}

	return targets, nil
}

// FindOrphans scans the Preferences directory for .plist files whose
// corresponding .app bundle no longer exists in the applications directory.
// For each orphaned plist it also checks Caches and Application Support for
// matching remnants and includes them in the results.
func (a *AppScanner) FindOrphans(ctx context.Context) ([]Target, error) {
	var targets []Target
	lib := a.library()

	prefsDir := filepath.Join(lib, "Preferences")
	if !utils.DirExists(prefsDir) {
		return targets, nil
	}

	entries, err := os.ReadDir(prefsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read preferences directory: %w", err)
	}

	installedApps := make(map[string]bool)
	for _, name := range a.ListApps() {
		installedApps[strings.ToLower(name)] = true
	}

	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return targets, ctx.Err()
		default:
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".plist") {
			continue
		}

		appName := extractAppName(name)
		if appName == "" {
			continue
		}

		if installedApps[strings.ToLower(appName)] {
			continue
		}

		// This plist has no matching installed app -- it is orphaned.
		plistPath := filepath.Join(prefsDir, name)
		info, err := entry.Info()
		if err != nil {
			continue
		}

		targets = append(targets, Target{
			Path:        plistPath,
			Size:        info.Size(),
			Category:    "Orphaned Preferences",
			Description: fmt.Sprintf("Orphaned plist (%s)", appName),
			Risk:        Safe,
			ModTime:     info.ModTime(),
			IsDir:       false,
		})

		// Also check Caches and Application Support for matching remnants.
		bundleID := strings.TrimSuffix(name, ".plist")
		relatedDirs := []string{"Caches", "Application Support"}
		for _, dir := range relatedDirs {
			dirPath := filepath.Join(lib, dir)
			if !utils.DirExists(dirPath) {
				continue
			}

			relEntries, err := os.ReadDir(dirPath)
			if err != nil {
				continue
			}

			bundleIDLower := strings.ToLower(bundleID)
			appNameLower := strings.ToLower(appName)

			for _, re := range relEntries {
				reLower := strings.ToLower(re.Name())
				if reLower != bundleIDLower && reLower != appNameLower {
					continue
				}

				rePath := filepath.Join(dirPath, re.Name())
				reInfo, err := re.Info()
				if err != nil {
					continue
				}

				var size int64
				if reInfo.IsDir() {
					size, _ = utils.DirSize(rePath)
				} else {
					size = reInfo.Size()
				}

				targets = append(targets, Target{
					Path:        rePath,
					Size:        size,
					Category:    "Orphaned Preferences",
					Description: fmt.Sprintf("Orphaned %s (%s)", dir, appName),
					Risk:        Safe,
					ModTime:     reInfo.ModTime(),
					IsDir:       reInfo.IsDir(),
				})
			}
		}
	}

	return targets, nil
}

// extractAppName attempts to derive an app name from a plist filename.
// For example, "com.example.MyApp.plist" returns "MyApp".
// It strips the ".plist" suffix and takes the last dot-separated component.
func extractAppName(plistFilename string) string {
	base := strings.TrimSuffix(plistFilename, ".plist")
	parts := strings.Split(base, ".")
	if len(parts) < 2 {
		return ""
	}
	return parts[len(parts)-1]
}
