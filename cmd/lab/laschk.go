//go:build ignore

package main

import (
	"fmt"
	"math"

	"mechbot/geo"
)

func main() {
	w, _ := geo.Load("/home/claude/kit/mechfight-kit/arena")
	w.SetMargin(30)
	path := [][4]float64{{0, 168, 84, 5}, {2, 190, 87, 5}, {4, 217, 94, 5}, {6, 238, 101, 5}, {18, 255, 89, 5}, {22, 267, 82, 5}, {24, 297, 72, 5}, {26, 324, 65, 5}, {28, 347, 64, 4}, {30, 367, 60, 4}, {32, 384, 55, 4}, {36, 395, 64, 4}, {40, 409, 67, 4}, {42, 422, 82, 4}, {50, 393, 97, 5}, {54, 392, 107, 4}, {64, 390, 78, 4}, {78, 380, 70, 4}, {80, 387, 39, 4}}
	las := [][3]float64{{221.4, -91.4, 5.1}, {271.9, -25.1, 4.5}, {110.4, -40.6, 4.5}, {369.4, -172.1, 2}, {454.4, -120.9, 1.9}, {79.5, -109.7, 3.8}}
	for _, p := range path {
		s := fmt.Sprintf("t=%2.0f (%.0f,%.0f):", p[0], p[1], p[2])
		for _, l := range las {
			d := math.Hypot(l[0]-p[1], l[1]-p[2])
			see := false; for _, up := range []float64{0, 15, 25, 40} { if d < 200 && w.Sees(p[1], p[2], p[3]+8+up, l[0], l[1], l[2]+1.0) { see = true; s += fmt.Sprintf(" up%.0f", up); break } }
			if see {
				s += fmt.Sprintf(" SEE(%.0f,%.0f d%.0f)", l[0], l[1], d)
			} else if d < 200 {
				s += fmt.Sprintf(" blk(d%.0f)", d)
			}
		}
		fmt.Println(s)
	}
}
