//go:build ignore

package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"math"
	"os"

	"mechbot/geo"
	"mechbot/proto"
)

func main() {
	w, _ := geo.Load("mapdata/arena")
	w.BuildNav(3.0, 9.5, 1.2)
	f, _ := os.Open("/home/claude/kit/mechfight-kit/check/stream.jsonl.gz")
	gz, _ := gzip.NewReader(f)
	sc := bufio.NewScanner(gz)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	n, bad, offnav := 0, 0, 0
	var maxd float64
	seen := map[int]bool{}
	for sc.Scan() {
		var o proto.Obs
		if json.Unmarshal(sc.Bytes(), &o) != nil || o.Type != "obs" {
			continue
		}
		s := o.Self
		if s.Grounded {
			n++
			surf, _ := w.Surface(s.Pos[0], s.Pos[1], s.Pos[2]+1.2)
			d := math.Abs(surf - s.Pos[2])
			if d > maxd {
				maxd = d
			}
			if d > 0.3 {
				bad++
				if bad < 6 {
					fmt.Printf("self t=%.1f pos=%v surf=%.2f ground=%.2f\n", o.T, s.Pos, surf, w.Ground(s.Pos[0], s.Pos[1]))
				}
			}
			if !w.Nav.Walkable(s.Pos[0], s.Pos[1]) {
				offnav++
			}
		}
		for _, m := range o.Mechs {
			if m.Vel[2] == 0 {
				surf, _ := w.Surface(m.Pos[0], m.Pos[1], m.Pos[2]+1.2)
				if math.Abs(surf-m.Pos[2]) > 0.3 && !seen[m.ID*100000+o.Tick] {
					fmt.Printf("mech %d t=%.1f pos=%v surf=%.2f top=%.2f\n", m.ID, o.T, m.Pos, surf, w.TopAt(m.Pos[0], m.Pos[1]))
				}
			}
		}
		for _, p := range o.Pickups {
			if !seen[p.ID] {
				seen[p.ID] = true
				surf, _ := w.Surface(p.Pos[0], p.Pos[1], p.Pos[2]+1.2)
				fmt.Printf("pickup %d %s pos=%v surf=%.2f ground=%.2f top=%.2f\n", p.ID, p.Kind, p.Pos, surf, w.Ground(p.Pos[0], p.Pos[1]), w.TopAt(p.Pos[0], p.Pos[1]))
			}
		}
		for _, p := range o.Crates {
			if !seen[p.ID] {
				seen[p.ID] = true
				surf, _ := w.Surface(p.Pos[0], p.Pos[1], p.Pos[2]+1.2)
				fmt.Printf("crate %d pos=%v surf=%.2f ground=%.2f top=%.2f\n", p.ID, p.Pos, surf, w.Ground(p.Pos[0], p.Pos[1]), w.TopAt(p.Pos[0], p.Pos[1]))
			}
		}
	}
	fmt.Println("grounded obs", n, "mismatch>0.3:", bad, "max", maxd, "offnav", offnav)
}
