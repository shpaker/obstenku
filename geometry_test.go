package obstenku

import "testing"

func TestOverlaps(t *testing.T) {
	wall := Rect{X: 10, Y: 0, W: 10, H: 10}
	tests := []struct {
		name string
		r    Rect
		want bool
	}{
		{"inside", Rect{X: 12, Y: 2, W: 4, H: 4}, true},
		{"crossing an edge", Rect{X: 8, Y: 2, W: 4, H: 4}, true},
		{"touching an edge", Rect{X: 6, Y: 2, W: 4, H: 4}, false},
		{"touching a corner", Rect{X: 6, Y: 10, W: 4, H: 4}, false},
		{"apart", Rect{X: 0, Y: 20, W: 4, H: 4}, false},
		{"sunk by rounding", Rect{X: 6 + 1e-12, Y: 2, W: 4, H: 4}, false},
		{"zero size", Rect{X: 15, Y: 5}, false},
	}
	for _, tt := range tests {
		if got := tt.r.Overlaps(wall); got != tt.want {
			t.Errorf("%s: Overlaps = %v, want %v", tt.name, got, tt.want)
		}
		if got := wall.Overlaps(tt.r); got != tt.want {
			t.Errorf("%s: reversed Overlaps = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestContainsPoint(t *testing.T) {
	r := Rect{X: 0, Y: 0, W: 8, H: 8}
	tests := []struct {
		p    Vec2
		want bool
	}{
		{Vec2{X: 0, Y: 0}, true},
		{Vec2{X: 4, Y: 4}, true},
		{Vec2{X: 8, Y: 4}, false},
		{Vec2{X: 4, Y: 8}, false},
		{Vec2{X: -1, Y: 4}, false},
	}
	for _, tt := range tests {
		if got := r.ContainsPoint(tt.p); got != tt.want {
			t.Errorf("ContainsPoint(%v) = %v, want %v", tt.p, got, tt.want)
		}
	}
}

func TestSweep(t *testing.T) {
	box := Rect{X: 0, Y: 0, W: 4, H: 4}
	wall := Rect{X: 10, Y: 0, W: 2, H: 10}
	tests := []struct {
		name  string
		r     Rect
		delta Vec2
		hit   bool
		want  Hit
	}{
		{
			name: "hits the face", r: box, delta: Vec2{X: 12},
			hit: true, want: Hit{Time: 0.5, Normal: Vec2{X: -1}, Touch: Vec2{X: 6}},
		},
		{
			name: "stops short", r: box, delta: Vec2{X: 5},
		},
		{
			name: "ends flush", r: box, delta: Vec2{X: 6},
		},
		{
			name: "passes a thin wall in one step", r: box, delta: Vec2{X: 1000},
			hit: true, want: Hit{Time: 0.006, Normal: Vec2{X: -1}, Touch: Vec2{X: 6}},
		},
		{
			name: "pushes from flush", r: box.At(Vec2{X: 6}), delta: Vec2{X: 1},
			hit: true, want: Hit{Normal: Vec2{X: -1}, Touch: Vec2{X: 6}},
		},
		{
			name: "leaves from flush", r: box.At(Vec2{X: 6}), delta: Vec2{X: -1},
		},
		{
			name: "slides along the face", r: box.At(Vec2{X: 6}), delta: Vec2{Y: 20},
		},
		{
			name: "from the right", r: box.At(Vec2{X: 20}), delta: Vec2{X: -20},
			hit: true, want: Hit{Time: 0.4, Normal: Vec2{X: 1}, Touch: Vec2{X: 12}},
		},
		{
			name: "from above", r: box.At(Vec2{X: 9, Y: -10}), delta: Vec2{Y: 10},
			hit: true, want: Hit{Time: 0.6, Normal: Vec2{Y: -1}, Touch: Vec2{X: 9, Y: -4}},
		},
		{
			name: "misses below", r: box.At(Vec2{Y: 20}), delta: Vec2{X: 20},
		},
		{
			name: "no move", r: box, delta: Vec2{},
		},
		{
			name: "overlapping, going deeper", r: box.At(Vec2{X: 7}), delta: Vec2{X: 1},
			hit: true, want: Hit{Normal: Vec2{X: -1}, Touch: Vec2{X: 7}, Overlapped: true},
		},
		{
			name: "overlapping, getting out", r: box.At(Vec2{X: 7}), delta: Vec2{X: -1},
			hit: true, want: Hit{Touch: Vec2{X: 7}, Overlapped: true},
		},
		{
			name: "overlapping, along the wall", r: box.At(Vec2{X: 7}), delta: Vec2{Y: 1},
			hit: true, want: Hit{Normal: Vec2{Y: -1}, Touch: Vec2{X: 7}, Overlapped: true},
		},
	}
	for _, tt := range tests {
		got, ok := Sweep(tt.r, tt.delta, wall)
		if ok != tt.hit {
			t.Errorf("%s: hit = %v, want %v (%+v)", tt.name, ok, tt.hit, got)
			continue
		}
		if ok && got != tt.want {
			t.Errorf("%s: %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

func TestSweepCornerPrefersX(t *testing.T) {
	hit, ok := Sweep(Rect{W: 4, H: 4}, Vec2{X: 10, Y: 10}, Rect{X: 8, Y: 8, W: 4, H: 4})
	if !ok || hit.Normal != (Vec2{X: -1}) || hit.Touch != (Vec2{X: 4, Y: 4}) {
		t.Errorf("corner: %+v, %v", hit, ok)
	}
}

func TestSegmentRect(t *testing.T) {
	r := Rect{X: 10, Y: 0, W: 10, H: 10}
	tests := []struct {
		name        string
		a, b        Vec2
		ok          bool
		enter, exit float64
	}{
		{"through", Vec2{X: 0, Y: 5}, Vec2{X: 40, Y: 5}, true, 0.25, 0.5},
		{"from inside", Vec2{X: 15, Y: 5}, Vec2{X: 25, Y: 5}, true, 0, 0.5},
		{"ends inside", Vec2{X: 0, Y: 5}, Vec2{X: 15, Y: 5}, true, 10.0 / 15, 1},
		{"along the edge", Vec2{X: 0, Y: 0}, Vec2{X: 40, Y: 0}, false, 0, 0},
		{"short", Vec2{X: 0, Y: 5}, Vec2{X: 10, Y: 5}, false, 0, 0},
		{"away", Vec2{X: 0, Y: 5}, Vec2{X: -10, Y: 5}, false, 0, 0},
	}
	for _, tt := range tests {
		enter, exit, ok := SegmentRect(tt.a, tt.b, r)
		if ok != tt.ok {
			t.Errorf("%s: ok = %v, want %v", tt.name, ok, tt.ok)
			continue
		}
		if ok && (!near(enter, tt.enter) || !near(exit, tt.exit)) {
			t.Errorf("%s: %v..%v, want %v..%v", tt.name, enter, exit, tt.enter, tt.exit)
		}
	}
}

func near(a, b float64) bool {
	d := a - b
	return d < 1e-6 && d > -1e-6
}
