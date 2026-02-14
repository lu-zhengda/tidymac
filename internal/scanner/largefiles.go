package scanner

import (
	"context"
	"os"
	"path/filepath"
	"time"
)

type LargeFileScanner struct {
	searchDirs []string
	minSize    int64
	minAge     time.Duration
}

func NewLargeFileScanner(searchDirs []string, minSize int64, minAge time.Duration) *LargeFileScanner {
	return &LargeFileScanner{searchDirs: searchDirs, minSize: minSize, minAge: minAge}
}

func (l *LargeFileScanner) Name() string        { return "Large & Old Files" }
func (l *LargeFileScanner) Description() string { return "Files exceeding size or age thresholds" }
func (l *LargeFileScanner) Risk() RiskLevel     { return Risky }

func (l *LargeFileScanner) Scan(ctx context.Context) ([]Target, error) {
	var targets []Target
	now := time.Now()

	for _, dir := range l.searchDirs {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if err != nil || info.IsDir() {
				return nil
			}

			matchesSize := l.minSize > 0 && info.Size() >= l.minSize
			matchesAge := l.minAge > 0 && now.Sub(info.ModTime()) >= l.minAge

			if !matchesSize && !matchesAge {
				return nil
			}

			risk := Moderate
			if matchesAge && !matchesSize {
				risk = Risky
			}

			desc := "Large file"
			if matchesAge && matchesSize {
				desc = "Large and old file"
			} else if matchesAge {
				desc = "Old file (not modified recently)"
			}

			targets = append(targets, Target{
				Path:        path,
				Size:        info.Size(),
				Category:    "Large & Old Files",
				Description: desc,
				Risk:        risk,
				ModTime:     info.ModTime(),
				IsDir:       false,
			})

			return nil
		})
		if err != nil {
			continue
		}
	}

	return targets, nil
}
