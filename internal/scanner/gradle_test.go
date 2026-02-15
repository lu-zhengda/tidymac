package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestGradleScanner_Name(t *testing.T) {
	s := NewGradleScanner("")
	if s.Name() != "Gradle" {
		t.Errorf("expected name %q, got %q", "Gradle", s.Name())
	}
}

func TestGradleScanner_ImplementsScanner(t *testing.T) {
	var _ Scanner = NewGradleScanner("")
}

func TestGradleScanner_FindsCaches(t *testing.T) {
	home := t.TempDir()
	caches := filepath.Join(home, ".gradle", "caches")
	if err := os.MkdirAll(caches, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(caches, "cache.bin"), make([]byte, 2048), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewGradleScanner(home)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var found bool
	for _, tgt := range targets {
		if tgt.Path == caches && tgt.Category == "Gradle" {
			found = true
			if tgt.Risk != Safe {
				t.Errorf("expected risk Safe, got %s", tgt.Risk)
			}
		}
	}
	if !found {
		t.Error("expected to find Gradle caches target")
	}
}

func TestGradleScanner_FindsWrapperDists(t *testing.T) {
	home := t.TempDir()
	dists := filepath.Join(home, ".gradle", "wrapper", "dists")
	if err := os.MkdirAll(dists, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dists, "gradle-8.zip"), make([]byte, 4096), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewGradleScanner(home)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var found bool
	for _, tgt := range targets {
		if tgt.Path == dists {
			found = true
		}
	}
	if !found {
		t.Error("expected to find Gradle wrapper dists target")
	}
}

func TestGradleScanner_NoDir(t *testing.T) {
	home := t.TempDir()
	s := NewGradleScanner(home)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) != 0 {
		t.Errorf("expected 0 targets, got %d", len(targets))
	}
}

func TestGradleScanner_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s := NewGradleScanner(t.TempDir())
	_, err := s.Scan(ctx)
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
