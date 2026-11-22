//go:build ignore

package main

import (
	"fmt"
	"os"
	"strconv"

	"mechbot/geo"
)

func main() {
	w, _ := geo.Load("mapdata/arena")
	w.SetMargin(30)
	w.BuildNav(3.0, 9.5, 1.2)
	nv := w.Nav
	for i := 2; i+1 < len(os.Args); i += 2 {
		x, _ := strconv.ParseFloat(os.Args[i], 64)
		y, _ := strconv.ParseFloat(os.Args[i+1], 64)
		c := nv.Cell(x, y)
		fmt.Printf("(%.0f,%.0f) cell %d ground %.1f", x, y, c, w.Ground(x, y))
		if c >= 0 {
			fmt.Printf(" walk %v comp %d main %d walkableMain %v exits %d", nv.Walk[c], nv.Comp[c], nv.Main, nv.WalkableMain(x, y), len(nv.Exits[nv.Comp[c]]))
		}
		fmt.Println()
	}
}
