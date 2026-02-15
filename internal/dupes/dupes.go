package dupes

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// partialHashSize is the number of bytes read for the partial hash pass.
const partialHashSize = 4096

// Group represents a set of duplicate files sharing the same content.
type Group struct {
	Size  int64
	Hash  string
	Files []string
}

// ProgressFunc is called with each file path as it is visited during scanning.
type ProgressFunc func(path string)

// Find scans dirs for duplicate files whose size is at least minSize bytes.
// It uses a three-pass algorithm: group by size, partial hash, then full hash.
func Find(ctx context.Context, dirs []string, minSize int64) ([]Group, error) {
	return FindWithProgress(ctx, dirs, minSize, nil)
}

// FindWithProgress is like Find but calls onProgress for each file visited.
func FindWithProgress(ctx context.Context, dirs []string, minSize int64, onProgress ProgressFunc) ([]Group, error) {
	// Pass 1: group files by size.
	sizeGroups, err := groupBySize(ctx, dirs, minSize, onProgress)
	if err != nil {
		return nil, fmt.Errorf("failed to group files by size: %w", err)
	}

	// Pass 2: partial hash (first 4KB SHA256) for same-size files.
	candidates, err := refineByHash(ctx, sizeGroups, true)
	if err != nil {
		return nil, fmt.Errorf("failed to compute partial hashes: %w", err)
	}

	// Pass 3: full SHA256 only for partial-hash matches.
	confirmed, err := refineByHash(ctx, candidates, false)
	if err != nil {
		return nil, fmt.Errorf("failed to compute full hashes: %w", err)
	}

	// Build result groups.
	var groups []Group
	for _, c := range confirmed {
		groups = append(groups, Group{
			Size:  c.size,
			Hash:  c.hash,
			Files: c.files,
		})
	}

	// Sort groups by total wasted size descending.
	sort.Slice(groups, func(i, j int) bool {
		wastedI := groups[i].Size * int64(len(groups[i].Files)-1)
		wastedJ := groups[j].Size * int64(len(groups[j].Files)-1)
		return wastedI > wastedJ
	})

	return groups, nil
}

// candidate holds a group of files that match on some criterion.
type candidate struct {
	size  int64
	hash  string
	files []string
}

// groupBySize walks all dirs and groups regular files by size, skipping
// files smaller than minSize. Returns only groups with 2+ files.
func groupBySize(ctx context.Context, dirs []string, minSize int64, onProgress ProgressFunc) ([]candidate, error) {
	sizeMap := make(map[int64][]string)

	for _, dir := range dirs {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // skip unreadable entries
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if d.IsDir() {
				// Skip .git directories entirely.
				if d.Name() == ".git" {
					return fs.SkipDir
				}
				// Skip git repository roots (directories containing .git).
				if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
					return fs.SkipDir
				}
				return nil
			}

			// Skip hidden files (dotfiles).
			if d.Name()[0] == '.' {
				return nil
			}

			// Skip symlinks.
			if d.Type()&os.ModeSymlink != 0 {
				return nil
			}

			info, err := d.Info()
			if err != nil {
				return nil
			}

			// Skip non-regular files.
			if !info.Mode().IsRegular() {
				return nil
			}

			size := info.Size()
			if size < minSize {
				return nil
			}

			if onProgress != nil {
				onProgress(path)
			}

			sizeMap[size] = append(sizeMap[size], path)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	// Keep only sizes with 2+ files.
	var candidates []candidate
	for size, files := range sizeMap {
		if len(files) >= 2 {
			candidates = append(candidates, candidate{size: size, files: files})
		}
	}

	return candidates, nil
}

// refineByHash takes candidate groups and sub-groups them by hash.
// If partial is true, only the first partialHashSize bytes are hashed.
// Returns only sub-groups with 2+ matching files.
func refineByHash(ctx context.Context, candidates []candidate, partial bool) ([]candidate, error) {
	var refined []candidate

	for _, c := range candidates {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		hashMap := make(map[string][]string)
		for _, f := range c.files {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}

			h, err := hashFile(f, partial)
			if err != nil {
				continue // skip unreadable files
			}
			hashMap[h] = append(hashMap[h], f)
		}

		for h, matched := range hashMap {
			if len(matched) >= 2 {
				refined = append(refined, candidate{
					size:  c.size,
					hash:  h,
					files: matched,
				})
			}
		}
	}

	return refined, nil
}

// hashFile computes the SHA256 hash of a file. If partial is true, only
// the first partialHashSize bytes are read.
func hashFile(path string, partial bool) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	h := sha256.New()

	if partial {
		_, err = io.CopyN(h, f, partialHashSize)
		if err != nil && err != io.EOF {
			return "", fmt.Errorf("failed to read file: %w", err)
		}
	} else {
		_, err = io.Copy(h, f)
		if err != nil {
			return "", fmt.Errorf("failed to read file: %w", err)
		}
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
