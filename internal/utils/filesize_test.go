package utils

import "testing"

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
	}

	for _, tt := range tests {
		result := FormatSize(tt.bytes)
		if result != tt.expected {
			t.Errorf("FormatSize(%d) = %q, want %q", tt.bytes, result, tt.expected)
		}
	}
}

func TestFormatSize_Boundaries(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{1023, "1023 B"},          // just below KB
		{1024, "1.0 KB"},         // exactly KB
		{1048575, "1024.0 KB"},   // just below MB
		{1048576, "1.0 MB"},      // exactly MB
		{1073741823, "1024.0 MB"}, // just below GB
		{1073741824, "1.0 GB"},   // exactly GB
	}

	for _, tt := range tests {
		result := FormatSize(tt.bytes)
		if result != tt.expected {
			t.Errorf("FormatSize(%d) = %q, want %q", tt.bytes, result, tt.expected)
		}
	}
}
