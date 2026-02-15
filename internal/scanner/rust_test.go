package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRustScanner_Name(t *testing.T) {
	s := NewRustScanner("", nil, 0)
	if s.Name() != "Rust" {
		t.Errorf("expected name %q, got %q", "Rust", s.Name())
	}
}

func TestRustScanner_ImplementsScanner(t *testing.T) {
	var _ Scanner = NewRustScanner("", nil, 0)
}

func TestRustScanner_FindsCargoRegistryCache(t *testing.T) {
	home := t.TempDir()
	cacheDir := filepath.Join(home, ".cargo", "registry", "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "crate.tar"), make([]byte, 1024), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewRustScanner(home, nil, 0)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var found bool
	for _, tgt := range targets {
		if tgt.Path == cacheDir && tgt.Category == "Rust" {
			found = true
		}
	}
	if !found {
		t.Error("expected to find cargo registry cache target")
	}
}

func TestRustScanner_FindsTargetDirs(t *testing.T) {
	searchDir := t.TempDir()
	targetDir := filepath.Join(searchDir, "my-project", "target")
	if err := os.MkdirAll(filepath.Join(targetDir, "debug"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(searchDir, "my-project", "Cargo.toml"), []byte("[package]"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "debug", "binary"), make([]byte, 2048), 0o644); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().Add(-60 * 24 * time.Hour)
	os.Chtimes(targetDir, oldTime, oldTime)

	s := NewRustScanner(t.TempDir(), []string{searchDir}, 30*24*time.Hour)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var found bool
	for _, tgt := range targets {
		if tgt.Path == targetDir {
			found = true
			if tgt.Risk != Moderate {
				t.Errorf("expected risk Moderate, got %s", tgt.Risk)
			}
		}
	}
	if !found {
		t.Error("expected to find stale target/ dir")
	}
}

func TestRustScanner_NoCargoDir(t *testing.T) {
	home := t.TempDir()
	s := NewRustScanner(home, nil, 0)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) != 0 {
		t.Errorf("expected 0 targets, got %d", len(targets))
	}
}

func TestRustScanner_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s := NewRustScanner(t.TempDir(), nil, 0)
	_, err := s.Scan(ctx)
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
