package bot

import (
	"fmt"
	"math"
	"os"
)

var hideDebug = os.Getenv("HIDE_DEBUG") != ""

func (b *Bot) hideTrace() {
	if b.o.Tick%20 != 0 {
		return
	}
	me := b.me
	z := &b.o.Zone
	x, y := me.Pos[0], me.Pos[1]
	l := hyp(x, y, z.NextCenter[0], z.NextCenter[1])
	gap := -1.0
	if l > 1 {
		ux, uy := (x-z.NextCenter[0])/l, (y-z.NextCenter[1])/l
		ex, ey := z.NextCenter[0]-z.Center[0], z.NextCenter[1]-z.Center[1]
		bb := ex*ux + ey*uy
		gap = -bb + math.Sqrt(math.Max(bb*bb-(ex*ex+ey*ey-z.Radius*z.Radius), 0)) - z.NextRadius
	}
	gd := -1.0
	if b.DbgHasGoal {
		gd = hyp(x, y, b.DbgGoal[0], b.DbgGoal[1])
	}
	near := 999.0
	for _, t := range b.trackL {
		if !t.NPC && !t.Ghost && b.t-t.Seen < 12 {
			near = math.Min(near, hyp(x, y, t.Pos[0], t.Pos[1]))
		}
	}
	fmt.Fprintf(os.Stderr, "HT %s %d %d %.0f %d %s %.0f %.0f %.1f %.1f %.0f %.0f %.0f %.2f %.0f %d %d %d %d %.0f %.0f %d\n", b.P.Name, b.seed, b.slot, b.t, b.o.Alive, b.mode, x, y, me.Pos[2], me.Pos[2]-b.w.Ground(x, y),
		z.Radius, z.NextRadius-l, gap, b.spotExpoZ(x, y, me.Pos[2]), near, b.hideN[0], b.hideN[1], b.hideN[2], b.hideN[3], me.HP, gd, len(b.noise))
}

func (b *Bot) hideOn() bool {
	return b.ffaSure && !b.practice && (b.P.HideCover > 0 || b.P.HideGeo > 0 || b.P.HideCorner > 0 || b.P.HideCtr > 0 || b.P.HideRoof > 0 || b.P.HideStick > 0)
}

func (b *Bot) spotExpo(x, y float64) float64 {
	c := b.w.Nav.Cell(x, y)
	if v, ok := b.hideExpo[c]; ok {
		return float64(v)
	}
	if b.hideExpo == nil {
		b.hideExpo = map[int]float32{}
	}
	e := b.spotExpoZ(x, y, b.w.Ground(x, y))
	b.hideExpo[c] = float32(e)
	return e
}

func (b *Bot) spotExpoZ(x, y, gz float64) float64 {
	nv := b.w.Nav
	h := b.me.Stats.Height
	tot, vis := 0, 0
	for ri, r := range []float64{40, 80, 120, 170} {
		for k := 0; k < 12; k++ {
			a := float64(k)*math.Pi/6 + 0.26*float64(ri)
			ox, oy := x+r*math.Cos(a), y+r*math.Sin(a)
			if !nv.WalkableMain(ox, oy) {
				continue
			}
			tot++
			ez := b.w.Ground(ox, oy) + 8
			if b.w.ClearCoarse(ox, oy, ez, x, y, gz+h-0.5) || b.w.ClearCoarse(ox, oy, ez, x, y, gz+h*0.5) {
				vis++
			}
		}
	}
	if tot == 0 {
		return 0
	}
	return float64(vis) / float64(tot)
}

var hideLattice = func() [][2]float64 {
	var l [][2]float64
	for r := 0; r <= 20; r++ {
		for i := -r; i <= r; i++ {
			for j := -r; j <= r; j++ {
				if (i == -r || i == r || j == -r || j == r) && i*i+j*j <= 400 {
					l = append(l, [2]float64{float64(i) * 8, float64(j) * 8})
				}
			}
		}
	}
	return l
}()

func (b *Bot) hideSurvey() {
	px, py := b.me.Pos[0], b.me.Pos[1]
	if hyp(px, py, b.hideSurvC[0], b.hideSurvC[1]) > 40 || b.hideSurvC == [2]float64{} {
		b.hideSurvC = [2]float64{math.Round(px/8) * 8, math.Round(py/8) * 8}
		b.hideSurvI = 0
	}
	nv := b.w.Nav
	if b.hideExpo == nil {
		b.hideExpo = map[int]float32{}
	}
	fresh := 0
	for n := 0; n < 80 && fresh < 5 && b.hideSurvI < len(hideLattice); n++ {
		o := hideLattice[b.hideSurvI]
		b.hideSurvI++
		x, y := b.hideSurvC[0]+o[0], b.hideSurvC[1]+o[1]
		if !nv.WalkableMain(x, y) {
			continue
		}
		if _, ok := b.hideExpo[nv.Cell(x, y)]; ok {
			continue
		}
		b.spotExpo(x, y)
		fresh++
	}
}

func (b *Bot) hideSpot() [2]float64 {
	me := b.me
	px, py := me.Pos[0], me.Pos[1]
	nv := b.w.Nav
	zs := b.zoneAt(b.t + 25)
	h := me.Stats.Height

	if b.P.HideCorner > 0 {
		var eyes [][3]float64
		looking := false
		for _, t := range b.trackL {
			if t.NPC || t.Ghost || b.t-t.Seen > 4 {
				continue
			}
			if d := hyp(px, py, t.Pos[0], t.Pos[1]); d > 230 {
				continue
			}
			eyes = append(eyes, [3]float64{t.Pos[0], t.Pos[1], t.Pos[2] + 8*t.Scale})
			if b.t-t.Seen < 1.0 {
				looking = true
			}
		}
		hidden := func(x, y float64) bool {
			gz := b.w.Ground(x, y)
			for _, e := range eyes {
				if b.bodyExposed(e, x, y, gz) {
					return false
				}
			}
			return true
		}
		if len(eyes) > 0 {
			if b.t < b.hideCornerT && hidden(b.hideCornerG[0], b.hideCornerG[1]) && zs.outside(b.hideCornerG[0], b.hideCornerG[1]) <= -8 {
				return b.hideCornerG
			}
			if looking || b.t < b.hideCornerT {
				best, bd, found := [2]float64{px, py}, math.Inf(1), false
				if hidden(px, py) && zs.outside(px, py) <= -8 {
					found, bd = true, 0
				}
				for r := 6.0; r <= b.P.HideCorner+0.1 && bd > r-6; r += 6 {
					for k := 0; k < 12; k++ {
						a := float64(k)*math.Pi/6 + 0.2*r
						x, y := px+r*math.Cos(a), py+r*math.Sin(a)
						if !nv.WalkableMain(x, y) || zs.outside(x, y) > -8 || !hidden(x, y) {
							continue
						}

						d := b.roughDist(x, y)
						for _, e := range eyes {
							if hyp(x, y, e[0], e[1]) < hyp(px, py, e[0], e[1])-8 {
								d += 25
							}
						}
						if d < bd {
							bd, best, found = d, [2]float64{x, y}, true
						}
					}
				}
				if found {
					b.hideCornerG, b.hideCornerT = best, b.t+5
					b.shyGoal, b.shyT = best, b.t+1
					b.hideN[1]++
					return best
				}
			}
		}
	}

	if b.t < b.shyT {
		return b.shyGoal
	}
	b.shyT = b.t + 2.5
	prev, hadPrev := b.shyGoal, b.shyGoal != [2]float64{}
	var oth [][2]float64
	for _, t := range b.trackL {
		if !t.NPC && !t.Ghost && b.t-t.Seen < 12 {
			oth = append(oth, [2]float64{t.Pos[0], t.Pos[1]})
		}
	}
	for i := range b.noise {
		if b.t-b.noise[i].t < 12 {
			oth = append(oth, b.noise[i].pos)
		}
	}
	z := &b.o.Zone
	zN := zoneC{z.NextCenter, z.NextRadius}
	geo := b.P.HideGeo > 0 && zN.r >= 40 && zN.r < z.Radius-1
	in := math.Min(15, zN.r*0.3)
	fresh := 16
	score := func(x, y float64) float64 {
		m := 260.0
		if b.P.HideStick > 0 {
			m = 200
		}
		for _, o := range oth {
			if d := hyp(x, y, o[0], o[1]); d < m {
				m = d
			}
		}
		if b.P.HideCover > 0 {
			if _, ok := b.hideExpo[b.w.Nav.Cell(x, y)]; !ok {
				if fresh <= 0 {
					return math.Inf(-1)
				}
				fresh--
			}
			m += b.P.HideCover * (1 - b.spotExpo(x, y))
		}
		if b.P.HideCtr > 0 && z.NextRadius < z.Radius-1 {
			gz := b.w.Ground(x, y)
			cz := b.w.Ground(zN.c[0], zN.c[1]) + 8
			if !b.w.ClearCoarse(zN.c[0], zN.c[1], cz, x, y, gz+h-0.5) && !b.w.ClearCoarse(zN.c[0], zN.c[1], cz, x, y, gz+h*0.5) {
				m += b.P.HideCtr
			}
		}
		if geo {
			vx, vy := x-zN.c[0], y-zN.c[1]
			l := math.Hypot(vx, vy)
			if l < 1 {
				vx, vy, l = 1, 0, 1
			}
			ux, uy := vx/l, vy/l
			depth := zN.r - l
			if depth < in {

				return -1e6 - l
			}

			ex, ey := zN.c[0]-z.Center[0], zN.c[1]-z.Center[1]
			bb := ex*ux + ey*uy
			disc := bb*bb - (ex*ex + ey*ey - z.Radius*z.Radius)
			gap := -bb + math.Sqrt(math.Max(disc, 0)) - zN.r
			m -= b.P.HideGeo * clamp(gap, 0, 200) / 200

			m -= b.P.HideGeo * 0.5 * clamp(depth/zN.r-0.3, 0, 0.7)

			for _, o := range oth {
				dj := hyp(o[0], o[1], zN.c[0], zN.c[1])
				if dj <= zN.r {
					continue
				}
				sx, sy := zN.c[0]-o[0], zN.c[1]-o[1]
				tt := clamp(((x-o[0])*sx+(y-o[1])*sy)/(sx*sx+sy*sy), 0, 1)
				if hyp(o[0]+tt*sx, o[1]+tt*sy, x, y) < 45 {
					m -= b.P.HideGeo * 0.6
				}
			}
		}
		return m
	}

	stick := 12.0
	if b.P.HideStick > 0 {
		stick = b.P.HideStick
	}
	b.shyGoal = [2]float64{px, py}
	bs := score(px, py) + stick
	if !nv.WalkableMain(px, py) || zs.outside(px, py) > -12 {
		bs = math.Inf(-1)
	}
	rings := []float64{30, 60, 90}
	if b.P.HideCover > 0 || b.P.HideGeo > 0 || b.P.HideCtr > 0 {
		rings = []float64{15, 30, 60, 90, 130}
	}
	for ri, r := range rings {
		for k := 0; k < 12; k++ {
			a := float64(k)*math.Pi/6 + 0.26*float64(ri%2)
			x, y := px+r*math.Cos(a), py+r*math.Sin(a)
			if !nv.WalkableMain(x, y) || zs.outside(x, y) > -12 {
				continue
			}
			if sc := score(x, y) - 0.15*r; sc > bs {
				bs, b.shyGoal = sc, [2]float64{x, y}
			}
		}
	}
	if b.P.HideCover > 0 {

		for _, o := range hideLattice {
			x, y := b.hideSurvC[0]+o[0], b.hideSurvC[1]+o[1]
			e, ok := b.hideExpo[nv.Cell(x, y)]
			if !ok || e > 0.25 || !nv.WalkableMain(x, y) || zs.outside(x, y) > -12 {
				continue
			}
			if sc := score(x, y) - 0.18*hyp(x, y, px, py); sc > bs {
				bs, b.shyGoal = sc, [2]float64{x, y}
			}
		}
	}
	if hadPrev {

		if dp := hyp(px, py, prev[0], prev[1]); dp > 4 && dp < 200 && nv.WalkableMain(prev[0], prev[1]) && zs.outside(prev[0], prev[1]) <= -12 {
			if sp := score(prev[0], prev[1]); sp > -1e5 && bs < sp+10 {
				b.shyGoal = prev
			}
		}
	}
	b.hideN[0]++
	return b.shyGoal
}
