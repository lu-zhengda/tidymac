package tui

import (
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lu-zhengda/macbroom/internal/scanner"
	"github.com/lu-zhengda/macbroom/internal/utils"
)

// ---------------------------------------------------------------------------
// Treemap types
// ---------------------------------------------------------------------------

type rect struct {
	x, y, w, h int
}

type treemapItem struct {
	name     string
	size     int64
	isDir    bool
	path     string
	colorIdx int
}

type treemapRect struct {
	rect
	item treemapItem
}

// ---------------------------------------------------------------------------
// layoutTreemap lays out items into a squarified treemap within bounds.
// ---------------------------------------------------------------------------

func layoutTreemap(items []treemapItem, bounds rect) []treemapRect {
	if len(items) == 0 || bounds.w <= 0 || bounds.h <= 0 {
		return nil
	}

	sorted := make([]treemapItem, len(items))
	copy(sorted, items)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].size > sorted[j].size
	})

	var totalSize int64
	for _, item := range sorted {
		totalSize += item.size
	}
	if totalSize == 0 {
		return nil
	}

	return squarify(sorted, totalSize, bounds)
}

// ---------------------------------------------------------------------------
// squarify implements the squarified treemap algorithm (Bruls et al. 2000).
// Items must be sorted by size descending.
// ---------------------------------------------------------------------------

func squarify(items []treemapItem, totalSize int64, bounds rect) []treemapRect {
	if len(items) == 0 || bounds.w <= 0 || bounds.h <= 0 {
		return nil
	}

	if len(items) == 1 {
		return []treemapRect{{rect: bounds, item: items[0]}}
	}

	horizontal := bounds.w >= bounds.h

	var row []treemapItem
	var rowSize int64
	bestWorst := float64(1e18)

	for i, item := range items {
		row = append(row, item)
		rowSize += item.size

		worst := worstAspectRatio(row, rowSize, totalSize, horizontal, bounds)

		if worst > bestWorst && i > 0 {
			// Adding this item made the aspect ratio worse; lay out the row
			// without it and recurse on the remainder.
			row = row[:len(row)-1]
			rowSize -= item.size

			rects := layoutRow(row, rowSize, totalSize, bounds, horizontal)

			rowFrac := float64(rowSize) / float64(totalSize)
			var remaining rect
			if horizontal {
				consumed := int(rowFrac * float64(bounds.w))
				if consumed < 1 {
					consumed = 1
				}
				remaining = rect{
					x: bounds.x + consumed,
					y: bounds.y,
					w: bounds.w - consumed,
					h: bounds.h,
				}
			} else {
				consumed := int(rowFrac * float64(bounds.h))
				if consumed < 1 {
					consumed = 1
				}
				remaining = rect{
					x: bounds.x,
					y: bounds.y + consumed,
					w: bounds.w,
					h: bounds.h - consumed,
				}
			}

			rest := squarify(items[i:], totalSize-rowSize, remaining)
			return append(rects, rest...)
		}
		bestWorst = worst
	}

	// All items fit into one row.
	return layoutRow(row, rowSize, totalSize, bounds, horizontal)
}

// worstAspectRatio returns the worst (highest) aspect ratio among all items
// in the current row candidate.
func worstAspectRatio(row []treemapItem, rowSize, totalSize int64, horizontal bool, bounds rect) float64 {
	worst := 0.0
	rowFrac := float64(rowSize) / float64(totalSize)

	for _, item := range row {
		itemFrac := float64(item.size) / float64(totalSize)
		var w, h int
		if horizontal {
			w = int(rowFrac * float64(bounds.w))
			h = int((itemFrac / rowFrac) * float64(bounds.h))
		} else {
			h = int(rowFrac * float64(bounds.h))
			w = int((itemFrac / rowFrac) * float64(bounds.w))
		}
		if w < 1 {
			w = 1
		}
		if h < 1 {
			h = 1
		}

		ratio := float64(w) / float64(h)
		if ratio < 1 {
			ratio = 1 / ratio
		}
		if ratio > worst {
			worst = ratio
		}
	}
	return worst
}

// layoutRow places a committed row of items within bounds along the short axis.
func layoutRow(row []treemapItem, rowSize, totalSize int64, bounds rect, horizontal bool) []treemapRect {
	var rects []treemapRect
	rowFrac := float64(rowSize) / float64(totalSize)

	if horizontal {
		w := int(rowFrac * float64(bounds.w))
		if w < 1 {
			w = 1
		}
		y := bounds.y
		for _, item := range row {
			itemFrac := float64(item.size) / float64(rowSize)
			h := int(itemFrac * float64(bounds.h))
			if h < 1 {
				h = 1
			}
			rects = append(rects, treemapRect{
				rect: rect{x: bounds.x, y: y, w: w, h: h},
				item: item,
			})
			y += h
		}
	} else {
		h := int(rowFrac * float64(bounds.h))
		if h < 1 {
			h = 1
		}
		x := bounds.x
		for _, item := range row {
			itemFrac := float64(item.size) / float64(rowSize)
			w := int(itemFrac * float64(bounds.w))
			if w < 1 {
				w = 1
			}
			rects = append(rects, treemapRect{
				rect: rect{x: x, y: bounds.y, w: w, h: h},
				item: item,
			})
			x += w
		}
	}

	return rects
}

// ---------------------------------------------------------------------------
// renderTreemap renders a set of SpaceLens nodes as a colored treemap string.
// selectedIdx highlights the item under the cursor (-1 for none).
// ---------------------------------------------------------------------------

func renderTreemap(nodes []scanner.SpaceLensNode, width, height int, selectedIdx int) string {
	if len(nodes) == 0 || width < 4 || height < 2 {
		return "No data to display.\n"
	}

	items := make([]treemapItem, len(nodes))
	for i, n := range nodes {
		items[i] = treemapItem{
			name:     n.Name,
			size:     n.Size,
			isDir:    n.IsDir,
			path:     n.Path,
			colorIdx: i % len(treemapColors),
		}
	}

	rects := layoutTreemap(items, rect{x: 0, y: 0, w: width, h: height})

	// Render to a 2D grid.
	grid := make([][]rune, height)
	colors := make([][]int, height)
	selected := make([][]bool, height)
	for y := 0; y < height; y++ {
		grid[y] = make([]rune, width)
		colors[y] = make([]int, width)
		selected[y] = make([]bool, width)
		for x := 0; x < width; x++ {
			grid[y][x] = ' '
			colors[y][x] = -1
		}
	}

	for ri, r := range rects {
		colorIdx := r.item.colorIdx
		isSel := ri == selectedIdx

		// Fill block.
		for y := r.y; y < r.y+r.h && y < height; y++ {
			for x := r.x; x < r.x+r.w && x < width; x++ {
				grid[y][x] = '\u2591' // light shade
				colors[y][x] = colorIdx
				selected[y][x] = isSel
			}
		}

		// Draw border.
		for x := r.x; x < r.x+r.w && x < width; x++ {
			if r.y < height {
				grid[r.y][x] = '\u2500' // horizontal line
			}
			if r.y+r.h-1 < height {
				grid[r.y+r.h-1][x] = '\u2500'
			}
		}
		for y := r.y; y < r.y+r.h && y < height; y++ {
			if r.x < width {
				grid[y][r.x] = '\u2502' // vertical line
			}
			if r.x+r.w-1 < width {
				grid[y][r.x+r.w-1] = '\u2502'
			}
		}

		// Write label inside block.
		label := r.item.name
		sizeTxt := utils.FormatSize(r.item.size)
		if r.w > 4 && r.h > 2 {
			maxLen := r.w - 3
			if len(label) > maxLen {
				if maxLen > 1 {
					label = label[:maxLen-1] + "\u2026"
				} else {
					label = label[:maxLen]
				}
			}
			writeText(grid, r.x+1, r.y+1, label, width, height)
			if r.h > 3 {
				writeText(grid, r.x+1, r.y+2, sizeTxt, width, height)
			}
		}
	}

	// Render grid to string with colors.
	var sb strings.Builder
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			ch := string(grid[y][x])
			ci := colors[y][x]
			if selected[y][x] {
				style := lipgloss.NewStyle().Bold(true).Reverse(true)
				if ci >= 0 && ci < len(treemapColors) {
					style = style.Foreground(treemapColors[ci])
				}
				sb.WriteString(style.Render(ch))
			} else if ci >= 0 && ci < len(treemapColors) {
				style := lipgloss.NewStyle().Foreground(treemapColors[ci])
				sb.WriteString(style.Render(ch))
			} else {
				sb.WriteString(ch)
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// writeText writes a string into the grid at (x, y), respecting bounds.
func writeText(grid [][]rune, x, y int, text string, maxW, maxH int) {
	if y >= maxH {
		return
	}
	for i, ch := range text {
		col := x + i
		if col >= maxW {
			break
		}
		grid[y][col] = ch
	}
}
