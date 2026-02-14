package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNodeScanner_Name(t *testing.T) {
	s := NewNodeScanner("/tmp", nil, 30*24*time.Hour)
	if s.Name() != "Node.js" {
		t.Errorf("expected name %q, got %q", "Node.js", s.Name())
	}
}

func TestNodeScanner_Description(t *testing.T) {
	s := NewNodeScanner("/tmp", nil, 30*24*time.Hour)
	want := "npm cache and stale node_modules"
	if s.Description() != want {
		t.Errorf("expected description %q, got %q", want, s.Description())
	}
}

func TestNodeScanner_Risk(t *testing.T) {
	s := NewNodeScanner("/tmp", nil, 30*24*time.Hour)
	if s.Risk() != Safe {
		t.Errorf("expected risk Safe, got %s", s.Risk())
	}
}

func TestNodeScanner_ImplementsScanner(t *testing.T) {
	var _ Scanner = NewNodeScanner("/tmp", nil, 30*24*time.Hour)
}

func TestNodeScanner_FindsNpmCache(t *testing.T) {
	// Create temp dir simulating home with .npm/_cacache
	home := t.TempDir()
	cacheDir := filepath.Join(home, ".npm", "_cacache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Write a file so the cache has measurable size
	if err := os.WriteFile(filepath.Join(cacheDir, "data.bin"), make([]byte, 1024), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewNodeScanner(home, nil, 30*24*time.Hour)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var found bool
	for _, tgt := range targets {
		if tgt.Path == filepath.Join(home, ".npm", "_cacache") {
			found = true
			if tgt.Category != "Node.js" {
				t.Errorf("expected category %q, got %q", "Node.js", tgt.Category)
			}
			if tgt.Risk != Safe {
				t.Errorf("expected risk Safe, got %s", tgt.Risk)
			}
			if tgt.Size == 0 {
				t.Error("expected non-zero size for npm cache")
			}
			if !tgt.IsDir {
				t.Error("expected IsDir to be true for npm cache")
			}
		}
	}
	if !found {
		t.Errorf("expected to find npm cache target at %s, got targets: %+v",
			filepath.Join(home, ".npm", "_cacache"), targets)
	}
}

func TestNodeScanner_FindsStaleNodeModules(t *testing.T) {
	// Create temp dir with a project containing old node_modules
	searchDir := t.TempDir()
	projectDir := filepath.Join(searchDir, "my-project")
	nmDir := filepath.Join(projectDir, "node_modules")
	if err := os.MkdirAll(nmDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Write a file inside node_modules
	if err := os.WriteFile(filepath.Join(nmDir, "package.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	// Set modification time to 60 days ago
	oldTime := time.Now().Add(-60 * 24 * time.Hour)
	if err := os.Chtimes(nmDir, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	maxAge := 30 * 24 * time.Hour
	s := NewNodeScanner(t.TempDir(), []string{searchDir}, maxAge)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var found bool
	for _, tgt := range targets {
		if tgt.Path == nmDir {
			found = true
			if tgt.Category != "Node.js" {
				t.Errorf("expected category %q, got %q", "Node.js", tgt.Category)
			}
			if tgt.Risk != Moderate {
				t.Errorf("expected risk Moderate, got %s", tgt.Risk)
			}
			if !tgt.IsDir {
				t.Error("expected IsDir to be true for node_modules")
			}
		}
	}
	if !found {
		t.Errorf("expected to find stale node_modules at %s, got targets: %+v", nmDir, targets)
	}
}

func TestNodeScanner_SkipsFreshNodeModules(t *testing.T) {
	// Create temp dir with a project containing fresh node_modules
	searchDir := t.TempDir()
	projectDir := filepath.Join(searchDir, "my-project")
	nmDir := filepath.Join(projectDir, "node_modules")
	if err := os.MkdirAll(nmDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nmDir, "package.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	// node_modules was just created — it's fresh

	maxAge := 30 * 24 * time.Hour
	s := NewNodeScanner(t.TempDir(), []string{searchDir}, maxAge)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, tgt := range targets {
		if tgt.Path == nmDir {
			t.Errorf("fresh node_modules should not be reported, but found: %+v", tgt)
		}
	}
}

func TestNodeScanner_NoNpmCache(t *testing.T) {
	// Home with no .npm directory — should not error
	home := t.TempDir()
	s := NewNodeScanner(home, nil, 30*24*time.Hour)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error when npm cache missing: %v", err)
	}
	if len(targets) != 0 {
		t.Errorf("expected 0 targets, got %d: %+v", len(targets), targets)
	}
}

func TestNodeScanner_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	home := t.TempDir()
	cacheDir := filepath.Join(home, ".npm", "_cacache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}

	s := NewNodeScanner(home, []string{home}, 30*24*time.Hour)
	_, err := s.Scan(ctx)
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestNodeScanner_SkipsNonExistentSearchPaths(t *testing.T) {
	s := NewNodeScanner(t.TempDir(), []string{"/nonexistent/path/that/does/not/exist"}, 30*24*time.Hour)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error for non-existent search path: %v", err)
	}
	if len(targets) != 0 {
		t.Errorf("expected 0 targets, got %d", len(targets))
	}
}

func TestNodeScanner_SkipsNestedNodeModules(t *testing.T) {
	// Create project/node_modules/some-pkg/node_modules (nested)
	// Only the top-level node_modules should be reported
	searchDir := t.TempDir()
	topNM := filepath.Join(searchDir, "project", "node_modules")
	nestedNM := filepath.Join(topNM, "some-pkg", "node_modules")
	if err := os.MkdirAll(nestedNM, 0o755); err != nil {
		t.Fatal(err)
	}

	// Make both old
	oldTime := time.Now().Add(-60 * 24 * time.Hour)
	if err := os.Chtimes(topNM, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(nestedNM, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	s := NewNodeScanner(t.TempDir(), []string{searchDir}, 30*24*time.Hour)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find only the top-level node_modules, not the nested one
	count := 0
	for _, tgt := range targets {
		if tgt.Risk == Moderate {
			count++
			if tgt.Path != topNM {
				t.Errorf("expected path %q, got %q", topNM, tgt.Path)
			}
		}
	}
	if count != 1 {
		t.Errorf("expected 1 stale node_modules target, got %d: %+v", count, targets)
	}
}
