package cli

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/lu-zhengda/macbroom/internal/scancache"
	"github.com/lu-zhengda/macbroom/internal/scanner"
)

// captureOutput redirects stdout via os.Pipe and returns whatever was written.
func captureOutput(fn func()) string {
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = origStdout

	data, _ := io.ReadAll(r)
	return string(data)
}

// ---------------------------------------------------------------------------
// truncatePath
// ---------------------------------------------------------------------------

func TestTruncatePath(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		maxLen int
		want   string
	}{
		{
			name:   "short path unchanged",
			path:   "/tmp/foo",
			maxLen: 20,
			want:   "/tmp/foo",
		},
		{
			name:   "exact length unchanged",
			path:   "abcdefghij",
			maxLen: 10,
			want:   "abcdefghij",
		},
		{
			name:   "long path truncated",
			path:   "/Users/home/very/long/path/to/file.txt",
			maxLen: 20,
			want:   ".../path/to/file.txt",
		},
		{
			name:   "empty path",
			path:   "",
			maxLen: 10,
			want:   "",
		},
		{
			name:   "maxLen equals 3",
			path:   "abcdef",
			maxLen: 3,
			want:   "...",
		},
		{
			name:   "maxLen equals 4",
			path:   "abcdef",
			maxLen: 4,
			want:   "...f",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncatePath(tt.path, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncatePath(%q, %d) = %q, want %q", tt.path, tt.maxLen, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// printScanResults — empty
// ---------------------------------------------------------------------------

func TestPrintScanResults_Empty(t *testing.T) {
	out := captureOutput(func() {
		printScanResults(nil, nil)
	})
	if !strings.Contains(out, "No junk files found") {
		t.Errorf("expected 'No junk files found', got %q", out)
	}
}

// ---------------------------------------------------------------------------
// printScanResults — grouped & sorted
// ---------------------------------------------------------------------------

func TestPrintScanResults_GroupedAndSorted(t *testing.T) {
	targets := []scanner.Target{
		{Path: "/small", Size: 100, Category: "Small Cat", Risk: scanner.Safe},
		{Path: "/medium", Size: 500, Category: "Small Cat", Risk: scanner.Moderate},
		{Path: "/big-a", Size: 2000, Category: "Big Cat", Risk: scanner.Risky},
		{Path: "/big-b", Size: 1000, Category: "Big Cat", Risk: scanner.Moderate},
		{Path: "/big-c", Size: 3000, Category: "Big Cat", Risk: scanner.Safe},
	}

	out := captureOutput(func() {
		printScanResults(targets, nil)
	})

	// Category with larger total (Big Cat = 6000) should appear before Small Cat (600).
	bigIdx := strings.Index(out, "Big Cat")
	smallIdx := strings.Index(out, "Small Cat")
	if bigIdx < 0 || smallIdx < 0 {
		t.Fatalf("expected both categories in output, got:\n%s", out)
	}
	if bigIdx >= smallIdx {
		t.Errorf("expected Big Cat (larger total) before Small Cat, bigIdx=%d smallIdx=%d", bigIdx, smallIdx)
	}

	// Within Big Cat, items should be ordered: 3000, 2000, 1000 (size descending).
	// Find the paths in output after "Big Cat" header.
	bigCatSection := out[bigIdx:]
	idxBigC := strings.Index(bigCatSection, "/big-c")
	idxBigA := strings.Index(bigCatSection, "/big-a")
	idxBigB := strings.Index(bigCatSection, "/big-b")
	if idxBigC < 0 || idxBigA < 0 || idxBigB < 0 {
		t.Fatalf("expected all Big Cat items in output, got:\n%s", bigCatSection)
	}
	if !(idxBigC < idxBigA && idxBigA < idxBigB) {
		t.Errorf("expected size-descending order /big-c, /big-a, /big-b; positions: c=%d a=%d b=%d",
			idxBigC, idxBigA, idxBigB)
	}

	// Within Small Cat, /medium (500) should come before /small (100).
	smallSection := out[smallIdx:]
	idxMedium := strings.Index(smallSection, "/medium")
	idxSmall := strings.Index(smallSection, "/small")
	if idxMedium < 0 || idxSmall < 0 {
		t.Fatalf("expected all Small Cat items in output, got:\n%s", smallSection)
	}
	if idxMedium >= idxSmall {
		t.Errorf("expected /medium before /small in Small Cat; positions: medium=%d small=%d",
			idxMedium, idxSmall)
	}

	// Total reclaimable line should be present.
	if !strings.Contains(out, "Total reclaimable") {
		t.Error("expected 'Total reclaimable' in output")
	}
}

// ---------------------------------------------------------------------------
// diffIndicator
// ---------------------------------------------------------------------------

func TestDiffIndicator_Grew(t *testing.T) {
	diff := &scancache.DiffResult{
		Categories: map[string]scancache.CategoryDiff{
			"System Junk": {PreviousSize: 1000, CurrentSize: 2000, Delta: 1000},
		},
	}
	got := diffIndicator("System Junk", diff)
	stripped := stripAnsi(got)
	if !strings.Contains(stripped, "+") {
		t.Errorf("expected '+' in grew indicator, got %q", stripped)
	}
}

func TestDiffIndicator_Shrank(t *testing.T) {
	diff := &scancache.DiffResult{
		Categories: map[string]scancache.CategoryDiff{
			"System Junk": {PreviousSize: 2000, CurrentSize: 1000, Delta: -1000},
		},
	}
	got := diffIndicator("System Junk", diff)
	stripped := stripAnsi(got)
	if !strings.Contains(stripped, "-") {
		t.Errorf("expected '-' in shrank indicator, got %q", stripped)
	}
}

func TestDiffIndicator_New(t *testing.T) {
	diff := &scancache.DiffResult{
		Categories: map[string]scancache.CategoryDiff{
			"System Junk": {CurrentSize: 1000, Delta: 1000, IsNew: true},
		},
	}
	got := diffIndicator("System Junk", diff)
	stripped := stripAnsi(got)
	if !strings.Contains(stripped, "new") {
		t.Errorf("expected 'new' in indicator, got %q", stripped)
	}
}

func TestDiffIndicator_Unchanged(t *testing.T) {
	diff := &scancache.DiffResult{
		Categories: map[string]scancache.CategoryDiff{
			"System Junk": {PreviousSize: 1000, CurrentSize: 1000, Delta: 0},
		},
	}
	got := diffIndicator("System Junk", diff)
	stripped := stripAnsi(got)
	if !strings.Contains(stripped, "unchanged") {
		t.Errorf("expected 'unchanged' in indicator, got %q", stripped)
	}
}

func TestDiffIndicator_Nil(t *testing.T) {
	got := diffIndicator("System Junk", nil)
	if got != "" {
		t.Errorf("expected empty string for nil diff, got %q", got)
	}
}

func TestDiffIndicator_NotFound(t *testing.T) {
	diff := &scancache.DiffResult{
		Categories: map[string]scancache.CategoryDiff{
			"Other": {PreviousSize: 1000, CurrentSize: 1000, Delta: 0},
		},
	}
	got := diffIndicator("System Junk", diff)
	if got != "" {
		t.Errorf("expected empty string for category not in diff, got %q", got)
	}
}
