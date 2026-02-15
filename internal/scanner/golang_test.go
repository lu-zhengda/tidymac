package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestGoScanner_Name(t *testing.T) {
	s := NewGoScanner("")
	if s.Name() != "Go" {
		t.Errorf("expected name %q, got %q", "Go", s.Name())
	}
}

func TestGoScanner_ImplementsScanner(t *testing.T) {
	var _ Scanner = NewGoScanner("")
}

func TestGoScanner_FindsModCache(t *testing.T) {
	home := t.TempDir()
	modCache := filepath.Join(home, "go", "pkg", "mod", "cache")
	if err := os.MkdirAll(modCache, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modCache, "module.zip"), make([]byte, 4096), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewGoScanner(home)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var found bool
	for _, tgt := range targets {
		if tgt.Path == modCache && tgt.Category == "Go" {
			found = true
			if tgt.Risk != Safe {
				t.Errorf("expected risk Safe, got %s", tgt.Risk)
			}
		}
	}
	if !found {
		t.Error("expected to find Go module cache target")
	}
}

func TestGoScanner_FindsBuildCache(t *testing.T) {
	home := t.TempDir()
	buildCache := filepath.Join(home, "Library", "Caches", "go-build")
	if err := os.MkdirAll(buildCache, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(buildCache, "cache.bin"), make([]byte, 2048), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewGoScanner(home)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var found bool
	for _, tgt := range targets {
		if tgt.Path == buildCache {
			found = true
		}
	}
	if !found {
		t.Error("expected to find Go build cache target")
	}
}

func TestGoScanner_NoGoDir(t *testing.T) {
	home := t.TempDir()
	s := NewGoScanner(home)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) != 0 {
		t.Errorf("expected 0 targets, got %d", len(targets))
	}
}

func TestGoScanner_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s := NewGoScanner(t.TempDir())
	_, err := s.Scan(ctx)
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
