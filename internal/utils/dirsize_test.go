package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirSize(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), make([]byte, 100), 0o644)
	os.WriteFile(filepath.Join(dir, "b.txt"), make([]byte, 200), 0o644)

	size, err := DirSize(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if size != 300 {
		t.Errorf("DirSize = %d, want 300", size)
	}
}

func TestDirSize_Empty(t *testing.T) {
	dir := t.TempDir()
	size, err := DirSize(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if size != 0 {
		t.Errorf("DirSize of empty dir = %d, want 0", size)
	}
}

func TestDirSize_Nested(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	os.Mkdir(sub, 0o755)
	os.WriteFile(filepath.Join(sub, "nested.txt"), make([]byte, 500), 0o644)

	size, err := DirSize(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if size != 500 {
		t.Errorf("DirSize = %d, want 500", size)
	}
}

func TestDirSizesParallel(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	os.WriteFile(filepath.Join(dir1, "a.txt"), make([]byte, 100), 0o644)
	os.WriteFile(filepath.Join(dir2, "b.txt"), make([]byte, 200), 0o644)

	result := DirSizesParallel([]string{dir1, dir2})
	if result[dir1] != 100 {
		t.Errorf("dir1 size = %d, want 100", result[dir1])
	}
	if result[dir2] != 200 {
		t.Errorf("dir2 size = %d, want 200", result[dir2])
	}
}
