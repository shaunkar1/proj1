//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"

	"mechbot/geo"
)

func main() {
	w, _ := geo.Load("mapdata/arena")
	w.BuildNav(3.0, 9.5, 1.2)
	raw, _ := os.ReadFile("mapdata/arena/v2/world.json")
	var j struct {
		Prisms []struct {
			Kind      string
			Base, Top float64
			Ring      []float64
		}
		Stadium map[string]json.RawMessage
	}
	json.Unmarshal(raw, &j)
	n := 0
	for _, p := range j.Prisms {
		if p.Kind == "stadium" {
			n++
			minx, maxx, miny, maxy := 1e9, -1e9, 1e9, -1e9
			for k := 0; k+1 < len(p.Ring); k += 2 {
				if p.Ring[k] < minx {
					minx = p.Ring[k]
				}
				if p.Ring[k] > maxx {
					maxx = p.Ring[k]
				}
				if p.Ring[k+1] < miny {
					miny = p.Ring[k+1]
				}
				if p.Ring[k+1] > maxy {
					maxy = p.Ring[k+1]
				}
			}
			if n <= 30 {
				fmt.Printf("stad base %.1f top %.1f verts %d bbox x[%.0f,%.0f] y[%.0f,%.0f]\n", p.Base, p.Top, len(p.Ring)/2, minx, maxx, miny, maxy)
			}
		}
	}
	fmt.Println(string(j.Stadium["openings"]))

	x0, x1, y0, y1 := -420.0, -130.0, -30.0, 290.0
	sc := 3
	img := image.NewRGBA(image.Rect(0, 0, int(x1-x0)*sc, int(y1-y0)*sc))
	for yi := 0; yi < int(y1-y0)*sc; yi++ {
		for xi := 0; xi < int(x1-x0)*sc; xi++ {
			x := x0 + float64(xi)/float64(sc)
			y := y1 - float64(yi)/float64(sc)
			g := w.Ground(x, y)
			t := w.TopAt(x, y)
			col := color.RGBA{uint8(90 + 6*g), uint8(90 + 6*g), uint8(90 + 6*g), 255}
			if t > g+1.2 {
				h := t - g
				col = color.RGBA{uint8(40 + 5*h), 40, uint8(200 - 4*h), 255}
			} else if w.Nav.WalkableMain(x, y) {
				col = color.RGBA{uint8(60 + 6*(g+10)), uint8(150 + 4*(g+10)), uint8(60 + 6*(g+10)), 255}
			} else if w.Nav.Walkable(x, y) {
				col = color.RGBA{220, 200, 60, 255}
			}
			img.Set(xi, yi, col)
		}
	}
	f, _ := os.Create("/tmp/stadium.png")
	png.Encode(f, img)
	f.Close()
}
