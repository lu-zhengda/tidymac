package cli

import (
	"testing"
	"time"

	"github.com/lu-zhengda/macbroom/internal/dupes"
	"github.com/lu-zhengda/macbroom/internal/history"
	"github.com/lu-zhengda/macbroom/internal/scancache"
	"github.com/lu-zhengda/macbroom/internal/scanner"
)

func TestBuildScanJSON(t *testing.T) {
	targets := []scanner.Target{
		{Path: "/a", Size: 1000, Category: "System Junk", Risk: scanner.Safe},
		{Path: "/b", Size: 2000, Category: "System Junk", Risk: scanner.Moderate},
		{Path: "/c", Size: 3000, Category: "Browser Cache", Risk: scanner.Risky},
	}

	result := buildScanJSON(targets, nil)

	if result.Version != version {
		t.Errorf("Version = %q, want %q", result.Version, version)
	}

	if result.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}

	if result.TotalSize != 6000 {
		t.Errorf("TotalSize = %d, want 6000", result.TotalSize)
	}

	if result.TotalItems != 3 {
		t.Errorf("TotalItems = %d, want 3", result.TotalItems)
	}

	if len(result.Categories) != 2 {
		t.Fatalf("len(Categories) = %d, want 2", len(result.Categories))
	}

	// Check risk summary.
	if result.RiskSummary.Safe != 1000 {
		t.Errorf("RiskSummary.Safe = %d, want 1000", result.RiskSummary.Safe)
	}
	if result.RiskSummary.Moderate != 2000 {
		t.Errorf("RiskSummary.Moderate = %d, want 2000", result.RiskSummary.Moderate)
	}
	if result.RiskSummary.Risky != 3000 {
		t.Errorf("RiskSummary.Risky = %d, want 3000", result.RiskSummary.Risky)
	}

	// Diff should be nil.
	if result.Diff != nil {
		t.Error("Diff should be nil when no diff passed")
	}
}

func TestBuildScanJSON_WithDiff(t *testing.T) {
	targets := []scanner.Target{
		{Path: "/a", Size: 1000, Category: "System Junk", Risk: scanner.Safe},
	}

	prev := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	diff := &scancache.DiffResult{
		PreviousTimestamp: prev,
		TotalDelta:        500,
		Categories: map[string]scancache.CategoryDiff{
			"System Junk": {PreviousSize: 500, CurrentSize: 1000, Delta: 500},
		},
	}

	result := buildScanJSON(targets, diff)

	if result.Diff == nil {
		t.Fatal("Diff should not be nil")
	}

	if !result.Diff.PreviousTimestamp.Equal(prev) {
		t.Errorf("Diff.PreviousTimestamp = %v, want %v", result.Diff.PreviousTimestamp, prev)
	}

	if result.Diff.TotalDelta != 500 {
		t.Errorf("Diff.TotalDelta = %d, want 500", result.Diff.TotalDelta)
	}

	cd, ok := result.Diff.Categories["System Junk"]
	if !ok {
		t.Fatal("expected 'System Junk' in diff categories")
	}
	if cd.Delta != 500 {
		t.Errorf("CategoryDiff.Delta = %d, want 500", cd.Delta)
	}
}

func TestBuildDupesJSON(t *testing.T) {
	groups := []dupes.Group{
		{
			Size:  1024,
			Hash:  "abc123",
			Files: []string{"/a/file.txt", "/b/file.txt", "/c/file.txt"},
		},
		{
			Size:  2048,
			Hash:  "def456",
			Files: []string{"/x/data.bin", "/y/data.bin"},
		},
	}

	result := buildDupesJSON(groups)

	if result.Version != version {
		t.Errorf("Version = %q, want %q", result.Version, version)
	}

	if result.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}

	if len(result.Groups) != 2 {
		t.Fatalf("len(Groups) = %d, want 2", len(result.Groups))
	}

	if result.TotalFiles != 5 {
		t.Errorf("TotalFiles = %d, want 5", result.TotalFiles)
	}

	// Waste: group1 = 1024 * 2 = 2048, group2 = 2048 * 1 = 2048, total = 4096
	if result.TotalWaste != 4096 {
		t.Errorf("TotalWaste = %d, want 4096", result.TotalWaste)
	}

	// Check first group.
	g := result.Groups[0]
	if g.Size != 1024 {
		t.Errorf("Groups[0].Size = %d, want 1024", g.Size)
	}
	if g.Hash != "abc123" {
		t.Errorf("Groups[0].Hash = %q, want %q", g.Hash, "abc123")
	}
	if len(g.Files) != 3 {
		t.Errorf("Groups[0].Files count = %d, want 3", len(g.Files))
	}
}

func TestBuildSpaceLensJSON(t *testing.T) {
	nodes := []scanner.SpaceLensNode{
		{Path: "/tmp/a", Name: "a", Size: 5000, IsDir: true, Depth: 0},
		{Path: "/tmp/b", Name: "b", Size: 2000, IsDir: false, Depth: 0},
	}

	result := buildSpaceLensJSON("/tmp", nodes)

	if result.Version != version {
		t.Errorf("Version = %q, want %q", result.Version, version)
	}
	if result.Path != "/tmp" {
		t.Errorf("Path = %q, want %q", result.Path, "/tmp")
	}
	if len(result.Nodes) != 2 {
		t.Errorf("len(Nodes) = %d, want 2", len(result.Nodes))
	}
	if result.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestBuildUninstallJSON(t *testing.T) {
	targets := []scanner.Target{
		{Path: "/app/a", Size: 1000, Risk: scanner.Safe},
		{Path: "/app/b", Size: 2000, Risk: scanner.Moderate},
	}

	result := buildUninstallJSON("TestApp", targets)

	if result.Version != version {
		t.Errorf("Version = %q, want %q", result.Version, version)
	}
	if result.AppName != "TestApp" {
		t.Errorf("AppName = %q, want %q", result.AppName, "TestApp")
	}
	if result.TotalSize != 3000 {
		t.Errorf("TotalSize = %d, want 3000", result.TotalSize)
	}
	if result.Items != 2 {
		t.Errorf("Items = %d, want 2", result.Items)
	}
	if len(result.Targets) != 2 {
		t.Errorf("len(Targets) = %d, want 2", len(result.Targets))
	}
	if result.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestBuildUninstallJSON_Empty(t *testing.T) {
	result := buildUninstallJSON("NoApp", nil)

	if result.AppName != "NoApp" {
		t.Errorf("AppName = %q, want %q", result.AppName, "NoApp")
	}
	if result.TotalSize != 0 {
		t.Errorf("TotalSize = %d, want 0", result.TotalSize)
	}
	if result.Items != 0 {
		t.Errorf("Items = %d, want 0", result.Items)
	}
}

func TestBuildStatsJSON(t *testing.T) {
	stats := history.Stats{
		TotalFreed:    1048576,
		TotalCleanups: 5,
		ByCategory: map[string]history.CategoryStats{
			"System Junk": {BytesFreed: 524288, Cleanups: 3},
			"Browser":     {BytesFreed: 524288, Cleanups: 2},
		},
		Recent: []history.Entry{
			{
				Timestamp:  time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC),
				Category:   "System Junk",
				Items:      10,
				BytesFreed: 100000,
				Method:     "trash",
			},
		},
	}

	result := buildStatsJSON(stats)

	if result.Version != version {
		t.Errorf("Version = %q, want %q", result.Version, version)
	}

	if result.TotalFreed != 1048576 {
		t.Errorf("TotalFreed = %d, want 1048576", result.TotalFreed)
	}

	if result.TotalCleanups != 5 {
		t.Errorf("TotalCleanups = %d, want 5", result.TotalCleanups)
	}

	if len(result.ByCategory) != 2 {
		t.Errorf("len(ByCategory) = %d, want 2", len(result.ByCategory))
	}

	if len(result.Recent) != 1 {
		t.Errorf("len(Recent) = %d, want 1", len(result.Recent))
	}
}
