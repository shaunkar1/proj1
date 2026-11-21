//go:build ignore

package main

import (
	"fmt"
	"math"

	"mechbot/geo"
)

func main() {
	w, err := geo.Load("/home/claude/kit/mechfight-kit/arena")
	if err != nil {
		panic(err)
	}
	w.SetMargin(30)
	w.BuildNav(3.0, 9.5, 1.2)
	px, py := -188.0, -178.0
	ex, ey, ez := -300.0, -210.0, -5.3+8
	fmt.Printf("ground me %.1f\n", w.Ground(px, py))
	best := 1e9
	n := 0
	for r := 4.0; r <= 120; r += 4 {
		for a := 0.0; a < 2*math.Pi; a += math.Pi / 18 {
			x, y := px+r*math.Cos(a), py+r*math.Sin(a)
			if !w.Nav.WalkableMain(x, y) {
				continue
			}
			g := w.Ground(x, y)
			hid := true
			for _, h := range []float64{1, 4.5, 8.5} {
				for _, o := range []float64{-2.2, 0, 2.2} {
					// lateral offsets across the line of sight
					dx, dy := x-ex, y-ey
					l := math.Hypot(dx, dy)
					ox, oy := -dy/l*o, dx/l*o
					if w.Clear(ex, ey, ez, x+ox, y+oy, g+h) {
						hid = false
					}
				}
			}
			if hid {
				n++
				if r < best {
					best = r
					fmt.Printf("nearest hidden spot r=%.0f at (%.0f,%.0f) bearing %.0f deg\n", r, x, y, a*180/math.Pi)
				}
				if n < 12 {
					fmt.Printf("  hidden (%.0f,%.0f) r=%.0f\n", x, y, r)
				}
			}
		}
	}
	fmt.Println("hidden spots:", n)
}
