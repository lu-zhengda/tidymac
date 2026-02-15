package trends

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseDF(t *testing.T) {
	dfOutput := `Filesystem   1024-blocks      Used Available Capacity iused ifree %iused  Mounted on
/dev/disk3s1  488245288 285438564 184654652    61% 3214567     0  100%   /
`

	snap, err := parseDF(dfOutput)
	if err != nil {
		t.Fatalf("parseDF failed: %v", err)
	}

	expectedTotal := int64(488245288) * 1024
	expectedUsed := int64(285438564) * 1024
	expectedAvail := int64(184654652) * 1024

	if snap.Total != expectedTotal {
		t.Errorf("total: got %d, want %d", snap.Total, expectedTotal)
	}
	if snap.Used != expectedUsed {
		t.Errorf("used: got %d, want %d", snap.Used, expectedUsed)
	}
	if snap.Available != expectedAvail {
		t.Errorf("available: got %d, want %d", snap.Available, expectedAvail)
	}
	if snap.UsedPct < 58.0 || snap.UsedPct > 59.0 {
		t.Errorf("used_pct: got %.1f, want ~58.5", snap.UsedPct)
	}
	if snap.Timestamp == "" {
		t.Error("timestamp should not be empty")
	}
}

func TestParseDFTooFewLines(t *testing.T) {
	_, err := parseDF("Filesystem   1024-blocks      Used Available")
	if err == nil {
		t.Error("expected error for too few lines")
	}
}

func TestParseDFTooFewFields(t *testing.T) {
	dfOutput := `Filesystem   1024-blocks
/dev/disk3s1  488245288
`
	_, err := parseDF(dfOutput)
	if err == nil {
		t.Error("expected error for too few fields")
	}
}

func TestTakeSnapshotIntegration(t *testing.T) {
	snap, err := TakeSnapshot()
	if err != nil {
		t.Fatalf("TakeSnapshot failed: %v", err)
	}

	if snap.Total <= 0 {
		t.Errorf("expected positive total bytes, got %d", snap.Total)
	}
	if snap.Used <= 0 {
		t.Errorf("expected positive used bytes, got %d", snap.Used)
	}
	if snap.Available <= 0 {
		t.Errorf("expected positive available bytes, got %d", snap.Available)
	}
	if snap.UsedPct <= 0 || snap.UsedPct > 100 {
		t.Errorf("expected used_pct between 0 and 100, got %.1f", snap.UsedPct)
	}
}

func TestStoreRecordAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "storage-trends.json")
	store := NewStore(path)

	snap, err := store.Record()
	if err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	if snap.Total <= 0 {
		t.Errorf("snapshot total should be positive, got %d", snap.Total)
	}

	snapshots, err := store.load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(snapshots) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(snapshots))
	}
}

func TestStoreMaxEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "storage-trends.json")
	store := NewStore(path)

	// Pre-fill with maxEntries snapshots.
	snapshots := make([]StorageSnapshot, maxEntries)
	for i := range snapshots {
		snapshots[i] = StorageSnapshot{
			Timestamp: time.Now().UTC().Add(-time.Duration(maxEntries-i) * 24 * time.Hour).Format(time.RFC3339),
			Total:     500000000000,
			Used:      int64(200000000000 + i*100000000),
			Available: int64(300000000000 - i*100000000),
			UsedPct:   40.0,
		}
	}
	if err := store.save(snapshots); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// Append one more via the append method with a synthetic snapshot.
	newSnap := &StorageSnapshot{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Total:     500000000000,
		Used:      250000000000,
		Available: 250000000000,
		UsedPct:   50.0,
	}
	_, err := store.append(newSnap)
	if err != nil {
		t.Fatalf("append failed: %v", err)
	}

	loaded, err := store.load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(loaded) != maxEntries {
		t.Errorf("expected %d entries after trim, got %d", maxEntries, len(loaded))
	}

	// The oldest entry should have been dropped; last entry should be our new one.
	last := loaded[len(loaded)-1]
	if last.Used != 250000000000 {
		t.Errorf("expected last entry used to be %d, got %d", int64(250000000000), last.Used)
	}
}

func TestStoreCreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deep", "nested", "storage-trends.json")
	store := NewStore(path)

	snap := &StorageSnapshot{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Total:     500000000000,
		Used:      200000000000,
		Available: 300000000000,
		UsedPct:   40.0,
	}
	_, err := store.append(snap)
	if err != nil {
		t.Fatalf("append with nested path failed: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected file to be created")
	}
}

func TestGetTrends(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "storage-trends.json")
	store := NewStore(path)

	now := time.Now().UTC()
	snapshots := []StorageSnapshot{
		{
			Timestamp: now.Add(-60 * 24 * time.Hour).Format(time.RFC3339),
			Total:     500000000000,
			Used:      200000000000,
			Available: 300000000000,
			UsedPct:   40.0,
		},
		{
			Timestamp: now.Add(-20 * 24 * time.Hour).Format(time.RFC3339),
			Total:     500000000000,
			Used:      210000000000,
			Available: 290000000000,
			UsedPct:   42.0,
		},
		{
			Timestamp: now.Add(-5 * 24 * time.Hour).Format(time.RFC3339),
			Total:     500000000000,
			Used:      220000000000,
			Available: 280000000000,
			UsedPct:   44.0,
		},
	}
	if err := store.save(snapshots); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	tests := []struct {
		name     string
		duration string
		want     int
	}{
		{"last 7 days", "7d", 1},
		{"last 30 days", "30d", 2},
		{"last 90 days", "90d", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := store.GetTrends(tt.duration)
			if err != nil {
				t.Fatalf("GetTrends(%q) failed: %v", tt.duration, err)
			}
			if len(got) != tt.want {
				t.Errorf("GetTrends(%q): got %d snapshots, want %d", tt.duration, len(got), tt.want)
			}
		})
	}
}

func TestGetTrendsEmptyStore(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.json")
	store := NewStore(path)

	got, err := store.GetTrends("30d")
	if err != nil {
		t.Fatalf("GetTrends on empty store failed: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for empty store, got %d items", len(got))
	}
}

func TestGetTrendsInvalidDuration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "storage-trends.json")
	store := NewStore(path)

	if err := store.save([]StorageSnapshot{}); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	_, err := store.GetTrends("invalid")
	if err == nil {
		t.Error("expected error for invalid duration")
	}
}

func TestForecastGrowingUsage(t *testing.T) {
	now := time.Now().UTC()
	snapshots := make([]StorageSnapshot, 10)
	for i := range snapshots {
		day := now.Add(-time.Duration(10-i) * 24 * time.Hour)
		snapshots[i] = StorageSnapshot{
			Timestamp: day.Format(time.RFC3339),
			Total:     500000000000,                                 // 500 GB
			Used:      int64(200000000000 + i*1000000000),           // grows by 1 GB/day
			Available: int64(300000000000 - i*1000000000),           // shrinks by 1 GB/day
			UsedPct:   float64(200000000000+i*1000000000) / 5000000000,
		}
	}

	fc := Forecast(snapshots)

	if fc.GrowthRatePerDay <= 0 {
		t.Errorf("expected positive growth rate, got %d", fc.GrowthRatePerDay)
	}
	if fc.DaysUntilFull <= 0 {
		t.Errorf("expected positive days until full, got %d", fc.DaysUntilFull)
	}
	if fc.ProjectedDate == "" {
		t.Error("expected projected date to be set")
	}
	if fc.Confidence != "medium" {
		t.Errorf("expected medium confidence for 10 points, got %q", fc.Confidence)
	}

	// Growth rate should be approximately 1 GB/day.
	expectedRate := int64(1000000000)
	tolerance := int64(200000000) // 200 MB tolerance
	if abs64(fc.GrowthRatePerDay-expectedRate) > tolerance {
		t.Errorf("growth rate: got %d, want ~%d (tolerance %d)", fc.GrowthRatePerDay, expectedRate, tolerance)
	}

	// Days until full: ~291 GB available / 1 GB per day = ~291 days.
	lastAvail := snapshots[len(snapshots)-1].Available
	expectedDays := int(lastAvail / expectedRate)
	if abs(fc.DaysUntilFull-expectedDays) > 5 {
		t.Errorf("days until full: got %d, want ~%d", fc.DaysUntilFull, expectedDays)
	}
}

func TestForecastShrinkingUsage(t *testing.T) {
	now := time.Now().UTC()
	snapshots := make([]StorageSnapshot, 10)
	for i := range snapshots {
		day := now.Add(-time.Duration(10-i) * 24 * time.Hour)
		snapshots[i] = StorageSnapshot{
			Timestamp: day.Format(time.RFC3339),
			Total:     500000000000,
			Used:      int64(300000000000 - i*2000000000),  // shrinking
			Available: int64(200000000000 + i*2000000000),
			UsedPct:   float64(300000000000-i*2000000000) / 5000000000,
		}
	}

	fc := Forecast(snapshots)

	if fc.GrowthRatePerDay >= 0 {
		t.Errorf("expected negative growth rate for shrinking usage, got %d", fc.GrowthRatePerDay)
	}
	if fc.DaysUntilFull != -1 {
		t.Errorf("expected -1 days until full for shrinking usage, got %d", fc.DaysUntilFull)
	}
	if fc.ProjectedDate != "" {
		t.Errorf("expected empty projected date for shrinking usage, got %q", fc.ProjectedDate)
	}
}

func TestForecastStableUsage(t *testing.T) {
	now := time.Now().UTC()
	snapshots := make([]StorageSnapshot, 40)
	for i := range snapshots {
		day := now.Add(-time.Duration(40-i) * 24 * time.Hour)
		snapshots[i] = StorageSnapshot{
			Timestamp: day.Format(time.RFC3339),
			Total:     500000000000,
			Used:      250000000000,
			Available: 250000000000,
			UsedPct:   50.0,
		}
	}

	fc := Forecast(snapshots)

	if fc.GrowthRatePerDay != 0 {
		t.Errorf("expected zero growth rate for stable usage, got %d", fc.GrowthRatePerDay)
	}
	if fc.DaysUntilFull != -1 {
		t.Errorf("expected -1 days until full for stable usage, got %d", fc.DaysUntilFull)
	}
	if fc.Confidence != "high" {
		t.Errorf("expected high confidence for 40 points, got %q", fc.Confidence)
	}
}

func TestForecastTooFewSnapshots(t *testing.T) {
	tests := []struct {
		name      string
		snapshots []StorageSnapshot
	}{
		{"zero snapshots", nil},
		{"one snapshot", []StorageSnapshot{
			{Timestamp: time.Now().UTC().Format(time.RFC3339), Total: 500000000000, Used: 200000000000, Available: 300000000000},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := Forecast(tt.snapshots)
			if fc.Confidence != "low" {
				t.Errorf("expected low confidence, got %q", fc.Confidence)
			}
			if fc.DaysUntilFull != -1 {
				t.Errorf("expected -1 days until full, got %d", fc.DaysUntilFull)
			}
		})
	}
}

func TestForecastConfidenceLevels(t *testing.T) {
	now := time.Now().UTC()

	makeSnapshots := func(n int) []StorageSnapshot {
		snaps := make([]StorageSnapshot, n)
		for i := range snaps {
			day := now.Add(-time.Duration(n-i) * 24 * time.Hour)
			snaps[i] = StorageSnapshot{
				Timestamp: day.Format(time.RFC3339),
				Total:     500000000000,
				Used:      int64(200000000000 + i*1000000000),
				Available: int64(300000000000 - i*1000000000),
			}
		}
		return snaps
	}

	tests := []struct {
		name       string
		points     int
		wantConf   string
	}{
		{"3 points - low", 3, "low"},
		{"6 points - low", 6, "low"},
		{"7 points - medium", 7, "medium"},
		{"15 points - medium", 15, "medium"},
		{"31 points - high", 31, "high"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := Forecast(makeSnapshots(tt.points))
			if fc.Confidence != tt.wantConf {
				t.Errorf("confidence for %d points: got %q, want %q", tt.points, fc.Confidence, tt.wantConf)
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input   string
		want    time.Duration
		wantErr bool
	}{
		{"7d", 7 * 24 * time.Hour, false},
		{"30d", 30 * 24 * time.Hour, false},
		{"90d", 90 * 24 * time.Hour, false},
		{"1d", 24 * time.Hour, false},
		{"0d", 0, false},
		{"24h", 24 * time.Hour, false},
		{"", 0, true},
		{"invalid", 0, true},
		{"-5d", 0, true},
		{"abcd", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDuration(%q): error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseDuration(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestStoreCorruptedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "storage-trends.json")

	if err := os.WriteFile(path, []byte("not json"), 0o644); err != nil {
		t.Fatalf("failed to write corrupt file: %v", err)
	}

	store := NewStore(path)

	// Record should recover from a corrupt file by starting fresh.
	snap, err := store.Record()
	if err != nil {
		t.Fatalf("Record after corrupt file failed: %v", err)
	}
	if snap.Total <= 0 {
		t.Error("expected valid snapshot after recovery")
	}

	loaded, err := store.load()
	if err != nil {
		t.Fatalf("load after recovery failed: %v", err)
	}
	if len(loaded) != 1 {
		t.Errorf("expected 1 snapshot after recovery, got %d", len(loaded))
	}
}

func TestSnapshotJSONRoundTrip(t *testing.T) {
	snap := StorageSnapshot{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Total:     500000000000,
		Used:      200000000000,
		Available: 300000000000,
		UsedPct:   40.0,
	}

	data, err := json.Marshal(snap)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var got StorageSnapshot
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if got.Total != snap.Total {
		t.Errorf("total: got %d, want %d", got.Total, snap.Total)
	}
	if got.Used != snap.Used {
		t.Errorf("used: got %d, want %d", got.Used, snap.Used)
	}
	if got.Available != snap.Available {
		t.Errorf("available: got %d, want %d", got.Available, snap.Available)
	}
}

func TestForecastJSONRoundTrip(t *testing.T) {
	fc := StorageForecast{
		GrowthRatePerDay: 1000000000,
		DaysUntilFull:    291,
		ProjectedDate:    "2026-12-03",
		Confidence:       "medium",
	}

	data, err := json.Marshal(fc)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var got StorageForecast
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if got.GrowthRatePerDay != fc.GrowthRatePerDay {
		t.Errorf("growth rate: got %d, want %d", got.GrowthRatePerDay, fc.GrowthRatePerDay)
	}
	if got.DaysUntilFull != fc.DaysUntilFull {
		t.Errorf("days until full: got %d, want %d", got.DaysUntilFull, fc.DaysUntilFull)
	}
	if got.ProjectedDate != fc.ProjectedDate {
		t.Errorf("projected date: got %q, want %q", got.ProjectedDate, fc.ProjectedDate)
	}
}

func abs64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
