// Package obstenku stops boxes at walls.
//
// It finds and resolves collisions of axis-aligned rectangles, the way
// classic grid games need it: a box that moves is swept along its whole
// path, so it never slips through a thin wall however far it goes in one
// tick, and it stops flush against what it hits.
//
// The game keeps its boxes in a [World] under any comparable key, moves them
// with [World.Move] and decides, pair by pair, how they meet with a [Filter]:
// stop at the wall ([Touch]), slide along it ([Slide]), pass through and be
// told ([Cross]) or not meet at all ([Ignore]). Queries find what lies in a
// rectangle, at a point or on a segment.
//
// Speeds, forces, input and drawing stay with the game. Nothing here keeps
// global state, reads the clock or draws random numbers: the same calls give
// the same results.
package obstenku

// eps is how deep two boxes may sink into each other and still only touch.
// It hides the rounding of fractional coordinates, so a box resting flush
// against a wall never counts as overlapping it.
const eps = 1e-9

// Vec2 is a point or a vector.
type Vec2 struct {
	X, Y float64
}

// Response is how a moving box meets another one.
type Response uint8

const (
	// Ignore: the boxes do not meet; nothing is reported.
	Ignore Response = iota
	// Touch: the box stops where it touches the other one.
	Touch
	// Slide: the box stops along the axis it hit and keeps moving along the
	// other one, as when it rubs along a wall.
	Slide
	// Cross: the box passes through; the meeting is reported.
	Cross
)

// String returns the name of the response.
func (r Response) String() string {
	switch r {
	case Ignore:
		return "Ignore"
	case Touch:
		return "Touch"
	case Slide:
		return "Slide"
	case Cross:
		return "Cross"
	}
	return "Response(?)"
}

// Filter tells how the moving item meets another one. A nil Filter slides
// along everything.
type Filter[I comparable] func(item, other I) Response
