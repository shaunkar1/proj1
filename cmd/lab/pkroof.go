//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"

	"mechbot/geo"
)

func main() {
	w, _ := geo.Load("mapdata/arena")
	w.SetMargin(30)
	w.BuildNav(3.0, 9.5, 1.2)
	b, _ := os.ReadFile(os.Args[1])
	var recs []struct {
		Hdr struct {
			Pickups []struct {
				Kind string     `json:"kind"`
				Pos  [3]float64 `json:"pos"`
			} `json:"pickups"`
			Crates []struct {
				Pos [3]float64 `json:"pos"`
			} `json:"crates"`
		} `json:"hdr"`
	}
	json.Unmarshal(b, &recs)
	n, roof, low, walk := 0, 0, 0, 0
	cn, croof := 0, 0
	hist := map[int]int{}
	for _, r := range recs {
		for _, p := range r.Hdr.Pickups {
			n++
			h := p.Pos[2] - w.Ground(p.Pos[0], p.Pos[1])
			if h > 2.5 {
				roof++
				if h < 9 {
					low++
				}
				hist[int(h/4)*4]++
			}
			if w.Nav.WalkableMain(p.Pos[0], p.Pos[1]) {
				walk++
			}
		}
		for _, c := range r.Hdr.Crates {
			cn++
			if c.Pos[2]-w.Ground(c.Pos[0], c.Pos[1]) > 2.5 {
				croof++
			}
		}
	}
	fmt.Printf("pickups %d: on roofs %d (%.0f%%), of which under 9 m: %d; walkable-main cell %d (%.0f%%)\n", n, roof, 100*float64(roof)/float64(n), low, walk, 100*float64(walk)/float64(n))
	fmt.Println("roof height histogram (m):", hist)
	fmt.Printf("crates %d: on roofs %d (%.0f%%)\n", cn, croof, 100*float64(croof)/float64(cn))
}
