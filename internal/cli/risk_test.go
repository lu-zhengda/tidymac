package cli

import (
	"testing"

	"github.com/lu-zhengda/macbroom/internal/scanner"
)

// ---------------------------------------------------------------------------
// riskSummary
// ---------------------------------------------------------------------------

func TestRiskSummary(t *testing.T) {
	targets := []scanner.Target{
		{Path: "/a", Size: 1000, Risk: scanner.Safe},
		{Path: "/b", Size: 2000, Risk: scanner.Moderate},
		{Path: "/c", Size: 3000, Risk: scanner.Risky},
		{Path: "/d", Size: 500, Risk: scanner.Safe},
		{Path: "/e", Size: 1500, Risk: scanner.Moderate},
	}

	rb := riskSummary(targets)

	if rb.Safe != 1500 {
		t.Errorf("Safe = %d, want 1500", rb.Safe)
	}
	if rb.Moderate != 3500 {
		t.Errorf("Moderate = %d, want 3500", rb.Moderate)
	}
	if rb.Risky != 3000 {
		t.Errorf("Risky = %d, want 3000", rb.Risky)
	}
	if rb.Total != 8000 {
		t.Errorf("Total = %d, want 8000", rb.Total)
	}
}

func TestRiskSummary_Empty(t *testing.T) {
	rb := riskSummary(nil)

	if rb.Total != 0 {
		t.Errorf("Total = %d, want 0", rb.Total)
	}
	if rb.Safe != 0 {
		t.Errorf("Safe = %d, want 0", rb.Safe)
	}
	if rb.Moderate != 0 {
		t.Errorf("Moderate = %d, want 0", rb.Moderate)
	}
	if rb.Risky != 0 {
		t.Errorf("Risky = %d, want 0", rb.Risky)
	}
}

// ---------------------------------------------------------------------------
// riskSummaryLine
// ---------------------------------------------------------------------------

func TestRiskSummaryLine(t *testing.T) {
	rb := RiskBreakdown{Safe: 5000, Moderate: 3000, Risky: 2000, Total: 10000}
	line := riskSummaryLine(rb)

	if line == "" {
		t.Fatal("expected non-empty risk summary line")
	}

	// Should contain "Safe" since Safe > 0.
	if !containsText(line, "Safe") {
		t.Error("expected 'Safe' in output")
	}
	// Should contain "Moderate" since Moderate > 0.
	if !containsText(line, "Moderate") {
		t.Error("expected 'Moderate' in output")
	}
	// Should contain "Risky" since Risky > 0.
	if !containsText(line, "Risky") {
		t.Error("expected 'Risky' in output")
	}
}

func TestRiskSummaryLine_Empty(t *testing.T) {
	rb := RiskBreakdown{}
	line := riskSummaryLine(rb)

	if line != "" {
		t.Errorf("expected empty string for zero total, got %q", line)
	}
}

func TestRiskSummaryLine_SafeOnly(t *testing.T) {
	rb := RiskBreakdown{Safe: 1000, Total: 1000}
	line := riskSummaryLine(rb)

	if !containsText(line, "Safe") {
		t.Error("expected 'Safe' in output")
	}
	// Moderate and Risky should not appear when zero.
	if containsText(line, "Moderate") {
		t.Error("did not expect 'Moderate' in output when zero")
	}
	if containsText(line, "Risky") {
		t.Error("did not expect 'Risky' in output when zero")
	}
}

// containsText strips ANSI escape sequences and checks for substring presence.
func containsText(s, sub string) bool {
	// Simple ANSI strip: remove escape sequences like \x1b[...m
	stripped := stripAnsi(s)
	return len(stripped) > 0 && len(sub) > 0 && contains(stripped, sub)
}

func stripAnsi(s string) string {
	var out []byte
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			// Skip to 'm'
			j := i + 2
			for j < len(s) && s[j] != 'm' {
				j++
			}
			if j < len(s) {
				i = j + 1
				continue
			}
		}
		out = append(out, s[i])
		i++
	}
	return string(out)
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && searchSubstring(s, sub)
}

func searchSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
