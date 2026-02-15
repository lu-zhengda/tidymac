# macbroom

A lightweight macOS cleanup tool for the terminal.

Scan and remove system junk, browser caches, Xcode artifacts, Docker waste, Node.js caches, Python/Rust/Go build artifacts, JetBrains IDE caches, and more — from the command line or an interactive TUI.

Scan and clean output is colored with category-specific colors, risk-level highlighting, and results sorted by size (largest first). Includes scan diff tracking, risk summary breakdown, JSON output, and config validation.

## Install

```bash
brew install lu-zhengda/tap/macbroom
```

Or with Go:

```bash
go install github.com/lu-zhengda/macbroom/cmd/macbroom@latest
```

Or from source:

```bash
git clone https://github.com/lu-zhengda/macbroom.git
cd macbroom
make install
```

## Usage

Launch the interactive TUI:

```bash
macbroom
```

Or use CLI subcommands:

```bash
# Scan for reclaimable space
macbroom scan
macbroom scan --system --browser
macbroom scan --docker --node --python --rust --go

# Clean junk files (moves to Trash)
macbroom clean
macbroom clean --xcode
macbroom clean --permanent    # permanent delete (requires typing "yes")
macbroom clean --dry-run      # preview what would be deleted

# Uninstall an app and all its related files
macbroom uninstall "Some App"

# Find and remove duplicate files
macbroom dupes
macbroom dupes --min-size 10MB
macbroom dupes --dry-run

# View cleanup history and statistics
macbroom stats

# Visualize disk usage
macbroom spacelens              # whole system
macbroom spacelens ~/Downloads  # specific path
macbroom spacelens -i           # interactive TUI mode

# Run maintenance tasks (DNS flush, Spotlight reindex, etc.)
macbroom maintain

# Schedule automatic cleaning
macbroom schedule enable
macbroom schedule disable
macbroom schedule status

# JSON output for scripting and automation
macbroom scan --json
macbroom clean --system --json
macbroom dupes --json
macbroom stats --json

# Validate your config file
macbroom config validate

# Use scan profiles
macbroom scan --dev               # all dev-tool caches
macbroom scan --caches            # system, browser, homebrew
macbroom scan --all               # everything
macbroom clean --dev --docker     # dev tools + Docker

# Exclude specific paths from scan or clean
macbroom scan --exclude "~/Projects/**" --exclude "*.iso"
macbroom clean --system --exclude "~/Library/Caches/com.important.app"
```

Scan output includes a risk summary breakdown and diff indicators when a previous scan exists:

```
System Junk (8.2 GB, 130 items)  +1.2 GB
Homebrew (2.1 GB, 117 items)  -200.0 MB
Browser Cache (1.8 GB, 3 items)  (new)

Total reclaimable: 12.1 GB
Safe: 4.0 GB (33%)  Moderate: 7.6 GB (63%)  Risky: 500.0 KB (4%)
+2.8 GB since Feb 13
```

### Flags

| Flag | Scope | Description |
|------|-------|-------------|
| `--config` | Global | Path to config file (default `~/.config/macbroom/config.yaml`) |
| `--json` | Global | Output as JSON (suppresses human-readable output) |
| `--yolo` | Global | Skip ALL confirmation prompts |
| `--yes, -y` | Per-command | Skip that command's confirmation |
| `--permanent` | clean, uninstall | Permanently delete instead of Trash |
| `--dry-run` | clean, uninstall, dupes | Show what would be deleted without deleting |
| `--quiet, -q` | clean | Suppress output (for scheduled runs) |
| `--system` | scan, clean | Filter to system junk only |
| `--browser` | scan, clean | Filter to browser caches only |
| `--xcode` | scan, clean | Filter to Xcode junk only |
| `--large` | scan, clean | Filter to large/old files only |
| `--docker` | scan, clean | Filter to Docker junk only |
| `--node` | scan, clean | Filter to Node.js cache only |
| `--homebrew` | scan, clean | Filter to Homebrew cache only |
| `--simulator` | scan, clean | Filter to iOS Simulator data only |
| `--python` | scan, clean | Filter to Python junk only |
| `--rust` | scan, clean | Filter to Rust junk only |
| `--go` | scan, clean | Filter to Go junk only |
| `--jetbrains` | scan, clean | Filter to JetBrains junk only |
| `--maven` | scan, clean | Filter to Maven cache only |
| `--gradle` | scan, clean | Filter to Gradle cache only |
| `--ruby` | scan, clean | Filter to Ruby cache only |
| `--dev` | scan, clean | Scan all dev-tool caches |
| `--caches` | scan, clean | Scan all general caches |
| `--all` | scan, clean | Scan everything |
| `--exclude` | scan, clean | Exclude paths matching pattern (glob or `dir/**`); repeatable |
| `--min-size` | dupes | Minimum file size for duplicate detection |
| `--depth N` | spacelens | Directory depth (default 2) |
| `-i` | spacelens | Interactive TUI mode |

## What it cleans

| Category | What | Risk |
|----------|------|------|
| System Junk | `~/Library/Caches/*`, `~/Library/Logs/*` | Safe |
| Browser Cache | Chrome, Safari, Firefox, Arc caches | Safe |
| Xcode Junk | DerivedData, Archives, old device support, simulators | Safe-Moderate |
| Large & Old Files | Files >100MB and >90 days in Downloads/Desktop | Moderate |
| Docker | Dangling images, build cache | Safe |
| Node.js | npm cache, stale `node_modules` | Safe |
| Homebrew | Old formula downloads and bottles | Safe |
| iOS Simulators | Unavailable simulator data and caches | Safe |
| Python | pip cache, conda packages, stale virtualenvs | Safe-Moderate |
| Rust | Cargo registry cache, stale `target/` directories | Safe-Moderate |
| Go | Module cache, build cache | Safe |
| JetBrains | IDE caches and logs (IntelliJ, GoLand, PyCharm, etc.) | Safe |
| Maven | Local repository (`~/.m2/repository`) | Safe |
| Gradle | Build caches, wrapper distributions | Safe |
| Ruby | Gem cache, Bundler cache | Safe |
| App Uninstall | App bundle + preferences, caches, support files | Moderate |
| Orphaned Preferences | Plist files for uninstalled apps | Safe |
| Duplicate Files | Identical files across Downloads, Desktop, Documents | Safe |

## Configuration

macbroom uses a YAML config file at `~/.config/macbroom/config.yaml`. A default config is created on first run.

```yaml
scanners:
  system: true
  browser: true
  xcode: true
  large_files: true
  docker: true
  node: true
  homebrew: true
  ios_simulators: true
  python: true
  rust: true
  go: true
  jetbrains: true
  maven: true
  gradle: true
  ruby: true

large_files:
  min_size: 100MB
  min_age: 90d
  paths:
    - ~/Downloads
    - ~/Desktop

dev_tools:
  search_paths:
    - ~/Documents
    - ~/Projects
    - ~/src
    - ~/code
    - ~/Developer
  min_age: 30d

exclude:
  - "~/Projects/important/**"
  - "*.iso"

schedule:
  enabled: false
  interval: daily
  time: "10:00"
  notify: true
  categories: []  # empty = all; or [system, browser, homebrew]
```

## Safety

- **Default: Move to Trash** — all deletions are recoverable via Trash
- **`--permanent`** — requires typing "yes" (not just "y") to confirm
- **`--dry-run`** — preview what would be deleted without touching anything
- **`--yolo`** — skips all confirmations with a visible warning banner
- **Risk labels** — TUI shows risk levels on items before you confirm
- **No root required** — only touches files in your home directory

## Architecture

```
cmd/macbroom/        Entry point
cmd/gendocs/         Man page generator
internal/
  scanner/           Modular scanners (System, Browser, Xcode, Apps, LargeFiles,
                     SpaceLens, Docker, Node, Homebrew, Simulator, Python,
                     Rust, Go, JetBrains, Maven, Gradle, Ruby)
  engine/            Orchestrates scanners with worker pool and live progress
  cli/               Cobra commands, flags, and JSON output
  tui/               Bubbletea interactive UI with bar list visualization,
                     per-scanner progress, and animated counters
  config/            YAML config loading, defaults, and validation
  scancache/         Scan snapshot persistence and diff computation
  dupes/             Duplicate file detection (three-pass: size, partial hash, full hash)
  history/           Cleanup history tracking and stats
  schedule/          LaunchAgent plist generation for scheduled cleaning
  trash/             macOS Trash integration (via Finder/osascript)
  maintain/          System maintenance tasks
  utils/             Shared utilities (dir sizing, formatting)
```

## Development

```bash
make build    # Build to ./bin/macbroom
make test     # Run tests with race detection
make lint     # Run golangci-lint
make man      # Generate man pages
make run      # Build and run
```

### Shell Completions

```bash
macbroom --generate-completion bash > /etc/bash_completion.d/macbroom
macbroom --generate-completion zsh > "${fpath[1]}/_macbroom"
macbroom --generate-completion fish > ~/.config/fish/completions/macbroom.fish
```

## Claude Code

Available as a skill in the [macos-toolkit](https://github.com/lu-zhengda/macos-toolkit) Claude Code plugin. Ask Claude "clean up my Mac" or "free disk space" and it runs macbroom automatically.

## License

MIT
