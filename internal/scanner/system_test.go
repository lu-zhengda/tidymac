package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSystemScanner_Scan(t *testing.T) {
	tmpDir := t.TempDir()

	cacheDir := filepath.Join(tmpDir, "Caches", "com.test.app")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "cache.db"), make([]byte, 1024), 0o644); err != nil {
		t.Fatal(err)
	}

	logDir := filepath.Join(tmpDir, "Logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(logDir, "test.log"), make([]byte, 512), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewSystemScanner(tmpDir)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) == 0 {
		t.Fatal("expected at least one target")
	}

	var totalSize int64
	for _, tgt := range targets {
		totalSize += tgt.Size
		if tgt.Category != "System Junk" {
			t.Errorf("expected category System Junk, got %s", tgt.Category)
		}
	}
	if totalSize == 0 {
		t.Error("expected non-zero total size")
	}
}

func TestSystemScanner_Name(t *testing.T) {
	s := NewSystemScanner("")
	if s.Name() != "System Junk" {
		t.Errorf("expected name System Junk, got %s", s.Name())
	}
}
