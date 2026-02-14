package scanner

import (
	"context"
	"os"
	"path/filepath"
	"sync"

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
	dirs := []struct {
		subpath     string
		description string
	}{
		{"Caches", "Application caches"},
		{"Logs", "System and application logs"},
	}

	// First pass: collect all entries and paths that need sizing
	type entryInfo struct {
		path        string
		description string
		info        os.FileInfo
	}
	var allEntries []entryInfo
	var dirPaths []string

	for _, d := range dirs {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
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

			allEntries = append(allEntries, entryInfo{
				path:        entryPath,
				description: d.description,
				info:        info,
			})
			if info.IsDir() {
				dirPaths = append(dirPaths, entryPath)
			}
		}
	}

	// Compute all directory sizes in parallel
	sizes := utils.DirSizesParallel(dirPaths)

	// Build targets using precomputed sizes
	var mu sync.Mutex
	var wg sync.WaitGroup
	targets := make([]Target, 0, len(allEntries))

	for _, e := range allEntries {
		wg.Add(1)
		go func(e entryInfo) {
			defer wg.Done()
			var size int64
			if e.info.IsDir() {
				size = sizes[e.path]
			} else {
				size = e.info.Size()
			}

			if size == 0 {
				return
			}

			t := Target{
				Path:        e.path,
				Size:        size,
				Category:    "System Junk",
				Description: e.description,
				Risk:        Safe,
				ModTime:     e.info.ModTime(),
				IsDir:       e.info.IsDir(),
			}

			mu.Lock()
			targets = append(targets, t)
			mu.Unlock()
		}(e)
	}

	wg.Wait()
	return targets, nil
}
