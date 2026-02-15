package scanner

import (
	"context"
	"path/filepath"

	"github.com/lu-zhengda/macbroom/internal/utils"
)

// GradleScanner detects Gradle caches and wrapper distributions.
type GradleScanner struct {
	home string
}

// NewGradleScanner returns a new GradleScanner.
func NewGradleScanner(home string) *GradleScanner {
	return &GradleScanner{home: home}
}

func (s *GradleScanner) Name() string        { return "Gradle" }
func (s *GradleScanner) Description() string { return "Gradle caches and wrapper distributions" }
func (s *GradleScanner) Risk() RiskLevel     { return Safe }

func (s *GradleScanner) Scan(ctx context.Context) ([]Target, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	var targets []Target

	caches := filepath.Join(s.home, ".gradle", "caches")
	if utils.DirExists(caches) {
		size, _ := utils.DirSize(caches)
		targets = append(targets, Target{
			Path:        caches,
			Size:        size,
			Category:    "Gradle",
			Description: "Gradle build caches",
			Risk:        Safe,
			IsDir:       true,
		})
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	dists := filepath.Join(s.home, ".gradle", "wrapper", "dists")
	if utils.DirExists(dists) {
		size, _ := utils.DirSize(dists)
		targets = append(targets, Target{
			Path:        dists,
			Size:        size,
			Category:    "Gradle",
			Description: "Gradle wrapper distributions",
			Risk:        Safe,
			IsDir:       true,
		})
	}

	return targets, nil
}
