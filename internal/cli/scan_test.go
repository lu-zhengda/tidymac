package cli

import (
	"strings"
	"testing"

	"github.com/lu-zhengda/macbroom/internal/scanner"
)

func TestFilterByThreshold(t *testing.T) {
	targets := []scanner.Target{
		{Path: "/a", Size: 100},
		{Path: "/b", Size: 500},
		{Path: "/c", Size: 1000},
		{Path: "/d", Size: 2000},
		{Path: "/e", Size: 50},
	}

	tests := []struct {
		name      string
		threshold int64
		wantCount int
		wantPaths []string
	}{
		{
			name:      "threshold 0 returns all",
			threshold: 0,
			wantCount: 5,
		},
		{
			name:      "threshold 500 returns 3",
			threshold: 500,
			wantCount: 3,
			wantPaths: []string{"/b", "/c", "/d"},
		},
		{
			name:      "threshold 1000 returns 2",
			threshold: 1000,
			wantCount: 2,
			wantPaths: []string{"/c", "/d"},
		},
		{
			name:      "threshold 5000 returns none",
			threshold: 5000,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterByThreshold(targets, tt.threshold)
			if len(result) != tt.wantCount {
				t.Errorf("filterByThreshold(threshold=%d) returned %d items, want %d",
					tt.threshold, len(result), tt.wantCount)
			}
			if tt.wantPaths != nil {
				for i, want := range tt.wantPaths {
					if i >= len(result) {
						break
					}
					if result[i].Path != want {
						t.Errorf("result[%d].Path = %q, want %q", i, result[i].Path, want)
					}
				}
			}
		})
	}
}

func TestFilterByThreshold_Empty(t *testing.T) {
	result := filterByThreshold(nil, 100)
	if len(result) != 0 {
		t.Errorf("expected empty result for nil targets, got %d", len(result))
	}
}

func TestPrintJSON(t *testing.T) {
	data := map[string]string{"key": "value"}
	out := captureOutput(func() {
		if err := printJSON(data); err != nil {
			t.Errorf("printJSON returned error: %v", err)
		}
	})

	if !strings.Contains(out, `"key"`) {
		t.Errorf("expected JSON with key, got %q", out)
	}
	if !strings.Contains(out, `"value"`) {
		t.Errorf("expected JSON with value, got %q", out)
	}
	// Should be indented.
	if !strings.Contains(out, "  ") {
		t.Error("expected indented JSON output")
	}
}
