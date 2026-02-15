package tui

import "testing"

func TestLayoutTreemap_SingleItem(t *testing.T) {
	items := []treemapItem{{name: "foo", size: 100}}
	rects := layoutTreemap(items, rect{x: 0, y: 0, w: 80, h: 24})

	if len(rects) != 1 {
		t.Fatalf("expected 1 rect, got %d", len(rects))
	}
	if rects[0].w != 80 || rects[0].h != 24 {
		t.Errorf("expected full rect (80x24), got (%dx%d)", rects[0].w, rects[0].h)
	}
}

func TestLayoutTreemap_TwoItems(t *testing.T) {
	items := []treemapItem{
		{name: "a", size: 75},
		{name: "b", size: 25},
	}
	rects := layoutTreemap(items, rect{x: 0, y: 0, w: 80, h: 24})

	if len(rects) != 2 {
		t.Fatalf("expected 2 rects, got %d", len(rects))
	}
	totalArea := 0
	for _, r := range rects {
		totalArea += r.w * r.h
	}
	expected := 80 * 24
	if totalArea != expected {
		t.Errorf("expected total area %d, got %d", expected, totalArea)
	}
}

func TestLayoutTreemap_Empty(t *testing.T) {
	rects := layoutTreemap(nil, rect{x: 0, y: 0, w: 80, h: 24})
	if len(rects) != 0 {
		t.Errorf("expected 0 rects, got %d", len(rects))
	}
}

func TestLayoutTreemap_AspectRatio(t *testing.T) {
	items := []treemapItem{
		{name: "a", size: 50},
		{name: "b", size: 30},
		{name: "c", size: 20},
	}
	rects := layoutTreemap(items, rect{x: 0, y: 0, w: 80, h: 24})

	for i, r := range rects {
		if r.w == 0 || r.h == 0 {
			t.Errorf("rect %d has zero dimension: %dx%d", i, r.w, r.h)
			continue
		}
		ratio := float64(r.w) / float64(r.h)
		if ratio > 20 || ratio < 0.05 {
			t.Errorf("rect %d has extreme aspect ratio: %dx%d (ratio=%.2f)", i, r.w, r.h, ratio)
		}
	}
}
