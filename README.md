# macbroom

A lightweight macOS cleanup tool for the terminal.

Scan and remove system junk, browser caches, Xcode artifacts, and more — from the command line or an interactive TUI.

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

# Clean junk files (moves to Trash)
macbroom clean
macbroom clean --xcode
macbroom clean --permanent    # permanent delete (requires typing "yes")

# Uninstall an app and all its related files
macbroom uninstall "Some App"

# Visualize disk usage
macbroom spacelens              # whole system
macbroom spacelens ~/Downloads  # specific path
macbroom spacelens -i           # interactive TUI mode

# Run maintenance tasks (DNS flush, Spotlight reindex, etc.)
macbroom maintain
```

### Flags

| Flag | Scope | Description |
|------|-------|-------------|
| `--yolo` | Global | Skip ALL confirmation prompts |
| `--yes, -y` | Per-command | Skip that command's confirmation |
| `--permanent` | clean, uninstall | Permanently delete instead of Trash |
| `--system` | scan, clean | Filter to system junk only |
| `--browser` | scan, clean | Filter to browser caches only |
| `--xcode` | scan, clean | Filter to Xcode junk only |
| `--large` | scan, clean | Filter to large/old files only |
| `--depth N` | spacelens | Directory depth (default 2) |
| `-i` | spacelens | Interactive TUI mode |

## What it cleans

| Category | What | Risk |
|----------|------|------|
| System Junk | `~/Library/Caches/*`, `~/Library/Logs/*` | Safe |
| Browser Cache | Chrome, Safari, Firefox, Arc caches | Safe |
| Xcode Junk | DerivedData, Archives, old device support, simulators | Safe-Moderate |
| Large & Old Files | Files >100MB and >90 days in Downloads/Desktop | Moderate |
| App Uninstall | App bundle + preferences, caches, support files | Moderate |

## Safety

- **Default: Move to Trash** — all deletions are recoverable via Trash
- **`--permanent`** — requires typing "yes" (not just "y") to confirm
- **`--yolo`** — skips all confirmations with a visible warning banner
- **Risk labels** — TUI shows risk levels on items before you confirm
- **No root required** — only touches files in your home directory

## Architecture

```
cmd/macbroom/        Entry point
internal/
  scanner/          Modular scanners (System, Browser, Xcode, Apps, LargeFiles, SpaceLens)
  engine/           Orchestrates scanners concurrently
  cli/              Cobra commands and flags
  tui/              Bubbletea interactive UI
  trash/            macOS Trash integration (via Finder/osascript)
  maintain/         System maintenance tasks
  utils/            Shared utilities (dir sizing, formatting)
```

## Development

```bash
make build    # Build to ./bin/macbroom
make test     # Run tests with race detection
make lint     # Run golangci-lint
make run      # Build and run
```

## License

MIT
