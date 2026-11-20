//go:build ignore

package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"time"

	"mechbot/geo"
)

func main() {
	t0 := time.Now()
	w, err := geo.Load("mapdata/arena")
	if err != nil {
		panic(err)
	}
	fmt.Println("load", time.Since(t0))
	t0 = time.Now()
	w.BuildNav(3.0, 9.5, 1.2)
	fmt.Println("nav", time.Since(t0))
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
	fmt.Println("walk", nw, "main", nm, "of", len(nv.Walk))
	img := image.NewRGBA(image.Rect(0, 0, nv.W, nv.H))
	for j := 0; j < nv.H; j++ {
		for i := 0; i < nv.W; i++ {
			c := j*nv.W + i
			col := color.RGBA{40, 40, 40, 255}
			if nv.Walk[c] {
				if nv.Comp[c] == nv.Main {
					v := uint8(120 + 6*float64(nv.Hgt[c]))
					col = color.RGBA{v, v, v, 255}
				} else {
					col = color.RGBA{200, 60, 60, 255}
				}
			} else {
				x, y := nv.Pos(c)
				if w.TopAt(x, y) > w.Ground(x, y)+1 {
					col = color.RGBA{30, 60, 140, 255}
				}
			}
			img.Set(i, nv.H-1-j, col)
		}
	}
	f, _ := os.Create("/tmp/nav.png")
	png.Encode(f, img)
	f.Close()
	t0 = time.Now()
	var p [][2]float64
	var cost float64
	for k := 0; k < 100; k++ {
		p, cost = nv.Path(320, -280, -350, 300, 400000, nil)
	}
	fmt.Println("path", len(p), cost, time.Since(t0)/100)
	t0 = time.Now()
	n := 0
	for k := 0; k < 100000; k++ {
		if w.Clear(float64(k%500-250), 0, 5, float64((k*7)%500-250), 150, 5) {
			n++
		}
	}
	fmt.Println("ray", n, time.Since(t0)/100000)
}
