package geo

import (
	"container/heap"
	"math"
)

const NavStep = 2.0

// Nav is a ground-level walkability grid (2 m cells) with A* search.
type Nav struct {
	W, H   int
	MinX   float64
	MinY   float64
	Walk   []bool
	Hgt    []float32 // walking surface height
	Cost   []float32 // extra cost for hugging obstacles
	Comp   []int32   // connected component id (-1 blocked)
	Main   int32     // id of the largest component
	g      []float32
	stamp  []uint32
	closed []uint32
	parent []int32
	gen    uint32
	open   nodeHeap
}

type node struct {
	f float32
	c int32
}
type nodeHeap []node

func (h nodeHeap) Len() int            { return len(h) }
func (h nodeHeap) Less(i, j int) bool  { return h[i].f < h[j].f }
func (h nodeHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *nodeHeap) Push(x interface{}) { *h = append(*h, x.(node)) }
func (h *nodeHeap) Pop() interface{} {
	o := *h
	n := len(o)
	x := o[n-1]
	*h = o[:n-1]
	return x
}

func (w *World) BuildNav(clearance, bodyH, stepUp float64) {
	nv := &Nav{W: (w.NX + 1) / 2, H: (w.NY + 1) / 2, MinX: w.MinX, MinY: w.MinY}
	n := nv.W * nv.H
	nv.Walk = make([]bool, n)
	nv.Hgt = make([]float32, n)
	nv.Cost = make([]float32, n)
	nv.Comp = make([]int32, n)
	nv.g = make([]float32, n)
	nv.stamp = make([]uint32, n)
	nv.closed = make([]uint32, n)
	nv.parent = make([]int32, n)
	r := int(math.Ceil(clearance))
	for j := 0; j < nv.H; j++ {
		for i := 0; i < nv.W; i++ {
			x := nv.MinX + float64(i)*NavStep
			y := nv.MinY + float64(j)*NavStep
			c := j*nv.W + i
			rc := w.cell(x, y)
			if rc < 0 || !w.InBounds(x, y) {
				continue
			}
			h := float64(w.H[rc])
			// thin deck at walking height: stand on it
			for _, iv := range w.iv[w.off[rc]:w.off[rc+1]] {
				if float64(iv.Top) > h && float64(iv.Top)-float64(iv.Base) <= 3.5 && float64(iv.Top) < h+12 && float64(iv.Base) < h+bodyH {
					h = float64(iv.Top)
				}
			}
			nv.Hgt[c] = float32(h)
			ok := true
		scan:
			for dj := -r; dj <= r; dj++ {
				for di := -r; di <= r; di++ {
					if float64(di*di+dj*dj) > clearance*clearance+0.01 {
						continue
					}
					px, py := x+float64(di), y+float64(dj)
					pc := w.cell(px, py)
					if pc < 0 || !w.InBounds(px, py) {
						ok = false
						break scan
					}
					th := float64(w.H[pc])
					if th > h+2.4 {
						ok = false
						break scan
					}
					for _, iv := range w.iv[w.off[pc]:w.off[pc+1]] {
						if float64(iv.Top) > h+stepUp && float64(iv.Base) < h+bodyH {
							ok = false
							break scan
						}
					}
				}
			}
			nv.Walk[c] = ok
		}
	}
	// obstacle-hugging cost: distance (in cells) to nearest blocked cell
	dist := make([]int8, n)
	q := make([]int32, 0, n)
	for c := 0; c < n; c++ {
		if !nv.Walk[c] {
			q = append(q, int32(c))
		} else {
			dist[c] = 127
		}
	}
	for qi := 0; qi < len(q); qi++ {
		c := int(q[qi])
		d := dist[c]
		if d >= 3 {
			continue
		}
		ci, cj := c%nv.W, c/nv.W
		for _, o := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			ni, nj := ci+o[0], cj+o[1]
			if ni < 0 || nj < 0 || ni >= nv.W || nj >= nv.H {
				continue
			}
			nc := nj*nv.W + ni
			if dist[nc] > d+1 {
				dist[nc] = d + 1
				q = append(q, int32(nc))
			}
		}
	}
	for c := 0; c < n; c++ {
		switch dist[c] {
		case 1:
			nv.Cost[c] = 1.6
		case 2:
			nv.Cost[c] = 0.6
		case 3:
			nv.Cost[c] = 0.2
		}
	}
	// components
	for c := range nv.Comp {
		nv.Comp[c] = -1
	}
	var id int32
	best, bestN := int32(-1), 0
	for c := 0; c < n; c++ {
		if !nv.Walk[c] || nv.Comp[c] >= 0 {
			continue
		}
		q = q[:0]
		q = append(q, int32(c))
		nv.Comp[c] = id
		for qi := 0; qi < len(q); qi++ {
			cc := int(q[qi])
			nv.neighbors(cc, func(nc int, _ float32) {
				if nv.Comp[nc] < 0 {
					nv.Comp[nc] = id
					q = append(q, int32(nc))
				}
			})
		}
		if len(q) > bestN {
			bestN = len(q)
			best = id
		}
		id++
	}
	nv.Main = best
	w.Nav = nv
}

func (nv *Nav) edgeOK(a, b int, lim float32) bool {
	d := nv.Hgt[a] - nv.Hgt[b]
	if d < 0 {
		d = -d
	}
	return d <= lim
}

func (nv *Nav) neighbors(c int, f func(nc int, cost float32)) {
	ci, cj := c%nv.W, c/nv.W
	for dj := -1; dj <= 1; dj++ {
		for di := -1; di <= 1; di++ {
			if di == 0 && dj == 0 {
				continue
			}
			ni, nj := ci+di, cj+dj
			if ni < 0 || nj < 0 || ni >= nv.W || nj >= nv.H {
				continue
			}
			nc := nj*nv.W + ni
			if !nv.Walk[nc] {
				continue
			}
			if di != 0 && dj != 0 {
				if !nv.Walk[cj*nv.W+ni] || !nv.Walk[nj*nv.W+ci] || !nv.edgeOK(c, nc, 2.6) {
					continue
				}
				f(nc, 1.4142*(1+nv.Cost[nc]))
			} else {
				if !nv.edgeOK(c, nc, 2.0) {
					continue
				}
				f(nc, 1+nv.Cost[nc])
			}
		}
	}
}

func (nv *Nav) Cell(x, y float64) int {
	i := int(math.Floor((x-nv.MinX)/NavStep + 0.5))
	j := int(math.Floor((y-nv.MinY)/NavStep + 0.5))
	if i < 0 || j < 0 || i >= nv.W || j >= nv.H {
		return -1
	}
	return j*nv.W + i
}

func (nv *Nav) Pos(c int) (float64, float64) {
	return nv.MinX + float64(c%nv.W)*NavStep, nv.MinY + float64(c/nv.W)*NavStep
}

func (nv *Nav) Walkable(x, y float64) bool {
	c := nv.Cell(x, y)
	return c >= 0 && nv.Walk[c]
}

// WalkableMain: walkable and part of the main connected area.
func (nv *Nav) WalkableMain(x, y float64) bool {
	c := nv.Cell(x, y)
	return c >= 0 && nv.Walk[c] && nv.Comp[c] == nv.Main
}

// Nearest returns the closest main-component walkable cell within maxR metres, or -1.
func (nv *Nav) Nearest(x, y, maxR float64) int {
	c := nv.Cell(x, y)
	if c >= 0 && nv.Walk[c] && nv.Comp[c] == nv.Main {
		return c
	}
	ci := int(math.Floor((x-nv.MinX)/NavStep + 0.5))
	cj := int(math.Floor((y-nv.MinY)/NavStep + 0.5))
	best, bd := -1, math.Inf(1)
	rr := int(maxR/NavStep) + 1
	for r := 1; r <= rr; r++ {
		for dj := -r; dj <= r; dj++ {
			for di := -r; di <= r; di++ {
				if di != -r && di != r && dj != -r && dj != r {
					continue
				}
				ni, nj := ci+di, cj+dj
				if ni < 0 || nj < 0 || ni >= nv.W || nj >= nv.H {
					continue
				}
				nc := nj*nv.W + ni
				if !nv.Walk[nc] || nv.Comp[nc] != nv.Main {
					continue
				}
				px, py := nv.Pos(nc)
				d := (px-x)*(px-x) + (py-y)*(py-y)
				if d < bd {
					bd = d
					best = nc
				}
			}
		}
		if best >= 0 && float64(r)*NavStep > math.Sqrt(bd) {
			break
		}
	}
	return best
}

// WalkLine reports whether the straight segment stays on walkable cells.
func (nv *Nav) WalkLine(ax, ay, bx, by float64) bool {
	dx, dy := bx-ax, by-ay
	l := math.Hypot(dx, dy)
	n := int(l/1.0) + 1
	prev := -1
	for s := 0; s <= n; s++ {
		t := float64(s) / float64(n)
		c := nv.Cell(ax+dx*t, ay+dy*t)
		if c < 0 || !nv.Walk[c] {
			return false
		}
		if prev >= 0 && c != prev && !nv.edgeOK(prev, c, 2.6) {
			return false
		}
		prev = c
	}
	return true
}

// Path runs A* from (sx,sy) to (gx,gy). Returns waypoints from start to goal
// (cell centres) and the path cost in metres; nil if unreachable.
// extra, if non-nil, adds a per-cell cost (danger fields).
func (nv *Nav) Path(sx, sy, gx, gy float64, maxExpand int, extra func(c int) float32) ([][2]float64, float64) {
	s := nv.Nearest(sx, sy, 40)
	g := nv.Nearest(gx, gy, 40)
	if s < 0 || g < 0 {
		return nil, 0
	}
	if s == g {
		return [][2]float64{{gx, gy}}, math.Hypot(gx-sx, gy-sy)
	}
	nv.gen++
	gen := nv.gen
	nv.open = nv.open[:0]
	gi, gj := g%nv.W, g/nv.W
	hfn := func(c int) float32 {
		dx := float64(c%nv.W - gi)
		dy := float64(c/nv.W - gj)
		if dx < 0 {
			dx = -dx
		}
		if dy < 0 {
			dy = -dy
		}
		if dx < dy {
			dx, dy = dy, dx
		}
		return float32(dx + 0.4142*dy)
	}
	nv.g[s] = 0
	nv.stamp[s] = gen
	nv.parent[s] = -1
	heap.Push(&nv.open, node{hfn(s), int32(s)})
	found := false
	exp := 0
	for len(nv.open) > 0 {
		cur := heap.Pop(&nv.open).(node)
		c := int(cur.c)
		if nv.closed[c] == gen {
			continue
		}
		nv.closed[c] = gen
		if c == g {
			found = true
			break
		}
		exp++
		if exp > maxExpand {
			break
		}
		gc := nv.g[c]
		nv.neighbors(c, func(nc int, cost float32) {
			if nv.closed[nc] == gen {
				return
			}
			if extra != nil {
				cost += extra(nc)
			}
			ng := gc + cost
			if nv.stamp[nc] != gen || ng < nv.g[nc] {
				nv.stamp[nc] = gen
				nv.g[nc] = ng
				nv.parent[nc] = int32(c)
				heap.Push(&nv.open, node{ng + hfn(nc), int32(nc)})
			}
		})
	}
	if !found {
		return nil, 0
	}
	var rev []int
	for c := g; c >= 0; c = int(nv.parent[c]) {
		rev = append(rev, c)
	}
	out := make([][2]float64, 0, len(rev))
	for i := len(rev) - 1; i >= 0; i-- {
		x, y := nv.Pos(rev[i])
		out = append(out, [2]float64{x, y})
	}
	out[len(out)-1] = [2]float64{gx, gy}
	if !nv.Walkable(gx, gy) {
		x, y := nv.Pos(g)
		out[len(out)-1] = [2]float64{x, y}
	}
	return out, float64(nv.g[g]) * NavStep
}

// CloneScratch returns a World sharing all static data but with its own
// search scratch space, so several goroutines can path-find at once.
func (w *World) CloneScratch() *World {
	c := *w
	nv := *w.Nav
	n := len(nv.Walk)
	nv.g = make([]float32, n)
	nv.stamp = make([]uint32, n)
	nv.closed = make([]uint32, n)
	nv.parent = make([]int32, n)
	nv.gen = 0
	nv.open = nil
	c.Nav = &nv
	return &c
}
