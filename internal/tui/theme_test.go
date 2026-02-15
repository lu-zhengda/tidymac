package tui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestCategoryColor(t *testing.T) {
	tests := []struct {
		name string
		want lipgloss.Color
	}{
		{"System Junk", lipgloss.Color("75")},
		{"Browser Cache", lipgloss.Color("214")},
		{"Docker", lipgloss.Color("39")},
		{"Python", lipgloss.Color("220")},
		{"Rust", lipgloss.Color("208")},
		{"Go", lipgloss.Color("75")},
		{"JetBrains", lipgloss.Color("171")},
		{"Unknown", colorPrimary},
		{"", colorPrimary},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := categoryColor(tt.name); got != tt.want {
				t.Errorf("categoryColor(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestBarColor(t *testing.T) {
	tests := []struct {
		ratio float64
		want  lipgloss.Color
	}{
		{1.00, barColorHigh},
		{0.80, barColorHigh},
		{0.75, barColorHigh},
		{0.74, barColorMedium},
		{0.50, barColorMedium},
		{0.40, barColorMedium},
		{0.39, barColorLow},
		{0.10, barColorLow},
		{0.00, barColorLow},
	}
	for _, tt := range tests {
		if got := barColor(tt.ratio); got != tt.want {
			t.Errorf("barColor(%.2f) = %v, want %v", tt.ratio, got, tt.want)
		}
	}
}
