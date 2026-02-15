package scanner

import (
	"context"
	"os"
	"path/filepath"
	"sort"

	"github.com/lu-zhengda/macbroom/internal/utils"
)

type SpaceLensNode struct {
	Path     string
	Name     string
	Size     int64
	IsDir    bool
	Children []SpaceLensNode
	Depth    int
}

// ProgressFunc is called with the name of each directory being analyzed.
type ProgressFunc func(name string)

type SpaceLens struct {
	root       string
	maxDepth   int
	onProgress ProgressFunc
}

func NewSpaceLens(root string, maxDepth int) *SpaceLens {
	return &SpaceLens{root: root, maxDepth: maxDepth}
}

// SetProgress sets a callback for reporting scan progress.
func (s *SpaceLens) SetProgress(fn ProgressFunc) {
	s.onProgress = fn
}

func (s *SpaceLens) Analyze(ctx context.Context) ([]SpaceLensNode, error) {
	return s.analyzeDir(ctx, s.root, 0)
}

func (s *SpaceLens) analyzeDir(ctx context.Context, dir string, depth int) ([]SpaceLensNode, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var nodes []SpaceLensNode
	for _, entry := range entries {
		entryPath := filepath.Join(dir, entry.Name())
		if s.onProgress != nil {
			s.onProgress(entry.Name())
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}

		node := SpaceLensNode{
			Path:  entryPath,
			Name:  entry.Name(),
			IsDir: info.IsDir(),
			Depth: depth,
		}

		if info.IsDir() {
			size, _ := utils.DirSize(entryPath)
			node.Size = size
			if depth < s.maxDepth {
				children, _ := s.analyzeDir(ctx, entryPath, depth+1)
				node.Children = children
			}
		} else {
			node.Size = info.Size()
		}

		nodes = append(nodes, node)
	}

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Size > nodes[j].Size
	})

	return nodes, nil
}
