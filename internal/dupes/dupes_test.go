package dupes_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/lu-zhengda/macbroom/internal/dupes"
)

func TestFindDuplicates(t *testing.T) {
	dir := t.TempDir()

	content := []byte("duplicate content here, long enough to pass minSize")

	// Two identical files.
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), content, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), content, 0644); err != nil {
		t.Fatal(err)
	}

	// One unique file.
	if err := os.WriteFile(filepath.Join(dir, "unique.txt"), []byte("totally different content!!!"), 0644); err != nil {
		t.Fatal(err)
	}

	groups, err := dupes.Find(context.Background(), []string{dir}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}

	g := groups[0]
	if len(g.Files) != 2 {
		t.Fatalf("expected 2 files in group, got %d", len(g.Files))
	}
	if g.Size != int64(len(content)) {
		t.Errorf("expected size %d, got %d", len(content), g.Size)
	}
	if g.Hash == "" {
		t.Error("expected non-empty hash")
	}
}

func TestNoDuplicates(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("file one"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("file two!"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "c.txt"), []byte("file three!!"), 0644); err != nil {
		t.Fatal(err)
	}

	groups, err := dupes.Find(context.Background(), []string{dir}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(groups) != 0 {
		t.Fatalf("expected 0 groups, got %d", len(groups))
	}
}

func TestSkipsSmallFiles(t *testing.T) {
	dir := t.TempDir()

	content := []byte("tiny")

	if err := os.WriteFile(filepath.Join(dir, "a.txt"), content, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), content, 0644); err != nil {
		t.Fatal(err)
	}

	// minSize larger than the file content.
	groups, err := dupes.Find(context.Background(), []string{dir}, 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(groups) != 0 {
		t.Fatalf("expected 0 groups (files below minSize), got %d", len(groups))
	}
}

func TestContextCancelled(t *testing.T) {
	dir := t.TempDir()

	content := []byte("some content for dupe test")
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), content, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), content, 0644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := dupes.Find(ctx, []string{dir}, 0)
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
}

func TestFindWithProgress(t *testing.T) {
	dir := t.TempDir()

	content := []byte("progress tracking content here!")
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), content, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), content, 0644); err != nil {
		t.Fatal(err)
	}

	var paths []string
	groups, err := dupes.FindWithProgress(context.Background(), []string{dir}, 0, func(path string) {
		paths = append(paths, path)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if len(paths) == 0 {
		t.Error("expected progress callback to be called at least once")
	}
}

func TestSkipsGitDirs(t *testing.T) {
	dir := t.TempDir()

	content := []byte("identical content in git and outside")

	// File outside .git.
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), content, 0644); err != nil {
		t.Fatal(err)
	}

	// File inside .git — should be ignored.
	gitDir := filepath.Join(dir, ".git", "objects")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "abc123"), content, 0644); err != nil {
		t.Fatal(err)
	}

	groups, err := dupes.Find(context.Background(), []string{dir}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only one file visible (a.txt), so no duplicates.
	if len(groups) != 0 {
		t.Fatalf("expected 0 groups (git files should be skipped), got %d", len(groups))
	}
}

func TestSkipsHiddenFiles(t *testing.T) {
	dir := t.TempDir()

	content := []byte("hidden file content that is duplicated")

	// Two hidden files with identical content.
	if err := os.WriteFile(filepath.Join(dir, ".localized"), content, 0644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, ".localized"), content, 0644); err != nil {
		t.Fatal(err)
	}

	groups, err := dupes.Find(context.Background(), []string{dir}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(groups) != 0 {
		t.Fatalf("expected 0 groups (hidden files should be skipped), got %d", len(groups))
	}
}

func TestSkipsGitRepoRoots(t *testing.T) {
	dir := t.TempDir()

	content := []byte("identical build artifacts in a git repo")

	// Create a git repo directory (has .git inside).
	repo := filepath.Join(dir, "myproject")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	// Build artifact inside repo.
	binDir := filepath.Join(repo, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "app"), content, 0644); err != nil {
		t.Fatal(err)
	}
	// Another copy at repo root.
	if err := os.WriteFile(filepath.Join(repo, "app"), content, 0644); err != nil {
		t.Fatal(err)
	}

	// File outside any repo — should still be found.
	if err := os.WriteFile(filepath.Join(dir, "standalone.txt"), content, 0644); err != nil {
		t.Fatal(err)
	}

	groups, err := dupes.Find(context.Background(), []string{dir}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only standalone.txt is visible; repo files are skipped entirely.
	// With just 1 file visible, no duplicates.
	if len(groups) != 0 {
		t.Fatalf("expected 0 groups (git repo files should be skipped), got %d", len(groups))
	}
}

func TestMultipleDirs(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	content := []byte("cross-directory duplicate content")
	if err := os.WriteFile(filepath.Join(dir1, "a.txt"), content, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir2, "b.txt"), content, 0644); err != nil {
		t.Fatal(err)
	}

	groups, err := dupes.Find(context.Background(), []string{dir1, dir2}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(groups) != 1 {
		t.Fatalf("expected 1 group across dirs, got %d", len(groups))
	}
	if len(groups[0].Files) != 2 {
		t.Fatalf("expected 2 files in group, got %d", len(groups[0].Files))
	}
}
