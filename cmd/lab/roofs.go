//go:build ignore

package main

import (
	"fmt"
	"math"
	"sort"

	"mechbot/geo"
)

func main() {
	w, _ := geo.Load("/home/claude/kit/mechfight-kit/arena")
	w.SetMargin(30)
	w.BuildNav(3.0, 9.5, 1.2)
	const cs = 2.0
	nx := int((w.BMaxX - w.BMinX) / cs)
	ny := int((w.BMaxY - w.BMinY) / cs)
	top := make([]float64, nx*ny)
	lab := make([]int, nx*ny)
	for j := 0; j < ny; j++ {
		for i := 0; i < nx; i++ {
			x, y := w.BMinX+(float64(i)+0.5)*cs, w.BMinY+(float64(j)+0.5)*cs
			t := w.TopAt(x, y)
			g := w.Ground(x, y)
			if t-g > 5 && w.InBounds(x, y) {
				top[j*nx+i] = t
			} else {
				top[j*nx+i] = math.NaN()
			}
			lab[j*nx+i] = -1
		}
	}
	type reg struct {
		n          int
		z, g       float64
		sx, sy     float64
		minx, maxx float64
	}
	var regs []reg
	for s := range top {
		if math.IsNaN(top[s]) || lab[s] >= 0 {
			continue
		}
		id := len(regs)
		r := reg{minx: 1e9, maxx: -1e9}
		st := []int{s}
		lab[s] = id
		for len(st) > 0 {
			c := st[len(st)-1]
			st = st[:len(st)-1]
			i, j := c%nx, c/nx
			x, y := w.BMinX+(float64(i)+0.5)*cs, w.BMinY+(float64(j)+0.5)*cs
			r.n++
			r.z += top[c]
			r.g += w.Ground(x, y)
			r.sx += x
			r.sy += y
			for _, o := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				a, b := i+o[0], j+o[1]
				if a < 0 || b < 0 || a >= nx || b >= ny {
					continue
				}
				d := b*nx + a
				if lab[d] < 0 && !math.IsNaN(top[d]) && math.Abs(top[d]-top[c]) < 1.0 {
					lab[d] = id
					st = append(st, d)
				}
			}
		}
		regs = append(regs, r)
	}
	sort.Slice(regs, func(a, b int) bool { return regs[a].n > regs[b].n })
	big := 0
	hist := map[int]int{}
	for _, r := range regs {
		h := r.z/float64(r.n) - r.g/float64(r.n)
		if r.n >= 60 {
			big++
			hist[int(h/10)*10]++
		}
	}
	fmt.Println("regions", len(regs), "with area>=240m2:", big, "height hist:", hist)
	for _, r := range regs[:25] {
		fmt.Printf("area %5.0f m2  top z %5.1f  height %5.1f  centre (%.0f,%.0f)\n", float64(r.n)*cs*cs, r.z/float64(r.n), r.z/float64(r.n)-r.g/float64(r.n), r.sx/float64(r.n), r.sy/float64(r.n))
	}
}
