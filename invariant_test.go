package obstenku

import (
	"math"
	"math/rand/v2"
	"testing"
)

// FuzzNoTunneling moves a box through random walls and checks the promise
// of the library: whatever the move, the box never passes through a wall
// that stops it, and never goes deeper into one it started inside.
func FuzzNoTunneling(f *testing.F) {
	f.Add(uint64(1), uint8(0))
	f.Add(uint64(2), uint8(1))
	f.Add(uint64(3), uint8(2))
	f.Fuzz(func(t *testing.T, seed uint64, kind uint8) {
		rng := rand.New(rand.NewPCG(seed, uint64(kind)))
		w, rects, responses := randomWorld(rng, kind)
		filter := func(_, other int) Response { return responses[other] }

		box := rects[0]
		goal := Vec2{X: coordinate(rng, kind) * 4, Y: coordinate(rng, kind) * 4}
		if kind%3 == 0 {
			// Four-way steering: one axis at a time
			goal.Y = box.Y
		}
		pos, cols, err := w.Move(0, goal, filter, nil)
		if err != nil {
			t.Fatal(err)
		}

		// The path is a polyline: from the start through every stop
		path := []Vec2{box.Pos()}
		for _, c := range cols {
			if c.Response == Touch || c.Response == Slide {
				path = append(path, c.Touch)
			}
		}
		path = append(path, pos)

		for i := 1; i < len(path); i++ {
			from, to := path[i-1], path[i]
			steps := int(math.Ceil(math.Hypot(to.X-from.X, to.Y-from.Y)*8)) + 1
			for s := 0; s <= steps; s++ {
				k := float64(s) / float64(steps)
				at := box.At(Vec2{X: from.X + (to.X-from.X)*k, Y: from.Y + (to.Y-from.Y)*k})
				for id := 1; id < len(rects); id++ {
					r := responses[id]
					if r != Touch && r != Slide {
						continue
					}
					was := overlapArea(box, rects[id])
					if now := overlapArea(at, rects[id]); now > was+1e-6 {
						t.Fatalf("goal %v: at %v the box sinks into wall %d %v (%v > %v), path %v",
							goal, at.Pos(), id, rects[id], now, was, path)
					}
				}
			}
		}
	})
}

// FuzzSameInAnyOrder adds the same walls in two orders and checks that a
// move ends at the same place: the order of adding never changes a game.
func FuzzSameInAnyOrder(f *testing.F) {
	f.Add(uint64(1), uint8(0))
	f.Add(uint64(2), uint8(1))
	f.Fuzz(func(t *testing.T, seed uint64, kind uint8) {
		rng := rand.New(rand.NewPCG(seed, uint64(kind)))
		w, rects, responses := randomWorld(rng, kind)
		filter := func(_, other int) Response { return responses[other] }

		reversed := NewWorld[int](float64(1 + rng.IntN(32)))
		for id := len(rects) - 1; id >= 0; id-- {
			_ = reversed.Add(id, rects[id])
		}

		goal := Vec2{X: coordinate(rng, kind) * 4, Y: coordinate(rng, kind) * 4}
		pos, cols, _ := w.Check(0, goal, filter, nil)
		pos2, cols2, _ := reversed.Check(0, goal, filter, nil)
		if pos != pos2 || len(cols) != len(cols2) {
			t.Fatalf("goal %v: %v after %d meetings, %v after %d in the other order",
				goal, pos, len(cols), pos2, len(cols2))
		}
	})
}

// randomWorld builds a box (item 0) and walls around it on a grid of
// quarter pixels or at any fractions, with a random response for each wall.
func randomWorld(rng *rand.Rand, kind uint8) (*World[int], []Rect, []Response) {
	w := NewWorld[int](float64(1 + rng.IntN(32)))
	n := 1 + rng.IntN(40)
	rects := make([]Rect, 0, n+1)
	responses := make([]Response, 0, n+1)
	for id := 0; id <= n; id++ {
		r := Rect{
			X: coordinate(rng, kind),
			Y: coordinate(rng, kind),
			W: size(rng, kind),
			H: size(rng, kind),
		}
		rects = append(rects, r)
		responses = append(responses, Response(rng.IntN(4)))
		_ = w.Add(id, r)
	}
	return w, rects, responses
}

func coordinate(rng *rand.Rand, kind uint8) float64 {
	if kind%2 == 0 {
		return float64(rng.IntN(400)-200) / 4
	}
	return rng.Float64()*100 - 50
}

func size(rng *rand.Rand, kind uint8) float64 {
	if kind%2 == 0 {
		return float64(1+rng.IntN(64)) / 4
	}
	return 0.25 + rng.Float64()*16
}

func overlapArea(a, b Rect) float64 {
	x := math.Min(a.X+a.W, b.X+b.W) - math.Max(a.X, b.X)
	y := math.Min(a.Y+a.H, b.Y+b.H) - math.Max(a.Y, b.Y)
	if x <= 0 || y <= 0 {
		return 0
	}
	return x * y
}
