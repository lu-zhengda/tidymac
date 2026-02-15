package trends

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// StorageSnapshot captures disk usage at a point in time.
type StorageSnapshot struct {
	Timestamp string  `json:"timestamp"`
	Total     int64   `json:"total_bytes"`
	Used      int64   `json:"used_bytes"`
	Available int64   `json:"available_bytes"`
	UsedPct   float64 `json:"used_pct"`
}

// StorageForecast predicts when disk will fill up.
type StorageForecast struct {
	GrowthRatePerDay int64  `json:"growth_rate_per_day_bytes"`
	DaysUntilFull    int    `json:"days_until_full"`
	ProjectedDate    string `json:"projected_full_date,omitempty"`
	Confidence       string `json:"confidence"` // high, medium, low
}

// TrendReport is the full output for the trends command.
type TrendReport struct {
	Snapshots []StorageSnapshot `json:"snapshots"`
	Current   StorageSnapshot   `json:"current"`
	Forecast  *StorageForecast  `json:"forecast,omitempty"`
}

// maxEntries is the maximum number of snapshots to retain (one per day for a year).
const maxEntries = 365

// TakeSnapshot captures current disk usage by parsing `df -k /`.
func TakeSnapshot() (*StorageSnapshot, error) {
	return takeSnapshotFromDF()
}

// takeSnapshotFromDF runs `df -k /` and parses the output.
func takeSnapshotFromDF() (*StorageSnapshot, error) {
	out, err := exec.Command("df", "-k", "/").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run df: %w", err)
	}
	return parseDF(string(out))
}

// parseDF parses the output of `df -k /` into a StorageSnapshot.
// Expected format (macOS):
//
//	Filesystem   1024-blocks      Used Available Capacity ...
//	/dev/disk3s1 488245288   285438564 184654652    61%  ...
func parseDF(output string) (*StorageSnapshot, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("unexpected df output: too few lines")
	}

	// Parse the data line (second line).
	fields := strings.Fields(lines[1])
	if len(fields) < 4 {
		return nil, fmt.Errorf("unexpected df output: too few fields in data line")
	}

	// Fields: Filesystem, 1024-blocks, Used, Available, ...
	totalKB, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse total blocks: %w", err)
	}

	usedKB, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse used blocks: %w", err)
	}

	availKB, err := strconv.ParseInt(fields[3], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse available blocks: %w", err)
	}

	totalBytes := totalKB * 1024
	usedBytes := usedKB * 1024
	availBytes := availKB * 1024

	var usedPct float64
	if totalBytes > 0 {
		usedPct = math.Round(float64(usedBytes)/float64(totalBytes)*1000) / 10
	}

	return &StorageSnapshot{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Total:     totalBytes,
		Used:      usedBytes,
		Available: availBytes,
		UsedPct:   usedPct,
	}, nil
}

// Store manages persistent snapshot storage.
type Store struct {
	path string
}

// NewStore creates a Store that reads/writes snapshots at the given path.
func NewStore(path string) *Store {
	return &Store{path: path}
}

// DefaultStorePath returns the default snapshot file location:
// ~/.config/macbroom/storage-trends.json
func DefaultStorePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "storage-trends.json"
	}
	return filepath.Join(home, ".config", "macbroom", "storage-trends.json")
}

// Record takes a snapshot and appends it to the store, keeping at most maxEntries.
func (s *Store) Record() (*StorageSnapshot, error) {
	snap, err := TakeSnapshot()
	if err != nil {
		return nil, fmt.Errorf("failed to take snapshot: %w", err)
	}

	return s.append(snap)
}

// append adds a snapshot to the store and trims to maxEntries.
func (s *Store) append(snap *StorageSnapshot) (*StorageSnapshot, error) {
	snapshots, err := s.load()
	if err != nil {
		// Start fresh on any load error.
		snapshots = nil
	}

	snapshots = append(snapshots, *snap)

	// Trim to maxEntries, keeping the most recent.
	if len(snapshots) > maxEntries {
		snapshots = snapshots[len(snapshots)-maxEntries:]
	}

	if err := s.save(snapshots); err != nil {
		return nil, fmt.Errorf("failed to save snapshots: %w", err)
	}

	return snap, nil
}

// GetTrends returns snapshots filtered by duration string (e.g. "7d", "30d", "90d").
func (s *Store) GetTrends(lastDuration string) ([]StorageSnapshot, error) {
	snapshots, err := s.load()
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to load snapshots: %w", err)
	}

	dur, err := ParseDuration(lastDuration)
	if err != nil {
		return nil, fmt.Errorf("invalid duration %q: %w", lastDuration, err)
	}

	cutoff := time.Now().UTC().Add(-dur)
	var filtered []StorageSnapshot
	for _, snap := range snapshots {
		t, err := time.Parse(time.RFC3339, snap.Timestamp)
		if err != nil {
			continue
		}
		if !t.Before(cutoff) {
			filtered = append(filtered, snap)
		}
	}

	return filtered, nil
}

// load reads all snapshots from disk.
func (s *Store) load() ([]StorageSnapshot, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, err
	}

	var snapshots []StorageSnapshot
	if err := json.Unmarshal(data, &snapshots); err != nil {
		return nil, fmt.Errorf("failed to parse snapshots: %w", err)
	}

	return snapshots, nil
}

// save writes snapshots to disk, creating parent directories as needed.
func (s *Store) save(snapshots []StorageSnapshot) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(snapshots, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal snapshots: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write snapshots: %w", err)
	}

	return nil
}

// Forecast calculates growth rate and predicts when disk fills up.
// Requires at least 3 snapshots for a meaningful forecast.
func Forecast(snapshots []StorageSnapshot) *StorageForecast {
	if len(snapshots) < 2 {
		return &StorageForecast{
			GrowthRatePerDay: 0,
			DaysUntilFull:    -1,
			Confidence:       "low",
		}
	}

	// Parse timestamps and compute daily growth rate from deltas.
	type dataPoint struct {
		t    time.Time
		used int64
	}

	points := make([]dataPoint, 0, len(snapshots))
	for _, snap := range snapshots {
		t, err := time.Parse(time.RFC3339, snap.Timestamp)
		if err != nil {
			continue
		}
		points = append(points, dataPoint{t: t, used: snap.Used})
	}

	if len(points) < 2 {
		return &StorageForecast{
			GrowthRatePerDay: 0,
			DaysUntilFull:    -1,
			Confidence:       "low",
		}
	}

	// Calculate growth rate: (last used - first used) / days elapsed.
	first := points[0]
	last := points[len(points)-1]
	daysDiff := last.t.Sub(first.t).Hours() / 24
	if daysDiff < 0.01 {
		// All snapshots on the same moment — no meaningful rate.
		return &StorageForecast{
			GrowthRatePerDay: 0,
			DaysUntilFull:    -1,
			Confidence:       "low",
		}
	}

	growthTotal := last.used - first.used
	growthPerDay := int64(float64(growthTotal) / daysDiff)

	// Determine confidence based on number of data points.
	confidence := "low"
	switch {
	case len(points) > 30:
		confidence = "high"
	case len(points) >= 7:
		confidence = "medium"
	}

	forecast := &StorageForecast{
		GrowthRatePerDay: growthPerDay,
		Confidence:       confidence,
	}

	// If growth rate is zero or negative, disk is not filling up.
	if growthPerDay <= 0 {
		forecast.DaysUntilFull = -1
		return forecast
	}

	// Project days until available reaches 0.
	lastSnap := snapshots[len(snapshots)-1]
	if lastSnap.Available <= 0 {
		forecast.DaysUntilFull = 0
		forecast.ProjectedDate = time.Now().UTC().Format("2006-01-02")
		return forecast
	}

	daysUntilFull := int(lastSnap.Available / growthPerDay)
	forecast.DaysUntilFull = daysUntilFull
	projected := time.Now().UTC().AddDate(0, 0, daysUntilFull)
	forecast.ProjectedDate = projected.Format("2006-01-02")

	return forecast
}

// ParseDuration parses duration strings like "7d", "30d", "90d" into time.Duration.
func ParseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty duration string")
	}

	if strings.HasSuffix(s, "d") {
		numStr := strings.TrimSuffix(s, "d")
		days, err := strconv.Atoi(numStr)
		if err != nil {
			return 0, fmt.Errorf("invalid day count %q: %w", numStr, err)
		}
		if days < 0 {
			return 0, fmt.Errorf("negative duration %q", s)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}

	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q: %w", s, err)
	}
	return d, nil
}
