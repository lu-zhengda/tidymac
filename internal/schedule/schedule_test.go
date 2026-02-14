package schedule

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseTime(t *testing.T) {
	tests := []struct {
		input   string
		hour    int
		minute  int
		wantErr bool
	}{
		{"10:00", 10, 0, false},
		{"0:00", 0, 0, false},
		{"23:59", 23, 59, false},
		{"09:30", 9, 30, false},
		{"24:00", 0, 0, true},
		{"10:60", 0, 0, true},
		{"-1:00", 0, 0, true},
		{"abc", 0, 0, true},
		{"10", 0, 0, true},
		{"", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			hour, minute, err := parseTime(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseTime(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("parseTime(%q) unexpected error: %v", tt.input, err)
				return
			}
			if hour != tt.hour {
				t.Errorf("parseTime(%q) hour = %d, want %d", tt.input, hour, tt.hour)
			}
			if minute != tt.minute {
				t.Errorf("parseTime(%q) minute = %d, want %d", tt.input, minute, tt.minute)
			}
		})
	}
}

func TestIntervalWeekday(t *testing.T) {
	tests := []struct {
		interval string
		want     int
	}{
		{"daily", 0},
		{"Daily", 0},
		{"weekly", 1},
		{"Weekly", 1},
		{"unknown", 0},
	}

	for _, tt := range tests {
		t.Run(tt.interval, func(t *testing.T) {
			got := intervalWeekday(tt.interval)
			if got != tt.want {
				t.Errorf("intervalWeekday(%q) = %d, want %d", tt.interval, got, tt.want)
			}
		})
	}
}

func TestGeneratePlist(t *testing.T) {
	plist := GeneratePlistWithBinary("10:00", "daily", "/usr/local/bin/macbroom")
	if plist == "" {
		t.Fatal("expected non-empty plist")
	}
	if !strings.Contains(plist, bundleID) {
		t.Error("expected bundle identifier")
	}
	if !strings.Contains(plist, "<integer>10</integer>") {
		t.Error("expected hour 10")
	}
	if !strings.Contains(plist, "<integer>0</integer>") {
		t.Error("expected minute 0")
	}
	if !strings.Contains(plist, "/usr/local/bin/macbroom") {
		t.Error("expected binary path")
	}
	if !strings.Contains(plist, "--quiet") {
		t.Error("expected --quiet flag")
	}
	if !strings.Contains(plist, "--yes") {
		t.Error("expected --yes flag")
	}
	// Daily should NOT have a Weekday key.
	if strings.Contains(plist, "Weekday") {
		t.Error("daily plist should not contain Weekday key")
	}
}

func TestGeneratePlistWeekly(t *testing.T) {
	plist := GeneratePlistWithBinary("14:30", "weekly", "/usr/local/bin/macbroom")
	if plist == "" {
		t.Fatal("expected non-empty plist")
	}
	if !strings.Contains(plist, "<integer>14</integer>") {
		t.Error("expected hour 14")
	}
	if !strings.Contains(plist, "<integer>30</integer>") {
		t.Error("expected minute 30")
	}
	if !strings.Contains(plist, "Weekday") {
		t.Error("weekly plist should contain Weekday key")
	}
	if !strings.Contains(plist, "<integer>1</integer>") {
		t.Error("expected weekday 1 (Monday)")
	}
}

func TestGeneratePlistInvalidTime(t *testing.T) {
	plist := GeneratePlistWithBinary("invalid", "daily", "/usr/local/bin/macbroom")
	if plist != "" {
		t.Error("expected empty plist for invalid time")
	}
}

func TestGeneratePlistValidXML(t *testing.T) {
	plist := GeneratePlistWithBinary("10:00", "daily", "/usr/local/bin/macbroom")
	if !strings.HasPrefix(plist, "<?xml version=") {
		t.Error("expected XML declaration at start")
	}
	if !strings.Contains(plist, "<!DOCTYPE plist") {
		t.Error("expected DOCTYPE declaration")
	}
	if !strings.Contains(plist, "<plist version=\"1.0\">") {
		t.Error("expected plist version tag")
	}
	if !strings.HasSuffix(strings.TrimSpace(plist), "</plist>") {
		t.Error("expected closing plist tag")
	}
}

func TestInstallWritesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, bundleID+".plist")

	err := InstallWithBinary(path, "10:00", "daily", "/usr/local/bin/macbroom")
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("plist file should exist: %v", err)
	}
	if info.Size() == 0 {
		t.Error("plist file should not be empty")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read plist: %v", err)
	}
	if !strings.Contains(string(data), bundleID) {
		t.Error("plist content should contain bundle identifier")
	}
}

func TestInstallInvalidTime(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, bundleID+".plist")

	err := InstallWithBinary(path, "bad", "daily", "/usr/local/bin/macbroom")
	if err == nil {
		t.Error("Install with invalid time should return error")
	}
}

func TestInstallCreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "dir", bundleID+".plist")

	err := InstallWithBinary(path, "10:00", "daily", "/usr/local/bin/macbroom")
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("plist should exist after install")
	}
}

func TestUninstallRemovesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, bundleID+".plist")

	// Create a file first.
	if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := Uninstall(path)
	if err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("plist should be removed after uninstall")
	}
}

func TestUninstallNonExistentFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.plist")

	err := Uninstall(path)
	if err != nil {
		t.Errorf("Uninstall of non-existent file should not error, got: %v", err)
	}
}

func TestStatusTrue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, bundleID+".plist")

	if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	if !Status(path) {
		t.Error("Status should return true when plist exists")
	}
}

func TestStatusFalse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.plist")

	if Status(path) {
		t.Error("Status should return false when plist does not exist")
	}
}

func TestInstallThenUninstall(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, bundleID+".plist")

	err := InstallWithBinary(path, "10:00", "daily", "/usr/local/bin/macbroom")
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}
	if !Status(path) {
		t.Error("Status should be true after install")
	}

	err = Uninstall(path)
	if err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}
	if Status(path) {
		t.Error("Status should be false after uninstall")
	}
}

func TestDefaultPath(t *testing.T) {
	path := DefaultPath()
	if path == "" {
		t.Fatal("DefaultPath should not be empty")
	}
	if !strings.Contains(path, "LaunchAgents") {
		t.Errorf("DefaultPath should contain LaunchAgents, got %q", path)
	}
	if !strings.HasSuffix(path, bundleID+".plist") {
		t.Errorf("DefaultPath should end with %s.plist, got %q", bundleID, path)
	}
}
