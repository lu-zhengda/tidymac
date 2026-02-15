package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandPaths(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{"tilde prefix", []string{"~/Documents"}, []string{filepath.Join(home, "Documents")}},
		{"tilde alone", []string{"~"}, []string{home}},
		{"absolute unchanged", []string{"/usr/local/bin"}, []string{"/usr/local/bin"}},
		{"mixed", []string{"~/foo", "/bar", "~"}, []string{filepath.Join(home, "foo"), "/bar", home}},
		{"empty", []string{}, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandPaths(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("expandPaths(%v) = %v, want %v", tt.input, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("expandPaths[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
