package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Entry represents a single cleanup operation recorded in the history.
type Entry struct {
	Timestamp  time.Time `json:"timestamp"`
	Category   string    `json:"category"`
	Items      int       `json:"items"`
	BytesFreed int64     `json:"bytes_freed"`
	Method     string    `json:"method"` // "trash" or "permanent"
}

// CategoryStats holds aggregate statistics for a single category.
type CategoryStats struct {
	BytesFreed int64 `json:"bytes_freed"`
	Cleanups   int   `json:"cleanups"`
}

// Stats holds aggregate cleanup statistics.
type Stats struct {
	TotalFreed    int64                    `json:"total_freed"`
	TotalCleanups int                      `json:"total_cleanups"`
	ByCategory    map[string]CategoryStats `json:"by_category"`
	Recent        []Entry                  `json:"recent"`
}

// History manages the cleanup history file.
type History struct {
	path string
}

// New creates a new History that reads/writes the given file path.
func New(path string) *History {
	return &History{path: path}
}

// DefaultPath returns the default history file location:
// ~/.local/share/macbroom/history.json
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "history.json"
	}
	return filepath.Join(home, ".local", "share", "macbroom", "history.json")
}

// Record appends a cleanup entry to the history file.
func (h *History) Record(e Entry) error {
	entries, err := h.Load()
	if err != nil && !os.IsNotExist(err) {
		// If the file simply doesn't exist, start fresh.
		// For other errors (e.g., corrupt JSON), still start fresh to avoid blocking.
		entries = nil
	}

	entries = append(entries, e)

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal history: %w", err)
	}

	dir := filepath.Dir(h.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create history directory: %w", err)
	}

	if err := os.WriteFile(h.path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write history file: %w", err)
	}

	return nil
}

// Load reads all entries from the history file.
func (h *History) Load() ([]Entry, error) {
	data, err := os.ReadFile(h.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to read history file: %w", err)
	}

	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("failed to parse history file: %w", err)
	}

	return entries, nil
}

// Stats computes aggregate statistics from the history.
func (h *History) Stats() Stats {
	entries, err := h.Load()
	if err != nil || len(entries) == 0 {
		return Stats{
			ByCategory: make(map[string]CategoryStats),
		}
	}

	s := Stats{
		TotalCleanups: len(entries),
		ByCategory:    make(map[string]CategoryStats),
	}

	for _, e := range entries {
		s.TotalFreed += e.BytesFreed

		cs := s.ByCategory[e.Category]
		cs.BytesFreed += e.BytesFreed
		cs.Cleanups++
		s.ByCategory[e.Category] = cs
	}

	// Sort entries by timestamp descending for recent list.
	sorted := make([]Entry, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Timestamp.After(sorted[j].Timestamp)
	})

	limit := 5
	if len(sorted) < limit {
		limit = len(sorted)
	}
	s.Recent = sorted[:limit]

	return s
}
