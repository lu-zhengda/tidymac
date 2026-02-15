package cli

import (
	"sort"
	"testing"
)

func TestSelectedCategories_DevProfile(t *testing.T) {
	cats := selectedCategories(
		false, false, false, false, false, false, false, false,
		false, false, false, false,
		false, false, false,
		true, false, false,
	)
	want := []string{"Go", "Gradle", "JetBrains", "Maven", "Node.js", "Python", "Ruby", "Rust"}
	sort.Strings(cats)
	if len(cats) != len(want) {
		t.Fatalf("--dev: got %v, want %v", cats, want)
	}
	for i := range want {
		if cats[i] != want[i] {
			t.Errorf("--dev[%d]: got %q, want %q", i, cats[i], want[i])
		}
	}
}

func TestSelectedCategories_CachesProfile(t *testing.T) {
	cats := selectedCategories(
		false, false, false, false, false, false, false, false,
		false, false, false, false,
		false, false, false,
		false, true, false,
	)
	want := []string{"Browser Cache", "Homebrew", "System Junk"}
	sort.Strings(cats)
	if len(cats) != len(want) {
		t.Fatalf("--caches: got %v, want %v", cats, want)
	}
	for i := range want {
		if cats[i] != want[i] {
			t.Errorf("--caches[%d]: got %q, want %q", i, cats[i], want[i])
		}
	}
}

func TestSelectedCategories_AllProfile(t *testing.T) {
	cats := selectedCategories(
		false, false, false, false, false, false, false, false,
		false, false, false, false,
		false, false, false,
		false, false, true,
	)
	if cats != nil {
		t.Errorf("--all should return nil (scan everything), got %v", cats)
	}
}

func TestSelectedCategories_DevPlusDocker(t *testing.T) {
	cats := selectedCategories(
		false, false, false, false, true, false, false, false,
		false, false, false, false,
		false, false, false,
		true, false, false,
	)
	sort.Strings(cats)
	want := []string{"Docker", "Go", "Gradle", "JetBrains", "Maven", "Node.js", "Python", "Ruby", "Rust"}
	if len(cats) != len(want) {
		t.Fatalf("--dev --docker: got %v, want %v", cats, want)
	}
	for i := range want {
		if cats[i] != want[i] {
			t.Errorf("--dev --docker[%d]: got %q, want %q", i, cats[i], want[i])
		}
	}
}

func TestSelectedCategories_DevAndCachesCombine(t *testing.T) {
	cats := selectedCategories(
		false, false, false, false, false, false, false, false,
		false, false, false, false,
		false, false, false,
		true, true, false,
	)
	sort.Strings(cats)
	want := []string{"Browser Cache", "Go", "Gradle", "Homebrew", "JetBrains", "Maven", "Node.js", "Python", "Ruby", "Rust", "System Junk"}
	if len(cats) != len(want) {
		t.Fatalf("--dev --caches: got %v, want %v", cats, want)
	}
	for i := range want {
		if cats[i] != want[i] {
			t.Errorf("--dev --caches[%d]: got %q, want %q", i, cats[i], want[i])
		}
	}
}

func TestSelectedCategories_NoneSelected(t *testing.T) {
	cats := selectedCategories(
		false, false, false, false, false, false, false, false,
		false, false, false, false,
		false, false, false,
		false, false, false,
	)
	if cats != nil {
		t.Errorf("no flags should return nil, got %v", cats)
	}
}
