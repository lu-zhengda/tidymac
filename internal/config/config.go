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
	DevTools   DevToolsConfig   `yaml:"dev_tools"`
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

// DevToolsConfig controls search paths and staleness for dev-tool scanners
// (Node.js, Python, Rust). These scanners walk directories looking for
// stale build artifacts (node_modules, virtualenvs, target/).
type DevToolsConfig struct {
	SearchPaths []string `yaml:"search_paths"`
	MinAge      string   `yaml:"min_age"`
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
	Maven         bool `yaml:"maven"`
	Gradle        bool `yaml:"gradle"`
	Ruby          bool `yaml:"ruby"`
}

// SpaceLensConfig controls the space-lens disk visualizer.
type SpaceLensConfig struct {
	DefaultPath string `yaml:"default_path"`
	Depth       int    `yaml:"depth"`
}

// ScheduleConfig controls automated/scheduled cleaning.
type ScheduleConfig struct {
	Enabled    bool     `yaml:"enabled"`
	Interval   string   `yaml:"interval"`
	Time       string   `yaml:"time"`
	Notify     bool     `yaml:"notify"`
	Categories []string `yaml:"categories"`
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
		DevTools: DevToolsConfig{
			SearchPaths: []string{"~/Documents", "~/Projects", "~/src", "~/code", "~/Developer"},
			MinAge:      "30d",
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
			Maven:         true,
			Gradle:        true,
			Ruby:          true,
		},
		SpaceLens: SpaceLensConfig{
			DefaultPath: "/",
			Depth:       2,
		},
		Schedule: ScheduleConfig{
			Enabled:    false,
			Interval:   "daily",
			Time:       "10:00",
			Notify:     true,
			Categories: []string{},
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
	{"T", 1024 * 1024 * 1024 * 1024},
	{"G", 1024 * 1024 * 1024},
	{"M", 1024 * 1024},
	{"K", 1024},
}

// ParseSize parses a human-readable size string like "100MB", "1GB",
// "500KB", "2TB", "100M", "1G", or a plain number (bytes) into int64 bytes.
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
	home, _ := os.UserHomeDir()
	for _, pattern := range c.Exclude {
		p := pattern
		if home != "" && strings.HasPrefix(p, "~/") {
			p = home + p[1:] // ~/foo -> /Users/x/foo
		} else if p == "~" && home != "" {
			p = home
		}

		// Handle "dir/**" as a prefix match.
		if strings.HasSuffix(p, "/**") {
			prefix := strings.TrimSuffix(p, "/**")
			if strings.HasPrefix(path, prefix+"/") || path == prefix {
				return true
			}
			continue
		}

		// Match against the full path.
		if matched, _ := filepath.Match(p, path); matched {
			return true
		}
		// Match against the base name (for patterns like "*.log").
		if matched, _ := filepath.Match(p, filepath.Base(path)); matched {
			return true
		}
	}
	return false
}

// Warning represents a non-fatal configuration issue.
type Warning struct {
	Field      string
	Message    string
	Suggestion string
}

// validScheduleCategories lists category names accepted in schedule.categories.
var validScheduleCategories = map[string]bool{
	"system": true, "browser": true, "xcode": true, "large": true,
	"docker": true, "node": true, "homebrew": true, "simulator": true,
	"python": true, "rust": true, "go": true, "jetbrains": true,
	"maven": true, "gradle": true, "ruby": true,
	"dev": true, "caches": true, "all": true,
}

// knownTopLevelKeys lists the accepted top-level YAML keys.
var knownTopLevelKeys = map[string]bool{
	"large_files": true, "dev_tools": true, "exclude": true,
	"scanners": true, "spacelens": true, "schedule": true,
}

// knownScannerKeys lists the accepted keys under the "scanners" map.
var knownScannerKeys = map[string]bool{
	"system": true, "browser": true, "xcode": true, "large_files": true,
	"docker": true, "node": true, "homebrew": true, "ios_simulators": true,
	"python": true, "rust": true, "go": true, "jetbrains": true,
	"maven": true, "gradle": true, "ruby": true,
}

// Validate checks the config for common issues and returns warnings.
func (c *Config) Validate() []Warning {
	var warnings []Warning

	// Validate exclude patterns.
	for _, pattern := range c.Exclude {
		// Skip "/**" suffix patterns — they use prefix matching, not filepath.Match.
		if strings.HasSuffix(pattern, "/**") {
			continue
		}
		// Expand ~ for validation.
		p := pattern
		if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(p, "~/") {
			p = home + p[1:]
		}
		if _, err := filepath.Match(p, "test"); err != nil {
			warnings = append(warnings, Warning{
				Field:      "exclude",
				Message:    fmt.Sprintf("invalid exclude pattern %q: %v", pattern, err),
				Suggestion: "Check glob syntax; avoid unmatched brackets",
			})
		}
	}

	// Validate large_files.paths — check for non-existent paths.
	for _, p := range c.LargeFiles.Paths {
		expanded := p
		if home, err := os.UserHomeDir(); err == nil {
			if strings.HasPrefix(expanded, "~/") {
				expanded = filepath.Join(home, expanded[2:])
			} else if expanded == "~" {
				expanded = home
			}
		}
		if _, err := os.Stat(expanded); err != nil {
			warnings = append(warnings, Warning{
				Field:      "large_files.paths",
				Message:    fmt.Sprintf("path %q does not exist", p),
				Suggestion: "Remove or correct the path",
			})
		}
	}

	// Validate dev_tools.search_paths — check for non-existent paths.
	for _, p := range c.DevTools.SearchPaths {
		expanded := p
		if home, err := os.UserHomeDir(); err == nil {
			if strings.HasPrefix(expanded, "~/") {
				expanded = filepath.Join(home, expanded[2:])
			} else if expanded == "~" {
				expanded = home
			}
		}
		if _, err := os.Stat(expanded); err != nil {
			warnings = append(warnings, Warning{
				Field:      "dev_tools.search_paths",
				Message:    fmt.Sprintf("path %q does not exist", p),
				Suggestion: "Remove or correct the path",
			})
		}
	}

	// Validate schedule.time.
	if c.Schedule.Time != "" {
		parts := strings.SplitN(c.Schedule.Time, ":", 2)
		valid := true
		if len(parts) != 2 {
			valid = false
		} else {
			hour, err := strconv.Atoi(parts[0])
			if err != nil || hour < 0 || hour > 23 {
				valid = false
			}
			minute, err := strconv.Atoi(parts[1])
			if err != nil || minute < 0 || minute > 59 {
				valid = false
			}
		}
		if !valid {
			warnings = append(warnings, Warning{
				Field:      "schedule.time",
				Message:    fmt.Sprintf("invalid schedule time %q: expected HH:MM (0-23:0-59)", c.Schedule.Time),
				Suggestion: "Use format HH:MM, e.g. \"10:00\" or \"14:30\"",
			})
		}
	}

	// Validate schedule.interval.
	if c.Schedule.Interval != "" {
		lower := strings.ToLower(c.Schedule.Interval)
		if lower != "daily" && lower != "weekly" {
			warnings = append(warnings, Warning{
				Field:      "schedule.interval",
				Message:    fmt.Sprintf("invalid schedule interval %q", c.Schedule.Interval),
				Suggestion: "Use \"daily\" or \"weekly\"",
			})
		}
	}

	// Validate schedule.categories.
	for _, cat := range c.Schedule.Categories {
		if !validScheduleCategories[cat] {
			warnings = append(warnings, Warning{
				Field:      "schedule.categories",
				Message:    fmt.Sprintf("unknown schedule category %q", cat),
				Suggestion: "Valid categories: system, browser, xcode, large, docker, node, homebrew, simulator, python, rust, go, jetbrains, maven, gradle, ruby, dev, caches, all",
			})
		}
	}

	return warnings
}

// LoadAndValidate unmarshals YAML data into a Config, detects unknown keys,
// and runs structural validation. It returns the config and any warnings.
func LoadAndValidate(data []byte) (*Config, []Warning) {
	cfg := Default()
	var warnings []Warning

	if err := yaml.Unmarshal(data, cfg); err != nil {
		warnings = append(warnings, Warning{
			Field:   "",
			Message: fmt.Sprintf("failed to parse config: %v", err),
		})
		return cfg, warnings
	}

	// Resolve MinSize from the string representation.
	if cfg.LargeFiles.MinSizeStr != "" {
		size, err := ParseSize(cfg.LargeFiles.MinSizeStr)
		if err != nil {
			warnings = append(warnings, Warning{
				Field:      "large_files.min_size",
				Message:    fmt.Sprintf("invalid min_size %q: %v", cfg.LargeFiles.MinSizeStr, err),
				Suggestion: "Use a size string like \"100MB\" or \"1GB\"",
			})
		} else {
			cfg.LargeFiles.MinSize = size
		}
	}

	// Detect unknown top-level keys.
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err == nil {
		for key := range raw {
			if !knownTopLevelKeys[key] {
				warnings = append(warnings, Warning{
					Field:      key,
					Message:    fmt.Sprintf("unknown config key %q", key),
					Suggestion: "Check spelling; valid keys: large_files, dev_tools, exclude, scanners, spacelens, schedule",
				})
			}
		}

		// Detect unknown scanner keys.
		if scannersRaw, ok := raw["scanners"]; ok {
			if scannersMap, ok := scannersRaw.(map[string]interface{}); ok {
				for key := range scannersMap {
					if !knownScannerKeys[key] {
						warnings = append(warnings, Warning{
							Field:      "scanners." + key,
							Message:    fmt.Sprintf("unknown scanner %q", key),
							Suggestion: "Valid scanners: system, browser, xcode, large_files, docker, node, homebrew, ios_simulators, python, rust, go, jetbrains, maven, gradle, ruby",
						})
					}
				}
			}
		}
	}

	// Run structural validation.
	warnings = append(warnings, cfg.Validate()...)

	return cfg, warnings
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
