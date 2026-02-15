package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all macbroom configuration.
type Config struct {
	LargeFiles LargeFilesConfig `yaml:"large_files"`
	Exclude    []string         `yaml:"exclude"`
	Scanners   ScannersConfig   `yaml:"scanners"`
	SpaceLens  SpaceLensConfig  `yaml:"spacelens"`
	Schedule   ScheduleConfig   `yaml:"schedule"`
}

// LargeFilesConfig controls the large/old file scanner.
type LargeFilesConfig struct {
	MinSize    int64    `yaml:"-"`
	MinSizeStr string   `yaml:"min_size"`
	MinAge     string   `yaml:"min_age"`
	Paths      []string `yaml:"paths"`
}

// ScannersConfig toggles individual scanners on or off.
type ScannersConfig struct {
	System        bool `yaml:"system"`
	Browser       bool `yaml:"browser"`
	Xcode         bool `yaml:"xcode"`
	LargeFiles    bool `yaml:"large_files"`
	Docker        bool `yaml:"docker"`
	Node          bool `yaml:"node"`
	Homebrew      bool `yaml:"homebrew"`
	IOSSimulators bool `yaml:"ios_simulators"`
	Python        bool `yaml:"python"`
	Rust          bool `yaml:"rust"`
	Go            bool `yaml:"go"`
	JetBrains     bool `yaml:"jetbrains"`
}

// SpaceLensConfig controls the space-lens disk visualizer.
type SpaceLensConfig struct {
	DefaultPath string `yaml:"default_path"`
	Depth       int    `yaml:"depth"`
}

// ScheduleConfig controls automated/scheduled cleaning.
type ScheduleConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Interval string `yaml:"interval"`
	Time     string `yaml:"time"`
	Notify   bool   `yaml:"notify"`
}

// Default returns a Config with all default values populated.
func Default() *Config {
	return &Config{
		LargeFiles: LargeFilesConfig{
			MinSize:    100 * 1024 * 1024,
			MinSizeStr: "100MB",
			MinAge:     "90d",
			Paths:      []string{"~/Downloads", "~/Desktop"},
		},
		Exclude: []string{},
		Scanners: ScannersConfig{
			System:        true,
			Browser:       true,
			Xcode:         true,
			LargeFiles:    true,
			Docker:        true,
			Node:          true,
			Homebrew:      true,
			IOSSimulators: true,
			Python:        true,
			Rust:          true,
			Go:            true,
			JetBrains:     true,
		},
		SpaceLens: SpaceLensConfig{
			DefaultPath: "/",
			Depth:       2,
		},
		Schedule: ScheduleConfig{
			Enabled:  false,
			Interval: "daily",
			Time:     "10:00",
			Notify:   true,
		},
	}
}

// Load loads config from the given path. If path is empty, it uses the
// default location (~/.config/macbroom/config.yaml). If the file does not
// exist, it creates it with default values.
func Load(path string) (*Config, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to determine home directory: %w", err)
		}
		path = filepath.Join(home, ".config", "macbroom", "config.yaml")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		cfg := Default()
		if err := cfg.Save(path); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		return cfg, nil
	}

	return LoadFrom(path)
}

// LoadFrom loads and parses config from the given path. Missing fields
// keep their default values.
func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := Default()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Resolve MinSize from the string representation.
	if cfg.LargeFiles.MinSizeStr != "" {
		size, err := ParseSize(cfg.LargeFiles.MinSizeStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse large_files.min_size %q: %w", cfg.LargeFiles.MinSizeStr, err)
		}
		cfg.LargeFiles.MinSize = size
	}

	return cfg, nil
}

// Save marshals the config to YAML and writes it to the given path,
// creating parent directories as needed.
func (c *Config) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

type sizeSuffix struct {
	suffix string
	mult   int64
}

var sizeSuffixes = []sizeSuffix{
	{"TB", 1024 * 1024 * 1024 * 1024},
	{"GB", 1024 * 1024 * 1024},
	{"MB", 1024 * 1024},
	{"KB", 1024},
}

// ParseSize parses a human-readable size string like "100MB", "1GB",
// "500KB", "2TB", or a plain number (bytes) into int64 bytes.
func ParseSize(s string) (int64, error) {
	if s == "" {
		return 0, fmt.Errorf("empty size string")
	}

	s = strings.TrimSpace(s)
	upper := strings.ToUpper(s)

	for _, ss := range sizeSuffixes {
		if strings.HasSuffix(upper, ss.suffix) {
			numStr := strings.TrimSuffix(upper, ss.suffix)
			if numStr == "" {
				return 0, fmt.Errorf("missing numeric value in %q", s)
			}
			n, err := strconv.ParseInt(numStr, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid size %q: %w", s, err)
			}
			if n < 0 {
				return 0, fmt.Errorf("negative size %q", s)
			}
			return n * ss.mult, nil
		}
	}

	// Plain number (bytes).
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size %q: %w", s, err)
	}
	if n < 0 {
		return 0, fmt.Errorf("negative size %q", s)
	}
	return n, nil
}

// IsExcluded checks if the given path matches any of the configured
// exclude glob patterns. Matching is done against the full path and
// against the base name. Patterns ending in "/**" are treated as
// directory prefix matches.
func (c *Config) IsExcluded(path string) bool {
	for _, pattern := range c.Exclude {
		// Handle "dir/**" as a prefix match.
		if strings.HasSuffix(pattern, "/**") {
			prefix := strings.TrimSuffix(pattern, "/**")
			if strings.HasPrefix(path, prefix+"/") || path == prefix {
				return true
			}
			continue
		}

		// Match against the full path.
		if matched, _ := filepath.Match(pattern, path); matched {
			return true
		}
		// Match against the base name (for patterns like "*.log").
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}
	}
	return false
}

// ParseDuration parses duration strings like "90d", "30d", "7d" into
// time.Duration. Falls back to time.ParseDuration for standard formats.
// Returns 90 days as the default for empty or unparseable strings.
func ParseDuration(s string) time.Duration {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "d") {
		numStr := strings.TrimSuffix(s, "d")
		days, err := strconv.Atoi(numStr)
		if err == nil {
			return time.Duration(days) * 24 * time.Hour
		}
	}

	d, err := time.ParseDuration(s)
	if err != nil {
		return 90 * 24 * time.Hour // fallback default
	}
	return d
}
