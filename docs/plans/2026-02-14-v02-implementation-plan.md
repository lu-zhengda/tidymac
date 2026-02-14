# macbroom v0.2 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Evolve macbroom into a comprehensive macOS maintenance suite with config, more scanners, duplicate detection, scheduled cleaning, and polished TUI.

**Architecture:** YAML config foundation that all scanners read from. Four new scanner implementations following the existing `Scanner` interface. New packages for config, history, scheduling, and duplicate detection. TUI extended with two new menu items (Uninstall, Duplicates) and Space Lens deletion.

**Tech Stack:** Go 1.25, Cobra CLI, Bubbletea TUI, gopkg.in/yaml.v3, crypto/sha256 (stdlib)

---

### Task 1: Config System

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`
- Modify: `go.mod` (add yaml.v3)
- Modify: `internal/cli/root.go` (load config, pass to engine)
- Modify: `internal/scanner/largefiles.go` (read thresholds from config)
- Modify: `internal/scanner/system.go` (read exclusions)

**Step 1: Add yaml.v3 dependency**

```bash
go get gopkg.in/yaml.v3
```

**Step 2: Write config tests**

```go
// internal/config/config_test.go
package config

import (
    "os"
    "path/filepath"
    "testing"
)

func TestDefaultConfig(t *testing.T) {
    cfg := Default()
    if cfg.LargeFiles.MinSize != 100*1024*1024 {
        t.Errorf("expected 100MB default, got %d", cfg.LargeFiles.MinSize)
    }
    if cfg.LargeFiles.MinAge != "90d" {
        t.Errorf("expected 90d default, got %s", cfg.LargeFiles.MinAge)
    }
    if !cfg.Scanners.System {
        t.Error("system scanner should be enabled by default")
    }
}

func TestLoadFromFile(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "config.yaml")
    os.WriteFile(path, []byte(`
large_files:
  min_size: 50MB
  min_age: 30d
  paths:
    - ~/Downloads
scanners:
  docker: false
exclude:
  - "*.important"
`), 0644)

    cfg, err := LoadFrom(path)
    if err != nil {
        t.Fatal(err)
    }
    if cfg.LargeFiles.MinSize != 50*1024*1024 {
        t.Errorf("expected 50MB, got %d", cfg.LargeFiles.MinSize)
    }
    if cfg.Scanners.Docker {
        t.Error("docker should be disabled")
    }
}

func TestLoadCreatesDefault(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "config.yaml")
    cfg, err := Load(path)
    if err != nil {
        t.Fatal(err)
    }
    // Should create file with defaults
    if _, err := os.Stat(path); os.IsNotExist(err) {
        t.Error("config file should have been created")
    }
    if cfg.LargeFiles.MinSize != 100*1024*1024 {
        t.Errorf("expected defaults")
    }
}

func TestParseSize(t *testing.T) {
    tests := []struct {
        input string
        want  int64
    }{
        {"100MB", 100 * 1024 * 1024},
        {"1GB", 1024 * 1024 * 1024},
        {"500KB", 500 * 1024},
        {"1024", 1024},
    }
    for _, tt := range tests {
        got, err := ParseSize(tt.input)
        if err != nil {
            t.Errorf("ParseSize(%q): %v", tt.input, err)
        }
        if got != tt.want {
            t.Errorf("ParseSize(%q) = %d, want %d", tt.input, got, tt.want)
        }
    }
}
```

Run: `go test ./internal/config/ -v`
Expected: FAIL (package doesn't exist)

**Step 3: Implement config package**

```go
// internal/config/config.go
package config

import (
    "fmt"
    "os"
    "path/filepath"
    "regexp"
    "strconv"
    "strings"

    "gopkg.in/yaml.v3"
)

type Config struct {
    LargeFiles LargeFilesConfig `yaml:"large_files"`
    Exclude    []string         `yaml:"exclude"`
    Scanners   ScannersConfig   `yaml:"scanners"`
    SpaceLens  SpaceLensConfig  `yaml:"spacelens"`
    Schedule   ScheduleConfig   `yaml:"schedule"`
}

type LargeFilesConfig struct {
    MinSize    int64    `yaml:"-"`
    MinSizeStr string   `yaml:"min_size"`
    MinAge     string   `yaml:"min_age"`
    Paths      []string `yaml:"paths"`
}

type ScannersConfig struct {
    System        bool `yaml:"system"`
    Browser       bool `yaml:"browser"`
    Xcode         bool `yaml:"xcode"`
    LargeFiles    bool `yaml:"large_files"`
    Docker        bool `yaml:"docker"`
    Node          bool `yaml:"node"`
    Homebrew      bool `yaml:"homebrew"`
    IOSSimulators bool `yaml:"ios_simulators"`
}

type SpaceLensConfig struct {
    DefaultPath string `yaml:"default_path"`
    Depth       int    `yaml:"depth"`
}

type ScheduleConfig struct {
    Enabled  bool   `yaml:"enabled"`
    Interval string `yaml:"interval"`
    Time     string `yaml:"time"`
    Notify   bool   `yaml:"notify"`
}

func Default() *Config {
    return &Config{
        LargeFiles: LargeFilesConfig{
            MinSize:    100 * 1024 * 1024,
            MinSizeStr: "100MB",
            MinAge:     "90d",
            Paths:      []string{"~/Downloads", "~/Desktop"},
        },
        Scanners: ScannersConfig{
            System: true, Browser: true, Xcode: true, LargeFiles: true,
            Docker: true, Node: true, Homebrew: true, IOSSimulators: true,
        },
        SpaceLens: SpaceLensConfig{DefaultPath: "/", Depth: 2},
        Schedule:  ScheduleConfig{Interval: "daily", Time: "10:00", Notify: true},
    }
}

func Load(path string) (*Config, error) {
    if path == "" {
        home, _ := os.UserHomeDir()
        path = filepath.Join(home, ".config", "macbroom", "config.yaml")
    }
    if _, err := os.Stat(path); os.IsNotExist(err) {
        cfg := Default()
        if err := cfg.Save(path); err != nil {
            return cfg, nil // return defaults even if save fails
        }
        return cfg, nil
    }
    return LoadFrom(path)
}

func LoadFrom(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("reading config: %w", err)
    }
    cfg := Default() // start with defaults so missing fields keep defaults
    if err := yaml.Unmarshal(data, cfg); err != nil {
        return nil, fmt.Errorf("parsing config: %w", err)
    }
    if cfg.LargeFiles.MinSizeStr != "" {
        size, err := ParseSize(cfg.LargeFiles.MinSizeStr)
        if err != nil {
            return nil, fmt.Errorf("invalid min_size %q: %w", cfg.LargeFiles.MinSizeStr, err)
        }
        cfg.LargeFiles.MinSize = size
    }
    return cfg, nil
}

func (c *Config) Save(path string) error {
    if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
        return err
    }
    data, err := yaml.Marshal(c)
    if err != nil {
        return err
    }
    return os.WriteFile(path, data, 0644)
}

var sizeRe = regexp.MustCompile(`^(\d+)\s*(KB|MB|GB|TB)?$`)

func ParseSize(s string) (int64, error) {
    s = strings.TrimSpace(strings.ToUpper(s))
    m := sizeRe.FindStringSubmatch(s)
    if m == nil {
        return 0, fmt.Errorf("invalid size format: %s", s)
    }
    n, _ := strconv.ParseInt(m[1], 10, 64)
    switch m[2] {
    case "KB":
        return n * 1024, nil
    case "MB":
        return n * 1024 * 1024, nil
    case "GB":
        return n * 1024 * 1024 * 1024, nil
    case "TB":
        return n * 1024 * 1024 * 1024 * 1024, nil
    default:
        return n, nil
    }
}

func (c *Config) IsExcluded(path string) bool {
    for _, pattern := range c.Exclude {
        if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
            return true
        }
    }
    return false
}
```

Run: `go test ./internal/config/ -v`
Expected: PASS

**Step 4: Wire config into CLI**

Modify `internal/cli/root.go`:
- Add `var cfg *config.Config` package variable
- Load config in `init()` or `PersistentPreRunE`
- Pass config to `buildEngine()` so scanners can use it
- Add `--config` flag for custom config path

Modify `internal/scanner/largefiles.go`:
- Accept `minSize int64` and `minAge time.Duration` in constructor instead of hardcoding
- Update `buildEngine()` call to pass values from config

**Step 5: Run full test suite and commit**

```bash
go test -race ./...
git add -A && git commit -m "feat: add YAML config system with defaults and validation"
```

---

### Task 2: Docker Scanner

**Files:**
- Create: `internal/scanner/docker.go`
- Create: `internal/scanner/docker_test.go`
- Modify: `internal/cli/root.go` (register scanner)

**Step 1: Write tests**

```go
// internal/scanner/docker_test.go
package scanner

import (
    "context"
    "testing"
)

func TestDockerScanner_Name(t *testing.T) {
    s := NewDockerScanner()
    if s.Name() != "Docker" {
        t.Errorf("expected Docker, got %s", s.Name())
    }
}

func TestDockerScanner_SkipsIfNotInstalled(t *testing.T) {
    // If Docker is not running, scanner should return empty, no error
    s := NewDockerScanner()
    targets, err := s.Scan(context.Background())
    if err != nil {
        t.Logf("Docker not available, scanner returned error (acceptable): %v", err)
    }
    // Should not panic regardless
    _ = targets
}
```

**Step 2: Implement scanner**

```go
// internal/scanner/docker.go
package scanner

import (
    "context"
    "encoding/json"
    "os/exec"
    "strings"
)

type DockerScanner struct{}

func NewDockerScanner() *DockerScanner {
    return &DockerScanner{}
}

func (s *DockerScanner) Name() string        { return "Docker" }
func (s *DockerScanner) Description() string { return "Docker images, containers, and build cache" }
func (s *DockerScanner) Risk() RiskLevel     { return Moderate }

func (s *DockerScanner) Scan(ctx context.Context) ([]Target, error) {
    // Check if docker is available
    if _, err := exec.LookPath("docker"); err != nil {
        return nil, nil
    }

    var targets []Target

    // Get dangling images
    out, err := exec.CommandContext(ctx, "docker", "images", "-f", "dangling=true", "--format", "{{.ID}}\t{{.Size}}").Output()
    if err != nil {
        return nil, nil // Docker not running, skip silently
    }
    for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
        if line == "" {
            continue
        }
        parts := strings.SplitN(line, "\t", 2)
        targets = append(targets, Target{
            Path:        "docker image " + parts[0],
            Description: "Dangling image",
            Category:    "Docker",
            Risk:        Moderate,
        })
    }

    // Get build cache size
    out, err = exec.CommandContext(ctx, "docker", "system", "df", "--format", "{{json .}}").Output()
    if err == nil {
        for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
            var df struct {
                Type        string `json:"Type"`
                Reclaimable string `json:"Reclaimable"`
                Size        string `json:"Size"`
            }
            if json.Unmarshal([]byte(line), &df) == nil && df.Type == "Build Cache" {
                targets = append(targets, Target{
                    Path:        "docker build cache",
                    Description: "Build cache (" + df.Size + ", " + df.Reclaimable + " reclaimable)",
                    Category:    "Docker",
                    Risk:        Safe,
                })
            }
        }
    }

    return targets, nil
}
```

**Step 3: Register in buildEngine and commit**

```bash
go test -race ./internal/scanner/ -v
git add -A && git commit -m "feat: add Docker scanner"
```

---

### Task 3: Node.js Scanner

**Files:**
- Create: `internal/scanner/node.go`
- Create: `internal/scanner/node_test.go`
- Modify: `internal/cli/root.go` (register)

**Step 1: Write tests**

```go
// internal/scanner/node_test.go
package scanner

import (
    "context"
    "os"
    "path/filepath"
    "testing"
    "time"
)

func TestNodeScanner_FindsNpmCache(t *testing.T) {
    dir := t.TempDir()
    cacheDir := filepath.Join(dir, ".npm", "_cacache")
    os.MkdirAll(cacheDir, 0755)
    os.WriteFile(filepath.Join(cacheDir, "data.bin"), make([]byte, 1024), 0644)

    s := NewNodeScanner(dir, nil, 90*24*time.Hour)
    targets, err := s.Scan(context.Background())
    if err != nil {
        t.Fatal(err)
    }
    if len(targets) == 0 {
        t.Error("expected to find npm cache")
    }
}

func TestNodeScanner_FindsStaleNodeModules(t *testing.T) {
    dir := t.TempDir()
    projDir := filepath.Join(dir, "old-project", "node_modules")
    os.MkdirAll(projDir, 0755)
    os.WriteFile(filepath.Join(projDir, "mod.js"), make([]byte, 100), 0644)
    // Set mod time to 6 months ago
    old := time.Now().Add(-180 * 24 * time.Hour)
    os.Chtimes(projDir, old, old)

    s := NewNodeScanner("", []string{dir}, 90*24*time.Hour)
    targets, err := s.Scan(context.Background())
    if err != nil {
        t.Fatal(err)
    }
    found := false
    for _, t := range targets {
        if t.IsDir && filepath.Base(filepath.Dir(t.Path)) == "old-project" {
            found = true
        }
    }
    if !found {
        t.Error("expected to find stale node_modules")
    }
}
```

**Step 2: Implement**

The scanner searches `~/.npm/_cacache` for npm cache and walks configured directories looking for `node_modules/` folders older than the configured threshold.

Constructor: `NewNodeScanner(home string, searchPaths []string, maxAge time.Duration)`

**Step 3: Test and commit**

```bash
go test -race ./internal/scanner/ -v -run TestNode
git add -A && git commit -m "feat: add Node.js scanner for npm cache and stale node_modules"
```

---

### Task 4: Homebrew Scanner

**Files:**
- Create: `internal/scanner/homebrew.go`
- Create: `internal/scanner/homebrew_test.go`
- Modify: `internal/cli/root.go`

**Implementation:**
- Run `brew --cache` to get cache directory path
- Walk cache directory for `.tar.gz`, `.bottle.tar.gz`, `.dmg` files
- Report each as a Target with file size
- Skip gracefully if `brew` not installed (`exec.LookPath`)

Constructor: `NewHomebrewScanner()`

**Step 1: Write test, Step 2: Implement, Step 3: Commit**

```bash
go test -race ./internal/scanner/ -v -run TestHomebrew
git add -A && git commit -m "feat: add Homebrew cache scanner"
```

---

### Task 5: iOS Simulator Scanner

**Files:**
- Create: `internal/scanner/simulator.go`
- Create: `internal/scanner/simulator_test.go`
- Modify: `internal/cli/root.go`

**Implementation:**
- Scan `~/Library/Developer/CoreSimulator/Devices/` for device data
- Scan `~/Library/Developer/CoreSimulator/Caches/`
- Run `xcrun simctl list devices unavailable -j` to find unavailable runtimes
- Skip if Xcode not installed
- Risk: Moderate

Constructor: `NewSimulatorScanner(libraryPath string)`

**Step 1: Write test, Step 2: Implement, Step 3: Commit**

```bash
go test -race ./internal/scanner/ -v -run TestSimulator
git add -A && git commit -m "feat: add iOS Simulator scanner"
```

---

### Task 6: Register All New Scanners

**Files:**
- Modify: `internal/cli/root.go:buildEngine()` — register Docker, Node, Homebrew, Simulator scanners
- Modify: `internal/cli/scan.go` — add `--docker`, `--node`, `--homebrew`, `--simulator` flags
- Modify: `internal/cli/clean.go` — add matching flags
- Modify: `internal/cli/root.go:selectedCategories()` — handle new categories

**Step 1: Update buildEngine to conditionally register based on config**

```go
func buildEngine() *engine.Engine {
    e := engine.New()
    cfg := appConfig // loaded in PersistentPreRunE

    if cfg.Scanners.System {
        e.Register(scanner.NewSystemScanner(""))
    }
    if cfg.Scanners.Browser {
        e.Register(scanner.NewBrowserScanner("", ""))
    }
    if cfg.Scanners.Xcode {
        e.Register(scanner.NewXcodeScanner(""))
    }
    if cfg.Scanners.LargeFiles {
        // Use config values for thresholds
        e.Register(scanner.NewLargeFileScanner(
            expandPaths(cfg.LargeFiles.Paths),
            cfg.LargeFiles.MinSize,
            parseDuration(cfg.LargeFiles.MinAge),
        ))
    }
    if cfg.Scanners.Docker {
        e.Register(scanner.NewDockerScanner())
    }
    if cfg.Scanners.Node {
        home := utils.HomeDir()
        e.Register(scanner.NewNodeScanner(home, expandPaths(cfg.LargeFiles.Paths), parseDuration(cfg.LargeFiles.MinAge)))
    }
    if cfg.Scanners.Homebrew {
        e.Register(scanner.NewHomebrewScanner())
    }
    if cfg.Scanners.IOSSimulators {
        e.Register(scanner.NewSimulatorScanner(""))
    }
    return e
}
```

**Step 2: Test and commit**

```bash
go build ./cmd/macbroom && go test -race ./...
git add -A && git commit -m "feat: register new scanners with config-driven toggles"
```

---

### Task 7: Dry-run Mode

**Files:**
- Modify: `internal/cli/clean.go` — add `--dry-run` flag, skip deletion when set
- Modify: `internal/cli/uninstall.go` — add `--dry-run` flag
- Modify: `internal/cli/output.go` — add `[DRY RUN]` banner

**Implementation:**
- Add `var dryRun bool` flag to both commands
- When `--dry-run` is set, scan and display results but skip the deletion loop
- Print `[DRY RUN] No files were deleted.` at the end

**Step 1: Implement, test manually, commit**

```bash
go build ./cmd/macbroom && ./bin/macbroom clean --dry-run
git add -A && git commit -m "feat: add --dry-run flag to clean and uninstall commands"
```

---

### Task 8: Scan History & Stats

**Files:**
- Create: `internal/history/history.go`
- Create: `internal/history/history_test.go`
- Create: `internal/cli/stats.go`
- Modify: `internal/cli/root.go` (register stats command)
- Modify: `internal/cli/clean.go` (record after clean)
- Modify: `internal/tui/app.go` (record after TUI clean)

**Step 1: Write tests**

```go
// internal/history/history_test.go
package history

import (
    "path/filepath"
    "testing"
    "time"
)

func TestRecordAndLoad(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "history.json")

    h := New(path)
    h.Record(Entry{
        Timestamp:  time.Now(),
        Category:   "System Junk",
        Items:      5,
        BytesFreed: 1024 * 1024,
        Method:     "trash",
    })

    entries, err := h.Load()
    if err != nil {
        t.Fatal(err)
    }
    if len(entries) != 1 {
        t.Fatalf("expected 1 entry, got %d", len(entries))
    }
    if entries[0].Category != "System Junk" {
        t.Errorf("expected System Junk, got %s", entries[0].Category)
    }
}

func TestStats(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "history.json")
    h := New(path)

    h.Record(Entry{Timestamp: time.Now(), Category: "System Junk", Items: 5, BytesFreed: 1000, Method: "trash"})
    h.Record(Entry{Timestamp: time.Now(), Category: "Browser Cache", Items: 3, BytesFreed: 2000, Method: "trash"})

    stats := h.Stats()
    if stats.TotalFreed != 3000 {
        t.Errorf("expected 3000 total, got %d", stats.TotalFreed)
    }
    if stats.TotalCleanups != 2 {
        t.Errorf("expected 2 cleanups, got %d", stats.TotalCleanups)
    }
}
```

**Step 2: Implement history package**

```go
// internal/history/history.go
package history

import (
    "encoding/json"
    "os"
    "path/filepath"
    "sort"
    "time"
)

type Entry struct {
    Timestamp  time.Time `json:"timestamp"`
    Category   string    `json:"category"`
    Items      int       `json:"items"`
    BytesFreed int64     `json:"bytes_freed"`
    Method     string    `json:"method"` // "trash" or "permanent"
}

type Stats struct {
    TotalFreed    int64
    TotalCleanups int
    ByCategory    map[string]CategoryStats
    Recent        []Entry
}

type CategoryStats struct {
    BytesFreed int64
    Cleanups   int
}

type History struct {
    path string
}

func New(path string) *History {
    return &History{path: path}
}

func DefaultPath() string {
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".local", "share", "macbroom", "history.json")
}

func (h *History) Record(e Entry) error {
    entries, _ := h.Load() // ignore error, start fresh if corrupt
    entries = append(entries, e)
    os.MkdirAll(filepath.Dir(h.path), 0755)
    data, err := json.Marshal(entries)
    if err != nil {
        return err
    }
    return os.WriteFile(h.path, data, 0644)
}

func (h *History) Load() ([]Entry, error) {
    data, err := os.ReadFile(h.path)
    if err != nil {
        return nil, nil
    }
    var entries []Entry
    return entries, json.Unmarshal(data, &entries)
}

func (h *History) Stats() Stats {
    entries, _ := h.Load()
    s := Stats{ByCategory: make(map[string]CategoryStats)}
    for _, e := range entries {
        s.TotalFreed += e.BytesFreed
        s.TotalCleanups++
        cs := s.ByCategory[e.Category]
        cs.BytesFreed += e.BytesFreed
        cs.Cleanups++
        s.ByCategory[e.Category] = cs
    }
    sort.Slice(entries, func(i, j int) bool {
        return entries[i].Timestamp.After(entries[j].Timestamp)
    })
    if len(entries) > 5 {
        entries = entries[:5]
    }
    s.Recent = entries
    return s
}
```

**Step 3: Create stats command**

```go
// internal/cli/stats.go
// Cobra command that loads history and prints formatted stats
```

**Step 4: Wire recording into clean operations**

Add `history.Record()` call after successful deletion in both `internal/cli/clean.go` and `internal/tui/app.go` (in `cleanDoneMsg` handler).

**Step 5: Test and commit**

```bash
go test -race ./internal/history/ -v
go build ./cmd/macbroom && ./bin/macbroom stats
git add -A && git commit -m "feat: add scan history tracking and stats command"
```

---

### Task 9: Space Lens Improvements

**Files:**
- Modify: `internal/tui/app.go` — scrolling, deletion, percentage, total size
- Modify: `internal/tui/spacelens.go` — match improvements for standalone mode

**Step 1: Add deletion support**

In `updateSpaceLens()`, add `"d"` key handler:
```go
case "d":
    if m.slCursor < len(m.slNodes) {
        m.slDeleteTarget = &m.slNodes[m.slCursor]
        m.currentView = viewSpaceLensConfirm
    }
```

Add new `viewSpaceLensConfirm` view state that shows "Delete <name> (<size>)? y/n".
On `y`, call `trash.MoveToTrash()`, then re-scan the current directory.

**Step 2: Add scrolling**

Replace the hardcoded `visible[:30]` cap with the same `scrollOffset` + `visibleItemCount()` pattern used in `viewCategory`.

**Step 3: Add percentage display**

Calculate `pct := float64(node.Size) / float64(totalDirSize) * 100` and show it:
```
> D  Applications              45.2 GB  38%  |##########.............|
```

**Step 4: Show total in header**

Change header from just path to: `/ (467 GB)` — sum all nodes' sizes.

**Step 5: Test and commit**

```bash
go build ./cmd/macbroom
# Manual test: navigate Space Lens, press d on a test file, verify deletion
git add -A && git commit -m "feat: Space Lens deletion, scrolling, percentages"
```

---

### Task 10: Duplicate File Finder

**Files:**
- Create: `internal/dupes/dupes.go`
- Create: `internal/dupes/dupes_test.go`
- Create: `internal/cli/dupes.go`
- Modify: `internal/cli/root.go` (register command)
- Modify: `internal/tui/app.go` (add Duplicates menu item)

**Step 1: Write tests**

```go
// internal/dupes/dupes_test.go
package dupes

import (
    "context"
    "os"
    "path/filepath"
    "testing"
)

func TestFindDuplicates(t *testing.T) {
    dir := t.TempDir()
    content := []byte("hello world duplicate content here")
    os.WriteFile(filepath.Join(dir, "file1.txt"), content, 0644)
    os.WriteFile(filepath.Join(dir, "file2.txt"), content, 0644)
    os.WriteFile(filepath.Join(dir, "unique.txt"), []byte("different"), 0644)

    groups, err := Find(context.Background(), []string{dir}, 1)
    if err != nil {
        t.Fatal(err)
    }
    if len(groups) != 1 {
        t.Fatalf("expected 1 duplicate group, got %d", len(groups))
    }
    if len(groups[0].Files) != 2 {
        t.Errorf("expected 2 files in group, got %d", len(groups[0].Files))
    }
}

func TestNoDuplicates(t *testing.T) {
    dir := t.TempDir()
    os.WriteFile(filepath.Join(dir, "a.txt"), []byte("aaa"), 0644)
    os.WriteFile(filepath.Join(dir, "b.txt"), []byte("bbb"), 0644)

    groups, err := Find(context.Background(), []string{dir}, 1)
    if err != nil {
        t.Fatal(err)
    }
    if len(groups) != 0 {
        t.Errorf("expected 0 groups, got %d", len(groups))
    }
}

func TestSkipsSmallFiles(t *testing.T) {
    dir := t.TempDir()
    // Files smaller than minSize should be skipped
    os.WriteFile(filepath.Join(dir, "tiny1.txt"), []byte("x"), 0644)
    os.WriteFile(filepath.Join(dir, "tiny2.txt"), []byte("x"), 0644)

    groups, err := Find(context.Background(), []string{dir}, 1024) // 1KB min
    if err != nil {
        t.Fatal(err)
    }
    if len(groups) != 0 {
        t.Errorf("expected 0 groups for tiny files, got %d", len(groups))
    }
}
```

**Step 2: Implement three-pass algorithm**

```go
// internal/dupes/dupes.go
package dupes

import (
    "context"
    "crypto/sha256"
    "io"
    "os"
    "path/filepath"
)

type Group struct {
    Size  int64
    Hash  string
    Files []string
}

type ProgressFunc func(path string)

func Find(ctx context.Context, dirs []string, minSize int64) ([]Group, error) {
    return FindWithProgress(ctx, dirs, minSize, nil)
}

func FindWithProgress(ctx context.Context, dirs []string, minSize int64, onProgress ProgressFunc) ([]Group, error) {
    // Pass 1: group by size
    sizeMap := make(map[int64][]string)
    for _, dir := range dirs {
        filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
            if err != nil || info.IsDir() {
                return nil
            }
            select {
            case <-ctx.Done():
                return ctx.Err()
            default:
            }
            if info.Size() < minSize {
                return nil
            }
            if onProgress != nil {
                onProgress(path)
            }
            sizeMap[info.Size()] = append(sizeMap[info.Size()], path)
            return nil
        })
    }

    // Pass 2: partial hash (first 4KB) for same-size files
    type hashKey struct {
        size int64
        hash string
    }
    partialMap := make(map[hashKey][]string)
    for size, files := range sizeMap {
        if len(files) < 2 {
            continue
        }
        for _, f := range files {
            h, err := hashFile(f, 4096)
            if err != nil {
                continue
            }
            key := hashKey{size, h}
            partialMap[key] = append(partialMap[key], f)
        }
    }

    // Pass 3: full hash only for partial matches
    fullMap := make(map[string]*Group)
    for key, files := range partialMap {
        if len(files) < 2 {
            continue
        }
        for _, f := range files {
            h, err := hashFile(f, 0) // 0 = full file
            if err != nil {
                continue
            }
            if g, ok := fullMap[h]; ok {
                g.Files = append(g.Files, f)
            } else {
                fullMap[h] = &Group{Size: key.size, Hash: h, Files: []string{f}}
            }
        }
    }

    var groups []Group
    for _, g := range fullMap {
        if len(g.Files) >= 2 {
            groups = append(groups, *g)
        }
    }
    return groups, nil
}

func hashFile(path string, limit int64) (string, error) {
    f, err := os.Open(path)
    if err != nil {
        return "", err
    }
    defer f.Close()
    h := sha256.New()
    if limit > 0 {
        io.CopyN(h, f, limit)
    } else {
        io.Copy(h, f)
    }
    return fmt.Sprintf("%x", h.Sum(nil)), nil
}
```

Note: add `"fmt"` to imports for `fmt.Sprintf`.

**Step 3: CLI command + TUI menu item**

Create `internal/cli/dupes.go` with Cobra command. Add "Duplicates" to TUI main menu with a new `viewDupes` and `viewDupesResult` view state.

**Step 4: Test and commit**

```bash
go test -race ./internal/dupes/ -v
git add -A && git commit -m "feat: add duplicate file finder with three-pass algorithm"
```

---

### Task 11: Uninstaller TUI + Orphan Scan

**Files:**
- Modify: `internal/tui/app.go` — add Uninstall menu item, text input view, results view
- Modify: `internal/scanner/apps.go` — add more search directories, orphan scan
- Modify: `internal/cli/scan.go` — add `--orphans` flag

**Step 1: Extend AppScanner search locations**

Add to `FindRelatedFiles()`:
- `~/Library/LaunchAgents/`
- `~/Library/Application Scripts/`
- `~/Library/Group Containers/`
- `~/Library/Cookies/`

**Step 2: Add orphan detection**

New method `FindOrphans() ([]Target, error)`:
- Read all `.plist` files in `~/Library/Preferences/`
- Extract bundle identifier from filename (e.g., `com.example.app.plist`)
- Check if matching `.app` exists in `/Applications/`
- Report orphaned prefs + associated caches/support dirs

**Step 3: Add TUI uninstall flow**

New view states: `viewUninstallInput`, `viewUninstallResults`, `viewUninstallConfirm`
- Text input (use bubbletea textinput bubble) for app name
- Show found files with checkboxes
- Confirm and delete

**Step 4: Test and commit**

```bash
go test -race ./internal/scanner/ -v -run TestApp
git add -A && git commit -m "feat: uninstaller TUI and orphan scan"
```

---

### Task 12: Scheduled Cleaning

**Files:**
- Create: `internal/schedule/schedule.go`
- Create: `internal/schedule/schedule_test.go`
- Create: `internal/cli/schedule.go`
- Modify: `internal/cli/root.go` (register command)
- Modify: `internal/cli/clean.go` (add `--quiet` flag)

**Step 1: Write tests**

```go
// internal/schedule/schedule_test.go
package schedule

import (
    "os"
    "path/filepath"
    "testing"
)

func TestGeneratePlist(t *testing.T) {
    plist := GeneratePlist("10:00", "daily")
    if plist == "" {
        t.Error("expected non-empty plist")
    }
    if !strings.Contains(plist, "com.macbroom.cleanup") {
        t.Error("expected bundle identifier")
    }
    if !strings.Contains(plist, "10") {
        t.Error("expected hour 10")
    }
}

func TestInstallUninstall(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "com.macbroom.cleanup.plist")

    err := Install(path, "10:00", "daily")
    if err != nil {
        t.Fatal(err)
    }
    if _, err := os.Stat(path); os.IsNotExist(err) {
        t.Error("plist should exist")
    }

    err = Uninstall(path)
    if err != nil {
        t.Fatal(err)
    }
    if _, err := os.Stat(path); !os.IsNotExist(err) {
        t.Error("plist should be removed")
    }
}
```

**Step 2: Implement schedule package**

```go
// internal/schedule/schedule.go
package schedule

// GeneratePlist(time, interval string) string — generates LaunchAgent plist XML
// Install(path, time, interval string) error — writes plist, runs launchctl load
// Uninstall(path string) error — runs launchctl unload, removes plist
// Status(path string) (enabled bool, err error) — checks if plist exists
// Note: Install/Uninstall use launchctl bootstrap/bootout (modern API)
// Notification: add osascript call to clean command when --quiet is set
```

**Step 3: Create CLI commands**

```go
// internal/cli/schedule.go
// macbroom schedule enable|disable|status
// Uses cobra sub-subcommands
```

**Step 4: Add --quiet flag to clean**

When `--quiet` is set:
- Suppress all stdout output
- Record to history as normal
- If `schedule.notify` is true in config, send macOS notification via osascript

**Step 5: Test and commit**

```bash
go test -race ./internal/schedule/ -v
git add -A && git commit -m "feat: add scheduled cleaning with LaunchAgent"
```

---

### Task 13: Shell Completions

**Files:**
- Modify: `internal/cli/root.go` — add `--generate-completion` flag

**Implementation:**

```go
// In init()
rootCmd.Flags().String("generate-completion", "", "Generate shell completion (bash, zsh, fish)")
rootCmd.Flags().MarkHidden("generate-completion")

// In RunE, check for flag before launching TUI
if shell, _ := cmd.Flags().GetString("generate-completion"); shell != "" {
    switch shell {
    case "bash":
        return rootCmd.GenBashCompletion(os.Stdout)
    case "zsh":
        return rootCmd.GenZshCompletion(os.Stdout)
    case "fish":
        return rootCmd.GenFishCompletion(os.Stdout, true)
    default:
        return fmt.Errorf("unsupported shell: %s (use bash, zsh, or fish)", shell)
    }
}
```

**Commit:**

```bash
go build ./cmd/macbroom && ./bin/macbroom --generate-completion zsh | head -5
git add -A && git commit -m "feat: add hidden --generate-completion flag for shell completions"
```

---

### Task 14: Man Page

**Files:**
- Create: `docs/man/` directory
- Modify: `Makefile` — add `man` target and install man page
- Modify: `.goreleaser.yaml` — include man page in archives

**Implementation:**

Add a `//go:generate` directive or a Makefile target:
```makefile
man:
	go run ./cmd/gendocs  # or use cobra/doc directly
```

Create a small `cmd/gendocs/main.go` that imports cobra/doc and generates `macbroom.1`:
```go
package main

import (
    "github.com/spf13/cobra/doc"
    "github.com/lu-zhengda/macbroom/internal/cli"
)

func main() {
    header := &doc.GenManHeader{Title: "MACBROOM", Section: "1"}
    doc.GenManTree(cli.RootCmd(), header, "./docs/man/")
}
```

Note: This requires exporting `RootCmd()` from the cli package.

**Commit:**

```bash
make man && man ./docs/man/macbroom.1
git add -A && git commit -m "feat: add man page generation"
```

---

### Task 15: Final Integration & Release

**Files:**
- Modify: `internal/tui/app.go` — ensure 5 menu items render correctly
- Modify: `README.md` — document all new features, commands, config
- Modify: `.goreleaser.yaml` — include man page in archives

**Steps:**

1. Run full test suite: `go test -race -cover ./...`
2. Build and manually test each feature
3. Update README with new features
4. Commit everything
5. Create PR, merge, tag v0.2.0, release

```bash
go test -race -cover ./...
go build ./cmd/macbroom && ./bin/macbroom --version
git add -A && git commit -m "chore: finalize v0.2.0 with all new features"
```
