package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestXcodeScanner_Scan(t *testing.T) {
	tmpDir := t.TempDir()

	derivedData := filepath.Join(tmpDir, "Developer", "Xcode", "DerivedData", "MyProject-abc123")
	if err := os.MkdirAll(derivedData, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(derivedData, "Build.db"), make([]byte, 4096), 0o644); err != nil {
		t.Fatal(err)
	}

	archives := filepath.Join(tmpDir, "Developer", "Xcode", "Archives")
	if err := os.MkdirAll(archives, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(archives, "old.xcarchive"), make([]byte, 8192), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewXcodeScanner(tmpDir)
	targets, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) == 0 {
		t.Fatal("expected at least one Xcode target")
	}
	for _, tgt := range targets {
		if tgt.Category != "Xcode Junk" {
			t.Errorf("expected category Xcode Junk, got %s", tgt.Category)
		}
	}
}
