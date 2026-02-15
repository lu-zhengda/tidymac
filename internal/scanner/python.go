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

// PythonScanner detects pip cache, conda packages, and stale virtualenvs.
type PythonScanner struct {
	home        string
	searchPaths []string
	maxAge      time.Duration
}

// NewPythonScanner returns a new PythonScanner.
//   - home: user home directory (pip cache lives at home/Library/Caches/pip)
//   - searchPaths: directories to walk looking for stale virtualenvs
//   - maxAge: threshold after which a virtualenv is considered stale
func NewPythonScanner(home string, searchPaths []string, maxAge time.Duration) *PythonScanner {
	return &PythonScanner{
		home:        home,
		searchPaths: searchPaths,
		maxAge:      maxAge,
	}
}

func (s *PythonScanner) Name() string { return "Python" }
func (s *PythonScanner) Description() string {
	return "pip cache, conda packages, and stale virtualenvs"
}
func (s *PythonScanner) Risk() RiskLevel { return Safe }

func (s *PythonScanner) Scan(ctx context.Context) ([]Target, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	var targets []Target

	// --- pip cache ---
	pipCache := filepath.Join(s.home, "Library", "Caches", "pip")
	if utils.DirExists(pipCache) {
		size, _ := utils.DirSize(pipCache)
		targets = append(targets, Target{
			Path:        pipCache,
			Size:        size,
			Category:    "Python",
			Description: "pip download cache",
			Risk:        Safe,
			IsDir:       true,
		})
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// --- conda package caches ---
	for _, condaRoot := range []string{"miniconda3", "anaconda3", "miniforge3"} {
		pkgsDir := filepath.Join(s.home, condaRoot, "pkgs")
		if utils.DirExists(pkgsDir) {
			size, _ := utils.DirSize(pkgsDir)
			targets = append(targets, Target{
				Path:        pkgsDir,
				Size:        size,
				Category:    "Python",
				Description: fmt.Sprintf("%s package cache", condaRoot),
				Risk:        Safe,
				IsDir:       true,
			})
		}
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// --- stale virtualenvs ---
	now := time.Now()
	for _, searchPath := range s.searchPaths {
		if !utils.DirExists(searchPath) {
			continue
		}

		err := filepath.WalkDir(searchPath, func(path string, d fs.DirEntry, err error) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if err != nil {
				return nil // skip inaccessible entries
			}
			if !d.IsDir() {
				return nil
			}

			name := d.Name()
			if name != ".venv" && name != "venv" {
				return nil
			}

			// Confirm it's a real virtualenv by checking for pyvenv.cfg.
			if _, statErr := os.Stat(filepath.Join(path, "pyvenv.cfg")); statErr != nil {
				return fs.SkipDir
			}

			info, infoErr := os.Stat(path)
			if infoErr != nil {
				return fs.SkipDir
			}

			age := now.Sub(info.ModTime())
			if s.maxAge > 0 && age < s.maxAge {
				return fs.SkipDir
			}

			size, _ := utils.DirSize(path)
			targets = append(targets, Target{
				Path:        path,
				Size:        size,
				Category:    "Python",
				Description: fmt.Sprintf("stale virtualenv (unused for %d days)", int(age.Hours()/24)),
				Risk:        Moderate,
				ModTime:     info.ModTime(),
				IsDir:       true,
			})

			// Don't descend into virtualenv directories regardless of staleness.
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
