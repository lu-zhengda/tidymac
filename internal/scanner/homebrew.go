package scanner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// HomebrewScanner detects Homebrew download cache files that can be
// safely removed to reclaim disk space.
type HomebrewScanner struct {
	// lookPath is used to check if brew is installed.
	// Defaults to exec.LookPath; override in tests.
	lookPath func(file string) (string, error)

	// runCmd executes a command and returns its stdout.
	// Defaults to exec.CommandContext(...).Output(); override in tests.
	runCmd func(ctx context.Context, name string, args ...string) ([]byte, error)
}

// NewHomebrewScanner returns a new HomebrewScanner with default command execution.
func NewHomebrewScanner() *HomebrewScanner {
	return &HomebrewScanner{
		lookPath: exec.LookPath,
		runCmd: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, name, args...).Output()
		},
	}
}

func (s *HomebrewScanner) Name() string        { return "Homebrew" }
func (s *HomebrewScanner) Description() string { return "Homebrew download cache" }
func (s *HomebrewScanner) Risk() RiskLevel     { return Safe }

func (s *HomebrewScanner) Scan(ctx context.Context) ([]Target, error) {
	if _, err := s.lookPath("brew"); err != nil {
		return nil, nil
	}

	out, err := s.runCmd(ctx, "brew", "--cache")
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("failed to get brew cache dir: %w", err)
	}

	cacheDir := strings.TrimSpace(string(out))
	if cacheDir == "" {
		return nil, nil
	}

	var targets []Target

	walkErr := filepath.WalkDir(cacheDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}

		if d.IsDir() {
			return nil
		}

		if !isCacheFile(path) {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil // skip files we cannot stat
		}

		targets = append(targets, Target{
			Path:        path,
			Size:        info.Size(),
			Category:    "Homebrew",
			Description: "Cached download: " + d.Name(),
			Risk:        Safe,
			ModTime:     info.ModTime(),
		})

		return nil
	})
	if walkErr != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("failed to walk brew cache: %w", walkErr)
	}

	return targets, nil
}

// isCacheFile returns true if the filename matches a Homebrew cache extension.
func isCacheFile(path string) bool {
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".bottle.tar.gz"):
		return true
	case strings.HasSuffix(lower, ".tar.gz"):
		return true
	case strings.HasSuffix(lower, ".dmg"):
		return true
	default:
		return false
	}
}
