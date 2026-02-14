package schedule

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
)

const (
	bundleID = "com.macbroom.cleanup"
	plistTpl = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>{{.Label}}</string>
	<key>ProgramArguments</key>
	<array>
		<string>{{.Binary}}</string>
		<string>clean</string>
		<string>--yes</string>
		<string>--quiet</string>
	</array>
	<key>StartCalendarInterval</key>
	<dict>
		<key>Hour</key>
		<integer>{{.Hour}}</integer>
		<key>Minute</key>
		<integer>{{.Minute}}</integer>{{if .Weekday}}
		<key>Weekday</key>
		<integer>{{.Weekday}}</integer>{{end}}
	</dict>
	<key>StandardOutPath</key>
	<string>{{.LogPath}}</string>
	<key>StandardErrorPath</key>
	<string>{{.LogPath}}</string>
</dict>
</plist>
`
)

// plistData holds the template fields for plist generation.
type plistData struct {
	Label   string
	Binary  string
	Hour    int
	Minute  int
	Weekday int // 0 = Sunday, 1 = Monday, ... 0 means omit (daily)
	LogPath string
}

// DefaultPath returns the default LaunchAgent plist file path.
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("Library", "LaunchAgents", bundleID+".plist")
	}
	return filepath.Join(home, "Library", "LaunchAgents", bundleID+".plist")
}

// BinaryPath returns the path to the macbroom binary. It first checks the
// currently running executable, then falls back to a well-known install path.
func BinaryPath() string {
	if exe, err := os.Executable(); err == nil {
		return exe
	}
	return "/usr/local/bin/macbroom"
}

// logPath returns the path for LaunchAgent log output.
func logPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/macbroom.log"
	}
	return filepath.Join(home, ".local", "share", "macbroom", "macbroom.log")
}

// parseTime splits a "HH:MM" string into hour and minute integers.
func parseTime(timeStr string) (int, int, error) {
	parts := strings.SplitN(timeStr, ":", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid time format %q, expected HH:MM", timeStr)
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return 0, 0, fmt.Errorf("invalid hour in %q: must be 0-23", timeStr)
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil || minute < 0 || minute > 59 {
		return 0, 0, fmt.Errorf("invalid minute in %q: must be 0-59", timeStr)
	}
	return hour, minute, nil
}

// intervalWeekday returns 0 for daily (no weekday restriction) or 1 for
// weekly (Monday). Other intervals default to daily.
func intervalWeekday(interval string) int {
	switch strings.ToLower(interval) {
	case "weekly":
		return 1 // Monday
	default:
		return 0 // daily (no weekday key)
	}
}

// GeneratePlist generates a LaunchAgent plist XML string for the given
// time and interval. The time must be in "HH:MM" format. The interval
// is either "daily" or "weekly".
func GeneratePlist(timeStr, interval string) string {
	return GeneratePlistWithBinary(timeStr, interval, BinaryPath())
}

// GeneratePlistWithBinary generates a plist using the specified binary path.
// This is useful for testing.
func GeneratePlistWithBinary(timeStr, interval, binary string) string {
	hour, minute, err := parseTime(timeStr)
	if err != nil {
		return ""
	}

	data := plistData{
		Label:   bundleID,
		Binary:  binary,
		Hour:    hour,
		Minute:  minute,
		Weekday: intervalWeekday(interval),
		LogPath: logPath(),
	}

	tmpl, err := template.New("plist").Parse(plistTpl)
	if err != nil {
		return ""
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return ""
	}
	return buf.String()
}

// Install writes the LaunchAgent plist file to the given path.
// It only writes the file; loading via launchctl is the caller's
// responsibility (e.g., the CLI layer).
func Install(path, timeStr, interval string) error {
	return InstallWithBinary(path, timeStr, interval, BinaryPath())
}

// InstallWithBinary writes the plist using a specified binary path.
func InstallWithBinary(path, timeStr, interval, binary string) error {
	plist := GeneratePlistWithBinary(timeStr, interval, binary)
	if plist == "" {
		return fmt.Errorf("failed to generate plist for time %q interval %q", timeStr, interval)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create LaunchAgents directory: %w", err)
	}

	if err := os.WriteFile(path, []byte(plist), 0o644); err != nil {
		return fmt.Errorf("failed to write plist file: %w", err)
	}

	return nil
}

// Uninstall removes the LaunchAgent plist file. Unloading via launchctl
// is the caller's responsibility.
func Uninstall(path string) error {
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil // already removed
		}
		return fmt.Errorf("failed to remove plist file: %w", err)
	}
	return nil
}

// Status checks whether the LaunchAgent plist file exists at the given path.
func Status(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Notify sends a macOS notification via osascript.
func Notify(title, message string) error {
	script := fmt.Sprintf(`display notification %q with title %q`, message, title)
	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	return nil
}
