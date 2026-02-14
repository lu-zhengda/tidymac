package scanner

import (
	"context"
	"os"
	"path/filepath"

	"github.com/zhengda-lu/tidymac/internal/utils"
)

type XcodeScanner struct {
	libraryBase string
}

func NewXcodeScanner(libraryBase string) *XcodeScanner {
	return &XcodeScanner{libraryBase: libraryBase}
}

func (x *XcodeScanner) Name() string        { return "Xcode Junk" }
func (x *XcodeScanner) Description() string { return "Xcode DerivedData, archives, and device support" }
func (x *XcodeScanner) Risk() RiskLevel     { return Safe }

func (x *XcodeScanner) base() string {
	if x.libraryBase != "" {
		return x.libraryBase
	}
	return utils.LibraryPath("")
}

func (x *XcodeScanner) Scan(ctx context.Context) ([]Target, error) {
	var targets []Target

	dirs := []struct {
		relPath     string
		description string
	}{
		{"Developer/Xcode/DerivedData", "Xcode DerivedData (build intermediates)"},
		{"Developer/Xcode/Archives", "Xcode Archives"},
		{"Developer/Xcode/iOS DeviceSupport", "iOS Device Support"},
		{"Developer/Xcode/watchOS DeviceSupport", "watchOS Device Support"},
		{"Developer/CoreSimulator/Devices", "Simulator data"},
		{"Caches/com.apple.dt.Xcode", "Xcode caches"},
	}

	for _, d := range dirs {
		select {
		case <-ctx.Done():
			return targets, ctx.Err()
		default:
		}

		dir := filepath.Join(x.base(), d.relPath)
		if !utils.DirExists(dir) {
			continue
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			entryPath := filepath.Join(dir, entry.Name())
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

			if size == 0 {
				continue
			}

			targets = append(targets, Target{
				Path:        entryPath,
				Size:        size,
				Category:    "Xcode Junk",
				Description: d.description,
				Risk:        Safe,
				ModTime:     info.ModTime(),
				IsDir:       info.IsDir(),
			})
		}
	}

	return targets, nil
}
