package engine

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/lu-zhengda/macbroom/internal/scanner"
)

type mockScanner struct {
	name    string
	targets []scanner.Target
	delay   time.Duration
	err     error
}

func (m *mockScanner) Name() string            { return m.name }
func (m *mockScanner) Description() string     { return "mock scanner" }
func (m *mockScanner) Risk() scanner.RiskLevel { return scanner.Safe }
func (m *mockScanner) Scan(ctx context.Context) ([]scanner.Target, error) {
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return m.targets, m.err
}

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

func TestScanGroupedWithProgress_Basic(t *testing.T) {
	e := New()
	e.Register(&mockScanner{name: "A", targets: []scanner.Target{{Path: "/a"}}})
	e.Register(&mockScanner{name: "B", targets: []scanner.Target{{Path: "/b"}}})

	var mu sync.Mutex
	var events []ScanProgress
	results := e.ScanGroupedWithProgress(context.Background(), 2, func(p ScanProgress) {
		mu.Lock()
		events = append(events, p)
		mu.Unlock()
	})

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	mu.Lock()
	defer mu.Unlock()
	startCount, doneCount := 0, 0
	for _, ev := range events {
		if ev.Status == ScanStarted {
			startCount++
		}
		if ev.Status == ScanDone {
			doneCount++
		}
	}
	if startCount != 2 {
		t.Errorf("expected 2 Started events, got %d", startCount)
	}
	if doneCount != 2 {
		t.Errorf("expected 2 Done events, got %d", doneCount)
	}
}

func TestScanGroupedWithProgress_ConcurrencyLimit(t *testing.T) {
	e := New()
	var mu sync.Mutex
	running := 0
	maxRunning := 0

	for i := 0; i < 4; i++ {
		e.Register(&mockScanner{name: "S", delay: 50 * time.Millisecond})
	}

	e.ScanGroupedWithProgress(context.Background(), 2, func(p ScanProgress) {
		mu.Lock()
		defer mu.Unlock()
		if p.Status == ScanStarted {
			running++
			if running > maxRunning {
				maxRunning = running
			}
		}
		if p.Status == ScanDone {
			running--
		}
	})

	if maxRunning > 2 {
		t.Errorf("expected max concurrency 2, got %d", maxRunning)
	}
}

func TestScanGroupedWithProgress_ContextCancelled(t *testing.T) {
	e := New()
	e.Register(&mockScanner{name: "Slow", delay: 5 * time.Second})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	results := e.ScanGroupedWithProgress(ctx, 1, nil)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Error == nil {
		t.Error("expected error for cancelled context")
	}
}
