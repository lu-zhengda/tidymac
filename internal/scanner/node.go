package scanner

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lu-zhengda/macbroom/internal/utils"
)

// NodeScanner detects npm cache and stale node_modules directories.
type NodeScanner struct {
	home        string
	searchPaths []string
	maxAge      time.Duration
}

// NewNodeScanner returns a new NodeScanner.
//   - home: user home directory (npm cache lives at home/.npm/_cacache)
//   - searchPaths: directories to walk looking for stale node_modules
//   - maxAge: threshold after which node_modules is considered stale
func NewNodeScanner(home string, searchPaths []string, maxAge time.Duration) *NodeScanner {
	return &NodeScanner{
		home:        home,
		searchPaths: searchPaths,
		maxAge:      maxAge,
	}
}

func (s *NodeScanner) Name() string        { return "Node.js" }
func (s *NodeScanner) Description() string { return "npm cache and stale node_modules" }
func (s *NodeScanner) Risk() RiskLevel     { return Safe }

func (s *NodeScanner) Scan(ctx context.Context) ([]Target, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	var targets []Target

	// --- npm cache ---
	npmCache := filepath.Join(s.home, ".npm", "_cacache")
	if utils.DirExists(npmCache) {
		size, err := utils.DirSize(npmCache)
		if err != nil {
			return nil, fmt.Errorf("failed to compute npm cache size: %w", err)
		}
		targets = append(targets, Target{
			Path:        npmCache,
			Size:        size,
			Category:    "Node.js",
			Description: "npm cache (_cacache)",
			Risk:        Safe,
			IsDir:       true,
		})
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// --- stale node_modules ---
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
			if d.Name() != "node_modules" {
				return nil
			}

			// Skip nested node_modules (those inside another node_modules)
			rel, relErr := filepath.Rel(searchPath, path)
			if relErr == nil {
				parts := strings.Split(rel, string(filepath.Separator))
				nested := false
				for i, p := range parts {
					if p == "node_modules" && i < len(parts)-1 {
						nested = true
						break
					}
				}
				if nested {
					return fs.SkipDir
				}
			}

			info, infoErr := os.Stat(path)
			if infoErr != nil {
				return nil
			}

			age := now.Sub(info.ModTime())
			if age >= s.maxAge {
				size, _ := utils.DirSize(path)
				targets = append(targets, Target{
					Path:        path,
					Size:        size,
					Category:    "Node.js",
					Description: fmt.Sprintf("stale node_modules (unused for %d days)", int(age.Hours()/24)),
					Risk:        Moderate,
					ModTime:     info.ModTime(),
					IsDir:       true,
				})
			}

			// Don't descend into node_modules regardless of staleness
			return fs.SkipDir
		})

		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			// Non-context errors during walk are non-fatal; skip this search path
		}
	}

	return targets, nil
}
