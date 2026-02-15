package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestMavenScanner_Name(t *testing.T) {
	s := NewMavenScanner("")
	if s.Name() != "Maven" {
		t.Errorf("expected name %q, got %q", "Maven", s.Name())
	}
}

func TestMavenScanner_ImplementsScanner(t *testing.T) {
	var _ Scanner = NewMavenScanner("")
}

func TestMavenScanner_FindsRepository(t *testing.T) {
	home := t.TempDir()
	repo := filepath.Join(home, ".m2", "repository")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "artifact.jar"), make([]byte, 4096), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewMavenScanner(home)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var found bool
	for _, tgt := range targets {
		if tgt.Path == repo && tgt.Category == "Maven" {
			found = true
			if tgt.Risk != Safe {
				t.Errorf("expected risk Safe, got %s", tgt.Risk)
			}
		}
	}
	if !found {
		t.Error("expected to find Maven repository target")
	}
}

func TestMavenScanner_NoDir(t *testing.T) {
	home := t.TempDir()
	s := NewMavenScanner(home)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) != 0 {
		t.Errorf("expected 0 targets, got %d", len(targets))
	}
}

func TestMavenScanner_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s := NewMavenScanner(t.TempDir())
	_, err := s.Scan(ctx)
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
