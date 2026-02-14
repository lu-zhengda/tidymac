package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestAppScanner_FindRelatedFiles(t *testing.T) {
	tmpDir := t.TempDir()

	appSupport := filepath.Join(tmpDir, "Application Support", "FakeApp")
	if err := os.MkdirAll(appSupport, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appSupport, "config.json"), make([]byte, 256), 0o644); err != nil {
		t.Fatal(err)
	}

	prefs := filepath.Join(tmpDir, "Preferences")
	if err := os.MkdirAll(prefs, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(prefs, "com.fake.FakeApp.plist"), make([]byte, 128), 0o644); err != nil {
		t.Fatal(err)
	}

	caches := filepath.Join(tmpDir, "Caches", "com.fake.FakeApp")
	if err := os.MkdirAll(caches, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(caches, "cache.db"), make([]byte, 512), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewAppScanner("", tmpDir)
	targets, err := s.FindRelatedFiles(context.Background(), "FakeApp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) < 2 {
		t.Fatalf("expected at least 2 related targets, got %d", len(targets))
	}
}

func TestAppScanner_FindRelatedFiles_NewDirs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create entries in the new search directories.
	newDirs := []string{"LaunchAgents", "Application Scripts", "Group Containers", "Cookies"}
	for _, dir := range newDirs {
		dirPath := filepath.Join(tmpDir, dir, "com.fake.TestApp")
		if err := os.MkdirAll(dirPath, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dirPath, "data"), make([]byte, 64), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	s := NewAppScanner("", tmpDir)
	targets, err := s.FindRelatedFiles(context.Background(), "TestApp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) < len(newDirs) {
		t.Fatalf("expected at least %d targets from new dirs, got %d", len(newDirs), len(targets))
	}

	// Verify each new directory contributed at least one target.
	found := make(map[string]bool)
	for _, tgt := range targets {
		for _, dir := range newDirs {
			if filepath.Dir(tgt.Path) == filepath.Join(tmpDir, dir) {
				found[dir] = true
			}
		}
	}
	for _, dir := range newDirs {
		if !found[dir] {
			t.Errorf("expected to find target in %s directory", dir)
		}
	}
}

func TestAppScanner_FindOrphans_DetectsOrphans(t *testing.T) {
	tmpDir := t.TempDir()
	appsDir := t.TempDir()

	// Create a plist for an app that is NOT installed.
	prefsDir := filepath.Join(tmpDir, "Preferences")
	if err := os.MkdirAll(prefsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(prefsDir, "com.removed.OldApp.plist"), make([]byte, 128), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create matching cache entry for the orphan.
	cachesDir := filepath.Join(tmpDir, "Caches", "com.removed.OldApp")
	if err := os.MkdirAll(cachesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cachesDir, "cache.db"), make([]byte, 256), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create matching Application Support entry.
	supportDir := filepath.Join(tmpDir, "Application Support", "OldApp")
	if err := os.MkdirAll(supportDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(supportDir, "data.json"), make([]byte, 512), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewAppScanner(appsDir, tmpDir)
	targets, err := s.FindOrphans(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expect: 1 orphaned plist + 1 cache dir + 1 app support dir = 3.
	if len(targets) < 3 {
		t.Fatalf("expected at least 3 orphan targets, got %d", len(targets))
	}

	for _, tgt := range targets {
		if tgt.Category != "Orphaned Preferences" {
			t.Errorf("expected category 'Orphaned Preferences', got %q", tgt.Category)
		}
	}
}

func TestAppScanner_FindOrphans_IgnoresInstalledApps(t *testing.T) {
	tmpDir := t.TempDir()
	appsDir := t.TempDir()

	// Create an installed app.
	if err := os.MkdirAll(filepath.Join(appsDir, "InstalledApp.app"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a plist for the installed app.
	prefsDir := filepath.Join(tmpDir, "Preferences")
	if err := os.MkdirAll(prefsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(prefsDir, "com.example.InstalledApp.plist"), make([]byte, 128), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewAppScanner(appsDir, tmpDir)
	targets, err := s.FindOrphans(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(targets) != 0 {
		t.Fatalf("expected 0 orphan targets for installed app, got %d", len(targets))
	}
}

func TestExtractAppName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"com.example.MyApp.plist", "MyApp"},
		{"com.apple.Safari.plist", "Safari"},
		{"org.mozilla.firefox.plist", "firefox"},
		{"single.plist", ""},
		{"nodots", ""},
		{"com.two.plist", "two"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := extractAppName(tc.input)
			if got != tc.want {
				t.Errorf("extractAppName(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestAppScanner_ListApps(t *testing.T) {
	tmpDir := t.TempDir()

	app1 := filepath.Join(tmpDir, "TestApp.app")
	if err := os.MkdirAll(app1, 0o755); err != nil {
		t.Fatal(err)
	}

	s := NewAppScanner(tmpDir, "")
	apps := s.ListApps()
	if len(apps) != 1 {
		t.Fatalf("expected 1 app, got %d", len(apps))
	}
	if apps[0] != "TestApp" {
		t.Errorf("expected TestApp, got %s", apps[0])
	}
}
