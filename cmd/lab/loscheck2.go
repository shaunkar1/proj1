//go:build ignore

package main

import (
	"fmt"

	"mechbot/geo"
)

func main() {
	w, _ := geo.Load("mapdata/arena")
	me := [3]float64{-105, -33, -5 + 4.5}
	for _, e := range [][3]float64{{-73, 56, -1 + 7}, {-71, 45, -2 + 7}, {-82, 22, -4 + 7}} {
		t, hit := w.RayHit(e[0], e[1], e[2], me[0], me[1], me[2])
		x, y, z := e[0]+(me[0]-e[0])*t, e[1]+(me[1]-e[1])*t, e[2]+(me[2]-e[2])*t
		fmt.Printf("from %v hit=%v at (%.1f,%.1f,%.1f) ground %.2f intervals %v\n", e, hit, x, y, z, w.Ground(x, y), w.Intervals(x, y))
	}
}
