package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Default()

	// LargeFiles defaults
	if cfg.LargeFiles.MinSize != 100*1024*1024 {
		t.Errorf("expected MinSize 100MB, got %d", cfg.LargeFiles.MinSize)
	}
	if cfg.LargeFiles.MinSizeStr != "100MB" {
		t.Errorf("expected MinSizeStr '100MB', got %q", cfg.LargeFiles.MinSizeStr)
	}
	if cfg.LargeFiles.MinAge != "90d" {
		t.Errorf("expected MinAge '90d', got %q", cfg.LargeFiles.MinAge)
	}
	if len(cfg.LargeFiles.Paths) != 2 {
		t.Fatalf("expected 2 default paths, got %d", len(cfg.LargeFiles.Paths))
	}
	if cfg.LargeFiles.Paths[0] != "~/Downloads" {
		t.Errorf("expected first path '~/Downloads', got %q", cfg.LargeFiles.Paths[0])
	}
	if cfg.LargeFiles.Paths[1] != "~/Desktop" {
		t.Errorf("expected second path '~/Desktop', got %q", cfg.LargeFiles.Paths[1])
	}

	// Scanners defaults — all enabled
	if !cfg.Scanners.System {
		t.Error("expected Scanners.System to be true")
	}
	if !cfg.Scanners.Browser {
		t.Error("expected Scanners.Browser to be true")
	}
	if !cfg.Scanners.Xcode {
		t.Error("expected Scanners.Xcode to be true")
	}
	if !cfg.Scanners.LargeFiles {
		t.Error("expected Scanners.LargeFiles to be true")
	}
	if !cfg.Scanners.Docker {
		t.Error("expected Scanners.Docker to be true")
	}
	if !cfg.Scanners.Node {
		t.Error("expected Scanners.Node to be true")
	}
	if !cfg.Scanners.Homebrew {
		t.Error("expected Scanners.Homebrew to be true")
	}
	if !cfg.Scanners.IOSSimulators {
		t.Error("expected Scanners.IOSSimulators to be true")
	}

	// DevTools defaults
	if len(cfg.DevTools.SearchPaths) != 5 {
		t.Fatalf("expected 5 DevTools search paths, got %d", len(cfg.DevTools.SearchPaths))
	}
	if cfg.DevTools.SearchPaths[0] != "~/Documents" {
		t.Errorf("expected first DevTools path '~/Documents', got %q", cfg.DevTools.SearchPaths[0])
	}
	if cfg.DevTools.MinAge != "30d" {
		t.Errorf("expected DevTools.MinAge '30d', got %q", cfg.DevTools.MinAge)
	}

	// SpaceLens defaults
	if cfg.SpaceLens.DefaultPath != "/" {
		t.Errorf("expected SpaceLens.DefaultPath '/', got %q", cfg.SpaceLens.DefaultPath)
	}
	if cfg.SpaceLens.Depth != 2 {
		t.Errorf("expected SpaceLens.Depth 2, got %d", cfg.SpaceLens.Depth)
	}

	// Schedule defaults
	if cfg.Schedule.Enabled {
		t.Error("expected Schedule.Enabled to be false")
	}
	if cfg.Schedule.Interval != "daily" {
		t.Errorf("expected Schedule.Interval 'daily', got %q", cfg.Schedule.Interval)
	}
	if cfg.Schedule.Time != "10:00" {
		t.Errorf("expected Schedule.Time '10:00', got %q", cfg.Schedule.Time)
	}
	if !cfg.Schedule.Notify {
		t.Error("expected Schedule.Notify to be true")
	}

	// Exclude defaults
	if cfg.Exclude == nil {
		t.Error("expected Exclude to be non-nil (empty slice)")
	}
}

func TestDefaultConfig_NewScanners(t *testing.T) {
	cfg := Default()
	if !cfg.Scanners.Python {
		t.Error("expected Python scanner enabled by default")
	}
	if !cfg.Scanners.Rust {
		t.Error("expected Rust scanner enabled by default")
	}
	if !cfg.Scanners.Go {
		t.Error("expected Go scanner enabled by default")
	}
	if !cfg.Scanners.JetBrains {
		t.Error("expected JetBrains scanner enabled by default")
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	content := `large_files:
  min_size: "500MB"
  min_age: "30d"
  paths:
    - "~/Documents"
exclude:
  - "*.log"
  - "/tmp/**"
scanners:
  system: false
  browser: true
  xcode: false
  large_files: true
  docker: false
  node: false
  homebrew: false
  ios_simulators: false
spacelens:
  default_path: "/Users"
  depth: 3
schedule:
  enabled: true
  interval: "weekly"
  time: "09:00"
  notify: false
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadFrom(cfgPath)
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}

	// Verify parsed values
	if cfg.LargeFiles.MinSizeStr != "500MB" {
		t.Errorf("expected MinSizeStr '500MB', got %q", cfg.LargeFiles.MinSizeStr)
	}
	if cfg.LargeFiles.MinSize != 500*1024*1024 {
		t.Errorf("expected MinSize 500MB (%d), got %d", 500*1024*1024, cfg.LargeFiles.MinSize)
	}
	if cfg.LargeFiles.MinAge != "30d" {
		t.Errorf("expected MinAge '30d', got %q", cfg.LargeFiles.MinAge)
	}
	if len(cfg.LargeFiles.Paths) != 1 || cfg.LargeFiles.Paths[0] != "~/Documents" {
		t.Errorf("expected paths [~/Documents], got %v", cfg.LargeFiles.Paths)
	}

	if len(cfg.Exclude) != 2 {
		t.Fatalf("expected 2 exclude patterns, got %d", len(cfg.Exclude))
	}

	if cfg.Scanners.System {
		t.Error("expected Scanners.System to be false")
	}
	if !cfg.Scanners.Browser {
		t.Error("expected Scanners.Browser to be true")
	}
	if cfg.Scanners.Xcode {
		t.Error("expected Scanners.Xcode to be false")
	}
	if cfg.Scanners.Docker {
		t.Error("expected Scanners.Docker to be false")
	}

	if cfg.SpaceLens.DefaultPath != "/Users" {
		t.Errorf("expected SpaceLens.DefaultPath '/Users', got %q", cfg.SpaceLens.DefaultPath)
	}
	if cfg.SpaceLens.Depth != 3 {
		t.Errorf("expected SpaceLens.Depth 3, got %d", cfg.SpaceLens.Depth)
	}

	if !cfg.Schedule.Enabled {
		t.Error("expected Schedule.Enabled to be true")
	}
	if cfg.Schedule.Interval != "weekly" {
		t.Errorf("expected Schedule.Interval 'weekly', got %q", cfg.Schedule.Interval)
	}
	if cfg.Schedule.Notify {
		t.Error("expected Schedule.Notify to be false")
	}
}

func TestLoadFromFilePartial(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	// Only override one field — rest should keep defaults
	content := `large_files:
  min_size: "200MB"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadFrom(cfgPath)
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}

	if cfg.LargeFiles.MinSize != 200*1024*1024 {
		t.Errorf("expected MinSize 200MB, got %d", cfg.LargeFiles.MinSize)
	}
	// Defaults should be preserved for unset fields
	if cfg.LargeFiles.MinAge != "90d" {
		t.Errorf("expected MinAge '90d' (default), got %q", cfg.LargeFiles.MinAge)
	}
	if !cfg.Scanners.System {
		t.Error("expected Scanners.System to keep default (true)")
	}
	if cfg.SpaceLens.Depth != 2 {
		t.Errorf("expected SpaceLens.Depth to keep default 2, got %d", cfg.SpaceLens.Depth)
	}
}

func TestLoadCreatesDefault(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "subdir", "config.yaml")

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Should have created the file with defaults
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Fatal("expected config file to be created")
	}

	// Returned config should be defaults
	if cfg.LargeFiles.MinSize != 100*1024*1024 {
		t.Errorf("expected default MinSize, got %d", cfg.LargeFiles.MinSize)
	}

	// Load it again — should parse the defaults file
	cfg2, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("second Load failed: %v", err)
	}
	if cfg2.LargeFiles.MinSize != cfg.LargeFiles.MinSize {
		t.Error("second load returned different MinSize")
	}
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		input   string
		want    int64
		wantErr bool
	}{
		{"100MB", 100 * 1024 * 1024, false},
		{"100mb", 100 * 1024 * 1024, false},
		{"1GB", 1024 * 1024 * 1024, false},
		{"1gb", 1024 * 1024 * 1024, false},
		{"500KB", 500 * 1024, false},
		{"500kb", 500 * 1024, false},
		{"1024", 1024, false},
		{"0", 0, false},
		{"2TB", 2 * 1024 * 1024 * 1024 * 1024, false},
		{"", 0, true},
		{"abc", 0, true},
		{"-1", 0, true},
		{"-100MB", 0, true},
		{"MB", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseSize(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseSize(%q) expected error, got %d", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseSize(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("ParseSize(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsExcluded(t *testing.T) {
	cfg := Default()
	cfg.Exclude = []string{"*.log", "/tmp/**", "*.DS_Store"}

	tests := []struct {
		path string
		want bool
	}{
		{"/var/log/system.log", true},
		{"/tmp/foo/bar", true},
		{"/home/.DS_Store", true},
		{"/home/user/file.txt", false},
		{"/home/user/photo.jpg", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := cfg.IsExcluded(tt.path)
			if got != tt.want {
				t.Errorf("IsExcluded(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestLoadExistingFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	// Create the file first with non-default values.
	content := `large_files:
  min_size: "250MB"
  min_age: "60d"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.LargeFiles.MinSize != 250*1024*1024 {
		t.Errorf("expected MinSize 250MB, got %d", cfg.LargeFiles.MinSize)
	}
	if cfg.LargeFiles.MinAge != "60d" {
		t.Errorf("expected MinAge '60d', got %q", cfg.LargeFiles.MinAge)
	}
}

func TestLoadFromInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	if err := os.WriteFile(cfgPath, []byte("{{invalid yaml"), 0o644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	_, err := LoadFrom(cfgPath)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoadFromNonexistentFile(t *testing.T) {
	_, err := LoadFrom("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	cfg := Default()
	cfg.LargeFiles.MinSizeStr = "250MB"
	cfg.LargeFiles.MinSize = 250 * 1024 * 1024
	cfg.Scanners.Docker = false
	cfg.Exclude = []string{"*.tmp"}

	if err := cfg.Save(cfgPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := LoadFrom(cfgPath)
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}

	if loaded.LargeFiles.MinSizeStr != "250MB" {
		t.Errorf("expected MinSizeStr '250MB', got %q", loaded.LargeFiles.MinSizeStr)
	}
	if loaded.LargeFiles.MinSize != 250*1024*1024 {
		t.Errorf("expected MinSize 250MB, got %d", loaded.LargeFiles.MinSize)
	}
	if loaded.Scanners.Docker {
		t.Error("expected Docker scanner to be disabled")
	}
	if len(loaded.Exclude) != 1 || loaded.Exclude[0] != "*.tmp" {
		t.Errorf("expected exclude [*.tmp], got %v", loaded.Exclude)
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
	}{
		{"90d", 90 * 24 * time.Hour},
		{"30d", 30 * 24 * time.Hour},
		{"1d", 24 * time.Hour},
		{"", 90 * 24 * time.Hour},    // default
		{"abc", 90 * 24 * time.Hour}, // fallback
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseDuration(tt.input)
			if got != tt.want {
				t.Errorf("ParseDuration(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
