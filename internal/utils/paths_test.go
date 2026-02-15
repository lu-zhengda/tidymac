package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHomeDir(t *testing.T) {
	home := HomeDir()
	if home == "" {
		t.Error("HomeDir should not return empty string")
	}
	expected, _ := os.UserHomeDir()
	if home != expected {
		t.Errorf("HomeDir = %q, want %q", home, expected)
	}
}

func TestLibraryPath(t *testing.T) {
	path := LibraryPath("Caches")
	home := HomeDir()
	want := filepath.Join(home, "Library", "Caches")
	if path != want {
		t.Errorf("LibraryPath(\"Caches\") = %q, want %q", path, want)
	}
}

func TestDirExists(t *testing.T) {
	dir := t.TempDir()
	if !DirExists(dir) {
		t.Error("DirExists should return true for existing dir")
	}
	if DirExists(filepath.Join(dir, "nope")) {
		t.Error("DirExists should return false for non-existent dir")
	}
	f := filepath.Join(dir, "file.txt")
	os.WriteFile(f, []byte("hi"), 0o644)
	if DirExists(f) {
		t.Error("DirExists should return false for a file")
	}
}

func TestFileExists(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "file.txt")
	os.WriteFile(f, []byte("hi"), 0o644)

	if !FileExists(f) {
		t.Error("FileExists should return true for existing file")
	}
	if FileExists(dir) {
		t.Error("FileExists should return false for a directory")
	}
	if FileExists(filepath.Join(dir, "nope")) {
		t.Error("FileExists should return false for non-existent path")
	}
}
