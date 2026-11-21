//go:build ignore

package main

import (
	"fmt"
	"math"
	"os"
	"strconv"

	"mechbot/geo"
)

func main() {
	w, _ := geo.Load("mapdata/arena")
	w.BuildNav(3.0, 9.5, 1.2)
	cx, _ := strconv.ParseFloat(os.Args[1], 64)
	cy, _ := strconv.ParseFloat(os.Args[2], 64)
	nv := w.Nav
	for y := cy + 24; y >= cy-24; y -= 2 {
		line := ""
		for x := cx - 40; x <= cx+40; x += 2 {
			c := nv.Cell(x, y)
			ch := "#"
			if c >= 0 && nv.Walk[c] {
				ch = "."
				if nv.Comp[c] != nv.Main {
					ch = "o"
				}
			} else if w.TopAt(x, y) > w.Ground(x, y)+1.3 {
				ch = "B"
			}
			if math.Abs(x-cx) < 1 && math.Abs(y-cy) < 1 {
				ch = "@"
			}
			line += ch
		}
		fmt.Printf("%6.0f %s\n", y, line)
	}
	for y := cy + 6; y >= cy-6; y -= 2 {
		for x := cx - 12; x <= cx+12; x += 2 {
			fmt.Printf("%5.1f ", w.Ground(x, y))
		}
		fmt.Println()
	}
}
