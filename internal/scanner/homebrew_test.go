package scanner

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestHomebrewScanner_Name(t *testing.T) {
	s := NewHomebrewScanner()
	if s.Name() != "Homebrew" {
		t.Errorf("expected name Homebrew, got %s", s.Name())
	}
}

func TestHomebrewScanner_Description(t *testing.T) {
	s := NewHomebrewScanner()
	want := "Homebrew download cache"
	if s.Description() != want {
		t.Errorf("expected description %q, got %q", want, s.Description())
	}
}

func TestHomebrewScanner_Risk(t *testing.T) {
	s := NewHomebrewScanner()
	if s.Risk() != Safe {
		t.Errorf("expected risk Safe, got %s", s.Risk())
	}
}

func TestHomebrewScanner_ImplementsScanner(t *testing.T) {
	var _ Scanner = NewHomebrewScanner()
}

func TestHomebrewScanner_SkipsIfNotInstalled(t *testing.T) {
	s := NewHomebrewScanner()
	s.lookPath = func(file string) (string, error) {
		return "", exec.ErrNotFound
	}

	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("expected no error when brew not installed, got %v", err)
	}
	if targets != nil {
		t.Errorf("expected nil targets when brew not installed, got %v", targets)
	}
}

func TestHomebrewScanner_FindsCacheFiles(t *testing.T) {
	// Create a temporary directory to act as the brew cache.
	cacheDir := t.TempDir()

	// Create test cache files with known extensions.
	testFiles := []struct {
		name string
		size int
	}{
		{"wget-1.21.4.tar.gz", 512},
		{"curl--8.4.0.bottle.tar.gz", 1024},
		{"firefox-120.0.dmg", 2048},
		{"README.txt", 100}, // should be ignored
	}

	for _, tf := range testFiles {
		data := make([]byte, tf.size)
		if err := os.WriteFile(filepath.Join(cacheDir, tf.name), data, 0o644); err != nil {
			t.Fatalf("failed to create test file %s: %v", tf.name, err)
		}
	}

	s := NewHomebrewScanner()
	s.lookPath = func(file string) (string, error) {
		return "/opt/homebrew/bin/brew", nil
	}
	s.runCmd = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte(cacheDir + "\n"), nil
	}

	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(targets) != 3 {
		t.Fatalf("expected 3 targets, got %d: %+v", len(targets), targets)
	}

	// Verify all targets have correct category and risk.
	for i, tgt := range targets {
		if tgt.Category != "Homebrew" {
			t.Errorf("target[%d]: expected category Homebrew, got %s", i, tgt.Category)
		}
		if tgt.Risk != Safe {
			t.Errorf("target[%d]: expected risk Safe, got %s", i, tgt.Risk)
		}
		if tgt.Size <= 0 {
			t.Errorf("target[%d]: expected positive size, got %d", i, tgt.Size)
		}
	}

	// Verify the specific files were found (collect paths).
	found := make(map[string]bool)
	for _, tgt := range targets {
		found[filepath.Base(tgt.Path)] = true
	}

	for _, want := range []string{"wget-1.21.4.tar.gz", "curl--8.4.0.bottle.tar.gz", "firefox-120.0.dmg"} {
		if !found[want] {
			t.Errorf("expected to find %s in targets", want)
		}
	}
	if found["README.txt"] {
		t.Error("did not expect README.txt in targets")
	}
}

func TestHomebrewScanner_FindsCacheFilesInSubdirs(t *testing.T) {
	// Homebrew caches downloads in subdirectories too.
	cacheDir := t.TempDir()
	subDir := filepath.Join(cacheDir, "downloads")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	data := make([]byte, 256)
	if err := os.WriteFile(filepath.Join(subDir, "pkg.tar.gz"), data, 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	s := NewHomebrewScanner()
	s.lookPath = func(file string) (string, error) {
		return "/opt/homebrew/bin/brew", nil
	}
	s.runCmd = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte(cacheDir + "\n"), nil
	}

	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d: %+v", len(targets), targets)
	}
	if filepath.Base(targets[0].Path) != "pkg.tar.gz" {
		t.Errorf("expected pkg.tar.gz, got %s", targets[0].Path)
	}
}

func TestHomebrewScanner_EmptyCache(t *testing.T) {
	cacheDir := t.TempDir()

	s := NewHomebrewScanner()
	s.lookPath = func(file string) (string, error) {
		return "/opt/homebrew/bin/brew", nil
	}
	s.runCmd = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte(cacheDir + "\n"), nil
	}

	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) != 0 {
		t.Errorf("expected 0 targets for empty cache, got %d", len(targets))
	}
}

func TestHomebrewScanner_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	s := NewHomebrewScanner()
	s.lookPath = func(file string) (string, error) {
		return "/opt/homebrew/bin/brew", nil
	}
	s.runCmd = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return nil, ctx.Err()
	}

	targets, err := s.Scan(ctx)
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled error, got %v", err)
	}
	if targets != nil {
		t.Errorf("expected nil targets on context cancellation, got %v", targets)
	}
}
