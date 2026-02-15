package scanner

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/lu-zhengda/macbroom/internal/utils"
)

// RustScanner detects cargo registry cache and stale target directories.
type RustScanner struct {
	home        string
	searchPaths []string
	maxAge      time.Duration
}

// NewRustScanner returns a new RustScanner.
//   - home: user home directory (cargo cache lives at home/.cargo)
//   - searchPaths: directories to walk looking for stale target/ dirs
//   - maxAge: threshold after which a target/ dir is considered stale
func NewRustScanner(home string, searchPaths []string, maxAge time.Duration) *RustScanner {
	return &RustScanner{home: home, searchPaths: searchPaths, maxAge: maxAge}
}

func (s *RustScanner) Name() string { return "Rust" }
func (s *RustScanner) Description() string {
	return "Cargo registry cache and stale target directories"
}
func (s *RustScanner) Risk() RiskLevel { return Safe }

func (s *RustScanner) Scan(ctx context.Context) ([]Target, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	var targets []Target

	// --- cargo registry caches ---
	for _, sub := range []string{"registry/cache", "registry/src"} {
		dir := filepath.Join(s.home, ".cargo", sub)
		if utils.DirExists(dir) {
			size, _ := utils.DirSize(dir)
			targets = append(targets, Target{
				Path:        dir,
				Size:        size,
				Category:    "Rust",
				Description: fmt.Sprintf("Cargo %s", sub),
				Risk:        Safe,
				IsDir:       true,
			})
		}
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// --- stale target/ directories ---
	now := time.Now()
	for _, searchPath := range s.searchPaths {
		if !utils.DirExists(searchPath) {
			continue
		}

		err := filepath.WalkDir(searchPath, func(path string, d fs.DirEntry, err error) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if err != nil || !d.IsDir() {
				return nil
			}
			if d.Name() != "target" {
				return nil
			}

			// Confirm it's a Rust project by checking for Cargo.toml in the parent.
			parent := filepath.Dir(path)
			if _, err := os.Stat(filepath.Join(parent, "Cargo.toml")); err != nil {
				return nil
			}

			info, err := os.Stat(path)
			if err != nil {
				return fs.SkipDir
			}

			if s.maxAge > 0 {
				age := now.Sub(info.ModTime())
				if age < s.maxAge {
					return fs.SkipDir
				}
			}

			size, _ := utils.DirSize(path)
			targets = append(targets, Target{
				Path:        path,
				Size:        size,
				Category:    "Rust",
				Description: fmt.Sprintf("Rust build artifacts (%s)", filepath.Base(parent)),
				Risk:        Moderate,
				ModTime:     info.ModTime(),
				IsDir:       true,
			})
			return fs.SkipDir
		})

		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			// Non-context errors during walk are non-fatal; skip this search path.
		}
	}

	return targets, nil
}
