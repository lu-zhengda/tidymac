package engine

import (
	"context"
	"fmt"
	"sync"

	"github.com/lu-zhengda/macbroom/internal/scanner"
)

type ScanResult struct {
	Category string
	Targets  []scanner.Target
	Error    error
}

type Engine struct {
	scanners    []scanner.Scanner
	excludeFunc func(string) bool
}

func New() *Engine {
	return &Engine{}
}

func (e *Engine) Register(s scanner.Scanner) {
	e.scanners = append(e.scanners, s)
}

func (e *Engine) SetExcludeFunc(fn func(string) bool) {
	e.excludeFunc = fn
}

func (e *Engine) Scanners() []scanner.Scanner {
	return e.scanners
}

func (e *Engine) filterExcluded(targets []scanner.Target) []scanner.Target {
	if e.excludeFunc == nil {
		return targets
	}
	filtered := make([]scanner.Target, 0, len(targets))
	for _, t := range targets {
		if !e.excludeFunc(t.Path) {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

func (e *Engine) ScanAll(ctx context.Context) ([]scanner.Target, error) {
	if len(e.scanners) == 0 {
		return nil, nil
	}

	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		targets []scanner.Target
		errs    []error
	)

	for _, s := range e.scanners {
		wg.Add(1)
		go func(s scanner.Scanner) {
			defer wg.Done()
			t, err := s.Scan(ctx)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", s.Name(), err))
				return
			}
			targets = append(targets, t...)
		}(s)
	}

	wg.Wait()

	if len(errs) > 0 {
		return e.filterExcluded(targets), fmt.Errorf("scan errors: %v", errs)
	}
	return e.filterExcluded(targets), nil
}

func (e *Engine) ScanByCategory(ctx context.Context, category string) ([]scanner.Target, error) {
	for _, s := range e.scanners {
		if s.Name() == category {
			targets, err := s.Scan(ctx)
			return e.filterExcluded(targets), err
		}
	}
	return nil, fmt.Errorf("unknown category: %s", category)
}

func (e *Engine) ScanGrouped(ctx context.Context) []ScanResult {
	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		results []ScanResult
	)

	for _, s := range e.scanners {
		wg.Add(1)
		go func(s scanner.Scanner) {
			defer wg.Done()
			targets, err := s.Scan(ctx)
			targets = e.filterExcluded(targets)
			mu.Lock()
			defer mu.Unlock()
			results = append(results, ScanResult{
				Category: s.Name(),
				Targets:  targets,
				Error:    err,
			})
		}(s)
	}

	wg.Wait()
	return results
}

// ScanStatus represents the state of a scanner in the progress callback.
type ScanStatus int

const (
	ScanWaiting ScanStatus = iota
	ScanStarted
	ScanDone
)

// ScanProgress is sent to the progress callback for each scanner event.
type ScanProgress struct {
	Name    string
	Status  ScanStatus
	Targets []scanner.Target
	Error   error
}

// ScanGroupedWithProgress runs scanners with a concurrency limit and calls
// onProgress for each scanner event (started, done).
func (e *Engine) ScanGroupedWithProgress(ctx context.Context, concurrency int, onProgress func(ScanProgress)) []ScanResult {
	if concurrency < 1 {
		concurrency = 1
	}

	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		results []ScanResult
		sem     = make(chan struct{}, concurrency)
	)

	for _, s := range e.scanners {
		wg.Add(1)
		go func(s scanner.Scanner) {
			defer wg.Done()

			sem <- struct{}{} // acquire
			if onProgress != nil {
				onProgress(ScanProgress{Name: s.Name(), Status: ScanStarted})
			}

			targets, err := s.Scan(ctx)
			targets = e.filterExcluded(targets)

			if onProgress != nil {
				onProgress(ScanProgress{
					Name:    s.Name(),
					Status:  ScanDone,
					Targets: targets,
					Error:   err,
				})
			}

			<-sem // release after Done callback to keep concurrency tracking consistent

			mu.Lock()
			results = append(results, ScanResult{
				Category: s.Name(),
				Targets:  targets,
				Error:    err,
			})
			mu.Unlock()
		}(s)
	}

	wg.Wait()
	return results
}
