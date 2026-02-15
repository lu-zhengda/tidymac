package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRubyScanner_Name(t *testing.T) {
	s := NewRubyScanner("")
	if s.Name() != "Ruby" {
		t.Errorf("expected name %q, got %q", "Ruby", s.Name())
	}
}

func TestRubyScanner_ImplementsScanner(t *testing.T) {
	var _ Scanner = NewRubyScanner("")
}

func TestRubyScanner_FindsGemDir(t *testing.T) {
	home := t.TempDir()
	gemDir := filepath.Join(home, ".gem")
	if err := os.MkdirAll(gemDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gemDir, "specs"), make([]byte, 2048), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewRubyScanner(home)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var found bool
	for _, tgt := range targets {
		if tgt.Path == gemDir && tgt.Category == "Ruby" {
			found = true
			if tgt.Risk != Safe {
				t.Errorf("expected risk Safe, got %s", tgt.Risk)
			}
		}
	}
	if !found {
		t.Error("expected to find Ruby gem directory target")
	}
}

func TestRubyScanner_FindsBundleCache(t *testing.T) {
	home := t.TempDir()
	bundleCache := filepath.Join(home, ".bundle", "cache")
	if err := os.MkdirAll(bundleCache, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundleCache, "gem.gem"), make([]byte, 4096), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewRubyScanner(home)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var found bool
	for _, tgt := range targets {
		if tgt.Path == bundleCache {
			found = true
		}
	}
	if !found {
		t.Error("expected to find Bundler cache target")
	}
}

func TestRubyScanner_NoDir(t *testing.T) {
	home := t.TempDir()
	s := NewRubyScanner(home)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) != 0 {
		t.Errorf("expected 0 targets, got %d", len(targets))
	}
}

func TestRubyScanner_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s := NewRubyScanner(t.TempDir())
	_, err := s.Scan(ctx)
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
