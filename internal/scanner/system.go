package scanner

import (
	"context"
	"os"
	"path/filepath"

	"github.com/zhengda-lu/macbroom/internal/utils"
)

type SystemScanner struct {
	libraryBase string
}

func NewSystemScanner(libraryBase string) *SystemScanner {
	return &SystemScanner{libraryBase: libraryBase}
}

func (s *SystemScanner) Name() string        { return "System Junk" }
func (s *SystemScanner) Description() string { return "System caches, logs, and temporary files" }
func (s *SystemScanner) Risk() RiskLevel     { return Safe }

func (s *SystemScanner) base() string {
	if s.libraryBase != "" {
		return s.libraryBase
	}
	return utils.LibraryPath("")
}

func (s *SystemScanner) Scan(ctx context.Context) ([]Target, error) {
	var targets []Target

	dirs := []struct {
		subpath     string
		description string
	}{
		{"Caches", "Application caches"},
		{"Logs", "System and application logs"},
	}

	for _, d := range dirs {
		select {
		case <-ctx.Done():
			return targets, ctx.Err()
		default:
		}

		dir := filepath.Join(s.base(), d.subpath)
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
				Category:    "System Junk",
				Description: d.description,
				Risk:        Safe,
				ModTime:     info.ModTime(),
				IsDir:       info.IsDir(),
			})
		}
	}

	return targets, nil
}
