package obstenku

import (
	"cmp"
	"errors"
	"slices"

	"github.com/shpaker/obstenku/internal/grid"
)

var (
	// ErrExists: the item is in the world already.
	ErrExists = errors.New("obstenku: item is in the world already")
	// ErrUnknown: the item is not in the world.
	ErrUnknown = errors.New("obstenku: item is not in the world")
	// ErrInvalid: a coordinate is NaN or infinite, or a size is negative.
	ErrInvalid = errors.New("obstenku: invalid geometry")
)

// Collision is what a move met on its way.
type Collision[I comparable] struct {
	// Other is the item met.
	Other I
	// Response is what the filter chose for the meeting.
	Response Response
	Hit
}

// SegmentHit is an item a segment passes through.
type SegmentHit[I comparable] struct {
	Item I
	// Enter and Exit are the parts of the segment, from 0 to 1, where it
	// enters and leaves the item.
	Enter, Exit float64
}

// World holds the rectangles of the game's items and finds who meets whom.
// Items are keyed by any comparable value: a pointer to an entity, an id.
//
// A World is not safe for concurrent use. Filters must not change it.
type World[I comparable] struct {
	grid  *grid.Grid[I]
	items map[I]entry
	seq   uint64
	// Query scratch: which items a grid walk has seen
	seen map[I]struct{}
}

// entry is an item's rectangle and its place in the order of adding.
type entry struct {
	rect Rect
	seq  uint64
}

// NewWorld returns an empty world. The cell size tunes the speed of queries,
// not their results: about the size of a typical item, or the step of the
// game's grid, is a good choice. It panics if the size is not a positive
// finite number.
func NewWorld[I comparable](cellSize float64) *World[I] {
	if !(cellSize > 0) || !finite(cellSize) {
		panic("obstenku: cell size must be a positive finite number")
	}
	return &World[I]{
		grid:  grid.New[I](cellSize),
		items: make(map[I]entry),
		seen:  make(map[I]struct{}),
	}
}

// Items

// Add puts an item into the world.
func (w *World[I]) Add(item I, r Rect) error {
	if !r.valid() {
		return ErrInvalid
	}
	if _, ok := w.items[item]; ok {
		return ErrExists
	}
	w.seq++
	w.items[item] = entry{rect: r, seq: w.seq}
	w.grid.Insert(item, r.X, r.Y, r.W, r.H)
	return nil
}

// Update gives an item a new rectangle, without collisions: a teleport, a
// change of size.
func (w *World[I]) Update(item I, r Rect) error {
	if !r.valid() {
		return ErrInvalid
	}
	e, ok := w.items[item]
	if !ok {
		return ErrUnknown
	}
	if e.rect == r {
		return nil
	}
	w.grid.Delete(item, e.rect.X, e.rect.Y, e.rect.W, e.rect.H)
	w.grid.Insert(item, r.X, r.Y, r.W, r.H)
	e.rect = r
	w.items[item] = e
	return nil
}

// Remove takes an item out of the world.
func (w *World[I]) Remove(item I) error {
	e, ok := w.items[item]
	if !ok {
		return ErrUnknown
	}
	w.grid.Delete(item, e.rect.X, e.rect.Y, e.rect.W, e.rect.H)
	delete(w.items, item)
	return nil
}

// Rect returns the rectangle of an item and whether it is in the world.
func (w *World[I]) Rect(item I) (Rect, bool) {
	e, ok := w.items[item]
	return e.rect, ok
}

// Has reports whether the item is in the world.
func (w *World[I]) Has(item I) bool {
	_, ok := w.items[item]
	return ok
}

// Len returns the number of items in the world.
func (w *World[I]) Len() int {
	return len(w.items)
}

// Movement

// Check moves an item towards goal, the new top-left corner, on paper: it
// returns where the item would end up and what it would meet on the way,
// appended to dst in the order of meeting. The world does not change.
//
// Every item on the way gets the filter's response. Touch stops the move,
// Slide stops it along the axis hit, Cross lets it go on; each item is met
// once. An item that already overlaps the moving one stops only a move that
// goes deeper into it, so the item can always get out.
func (w *World[I]) Check(
	item I,
	goal Vec2,
	filter Filter[I],
	dst []Collision[I],
) (Vec2, []Collision[I], error) {
	e, ok := w.items[item]
	if !ok {
		return goal, dst, ErrUnknown
	}
	if !finite(goal.X) || !finite(goal.Y) {
		return goal, dst, ErrInvalid
	}

	r := e.rect
	met := []I{item}
	var candidates []I
	for {
		delta := Vec2{X: goal.X - r.X, Y: goal.Y - r.Y}
		candidates = w.candidates(r.union(r.At(goal)), candidates[:0])

		var (
			first      Collision[I]
			firstEntry entry
			found      bool
		)
		for _, other := range candidates {
			if slices.Contains(met, other) {
				continue
			}
			response := Slide
			if filter != nil {
				response = filter(item, other)
			}
			if response == Ignore {
				continue
			}
			o := w.items[other]
			hit, ok := Sweep(r, delta, o.rect)
			if !ok {
				continue
			}
			// An overlap the move leads out of stops nothing
			if response != Cross && hit.Overlapped && hit.Normal == (Vec2{}) {
				continue
			}
			if found && !sooner(hit, o, first.Hit, firstEntry) {
				continue
			}
			first = Collision[I]{Other: other, Response: response, Hit: hit}
			firstEntry, found = o, true
		}
		if !found {
			return goal, dst, nil
		}

		met = append(met, first.Other)
		dst = append(dst, first)
		switch first.Response {
		case Touch:
			r = r.At(first.Touch)
			goal = first.Touch
		case Slide:
			r = r.At(first.Touch)
			if first.Normal.X != 0 {
				goal.X = first.Touch.X
			}
			if first.Normal.Y != 0 {
				goal.Y = first.Touch.Y
			}
		}
	}
}

// Move moves an item towards goal like Check does and puts it where it ends
// up.
func (w *World[I]) Move(
	item I,
	goal Vec2,
	filter Filter[I],
	dst []Collision[I],
) (Vec2, []Collision[I], error) {
	pos, dst, err := w.Check(item, goal, filter, dst)
	if err != nil {
		return pos, dst, err
	}
	return pos, dst, w.Update(item, w.items[item].rect.At(pos))
}

// sooner orders meetings: by time, then by the place of the other item, so
// the order of adding never changes where a move ends; identical rectangles
// go in the order of adding.
func sooner(a Hit, ae entry, b Hit, be entry) bool {
	if c := cmp.Compare(a.Time, b.Time); c != 0 {
		return c < 0
	}
	if c := compareRects(ae.rect, be.rect); c != 0 {
		return c < 0
	}
	return ae.seq < be.seq
}

func compareRects(a, b Rect) int {
	if c := cmp.Compare(a.X, b.X); c != 0 {
		return c
	}
	if c := cmp.Compare(a.Y, b.Y); c != 0 {
		return c
	}
	if c := cmp.Compare(a.W, b.W); c != 0 {
		return c
	}
	return cmp.Compare(a.H, b.H)
}

// Queries

// QueryRect appends to dst the items that overlap r and that keep returns
// true for (all of them for a nil keep), in the order of adding.
func (w *World[I]) QueryRect(r Rect, keep func(I) bool, dst []I) []I {
	start := len(dst)
	dst = w.candidates(r, dst)
	found := dst[:start]
	for _, item := range dst[start:] {
		if w.items[item].rect.Overlaps(r) && (keep == nil || keep(item)) {
			found = append(found, item)
		}
	}
	w.sortBySeq(found[start:])
	return found
}

// QueryPoint appends to dst the items that contain p (see
// [Rect.ContainsPoint]) and that keep returns true for (all of them for a
// nil keep), in the order of adding.
func (w *World[I]) QueryPoint(p Vec2, keep func(I) bool, dst []I) []I {
	start := len(dst)
	dst = w.candidates(Rect{X: p.X, Y: p.Y}, dst)
	found := dst[:start]
	for _, item := range dst[start:] {
		if w.items[item].rect.ContainsPoint(p) && (keep == nil || keep(item)) {
			found = append(found, item)
		}
	}
	w.sortBySeq(found[start:])
	return found
}

// QuerySegment appends to dst the items the segment from a to b passes
// through (see [SegmentRect]) and that keep returns true for (all of them
// for a nil keep), nearest to a first: a line of sight, a laser.
func (w *World[I]) QuerySegment(
	a, b Vec2,
	keep func(I) bool,
	dst []SegmentHit[I],
) []SegmentHit[I] {
	start := len(dst)
	box := Rect{X: a.X, Y: a.Y}.union(Rect{X: b.X, Y: b.Y})
	for _, item := range w.candidates(box, nil) {
		enter, exit, ok := SegmentRect(a, b, w.items[item].rect)
		if ok && (keep == nil || keep(item)) {
			dst = append(dst, SegmentHit[I]{Item: item, Enter: enter, Exit: exit})
		}
	}
	slices.SortFunc(dst[start:], func(p, q SegmentHit[I]) int {
		if c := cmp.Compare(p.Enter, q.Enter); c != 0 {
			return c
		}
		return cmp.Compare(w.items[p.Item].seq, w.items[q.Item].seq)
	})
	return dst
}

// candidates appends to dst every item in the cells r touches, once.
func (w *World[I]) candidates(r Rect, dst []I) []I {
	clear(w.seen)
	w.grid.Visit(r.X, r.Y, r.W, r.H, func(item I) {
		if _, ok := w.seen[item]; ok {
			return
		}
		w.seen[item] = struct{}{}
		dst = append(dst, item)
	})
	return dst
}

func (w *World[I]) sortBySeq(items []I) {
	slices.SortFunc(items, func(p, q I) int {
		return cmp.Compare(w.items[p].seq, w.items[q].seq)
	})
}
