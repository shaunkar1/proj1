//go:build ignore

package main

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"mechbot/geo"
)

func main() {
	w, _ := geo.Load("/home/claude/kit/mechfight-kit/arena")
	w.SetMargin(30)
	w.BuildNav(3.0, 9.5, 1.2)
	dir := os.Args[1]
	pkb, _ := os.ReadFile(dir + "/pk.txt")
	trb, _ := os.ReadFile(dir + "/trk.txt")
	type pk struct {
		id   int
		kind string
		p    [3]float64
	}
	var pks []pk
	for _, f := range strings.Fields(string(pkb)) {
		s := strings.Split(f, ",")
		id, _ := strconv.Atoi(s[0])
		x, _ := strconv.ParseFloat(s[2], 64)
		y, _ := strconv.ParseFloat(s[3], 64)
		z, _ := strconv.ParseFloat(s[4], 64)
		pks = append(pks, pk{id, s[1], [3]float64{x, y, z}})
	}
	for _, p := range pks {
		g := w.Ground(p.p[0], p.p[1])
		fmt.Printf("pickup %d %s (%.0f,%.0f,%.1f) ground %.1f top %.1f above-ground %.1f walkable %v\n", p.id, p.kind, p.p[0], p.p[1], p.p[2], g, w.TopAt(p.p[0], p.p[1]), p.p[2]-g, w.Nav.WalkableMain(p.p[0], p.p[1]))
	}
	vis := map[int]int{}
	first := map[int]float64{}
	mind := map[int]float64{}
	for _, f := range strings.Fields(string(trb)) {
		s := strings.Split(f, ",")
		v := make([]float64, len(s))
		for i := range s {
			v[i], _ = strconv.ParseFloat(s[i], 64)
		}
		t, x, y, z, yaw := v[0], v[1], v[2], v[3], v[4]
		for _, p := range pks {
			d := math.Hypot(p.p[0]-x, p.p[1]-y)
			if d > 200 {
				continue
			}
			off := math.Abs(math.Mod(math.Atan2(p.p[1]-y, p.p[0]-x)-yaw+3*math.Pi, 2*math.Pi) - math.Pi)
			if off > math.Pi/3 && d > 45 {
				continue
			}
			if w.Clear(x, y, z+8, p.p[0], p.p[1], p.p[2]+0.5) {
				vis[p.id]++
				if _, ok := first[p.id]; !ok {
					first[p.id] = t
				}
				if m, ok := mind[p.id]; !ok || d < m {
					mind[p.id] = d
				}
			}
		}
	}
	for _, p := range pks {
		if vis[p.id] > 0 {
			fmt.Printf("VISIBLE %d %s: %d samples (%.1fs), first at t=%.1f, min dist %.0f\n", p.id, p.kind, vis[p.id], float64(vis[p.id])*0.5, first[p.id], mind[p.id])
		}
	}
}
