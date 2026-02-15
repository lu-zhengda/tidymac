package scanner

import (
	"context"
	"path/filepath"

	"github.com/lu-zhengda/macbroom/internal/utils"
)

// GoScanner detects Go module cache and build cache.
type GoScanner struct {
	home string
}

// NewGoScanner returns a new GoScanner.
//   - home: user home directory (module cache at home/go/pkg/mod/cache,
//     build cache at home/Library/Caches/go-build)
func NewGoScanner(home string) *GoScanner {
	return &GoScanner{home: home}
}

func (s *GoScanner) Name() string        { return "Go" }
func (s *GoScanner) Description() string { return "Go module cache and build cache" }
func (s *GoScanner) Risk() RiskLevel     { return Safe }

func (s *GoScanner) Scan(ctx context.Context) ([]Target, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	var targets []Target

	modCache := filepath.Join(s.home, "go", "pkg", "mod", "cache")
	if utils.DirExists(modCache) {
		size, _ := utils.DirSize(modCache)
		targets = append(targets, Target{
			Path:        modCache,
			Size:        size,
			Category:    "Go",
			Description: "Go module cache",
			Risk:        Safe,
			IsDir:       true,
		})
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	buildCache := filepath.Join(s.home, "Library", "Caches", "go-build")
	if utils.DirExists(buildCache) {
		size, _ := utils.DirSize(buildCache)
		targets = append(targets, Target{
			Path:        buildCache,
			Size:        size,
			Category:    "Go",
			Description: "Go build cache",
			Risk:        Safe,
			IsDir:       true,
		})
	}

	return targets, nil
}
