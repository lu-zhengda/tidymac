package engine

import (
	"context"
	"testing"

	"github.com/zhengda-lu/tidymac/internal/scanner"
)

type mockScanner struct {
	name    string
	targets []scanner.Target
	err     error
}

func (m *mockScanner) Name() string                                    { return m.name }
func (m *mockScanner) Description() string                             { return "mock scanner" }
func (m *mockScanner) Risk() scanner.RiskLevel                         { return scanner.Safe }
func (m *mockScanner) Scan(_ context.Context) ([]scanner.Target, error) { return m.targets, m.err }

func TestScanAll(t *testing.T) {
	targets := []scanner.Target{
		{Path: "/tmp/cache1", Size: 1024, Category: "test"},
		{Path: "/tmp/cache2", Size: 2048, Category: "test"},
	}

	e := New()
	e.Register(&mockScanner{name: "test", targets: targets})

	results, err := e.ScanAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(results))
	}
	if results[0].Path != "/tmp/cache1" {
		t.Errorf("expected /tmp/cache1, got %s", results[0].Path)
	}
}

func TestScanAllEmpty(t *testing.T) {
	e := New()
	results, err := e.ScanAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 targets, got %d", len(results))
	}
}

func TestScanByCategory(t *testing.T) {
	systemTargets := []scanner.Target{
		{Path: "/tmp/sys", Size: 512, Category: "System Junk"},
	}
	browserTargets := []scanner.Target{
		{Path: "/tmp/browser", Size: 1024, Category: "Browser Cache"},
	}

	e := New()
	e.Register(&mockScanner{name: "System Junk", targets: systemTargets})
	e.Register(&mockScanner{name: "Browser Cache", targets: browserTargets})

	results, err := e.ScanByCategory(context.Background(), "System Junk")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 target, got %d", len(results))
	}
	if results[0].Category != "System Junk" {
		t.Errorf("expected System Junk, got %s", results[0].Category)
	}
}
