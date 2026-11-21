// Package geo holds the static map: terrain, solid prisms, trees, and the
// queries built on them (shot raycasts, sight lines, surfaces, navigation).
package geo

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
)

type Interval struct{ Base, Top float32 }

type Tree struct {
	X, Y, Z, Height, CrownR, CrownBase, TrunkR, CX, CY float64
}

type World struct {
	MinX, MinY    float64
	NX, NY        int
	H             []float32 // terrain heights, vertex grid, 1 m
	off           []int32   // per cell offset into iv (len NX*NY+1)
	iv            []Interval
	BMinX, BMaxX  float64
	BMinY, BMaxY  float64
	Trees         []Tree
	treeCell      float64
	tgw, tgh      int
	tgrid         [][]int32
	CanopyDensity float64
	CanopyBlockT  float64
	MaxTop        float32
	margin        float64
	Nav           *Nav
}

type prismJSON struct {
	Kind  string      `json:"kind"`
	Base  float64     `json:"base"`
	Top   float64     `json:"top"`
	Ring  []float64   `json:"ring"`
	Holes [][]float64 `json:"holes"`
}

type worldJSON struct {
	Bounds struct{ MinX, MaxX, MinY, MaxY float64 } `json:"bounds"`
	Rules  struct {
		CanopyDensity float64 `json:"canopyDensity"`
		CanopyBlockT  float64 `json:"canopyBlockT"`
	} `json:"rules"`
	Terrain struct {
		MinX, MinY, Step float64
		NX               int `json:"nx"`
		NY               int `json:"ny"`
	} `json:"terrain"`
	Prisms []prismJSON `json:"prisms"`
	Trees  struct {
		Fields []string  `json:"fields"`
		Data   []float64 `json:"data"`
	} `json:"trees"`
}

func Load(dir string) (*World, error) {
	wj, err := os.ReadFile(filepath.Join(dir, "v2", "world.json"))
	if err != nil {
		return nil, err
	}
	tr, err := os.ReadFile(filepath.Join(dir, "v2", "terrain.i16"))
	if err != nil {
		return nil, err
	}
	return FromBytes(wj, tr)
}

func FromBytes(wj, tr []byte) (*World, error) {
	var j worldJSON
	if err := json.Unmarshal(wj, &j); err != nil {
		return nil, err
	}
	w := &World{MinX: j.Terrain.MinX, MinY: j.Terrain.MinY, NX: j.Terrain.NX, NY: j.Terrain.NY}
	if w.NX <= 0 || w.NY <= 0 || len(tr) < w.NX*w.NY*2 {
		return nil, fmt.Errorf("bad terrain %d x %d, %d bytes", w.NX, w.NY, len(tr))
	}
	if j.Terrain.Step != 0 && j.Terrain.Step != 1 {
		return nil, fmt.Errorf("unsupported terrain step %v", j.Terrain.Step)
	}
	w.BMinX, w.BMaxX, w.BMinY, w.BMaxY = j.Bounds.MinX, j.Bounds.MaxX, j.Bounds.MinY, j.Bounds.MaxY
	w.CanopyDensity, w.CanopyBlockT = j.Rules.CanopyDensity, j.Rules.CanopyBlockT
	if w.CanopyDensity <= 0 {
		w.CanopyDensity = 0.35
	}
	if w.CanopyBlockT <= 0 {
		w.CanopyBlockT = 0.25
	}
	n := w.NX * w.NY
	w.H = make([]float32, n)
	for i := 0; i < n; i++ {
		w.H[i] = float32(int16(binary.LittleEndian.Uint16(tr[2*i:]))) * 0.01
	}
	// rasterize prisms
	cnt := make([]int32, n+1)
	type cellIv struct {
		c  int32
		iv Interval
	}
	var all []cellIv
	for _, p := range j.Prisms {
		if len(p.Ring) < 6 {
			continue
		}
		minx, maxx, miny, maxy := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
		for k := 0; k+1 < len(p.Ring); k += 2 {
			minx = math.Min(minx, p.Ring[k])
			maxx = math.Max(maxx, p.Ring[k])
			miny = math.Min(miny, p.Ring[k+1])
			maxy = math.Max(maxy, p.Ring[k+1])
		}
		i0 := int(math.Ceil(minx - w.MinX))
		i1 := int(math.Floor(maxx - w.MinX))
		j0 := int(math.Ceil(miny - w.MinY))
		j1 := int(math.Floor(maxy - w.MinY))
		if i0 < 0 {
			i0 = 0
		}
		if j0 < 0 {
			j0 = 0
		}
		if i1 >= w.NX {
			i1 = w.NX - 1
		}
		if j1 >= w.NY {
			j1 = w.NY - 1
		}
		if float32(p.Top) > w.MaxTop {
			w.MaxTop = float32(p.Top)
		}
		for jj := j0; jj <= j1; jj++ {
			y := w.MinY + float64(jj)
			for ii := i0; ii <= i1; ii++ {
				x := w.MinX + float64(ii)
				if !pointInRing(p.Ring, x, y) {
					continue
				}
				inHole := false
				for _, h := range p.Holes {
					if len(h) >= 6 && pointInRing(h, x, y) {
						inHole = true
						break
					}
				}
				if inHole {
					continue
				}
				c := int32(jj*w.NX + ii)
				all = append(all, cellIv{c, Interval{float32(p.Base), float32(p.Top)}})
				cnt[c+1]++
			}
		}
	}
	for i := 0; i < n; i++ {
		cnt[i+1] += cnt[i]
	}
	w.off = cnt
	w.iv = make([]Interval, len(all))
	fill := make([]int32, n)
	for _, a := range all {
		w.iv[w.off[a.c]+fill[a.c]] = a.iv
		fill[a.c]++
	}
	// trees
	nf := len(j.Trees.Fields)
	idx := map[string]int{}
	for i, f := range j.Trees.Fields {
		idx[f] = i
	}
	if nf > 0 {
		get := func(row int, name string, def float64) float64 {
			k, ok := idx[name]
			if !ok {
				return def
			}
			return j.Trees.Data[row*nf+k]
		}
		for r := 0; (r+1)*nf <= len(j.Trees.Data); r++ {
			t := Tree{X: get(r, "x", 0), Y: get(r, "y", 0), Z: get(r, "z", 0), Height: get(r, "height", 8),
				CrownR: get(r, "crownR", 3), CrownBase: get(r, "crownBase", 3), TrunkR: get(r, "trunkR", 0.4)}
			t.CX, t.CY = get(r, "crownX", t.X), get(r, "crownY", t.Y)
			w.Trees = append(w.Trees, t)
		}
	}
	w.treeCell = 8
	w.tgw = w.NX/8 + 2
	w.tgh = w.NY/8 + 2
	w.tgrid = make([][]int32, w.tgw*w.tgh)
	for ti, t := range w.Trees {
		r := t.CrownR + 0.5
		x0 := int((math.Min(t.X, t.CX) - r - w.MinX) / w.treeCell)
		x1 := int((math.Max(t.X, t.CX) + r - w.MinX) / w.treeCell)
		y0 := int((math.Min(t.Y, t.CY) - r - w.MinY) / w.treeCell)
		y1 := int((math.Max(t.Y, t.CY) + r - w.MinY) / w.treeCell)
		for yy := y0; yy <= y1; yy++ {
			for xx := x0; xx <= x1; xx++ {
				if xx < 0 || yy < 0 || xx >= w.tgw || yy >= w.tgh {
					continue
				}
				w.tgrid[yy*w.tgw+xx] = append(w.tgrid[yy*w.tgw+xx], int32(ti))
			}
		}
	}
	return w, nil
}

func pointInRing(r []float64, x, y float64) bool {
	in := false
	n := len(r) / 2
	jx, jy := r[2*(n-1)], r[2*(n-1)+1]
	for i := 0; i < n; i++ {
		ix, iy := r[2*i], r[2*i+1]
		if (iy > y) != (jy > y) && x < (jx-ix)*(y-iy)/(jy-iy)+ix {
			in = !in
		}
		jx, jy = ix, iy
	}
	return in
}

// cell returns the raster index of the vertex nearest (x,y), or -1 outside.
func (w *World) cell(x, y float64) int {
	i := int(math.Floor(x - w.MinX + 0.5))
	j := int(math.Floor(y - w.MinY + 0.5))
	if i < 0 || j < 0 || i >= w.NX || j >= w.NY {
		return -1
	}
	return j*w.NX + i
}

// SetMargin shrinks the playable area: the arena wall stands this far inside
// the map bounds (config match.arena_margin). Call before BuildNav.
func (w *World) SetMargin(m float64) {
	w.BMinX += m - w.margin
	w.BMaxX -= m - w.margin
	w.BMinY += m - w.margin
	w.BMaxY -= m - w.margin
	w.margin = m
}

func (w *World) InBounds(x, y float64) bool {
	return x >= w.BMinX && x <= w.BMaxX && y >= w.BMinY && y <= w.BMaxY
}

// Ground is the bilinear terrain height.
func (w *World) Ground(x, y float64) float64 {
	fx := x - w.MinX
	fy := y - w.MinY
	if fx < 0 {
		fx = 0
	}
	if fy < 0 {
		fy = 0
	}
	if fx > float64(w.NX-1)-1e-6 {
		fx = float64(w.NX-1) - 1e-6
	}
	if fy > float64(w.NY-1)-1e-6 {
		fy = float64(w.NY-1) - 1e-6
	}
	i := int(fx)
	j := int(fy)
	tx := fx - float64(i)
	ty := fy - float64(j)
	k := j*w.NX + i
	h00 := float64(w.H[k])
	h10 := float64(w.H[k+1])
	h01 := float64(w.H[k+w.NX])
	h11 := float64(w.H[k+w.NX+1])
	return (h00*(1-tx)+h10*tx)*(1-ty) + (h01*(1-tx)+h11*tx)*ty
}

// Intervals returns the solid prism spans over the cell at (x,y).
func (w *World) Intervals(x, y float64) []Interval {
	c := w.cell(x, y)
	if c < 0 {
		return nil
	}
	return w.iv[w.off[c]:w.off[c+1]]
}

// SolidAt reports whether the point is inside terrain or a prism.
func (w *World) SolidAt(x, y, z float64) bool {
	c := w.cell(x, y)
	if c < 0 {
		return true
	}
	if z < float64(w.H[c]) {
		return true
	}
	for _, iv := range w.iv[w.off[c]:w.off[c+1]] {
		if z >= float64(iv.Base) && z <= float64(iv.Top) {
			return true
		}
	}
	return false
}

// Surface is the highest standable surface at (x,y) not above zmax
// (terrain or a prism top). ok=false if even the terrain is above zmax.
func (w *World) Surface(x, y, zmax float64) (float64, bool) {
	g := w.Ground(x, y)
	if g > zmax {
		return g, false
	}
	s := g
	for _, iv := range w.Intervals(x, y) {
		t := float64(iv.Top)
		if t <= zmax && t > s {
			s = t
		}
	}
	return s, true
}

// BodyBlocked reports whether a prism occupies any part of the column
// (z0,z1) at (x,y), or the map edge is crossed.
func (w *World) BodyBlocked(x, y, z0, z1 float64) bool {
	if !w.InBounds(x, y) {
		return true
	}
	for _, iv := range w.Intervals(x, y) {
		if float64(iv.Top) > z0 && float64(iv.Base) < z1 {
			return true
		}
	}
	return false
}

// TopAt returns the highest prism top over the cell (roof height), or ground.
func (w *World) TopAt(x, y float64) float64 {
	s := w.Ground(x, y)
	for _, iv := range w.Intervals(x, y) {
		if float64(iv.Top) > s {
			s = float64(iv.Top)
		}
	}
	return s
}

// RayHit marches a shot from a to b and returns the fraction (0..1) at which
// it first enters terrain or a prism; 1 and false if it is clear.
func (w *World) RayHit(ax, ay, az, bx, by, bz float64) (float64, bool) {
	dx, dy, dz := bx-ax, by-ay, bz-az
	l := math.Sqrt(dx*dx + dy*dy + dz*dz)
	if l < 1e-6 {
		return 1, false
	}
	n := int(l/0.5) + 1
	inv := 1 / float64(n)
	maxTop := float64(w.MaxTop)
	for s := 1; s <= n; s++ {
		t := float64(s) * inv
		x, y, z := ax+dx*t, ay+dy*t, az+dz*t
		c := w.cell(x, y)
		if c < 0 {
			return t, true
		}
		if z < float64(w.H[c]) {
			return t, true
		}
		if z > maxTop {
			continue
		}
		for _, iv := range w.iv[w.off[c]:w.off[c+1]] {
			if z >= float64(iv.Base) && z <= float64(iv.Top) {
				return t, true
			}
		}
	}
	return 1, false
}

// Clear reports whether a straight shot from a to b meets no terrain or prism.
func (w *World) Clear(ax, ay, az, bx, by, bz float64) bool {
	_, hit := w.RayHit(ax, ay, az, bx, by, bz)
	return !hit
}

// ClearCoarse is Clear with 1 m steps (cheaper, for tactical sampling).
func (w *World) ClearCoarse(ax, ay, az, bx, by, bz float64) bool {
	dx, dy, dz := bx-ax, by-ay, bz-az
	l := math.Sqrt(dx*dx + dy*dy + dz*dz)
	if l < 1e-6 {
		return true
	}
	n := int(l) + 1
	inv := 1 / float64(n)
	maxTop := float64(w.MaxTop)
	for s := 1; s < n; s++ {
		t := float64(s) * inv
		x, y, z := ax+dx*t, ay+dy*t, az+dz*t
		c := w.cell(x, y)
		if c < 0 {
			return false
		}
		if z < float64(w.H[c]) {
			return false
		}
		if z > maxTop {
			continue
		}
		for _, iv := range w.iv[w.off[c]:w.off[c+1]] {
			if z >= float64(iv.Base) && z <= float64(iv.Top) {
				return false
			}
		}
	}
	return true
}

// Sees reports a sight line: buildings and terrain block, tree trunks block,
// and crowns block once enough foliage has been crossed.
func (w *World) Sees(ax, ay, az, bx, by, bz float64) bool {
	if !w.Clear(ax, ay, az, bx, by, bz) {
		return false
	}
	return w.foliageOK(ax, ay, az, bx, by, bz)
}

func (w *World) foliageOK(ax, ay, az, bx, by, bz float64) bool {
	dx, dy, dz := bx-ax, by-ay, bz-az
	l := math.Sqrt(dx*dx + dy*dy + dz*dz)
	if l < 1e-6 {
		return true
	}
	limit := -math.Log(w.CanopyBlockT) / w.CanopyDensity // metres of crown
	n := int(l) + 1
	step := l / float64(n)
	inv := 1 / float64(n)
	acc := 0.0
	for s := 0; s <= n; s++ {
		t := float64(s) * inv
		x, y, z := ax+dx*t, ay+dy*t, az+dz*t
		gx := int((x - w.MinX) / w.treeCell)
		gy := int((y - w.MinY) / w.treeCell)
		if gx < 0 || gy < 0 || gx >= w.tgw || gy >= w.tgh {
			continue
		}
		in := false
		for _, ti := range w.tgrid[gy*w.tgw+gx] {
			tr := &w.Trees[ti]
			if z < tr.Z || z > tr.Z+tr.Height {
				continue
			}
			if z < tr.Z+tr.CrownBase {
				ex, ey := x-tr.X, y-tr.Y
				if ex*ex+ey*ey < tr.TrunkR*tr.TrunkR {
					return false
				}
				continue
			}
			ex, ey := x-tr.CX, y-tr.CY
			if ex*ex+ey*ey < tr.CrownR*tr.CrownR {
				in = true
			}
		}
		if in {
			acc += step
			if acc > limit {
				return false
			}
		}
	}
	return true
}
