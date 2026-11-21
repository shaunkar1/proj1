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
	w.BuildNav(2.6, 9.5, 1.2)
	cx, _ := strconv.ParseFloat(os.Args[1], 64)
	cy, _ := strconv.ParseFloat(os.Args[2], 64)
	nv := w.Nav
	for y := cy + 34; y >= cy-24; y -= 2 {
		fmt.Printf("%5.0f ", y)
		for x := cx - 60; x <= cx+60; x += 2 {
			c := nv.Cell(x, y)
			ch := "#"
			if nv.Walk[c] {
				ch = "o"
				if nv.Comp[c] == nv.Main {
					ch = "."
				}
				if nv.Adj[c] != 255 {
					if nv.Comp[c] == nv.Main {
						ch = ","
					} else {
						ch = "c"
					}
				}
			}
			fmt.Print(ch)
		}
		fmt.Println()
	}
}
