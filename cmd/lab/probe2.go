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
	for y := cy + 30; y >= cy-30; y -= 2 {
		fmt.Printf("%5.0f ", y)
		for x := cx - 26; x <= cx+26; x += 2 {
			g := w.Ground(x, y)
			iv := w.Intervals(x, y)
			ch := " "
			if w.Nav.WalkableMain(x, y) {
				ch = "."
			} else if w.Nav.Walkable(x, y) {
				ch = "o"
			}
			if len(iv) > 0 {
				fmt.Printf("%s[%3.0f]", ch, float64(iv[0].Top)-g)
			} else {
				fmt.Printf("%s%4.0f ", ch, g)
			}
		}
		fmt.Println()
	}
}
