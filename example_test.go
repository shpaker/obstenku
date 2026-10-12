package obstenku_test

import (
	"fmt"

	"github.com/shpaker/obstenku"
)

func Example() {
	type item string
	world := obstenku.NewWorld[item](16) // cell size: about one item

	_ = world.Add("tank", obstenku.Rect{X: 0, Y: 0, W: 16, H: 16})
	_ = world.Add("brick", obstenku.Rect{X: 40, Y: 0, W: 4, H: 16})
	_ = world.Add("star", obstenku.Rect{X: 20, Y: 4, W: 8, H: 8})

	// The game decides how its items meet
	filter := func(_, other item) obstenku.Response {
		if other == "star" {
			return obstenku.Cross // picked up on the way
		}
		return obstenku.Slide
	}

	// However far the tank goes in one tick, it stops at the brick
	pos, cols, _ := world.Move("tank", obstenku.Vec2{X: 1000, Y: 0}, filter, nil)
	fmt.Println("tank at", pos)
	for _, c := range cols {
		fmt.Println("met", c.Other, c.Response)
	}
	// Output:
	// tank at {24 0}
	// met star Cross
	// met brick Slide
}
