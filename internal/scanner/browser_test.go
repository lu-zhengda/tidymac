package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestBrowserScanner_Scan(t *testing.T) {
	tmpDir := t.TempDir()

	chromeCache := filepath.Join(tmpDir, "Google", "Chrome", "Default", "Cache")
	if err := os.MkdirAll(chromeCache, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(chromeCache, "data_0"), make([]byte, 2048), 0o644); err != nil {
		t.Fatal(err)
	}

	safariCache := filepath.Join(tmpDir, "Safari")
	if err := os.MkdirAll(safariCache, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(safariCache, "Cache.db"), make([]byte, 4096), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewBrowserScanner(tmpDir, tmpDir)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) == 0 {
		t.Fatal("expected at least one browser target")
	}
	for _, tgt := range targets {
		if tgt.Category != "Browser Cache" {
			t.Errorf("expected category Browser Cache, got %s", tgt.Category)
		}
	}
}
