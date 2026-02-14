package engine

import (
	"context"
	"fmt"
	"sync"

	"github.com/zhengda-lu/macbroom/internal/scanner"
)

type ScanResult struct {
	Category string
	Targets  []scanner.Target
	Error    error
}

type Engine struct {
	scanners []scanner.Scanner
}

func New() *Engine {
	return &Engine{}
}

func (e *Engine) Register(s scanner.Scanner) {
	e.scanners = append(e.scanners, s)
}

func (e *Engine) Scanners() []scanner.Scanner {
	return e.scanners
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
		return targets, fmt.Errorf("scan errors: %v", errs)
	}
	return targets, nil
}

func (e *Engine) ScanByCategory(ctx context.Context, category string) ([]scanner.Target, error) {
	for _, s := range e.scanners {
		if s.Name() == category {
			return s.Scan(ctx)
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
