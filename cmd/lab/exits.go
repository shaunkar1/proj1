//go:build ignore

package main

import (
	"fmt"
	"time"

	"mechbot/geo"
)

func main() {
	w, _ := geo.Load("mapdata/arena")
	t0 := time.Now()
	w.BuildNav(3.0, 9.5, 1.2)
	fmt.Println("nav+exits", time.Since(t0))
	nv := w.Nav
	for comp, ex := range nv.Exits {
		n := 0
		for c := range nv.Walk {
			if nv.Walk[c] && nv.Comp[c] == comp {
				n++
			}
		}
		be := ex[0]
		for _, e := range ex {
			if e.Energy < be.Energy {
				be = e
			}
		}
		if n > 100 {
			fmt.Printf("comp %d cells %d exits %d best: L(%.0f,%.0f) T(%.0f,%.0f) H %.1f E %.0f\n", comp, n, len(ex), be.LX, be.LY, be.TX, be.TY, be.H, be.Energy)
		}
	}
	fc := nv.Cell(-290, 138)
	fmt.Println("field comp", nv.Comp[fc], "exits", len(nv.Exits[nv.Comp[fc]]))
	n := 0
	tot := 0
	for c := range nv.Walk {
		if nv.Walk[c] && nv.Comp[c] != nv.Main {
			tot++
			if len(nv.Exits[nv.Comp[c]]) == 0 {
				n++
			}
		}
	}
	fmt.Println("non-main cells", tot, "without exits", n)
}
