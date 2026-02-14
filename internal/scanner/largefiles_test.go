package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLargeFileScanner_Scan(t *testing.T) {
	tmpDir := t.TempDir()

	largePath := filepath.Join(tmpDir, "bigfile.iso")
	if err := os.WriteFile(largePath, make([]byte, 200*1024*1024), 0o644); err != nil {
		t.Fatal(err)
	}

	smallPath := filepath.Join(tmpDir, "small.txt")
	if err := os.WriteFile(smallPath, make([]byte, 100), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewLargeFileScanner([]string{tmpDir}, 100*1024*1024, 0)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}
	if targets[0].Path != largePath {
		t.Errorf("expected %s, got %s", largePath, targets[0].Path)
	}
}

func TestLargeFileScanner_OldFiles(t *testing.T) {
	tmpDir := t.TempDir()

	oldFile := filepath.Join(tmpDir, "old.dmg")
	if err := os.WriteFile(oldFile, make([]byte, 1024), 0o644); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().Add(-100 * 24 * time.Hour)
	if err := os.Chtimes(oldFile, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	s := NewLargeFileScanner([]string{tmpDir}, 0, 90*24*time.Hour)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}
}
