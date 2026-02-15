package scancache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Snapshot captures the state of a scan at a point in time.
type Snapshot struct {
	Timestamp  time.Time          `json:"timestamp"`
	Categories []CategorySnapshot `json:"categories"`
	TotalSize  int64              `json:"total_size"`
}

// CategorySnapshot captures the size and item count for a single category.
type CategorySnapshot struct {
	Name  string `json:"name"`
	Size  int64  `json:"size"`
	Items int    `json:"items"`
}

// CategoryDiff describes how a category changed between two snapshots.
type CategoryDiff struct {
	PreviousSize int64 `json:"previous_size"`
	CurrentSize  int64 `json:"current_size"`
	Delta        int64 `json:"delta"`
	IsNew        bool  `json:"is_new,omitempty"`
}

// DiffResult describes the differences between two snapshots.
type DiffResult struct {
	PreviousTimestamp time.Time               `json:"previous_timestamp"`
	TotalDelta        int64                   `json:"total_delta"`
	Categories        map[string]CategoryDiff `json:"categories"`
}

// DefaultPath returns the default scan cache file location:
// ~/.local/share/macbroom/last-scan.json
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "last-scan.json"
	}
	return filepath.Join(home, ".local", "share", "macbroom", "last-scan.json")
}

// Save writes a snapshot to the given path as indented JSON.
// It creates parent directories if they don't exist.
func Save(path string, snap Snapshot) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create scan cache directory: %w", err)
	}

	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal scan snapshot: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write scan cache file: %w", err)
	}

	return nil
}

// Load reads a snapshot from the given path.
func Load(path string) (Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Snapshot{}, fmt.Errorf("failed to read scan cache file: %w", err)
	}

	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return Snapshot{}, fmt.Errorf("failed to parse scan cache file: %w", err)
	}

	return snap, nil
}

// Diff computes per-category differences between two snapshots.
// New categories in curr get IsNew: true. Categories present in prev
// but absent in curr get a negative delta.
func Diff(prev, curr Snapshot) DiffResult {
	result := DiffResult{
		PreviousTimestamp: prev.Timestamp,
		TotalDelta:        curr.TotalSize - prev.TotalSize,
		Categories:        make(map[string]CategoryDiff),
	}

	// Index previous categories by name.
	prevMap := make(map[string]int64, len(prev.Categories))
	for _, c := range prev.Categories {
		prevMap[c.Name] = c.Size
	}

	// Process current categories.
	for _, c := range curr.Categories {
		prevSize, existed := prevMap[c.Name]
		result.Categories[c.Name] = CategoryDiff{
			PreviousSize: prevSize,
			CurrentSize:  c.Size,
			Delta:        c.Size - prevSize,
			IsNew:        !existed,
		}
		delete(prevMap, c.Name)
	}

	// Remaining entries in prevMap are categories that were removed.
	for name, prevSize := range prevMap {
		result.Categories[name] = CategoryDiff{
			PreviousSize: prevSize,
			CurrentSize:  0,
			Delta:        -prevSize,
		}
	}

	return result
}
