package grid

import (
	"slices"
	"testing"
)

func visited(g *Grid[string], x, y, w, h float64) []string {
	var items []string
	g.Visit(x, y, w, h, func(item string) {
		if !slices.Contains(items, item) {
			items = append(items, item)
		}
	})
	slices.Sort(items)
	return items
}

func TestGrid(t *testing.T) {
	g := New[string](8)
	g.Insert("a", 0, 0, 4, 4)
	g.Insert("wide", -20, 0, 40, 4)
	g.Insert("far", 100, 100, 4, 4)

	if got := visited(g, 1, 1, 1, 1); !slices.Equal(got, []string{"a", "wide"}) {
		t.Errorf("near the origin: %v", got)
	}
	if got := visited(g, -1e9, -1e9, 2e9, 2e9); !slices.Equal(got, []string{"a", "far", "wide"}) {
		t.Errorf("everything: %v", got)
	}

	g.Delete("wide", -20, 0, 40, 4)
	if got := visited(g, -20, 0, 40, 4); !slices.Equal(got, []string{"a"}) {
		t.Errorf("after Delete: %v", got)
	}
	g.Delete("a", 0, 0, 4, 4)
	g.Delete("far", 100, 100, 4, 4)
	if len(g.cells) != 0 {
		t.Errorf("empty cells kept: %d", len(g.cells))
	}
}
