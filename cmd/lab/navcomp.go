//go:build ignore

package main

import (
	"fmt"

	"mechbot/geo"
)

func main() {
	w, _ := geo.Load("mapdata/arena")
	for _, cl := range []float64{2.6, 3.0, 3.9} {
		w.BuildNav(cl, 9.5, 1.2)
		nv := w.Nav
		nw, nm := 0, 0
		for c := range nv.Walk {
			if nv.Walk[c] {
				nw++
				if nv.Comp[c] == nv.Main {
					nm++
				}
			}
		}
		fmt.Printf("clear %.1f walk %d main %d stadium field main=%v\n", cl, nw, nm, nv.WalkableMain(-276, 128))
	}
}
