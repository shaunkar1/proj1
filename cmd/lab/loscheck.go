//go:build ignore

package main

import (
	"fmt"

	"mechbot/geo"
)

func main() {
	w, _ := geo.Load("mapdata/arena")
	w.SetMargin(30)
	w.BuildNav(3.0, 9.5, 1.2)
	me := [3]float64{-105, -33, -5}
	for _, e := range [][3]float64{{-70, 72, -1}, {-73, 56, -1}, {-71, 45, -2}, {-88, 36, -4}, {-82, 22, -4}} {
		fmt.Printf("enemy %v: shot clear me->e %v e->me %v | sees me->e %v e->me %v | ground me %.1f walk %v\n", e,
			w.Clear(me[0], me[1], me[2]+7, e[0], e[1], e[2]+4.5), w.Clear(e[0], e[1], e[2]+7, me[0], me[1], me[2]+4.5),
			w.Sees(me[0], me[1], me[2]+8, e[0], e[1], e[2]+4.5) || w.Sees(me[0], me[1], me[2]+8, e[0], e[1], e[2]+9),
			w.Sees(e[0], e[1], e[2]+8, me[0], me[1], me[2]+4.5) || w.Sees(e[0], e[1], e[2]+8, me[0], me[1], me[2]+9), w.Ground(me[0], me[1]), w.Nav.Walkable(me[0], me[1]))
	}
}
