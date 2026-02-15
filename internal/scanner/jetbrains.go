package scanner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lu-zhengda/macbroom/internal/utils"
)

// JetBrainsScanner detects JetBrains IDE caches and logs.
type JetBrainsScanner struct {
	home string
}

// NewJetBrainsScanner returns a new JetBrainsScanner.
//   - home: user home directory (caches at home/Library/Caches/JetBrains,
//     logs at home/Library/Logs/JetBrains)
func NewJetBrainsScanner(home string) *JetBrainsScanner {
	return &JetBrainsScanner{home: home}
}

func (s *JetBrainsScanner) Name() string        { return "JetBrains" }
func (s *JetBrainsScanner) Description() string { return "JetBrains IDE caches and logs" }
func (s *JetBrainsScanner) Risk() RiskLevel     { return Safe }

func (s *JetBrainsScanner) Scan(ctx context.Context) ([]Target, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	var targets []Target

	dirs := []struct {
		base string
		desc string
	}{
		{filepath.Join(s.home, "Library", "Caches", "JetBrains"), "cache"},
		{filepath.Join(s.home, "Library", "Logs", "JetBrains"), "logs"},
	}

	for _, d := range dirs {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if !utils.DirExists(d.base) {
			continue
		}
		entries, err := os.ReadDir(d.base)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			ideDir := filepath.Join(d.base, entry.Name())
			size, _ := utils.DirSize(ideDir)
			targets = append(targets, Target{
				Path:        ideDir,
				Size:        size,
				Category:    "JetBrains",
				Description: fmt.Sprintf("%s %s", entry.Name(), d.desc),
				Risk:        Safe,
				IsDir:       true,
			})
		}
	}

	return targets, nil
}
