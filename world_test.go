package obstenku

import (
	"errors"
	"slices"
	"testing"
)

func newTestWorld(t *testing.T, rects map[string]Rect) *World[string] {
	t.Helper()
	w := NewWorld[string](8)
	names := make([]string, 0, len(rects))
	for name := range rects {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		if err := w.Add(name, rects[name]); err != nil {
			t.Fatal(err)
		}
	}
	return w
}

func responses(byName map[string]Response) Filter[string] {
	return func(_, other string) Response {
		if r, ok := byName[other]; ok {
			return r
		}
		return Slide
	}
}

func others(cols []Collision[string]) []string {
	names := make([]string, len(cols))
	for i, c := range cols {
		names[i] = c.Other
	}
	return names
}

func TestItems(t *testing.T) {
	w := NewWorld[int](16)
	if err := w.Add(1, Rect{W: 4, H: 4}); err != nil {
		t.Fatal(err)
	}
	if err := w.Add(1, Rect{W: 4, H: 4}); !errors.Is(err, ErrExists) {
		t.Errorf("Add twice: %v", err)
	}
	if err := w.Add(2, Rect{W: -1, H: 4}); !errors.Is(err, ErrInvalid) {
		t.Errorf("Add with a negative size: %v", err)
	}
	if err := w.Update(3, Rect{}); !errors.Is(err, ErrUnknown) {
		t.Errorf("Update unknown: %v", err)
	}
	if err := w.Update(1, Rect{X: 100, W: 4, H: 4}); err != nil {
		t.Fatal(err)
	}
	if got := w.QueryPoint(Vec2{X: 1, Y: 1}, nil, nil); len(got) != 0 {
		t.Errorf("old place still holds %v", got)
	}
	if got := w.QueryPoint(Vec2{X: 101, Y: 1}, nil, nil); !slices.Equal(got, []int{1}) {
		t.Errorf("new place holds %v", got)
	}
	if err := w.Remove(1); err != nil || w.Has(1) || w.Len() != 0 {
		t.Errorf("Remove: %v, has %v, len %d", err, w.Has(1), w.Len())
	}
	if _, _, err := w.Move(1, Vec2{}, nil, nil); !errors.Is(err, ErrUnknown) {
		t.Errorf("Move removed: %v", err)
	}
}

func TestMoveResponses(t *testing.T) {
	tests := []struct {
		name     string
		response Response
		goal     Vec2
		want     Vec2
		met      []string
	}{
		{"touch stops", Touch, Vec2{X: 40, Y: 10}, Vec2{X: 16, Y: 4}, []string{"wall"}},
		{"slide keeps the other axis", Slide, Vec2{X: 40, Y: 10}, Vec2{X: 16, Y: 10}, []string{"wall"}},
		{"cross passes", Cross, Vec2{X: 40, Y: 10}, Vec2{X: 40, Y: 10}, []string{"wall"}},
		{"ignore passes silently", Ignore, Vec2{X: 40, Y: 10}, Vec2{X: 40, Y: 10}, nil},
	}
	for _, tt := range tests {
		w := newTestWorld(t, map[string]Rect{
			"box":  {X: 0, Y: 0, W: 4, H: 4},
			"wall": {X: 20, Y: -100, W: 4, H: 200},
		})
		filter := responses(map[string]Response{"wall": tt.response})
		pos, cols, err := w.Move("box", tt.goal, filter, nil)
		if err != nil {
			t.Fatal(err)
		}
		if pos != tt.want {
			t.Errorf("%s: at %v, want %v", tt.name, pos, tt.want)
		}
		if got := others(cols); !slices.Equal(got, tt.met) {
			t.Errorf("%s: met %v, want %v", tt.name, got, tt.met)
		}
		if r, _ := w.Rect("box"); r.Pos() != pos {
			t.Errorf("%s: world keeps %v, move ended at %v", tt.name, r.Pos(), pos)
		}
	}
}

func TestMoveThroughThinWallInOneStep(t *testing.T) {
	w := newTestWorld(t, map[string]Rect{
		"bullet": {X: 0, Y: 0, W: 4, H: 4},
		"brick":  {X: 100, Y: 0, W: 1, H: 4},
	})
	pos, cols, _ := w.Move("bullet", Vec2{X: 10000}, nil, nil)
	if pos != (Vec2{X: 96}) || !slices.Equal(others(cols), []string{"brick"}) {
		t.Errorf("at %v, met %v", pos, others(cols))
	}
}

func TestMoveMeetsInOrder(t *testing.T) {
	w := newTestWorld(t, map[string]Rect{
		"box":   {X: 0, Y: 0, W: 4, H: 4},
		"coin":  {X: 10, Y: 0, W: 4, H: 4},
		"far":   {X: 40, Y: 0, W: 4, H: 4},
		"brick": {X: 20, Y: 0, W: 4, H: 4},
	})
	filter := responses(map[string]Response{"coin": Cross, "brick": Touch, "far": Touch})
	pos, cols, _ := w.Move("box", Vec2{X: 100}, filter, nil)
	if pos != (Vec2{X: 16}) || !slices.Equal(others(cols), []string{"coin", "brick"}) {
		t.Errorf("at %v, met %v", pos, others(cols))
	}
}

func TestMoveSlidesAlongTwoWalls(t *testing.T) {
	// A corridor of two blocks with a seam between them
	w := newTestWorld(t, map[string]Rect{
		"box": {X: 0, Y: 0, W: 4, H: 4},
		"a":   {X: 4, Y: 0, W: 4, H: 8},
		"b":   {X: 4, Y: 8, W: 4, H: 8},
	})
	pos, cols, _ := w.Move("box", Vec2{X: 2, Y: 12}, nil, nil)
	if pos != (Vec2{X: 0, Y: 12}) || len(cols) != 1 {
		t.Errorf("at %v, met %v", pos, others(cols))
	}
}

func TestMoveOutOfOverlap(t *testing.T) {
	w := newTestWorld(t, map[string]Rect{
		"spawned": {X: 0, Y: 0, W: 16, H: 16},
		"tank":    {X: 8, Y: 0, W: 16, H: 16},
	})
	pos, cols, _ := w.Check("spawned", Vec2{X: 4}, nil, nil)
	if pos != (Vec2{X: 0}) || len(cols) != 1 || !cols[0].Overlapped {
		t.Errorf("deeper: at %v, met %+v", pos, cols)
	}
	pos, cols, _ = w.Check("spawned", Vec2{X: -4}, nil, nil)
	if pos != (Vec2{X: -4}) || len(cols) != 0 {
		t.Errorf("out: at %v, met %+v", pos, cols)
	}
	pos, cols, _ = w.Check("spawned", Vec2{}, responses(map[string]Response{"tank": Cross}), nil)
	if pos != (Vec2{}) || !slices.Equal(others(cols), []string{"tank"}) {
		t.Errorf("standing on a bonus: at %v, met %v", pos, others(cols))
	}
}

func TestMoveAcrossCells(t *testing.T) {
	w := NewWorld[string](8)
	_ = w.Add("box", Rect{X: -100, Y: 3, W: 2, H: 2})
	_ = w.Add("wall", Rect{X: 70, Y: 0, W: 1, H: 8})
	pos, _, _ := w.Move("box", Vec2{X: 300, Y: 3}, nil, nil)
	if pos != (Vec2{X: 68, Y: 3}) {
		t.Errorf("at %v", pos)
	}
}

func TestCheckLeavesWorld(t *testing.T) {
	w := newTestWorld(t, map[string]Rect{"box": {W: 4, H: 4}})
	if _, _, err := w.Check("box", Vec2{X: 50}, nil, nil); err != nil {
		t.Fatal(err)
	}
	if r, _ := w.Rect("box"); r.Pos() != (Vec2{}) {
		t.Errorf("Check moved the item to %v", r.Pos())
	}
}

func TestQueries(t *testing.T) {
	w := newTestWorld(t, map[string]Rect{
		"a": {X: 0, Y: 0, W: 8, H: 8},
		"b": {X: 8, Y: 0, W: 8, H: 8},
		"c": {X: 32, Y: 0, W: 8, H: 8},
	})
	if got := w.QueryRect(Rect{X: 4, Y: 4, W: 8, H: 2}, nil, nil); !slices.Equal(got, []string{"a", "b"}) {
		t.Errorf("QueryRect: %v", got)
	}
	notA := func(s string) bool { return s != "a" }
	if got := w.QueryRect(Rect{X: 4, Y: 4, W: 8, H: 2}, notA, nil); !slices.Equal(got, []string{"b"}) {
		t.Errorf("QueryRect with keep: %v", got)
	}
	if got := w.QueryPoint(Vec2{X: 8, Y: 4}, nil, nil); !slices.Equal(got, []string{"b"}) {
		t.Errorf("QueryPoint on a seam: %v", got)
	}
	hits := w.QuerySegment(Vec2{X: 100, Y: 4}, Vec2{X: 4, Y: 4}, nil, nil)
	var names []string
	for _, h := range hits {
		names = append(names, h.Item)
	}
	if !slices.Equal(names, []string{"c", "b", "a"}) {
		t.Errorf("QuerySegment: %v", names)
	}
	if got := w.QueryRect(Rect{X: -1e6, Y: -1e6, W: 2e6, H: 2e6}, nil, []string{"x"}); !slices.Equal(got, []string{"x", "a", "b", "c"}) {
		t.Errorf("QueryRect over everything, appended: %v", got)
	}
}
