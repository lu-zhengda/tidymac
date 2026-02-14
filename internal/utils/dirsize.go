package utils

import (
	"io/fs"
	"path/filepath"
	"sync"
	"sync/atomic"
)

// DirSize calculates the total size of all files in a directory tree.
// Uses filepath.WalkDir for performance (avoids redundant stat calls).
func DirSize(path string) (int64, error) {
	var size int64
	err := filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip inaccessible files
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return nil
			}
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// DirSizesParallel computes sizes for multiple paths concurrently.
// Returns a map of path -> size.
func DirSizesParallel(paths []string) map[string]int64 {
	result := make(map[string]int64, len(paths))
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Limit concurrency to avoid overwhelming the filesystem
	sem := make(chan struct{}, 8)

	for _, p := range paths {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			size, _ := DirSize(path)
			mu.Lock()
			result[path] = size
			mu.Unlock()
		}(p)
	}

	wg.Wait()
	return result
}

// DirSizeAtomic is a concurrent-safe version for use in goroutines.
func DirSizeAtomic(path string, total *atomic.Int64) {
	size, _ := DirSize(path)
	total.Add(size)
}
