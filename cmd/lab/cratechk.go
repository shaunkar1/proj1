//go:build ignore

package main

import (
	"fmt"

	"mechbot/geo"
)

func main() {
	w, err := geo.Load("/home/claude/kit/mechfight-kit/arena")
	if err != nil {
		w, err = geo.Load("/home/claude/mech/mapdata/arena")
	}
	if err != nil {
		panic(err)
	}
	w.SetMargin(30)
	cx, cy, cz := 236.73, 101.05, 4.96
	fmt.Printf("ground at crate %.2f  intervals %v\n", w.Ground(cx, cy), w.Intervals(cx, cy))
	pts := [][3]float64{{190.22, 86.51, 5.11}, {205.57, 90.45, 4.94}, {216.67, 94.06, 4.83}, {227.64, 98.10, 4.89}, {233.05, 100.34, 4.92}, {238.5, 100.86, 4.99}}
	for _, p := range pts {
		for _, dz := range []float64{0.3, 1.2, 2.0} {
			fmt.Printf("from (%.1f,%.1f) dz %.1f: clear=%v sees=%v ground@me %.2f\n", p[0], p[1], dz,
				w.Clear(p[0], p[1], p[2]+7, cx, cy, cz+dz), w.Sees(p[0], p[1], p[2]+8, cx, cy, cz+dz), w.Ground(p[0], p[1]))
		}
	}
}
