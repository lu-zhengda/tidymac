package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestJetBrainsScanner_Name(t *testing.T) {
	s := NewJetBrainsScanner("")
	if s.Name() != "JetBrains" {
		t.Errorf("expected name %q, got %q", "JetBrains", s.Name())
	}
}

func TestJetBrainsScanner_ImplementsScanner(t *testing.T) {
	var _ Scanner = NewJetBrainsScanner("")
}

func TestJetBrainsScanner_FindsCaches(t *testing.T) {
	home := t.TempDir()
	cacheDir := filepath.Join(home, "Library", "Caches", "JetBrains", "IntelliJIdea2024.1")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "index.dat"), make([]byte, 4096), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewJetBrainsScanner(home)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var found bool
	for _, tgt := range targets {
		if tgt.Path == cacheDir && tgt.Category == "JetBrains" {
			found = true
			if tgt.Risk != Safe {
				t.Errorf("expected risk Safe, got %s", tgt.Risk)
			}
		}
	}
	if !found {
		t.Error("expected to find JetBrains cache target")
	}
}

func TestJetBrainsScanner_FindsLogs(t *testing.T) {
	home := t.TempDir()
	logDir := filepath.Join(home, "Library", "Logs", "JetBrains", "PyCharm2024.2")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(logDir, "idea.log"), make([]byte, 2048), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewJetBrainsScanner(home)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var found bool
	for _, tgt := range targets {
		if tgt.Path == logDir {
			found = true
		}
	}
	if !found {
		t.Error("expected to find JetBrains logs target")
	}
}

func TestJetBrainsScanner_NoJetBrainsDirs(t *testing.T) {
	s := NewJetBrainsScanner(t.TempDir())
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) != 0 {
		t.Errorf("expected 0 targets, got %d", len(targets))
	}
}

func TestJetBrainsScanner_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s := NewJetBrainsScanner(t.TempDir())
	_, err := s.Scan(ctx)
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
