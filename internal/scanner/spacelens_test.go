package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSpaceLens_Analyze(t *testing.T) {
	tmpDir := t.TempDir()

	sub1 := filepath.Join(tmpDir, "big-folder")
	if err := os.MkdirAll(sub1, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub1, "data.bin"), make([]byte, 4096), 0o644); err != nil {
		t.Fatal(err)
	}

	sub2 := filepath.Join(tmpDir, "small-folder")
	if err := os.MkdirAll(sub2, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub2, "tiny.txt"), make([]byte, 64), 0o644); err != nil {
		t.Fatal(err)
	}

	sl := NewSpaceLens(tmpDir, 1)
	nodes, err := sl.Analyze(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nodes) < 2 {
		t.Fatalf("expected at least 2 nodes, got %d", len(nodes))
	}

	if nodes[0].Size < nodes[1].Size {
		t.Error("expected nodes sorted by size descending")
	}
}
