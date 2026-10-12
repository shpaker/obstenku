package obstenku

import "math"

// Rect is an axis-aligned rectangle: the top-left corner and the size. Y
// grows downwards, as on the screen.
type Rect struct {
	X, Y, W, H float64
}

// Pos returns the top-left corner.
func (r Rect) Pos() Vec2 {
	return Vec2{X: r.X, Y: r.Y}
}

// At returns the same rectangle with its top-left corner at p.
func (r Rect) At(p Vec2) Rect {
	r.X, r.Y = p.X, p.Y
	return r
}

// Translate returns the rectangle moved by d.
func (r Rect) Translate(d Vec2) Rect {
	r.X += d.X
	r.Y += d.Y
	return r
}

// Center returns the middle of the rectangle.
func (r Rect) Center() Vec2 {
	return Vec2{X: r.X + r.W/2, Y: r.Y + r.H/2}
}

// Overlaps reports whether the rectangles share some area. Rectangles that
// only touch with an edge or a corner do not overlap, and neither does a
// rectangle of zero width or height.
func (r Rect) Overlaps(o Rect) bool {
	return math.Min(r.X+r.W, o.X+o.W)-math.Max(r.X, o.X) > eps &&
		math.Min(r.Y+r.H, o.Y+o.H)-math.Max(r.Y, o.Y) > eps
}

// ContainsPoint reports whether p lies inside the rectangle. The top and
// left edges belong to it, the bottom and right ones do not, so a point on
// a grid line belongs to exactly one cell of a grid.
func (r Rect) ContainsPoint(p Vec2) bool {
	return r.X <= p.X && p.X < r.X+r.W && r.Y <= p.Y && p.Y < r.Y+r.H
}

// union returns the smallest rectangle that holds both.
func (r Rect) union(o Rect) Rect {
	x0, y0 := math.Min(r.X, o.X), math.Min(r.Y, o.Y)
	x1, y1 := math.Max(r.X+r.W, o.X+o.W), math.Max(r.Y+r.H, o.Y+o.H)
	return Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

// valid reports whether the rectangle has finite coordinates and a size that
// is not negative.
func (r Rect) valid() bool {
	return finite(r.X) && finite(r.Y) && finite(r.W) && finite(r.H) &&
		r.W >= 0 && r.H >= 0
}

// Hit is the meeting of a moving rectangle with another one.
type Hit struct {
	// Time is the part of the move done when the rectangles met, from 0 to 1.
	Time float64
	// Normal points away from the other rectangle along the axis the moving
	// one hit: (-1, 0) for a hit while moving right. For rectangles that
	// overlapped before the move it has a component for every axis the move
	// goes deeper along, and is zero when the move leads out.
	Normal Vec2
	// Touch is where the moving rectangle's top-left corner is at Time.
	Touch Vec2
	// Overlapped tells that the rectangles overlapped before the move.
	Overlapped bool
}

// Sweep moves r by delta and reports whether it meets target on the way, and
// how. Rectangles that only touch do not meet: r slides freely along a face
// of target. For rectangles that already overlap, the hit is at Time 0. A
// rectangle of zero width or height meets nothing.
func Sweep(r Rect, delta Vec2, target Rect) (Hit, bool) {
	if r.W <= eps || r.H <= eps || target.W <= eps || target.H <= eps {
		return Hit{}, false
	}
	// Minkowski difference: the top-left corner of r against target grown
	// by the size of r, minus the touching tolerance
	x0, x1 := target.X-r.W+eps, target.X+target.W-eps
	y0, y1 := target.Y-r.H+eps, target.Y+target.H-eps
	p := r.Pos()

	if p.X > x0 && p.X < x1 && p.Y > y0 && p.Y < y1 {
		// Deeper is towards the middle of target
		c, tc := r.Center(), target.Center()
		var n Vec2
		if delta.X > 0 && c.X < tc.X || delta.X < 0 && c.X > tc.X {
			n.X = -math.Copysign(1, delta.X)
		}
		if delta.Y > 0 && c.Y < tc.Y || delta.Y < 0 && c.Y > tc.Y {
			n.Y = -math.Copysign(1, delta.Y)
		}
		return Hit{Normal: n, Touch: p, Overlapped: true}, true
	}

	enterX, exitX, ok := slab(p.X, delta.X, x0, x1)
	if !ok {
		return Hit{}, false
	}
	enterY, exitY, ok := slab(p.Y, delta.Y, y0, y1)
	if !ok {
		return Hit{}, false
	}
	enter, exit := math.Max(enterX, enterY), math.Min(exitX, exitY)
	if enter >= exit || enter < 0 || enter >= 1 {
		return Hit{}, false
	}

	// The axis entered last is the one hit; on a tie, X. The touch lies
	// exactly on the face, so rounding never sinks it in, and the time is
	// the one of reaching the face, not the tolerance before it.
	var hit Hit
	if enterX >= enterY {
		face := target.X - r.W
		if delta.X < 0 {
			face = target.X + target.W
		}
		hit.Time = math.Max(0, (face-p.X)/delta.X)
		hit.Normal.X = -math.Copysign(1, delta.X)
		hit.Touch = Vec2{X: face, Y: p.Y + delta.Y*hit.Time}
	} else {
		face := target.Y - r.H
		if delta.Y < 0 {
			face = target.Y + target.H
		}
		hit.Time = math.Max(0, (face-p.Y)/delta.Y)
		hit.Normal.Y = -math.Copysign(1, delta.Y)
		hit.Touch = Vec2{X: p.X + delta.X*hit.Time, Y: face}
	}
	return hit, true
}

// SegmentRect reports whether the segment from a to b passes through the
// inside of r, and the parts of the way, from 0 to 1, where it enters and
// leaves it. A segment that only runs along an edge does not pass through.
func SegmentRect(a, b Vec2, r Rect) (enter, exit float64, ok bool) {
	x0, x1 := r.X+eps, r.X+r.W-eps
	y0, y1 := r.Y+eps, r.Y+r.H-eps
	if x0 >= x1 || y0 >= y1 {
		return 0, 0, false
	}
	enterX, exitX, okX := slab(a.X, b.X-a.X, x0, x1)
	enterY, exitY, okY := slab(a.Y, b.Y-a.Y, y0, y1)
	if !okX || !okY {
		return 0, 0, false
	}
	enter = math.Max(0, math.Max(enterX, enterY))
	exit = math.Min(1, math.Min(exitX, exitY))
	if enter >= exit {
		return 0, 0, false
	}
	return enter, exit, true
}

// slab returns the times a point moving from p by d is strictly between lo
// and hi. A point that does not move is there always or never.
func slab(p, d, lo, hi float64) (enter, exit float64, ok bool) {
	if d == 0 {
		if p > lo && p < hi {
			return math.Inf(-1), math.Inf(1), true
		}
		return 0, 0, false
	}
	t0, t1 := (lo-p)/d, (hi-p)/d
	return math.Min(t0, t1), math.Max(t0, t1), true
}

func finite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}
