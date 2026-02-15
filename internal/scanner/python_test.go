package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPythonScanner_Name(t *testing.T) {
	s := NewPythonScanner("", nil, 30*24*time.Hour)
	if s.Name() != "Python" {
		t.Errorf("expected name %q, got %q", "Python", s.Name())
	}
}

func TestPythonScanner_ImplementsScanner(t *testing.T) {
	var _ Scanner = NewPythonScanner("", nil, 30*24*time.Hour)
}

func TestPythonScanner_FindsPipCache(t *testing.T) {
	home := t.TempDir()
	cacheDir := filepath.Join(home, "Library", "Caches", "pip")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "data.bin"), make([]byte, 1024), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewPythonScanner(home, nil, 30*24*time.Hour)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var found bool
	for _, tgt := range targets {
		if tgt.Path == cacheDir {
			found = true
			if tgt.Category != "Python" {
				t.Errorf("expected category Python, got %q", tgt.Category)
			}
			if tgt.Risk != Safe {
				t.Errorf("expected risk Safe, got %s", tgt.Risk)
			}
		}
	}
	if !found {
		t.Errorf("expected to find pip cache target")
	}
}

func TestPythonScanner_FindsCondaPkgs(t *testing.T) {
	home := t.TempDir()
	condaDir := filepath.Join(home, "miniconda3", "pkgs")
	if err := os.MkdirAll(condaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(condaDir, "pkg.tar.bz2"), make([]byte, 2048), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewPythonScanner(home, nil, 30*24*time.Hour)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var found bool
	for _, tgt := range targets {
		if tgt.Path == condaDir {
			found = true
		}
	}
	if !found {
		t.Errorf("expected to find conda pkgs target")
	}
}

func TestPythonScanner_FindsStaleVenvs(t *testing.T) {
	searchDir := t.TempDir()
	venvDir := filepath.Join(searchDir, "my-project", ".venv")
	if err := os.MkdirAll(filepath.Join(venvDir, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(venvDir, "pyvenv.cfg"), []byte("home = /usr/bin"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().Add(-60 * 24 * time.Hour)
	os.Chtimes(venvDir, oldTime, oldTime)

	s := NewPythonScanner(t.TempDir(), []string{searchDir}, 30*24*time.Hour)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var found bool
	for _, tgt := range targets {
		if tgt.Path == venvDir {
			found = true
			if tgt.Risk != Moderate {
				t.Errorf("expected risk Moderate, got %s", tgt.Risk)
			}
		}
	}
	if !found {
		t.Errorf("expected to find stale venv target at %s", venvDir)
	}
}

func TestPythonScanner_SkipsFreshVenvs(t *testing.T) {
	searchDir := t.TempDir()
	venvDir := filepath.Join(searchDir, "my-project", ".venv")
	if err := os.MkdirAll(venvDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(venvDir, "pyvenv.cfg"), []byte("home = /usr/bin"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewPythonScanner(t.TempDir(), []string{searchDir}, 30*24*time.Hour)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, tgt := range targets {
		if tgt.Path == venvDir {
			t.Error("fresh venv should not be reported")
		}
	}
}

func TestPythonScanner_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	home := t.TempDir()
	cacheDir := filepath.Join(home, "Library", "Caches", "pip")
	os.MkdirAll(cacheDir, 0o755)

	s := NewPythonScanner(home, nil, 30*24*time.Hour)
	_, err := s.Scan(ctx)
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
