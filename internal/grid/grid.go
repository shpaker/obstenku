// Package grid is a uniform spatial hash: it remembers which items cover
// which square cells, so a query looks only at the cells around it.
package grid

import "math"

// limit keeps cell coordinates within int range for any finite input.
const limit = 1e15

// Cell is the address of one square cell.
type Cell struct {
	X, Y int
}

// Grid maps cells to the items that cover them. The zero value is not
// usable; call New.
type Grid[I comparable] struct {
	size  float64
	cells map[Cell][]I
}

// New returns an empty grid of square cells with the given side.
func New[I comparable](size float64) *Grid[I] {
	return &Grid[I]{size: size, cells: make(map[Cell][]I)}
}

// Insert puts the item into every cell the box touches.
func (g *Grid[I]) Insert(item I, x, y, w, h float64) {
	x0, y0, x1, y1 := g.span(x, y, w, h)
	for cy := y0; cy <= y1; cy++ {
		for cx := x0; cx <= x1; cx++ {
			c := Cell{X: cx, Y: cy}
			g.cells[c] = append(g.cells[c], item)
		}
	}
}

// Delete takes the item out of every cell the box touches. The box must be
// the one the item was inserted with.
func (g *Grid[I]) Delete(item I, x, y, w, h float64) {
	x0, y0, x1, y1 := g.span(x, y, w, h)
	for cy := y0; cy <= y1; cy++ {
		for cx := x0; cx <= x1; cx++ {
			c := Cell{X: cx, Y: cy}
			items := g.cells[c]
			for i, it := range items {
				if it != item {
					continue
				}
				last := len(items) - 1
				items[i] = items[last]
				var zero I
				items[last] = zero
				items = items[:last]
				break
			}
			if len(items) == 0 {
				delete(g.cells, c)
			} else {
				g.cells[c] = items
			}
		}
	}
}

// Visit calls visit for every item in the cells the box touches. An item
// that covers several of them is visited several times.
func (g *Grid[I]) Visit(x, y, w, h float64, visit func(I)) {
	x0, y0, x1, y1 := g.span(x, y, w, h)
	// A box wider than the occupied cells: walk the cells, not the box
	if float64(x1-x0+1)*float64(y1-y0+1) > float64(len(g.cells)) {
		for c, items := range g.cells {
			if c.X < x0 || c.X > x1 || c.Y < y0 || c.Y > y1 {
				continue
			}
			for _, it := range items {
				visit(it)
			}
		}
		return
	}
	for cy := y0; cy <= y1; cy++ {
		for cx := x0; cx <= x1; cx++ {
			for _, it := range g.cells[Cell{X: cx, Y: cy}] {
				visit(it)
			}
		}
	}
}

// span returns the first and the last cell the box touches, both inclusive.
func (g *Grid[I]) span(x, y, w, h float64) (x0, y0, x1, y1 int) {
	return g.cell(x), g.cell(y), g.cell(x + w), g.cell(y + h)
}

func (g *Grid[I]) cell(v float64) int {
	return int(math.Max(-limit, math.Min(limit, math.Floor(v/g.size))))
}
