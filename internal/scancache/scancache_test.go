package scancache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Save and Load round-trip
// ---------------------------------------------------------------------------

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test-scan.json")

	ts := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	snap := Snapshot{
		Timestamp: ts,
		TotalSize: 5000,
		Categories: []CategorySnapshot{
			{Name: "Browser Cache", Size: 3000, Items: 10},
			{Name: "System Junk", Size: 2000, Items: 5},
		},
	}

	if err := Save(path, snap); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if !loaded.Timestamp.Equal(ts) {
		t.Errorf("Timestamp = %v, want %v", loaded.Timestamp, ts)
	}
	if loaded.TotalSize != 5000 {
		t.Errorf("TotalSize = %d, want 5000", loaded.TotalSize)
	}
	if len(loaded.Categories) != 2 {
		t.Fatalf("len(Categories) = %d, want 2", len(loaded.Categories))
	}
	if loaded.Categories[0].Name != "Browser Cache" {
		t.Errorf("Categories[0].Name = %q, want %q", loaded.Categories[0].Name, "Browser Cache")
	}
	if loaded.Categories[0].Size != 3000 {
		t.Errorf("Categories[0].Size = %d, want 3000", loaded.Categories[0].Size)
	}
	if loaded.Categories[0].Items != 10 {
		t.Errorf("Categories[0].Items = %d, want 10", loaded.Categories[0].Items)
	}
	if loaded.Categories[1].Name != "System Junk" {
		t.Errorf("Categories[1].Name = %q, want %q", loaded.Categories[1].Name, "System Junk")
	}
}

// ---------------------------------------------------------------------------
// Load — missing file
// ---------------------------------------------------------------------------

func TestLoad_NotExist(t *testing.T) {
	_, err := Load("/tmp/nonexistent-macbroom-scan-cache-test.json")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

// ---------------------------------------------------------------------------
// Diff
// ---------------------------------------------------------------------------

func TestDiff(t *testing.T) {
	prev := Snapshot{
		Timestamp: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		TotalSize: 10000,
		Categories: []CategorySnapshot{
			{Name: "Browser Cache", Size: 5000, Items: 20},
			{Name: "System Junk", Size: 3000, Items: 10},
			{Name: "Removed Cat", Size: 2000, Items: 5},
		},
	}

	curr := Snapshot{
		Timestamp: time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
		TotalSize: 12000,
		Categories: []CategorySnapshot{
			{Name: "Browser Cache", Size: 7000, Items: 25}, // grew +2000
			{Name: "System Junk", Size: 2000, Items: 8},    // shrank -1000
			{Name: "New Cat", Size: 3000, Items: 12},        // new
			// "Removed Cat" is absent — should get negative delta
		},
	}

	result := Diff(prev, curr)

	if !result.PreviousTimestamp.Equal(prev.Timestamp) {
		t.Errorf("PreviousTimestamp = %v, want %v", result.PreviousTimestamp, prev.Timestamp)
	}

	if result.TotalDelta != 2000 {
		t.Errorf("TotalDelta = %d, want 2000", result.TotalDelta)
	}

	// Browser Cache grew.
	bc, ok := result.Categories["Browser Cache"]
	if !ok {
		t.Fatal("missing Browser Cache in diff")
	}
	if bc.PreviousSize != 5000 || bc.CurrentSize != 7000 || bc.Delta != 2000 {
		t.Errorf("Browser Cache diff: prev=%d curr=%d delta=%d", bc.PreviousSize, bc.CurrentSize, bc.Delta)
	}
	if bc.IsNew {
		t.Error("Browser Cache should not be marked as new")
	}

	// System Junk shrank.
	sj, ok := result.Categories["System Junk"]
	if !ok {
		t.Fatal("missing System Junk in diff")
	}
	if sj.Delta != -1000 {
		t.Errorf("System Junk delta = %d, want -1000", sj.Delta)
	}

	// New Cat is new.
	nc, ok := result.Categories["New Cat"]
	if !ok {
		t.Fatal("missing New Cat in diff")
	}
	if !nc.IsNew {
		t.Error("New Cat should be marked as new")
	}
	if nc.Delta != 3000 {
		t.Errorf("New Cat delta = %d, want 3000", nc.Delta)
	}

	// Removed Cat has negative delta.
	rc, ok := result.Categories["Removed Cat"]
	if !ok {
		t.Fatal("missing Removed Cat in diff")
	}
	if rc.Delta != -2000 {
		t.Errorf("Removed Cat delta = %d, want -2000", rc.Delta)
	}
	if rc.CurrentSize != 0 {
		t.Errorf("Removed Cat current size = %d, want 0", rc.CurrentSize)
	}
}

// ---------------------------------------------------------------------------
// DefaultPath
// ---------------------------------------------------------------------------

func TestDefaultPath(t *testing.T) {
	p := DefaultPath()
	if p == "" {
		t.Fatal("DefaultPath returned empty string")
	}
	if !filepath.IsAbs(p) {
		t.Errorf("DefaultPath returned non-absolute path: %q", p)
	}

	// Clean up: ensure we didn't accidentally create the file.
	os.Remove(p)
}
