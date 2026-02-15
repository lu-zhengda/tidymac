package scanner

import (
	"context"
	"path/filepath"

	"github.com/lu-zhengda/macbroom/internal/utils"
)

// MavenScanner detects the Maven local repository (~/.m2/repository).
type MavenScanner struct {
	home string
}

// NewMavenScanner returns a new MavenScanner.
func NewMavenScanner(home string) *MavenScanner {
	return &MavenScanner{home: home}
}

func (s *MavenScanner) Name() string        { return "Maven" }
func (s *MavenScanner) Description() string { return "Maven local repository" }
func (s *MavenScanner) Risk() RiskLevel     { return Safe }

func (s *MavenScanner) Scan(ctx context.Context) ([]Target, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	var targets []Target

	repo := filepath.Join(s.home, ".m2", "repository")
	if utils.DirExists(repo) {
		size, _ := utils.DirSize(repo)
		targets = append(targets, Target{
			Path:        repo,
			Size:        size,
			Category:    "Maven",
			Description: "Maven local repository",
			Risk:        Safe,
			IsDir:       true,
		})
	}

	return targets, nil
}
