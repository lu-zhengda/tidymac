package scanner

import (
	"context"
	"path/filepath"

	"github.com/lu-zhengda/macbroom/internal/utils"
)

// RubyScanner detects Ruby gem cache and Bundler cache.
type RubyScanner struct {
	home string
}

// NewRubyScanner returns a new RubyScanner.
func NewRubyScanner(home string) *RubyScanner {
	return &RubyScanner{home: home}
}

func (s *RubyScanner) Name() string        { return "Ruby" }
func (s *RubyScanner) Description() string { return "Ruby gem and Bundler cache" }
func (s *RubyScanner) Risk() RiskLevel     { return Safe }

func (s *RubyScanner) Scan(ctx context.Context) ([]Target, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	var targets []Target

	gemDir := filepath.Join(s.home, ".gem")
	if utils.DirExists(gemDir) {
		size, _ := utils.DirSize(gemDir)
		targets = append(targets, Target{
			Path:        gemDir,
			Size:        size,
			Category:    "Ruby",
			Description: "Ruby gem cache",
			Risk:        Safe,
			IsDir:       true,
		})
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	bundleCache := filepath.Join(s.home, ".bundle", "cache")
	if utils.DirExists(bundleCache) {
		size, _ := utils.DirSize(bundleCache)
		targets = append(targets, Target{
			Path:        bundleCache,
			Size:        size,
			Category:    "Ruby",
			Description: "Bundler cache",
			Risk:        Safe,
			IsDir:       true,
		})
	}

	return targets, nil
}
