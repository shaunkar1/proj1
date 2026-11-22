package bot

import (
	"fmt"
	"math"
	"os"

	"mechbot/geo"
)

type threat struct {
	t      *Track
	eye    [3]float64
	weight float64
	isTgt  bool
}

type moveCtx struct {
	goal     *[2]float64
	threats  []threat
	target   *Track
	wantLOS  float64
	pref     float64
	strafe   bool
	hide     bool
	boost    bool
	wGoal    float64
	arrive   float64
	mustMove bool
	keepOff  float64
	brave    bool
}

type zoneC struct {
	c [2]float64
	r float64
}

func (b *Bot) zoneAt(t float64) zoneC {
	z := &b.o.Zone
	if t <= z.ShrinkStart || z.ShrinkEnd <= z.ShrinkStart {
		if t >= z.ShrinkEnd && z.ShrinkEnd > 0 && z.ShrinkEnd <= z.ShrinkStart {
			return zoneC{z.NextCenter, z.NextRadius}
		}
		return zoneC{z.Center, z.Radius}
	}
	if t >= z.ShrinkEnd {
		return zoneC{z.NextCenter, z.NextRadius}
	}

	f := (t - b.t) / (z.ShrinkEnd - b.t)
	if b.t < z.ShrinkStart {
		f = (t - z.ShrinkStart) / (z.ShrinkEnd - z.ShrinkStart)
	}
	f = clamp(f, 0, 1)
	return zoneC{[2]float64{z.Center[0] + (z.NextCenter[0]-z.Center[0])*f, z.Center[1] + (z.NextCenter[1]-z.Center[1])*f}, z.Radius + (z.NextRadius-z.Radius)*f}
}

func (z zoneC) outside(x, y float64) float64 { return hyp(x, y, z.c[0], z.c[1]) - z.r }

func (b *Bot) navTo(gx, gy float64) (float64, float64, bool) {
	nv := b.w.Nav
	me := b.me
	if b.path == nil || hyp(gx, gy, b.pathGoal[0], b.pathGoal[1]) > 5 || b.t-b.pathT > 0.8 || !b.pathOK {
		if (b.t-b.pathT > 0.25 || hyp(gx, gy, b.pathGoal[0], b.pathGoal[1]) > 5) && (b.searchTick != b.o.Tick || b.path == nil) {
			b.path, b.pathCost = nv.Path(me.Pos[0], me.Pos[1], gx, gy, 30000, b.navExtra())
			b.searchTick = b.o.Tick
			b.pathGoal = [2]float64{gx, gy}
			b.pathT = b.t
			b.pathOK = b.path != nil
		}
	}
	if b.path == nil {
		return gx, gy, false
	}

	bi, bd := 0, math.Inf(1)
	for i, p := range b.path {
		d := hyp(p[0], p[1], me.Pos[0], me.Pos[1])
		if d < bd {
			bd, bi = d, i
		}
	}

	best := bi
	if bi+1 < len(b.path) {
		best = bi + 1
	}
	acc := 0.0
	for i := bi + 1; i < len(b.path); i++ {
		acc += hyp(b.path[i][0], b.path[i][1], b.path[i-1][0], b.path[i-1][1])
		if acc > 26 {
			break
		}
		if nv.WalkLine(me.Pos[0], me.Pos[1], b.path[i][0], b.path[i][1]) {
			best = i
		}
	}
	return b.path[best][0], b.path[best][1], true
}

func (b *Bot) navExtra() func(int) float32 {
	nv := b.w.Nav
	z := b.zoneAt(b.t + 4)
	r2 := (z.r - 3) * (z.r - 3)
	nb := len(b.blockedC)
	oc := float32(b.P.OpenCost)
	return func(c int) float32 {
		var e float32
		if nb > 0 {
			if _, bad := b.blockedC[c]; bad {
				e += 30
			}
		}
		x, y := nv.Pos(c)
		dx, dy := x-z.c[0], y-z.c[1]
		if dx*dx+dy*dy > r2 {
			e += 12
		}
		if oc > 0 {

			if op := nv.Open[c]; op > 24 {
				e += oc * float32(math.Min(float64(op-24)/30, 1))
			}
		}
		return e
	}
}

func (b *Bot) pathCostTo(gx, gy float64) float64 {
	b.searchTick = b.o.Tick
	p, c := b.w.Nav.Path(b.me.Pos[0], b.me.Pos[1], gx, gy, 30000, b.navExtra())
	if p == nil {
		return -1
	}
	return c
}

func (b *Bot) roughDist(x, y float64) float64 {
	d := hyp(x, y, b.me.Pos[0], b.me.Pos[1])
	if d < 12 || b.w.Nav.WalkLine(b.me.Pos[0], b.me.Pos[1], x, y) {
		return d
	}
	return d * 1.35
}

func (b *Bot) exposed(eye [3]float64, x, y, z float64) bool {
	return b.w.ClearCoarse(eye[0], eye[1], eye[2], x, y, z)
}

func (b *Bot) peekOn() bool {
	return !b.P.NoPeek && !(b.practice && b.P.PracNoPeek)
}

func (b *Bot) bodyExposed(eye [3]float64, x, y, gz float64) bool {
	me := b.me
	h := me.Stats.Height
	cz := gz + h*0.5
	if b.w.ClearCoarse(eye[0], eye[1], eye[2], x, y, cz) || b.w.ClearCoarse(eye[0], eye[1], eye[2], x, y, gz+h-0.5) {
		return true
	}
	dx, dy := x-eye[0], y-eye[1]
	l := math.Hypot(dx, dy)
	if l < 1 {
		return true
	}
	r := me.Stats.Radius + 0.4
	px, py := -dy/l*r, dx/l*r
	return b.w.ClearCoarse(eye[0], eye[1], eye[2], x+px, y+py, cz) || b.w.ClearCoarse(eye[0], eye[1], eye[2], x-px, y-py, cz)
}

func (b *Bot) steer(ctx *moveCtx) (mx, my float64, boost bool) {
	me := b.me
	nv := b.w.Nav
	px, py := me.Pos[0], me.Pos[1]
	speed := me.Stats.WalkSpeed
	onNav := nv.Walkable(px, py)

	if !onNav {
		c := nv.Nearest(px, py, 80)
		if c >= 0 {
			tx, ty := nv.Pos(c)
			if ctx.goal != nil {

				gc := nv.Nearest(px+(ctx.goal[0]-px)*0.15, py+(ctx.goal[1]-py)*0.15, 60)
				if gc >= 0 {
					gx, gy := nv.Pos(gc)
					if hyp(gx, gy, px, py) < hyp(tx, ty, px, py)*1.6+4 {
						tx, ty = gx, gy
					}
				}
			}
			d := hyp(tx, ty, px, py)
			if d > 0.3 {
				return (tx - px) / d, (ty - py) / d, false
			}
		}
	}
	var carrot [2]float64
	hasCarrot := false
	if ctx.goal != nil {
		gd := hyp(ctx.goal[0], ctx.goal[1], px, py)
		if gd > ctx.arrive {
			cx, cy, _ := b.navTo(ctx.goal[0], ctx.goal[1])
			carrot = [2]float64{cx, cy}
			hasCarrot = true
		}
	}
	if Debug && (DebugSlot < 0 || b.slot == DebugSlot) && os.Getenv("MECH_DEBUG") == "nav" {
		gx, gy := 0.0, 0.0
		if ctx.goal != nil {
			gx, gy = ctx.goal[0], ctx.goal[1]
		}
		z := b.o.Zone
		fmt.Fprintf(os.Stderr, "NAV t=%.1f pos=(%.0f,%.0f) goal=(%.0f,%.0f) carrot=(%.0f,%.0f) has=%v pathlen=%d ok=%v zone=(%.0f,%.0f,%.0f)->(%.0f,%.0f,%.0f) %0.f-%.0f\n", b.t, px, py, gx, gy, carrot[0], carrot[1], hasCarrot, len(b.path), b.pathOK, z.Center[0], z.Center[1], z.Radius, z.NextCenter[0], z.NextCenter[1], z.NextRadius, z.ShrinkStart, z.ShrinkEnd)
	}

	if b.t >= b.strafeT {
		b.strafeSign = -b.strafeSign
		if b.rng.Float64() < 0.3 {
			b.strafeSign = -b.strafeSign
		}
		b.strafeT = b.t + b.P.JukeMin + b.rng.Float64()*(b.P.JukeMax-b.P.JukeMin)
	}
	var sdx, sdy float64
	if ctx.strafe && ctx.target != nil {
		tx, ty := ctx.target.Pos[0]-px, ctx.target.Pos[1]-py
		l := math.Hypot(tx, ty)
		if l > 0.1 {
			sdx, sdy = -ty/l*b.strafeSign, tx/l*b.strafeSign
		}
	}
	zNow := b.zoneAt(b.t + 1.0)
	zSoon := b.zoneAt(b.t + 4.0)
	myH := me.Stats.Height * 0.5

	var incs []inc
	var dodgePen [17]float64
	haveDodge := false
	for i := range b.o.Projectiles {
		p := &b.o.Projectiles[i]
		rx, ry := px-p.Pos[0], py-p.Pos[1]
		if p.Kind == "mortar" {

			x, y, z := p.Pos[0], p.Pos[1], p.Pos[2]
			vz := p.Vel[2]
			tt := 0.0
			g := b.cfg.Weapons["mortar"].Gravity
			if g == 0 {
				g = 22
			}
			for tt < 7 {
				x += p.Vel[0] * 0.05
				y += p.Vel[1] * 0.05
				vz -= g * 0.05
				z += vz * 0.05
				tt += 0.05
				if b.w.SolidAt(x, y, z) {
					break
				}
			}
			if hyp(x, y, px, py) < 30 && tt < 3 {
				incs = append(incs, inc{p: p.Pos, v: p.Vel, mortar: true, ix: x, iy: y, it: tt})
			}
			continue
		}

		vl := math.Hypot(p.Vel[0], p.Vel[1])
		if vl < 1 {
			continue
		}
		along := (rx*p.Vel[0] + ry*p.Vel[1]) / vl
		if along < 0 || along > 260 {
			continue
		}
		perp := math.Abs(rx*p.Vel[1]-ry*p.Vel[0]) / vl
		if perp > 14 {
			continue
		}
		incs = append(incs, inc{p: p.Pos, v: p.Vel})
	}

	if len(incs) > 0 && !b.P.NoDodge {
		dodgePen = b.dodgePlan(incs, speed)
		haveDodge = true
	}
	if Debug && b.P.NoFire {
		fmt.Fprintf(os.Stderr, "t=%.2f pos=(%.1f,%.1f) vel=(%.1f,%.1f) nproj=%d incs=%d pen=%.0f\n", b.t, px, py, me.Vel[0], me.Vel[1], len(b.o.Projectiles), len(incs), dodgePen)
		for _, p := range b.o.Projectiles {
			fmt.Fprintf(os.Stderr, "    proj %d pos=(%.1f,%.1f,%.1f) vel=(%.1f,%.1f,%.1f)\n", p.ID, p.Pos[0], p.Pos[1], p.Pos[2], p.Vel[0], p.Vel[1], p.Vel[2])
		}
	}
	type cand struct {
		dx, dy, r float64
		free      bool
	}
	cands := make([]cand, 0, 40)
	cands = append(cands, cand{0, 0, 0, false})
	for k := 0; k < 16; k++ {
		a := float64(k) * math.Pi / 8
		cands = append(cands, cand{math.Cos(a), math.Sin(a), 2.5, false}, cand{math.Cos(a), math.Sin(a), 5, false}, cand{math.Cos(a), math.Sin(a), 11, false})
	}
	if hasCarrot {
		d := hyp(carrot[0], carrot[1], px, py)
		if d > 0.5 {
			cands = append(cands, cand{(carrot[0] - px) / d, (carrot[1] - py) / d, math.Min(d, 11), true})
		}
	}
	bestS := math.Inf(-1)
	var bx, by float64
	_ = 9.0

	peek := b.peekOn()
	myLasReady, haveLas := false, false
	lasVal := 0.0
	if las, _ := b.weapon("laser"); las != nil {
		haveLas = true
		if las.Charge >= las.Stats.Charge-0.25 && las.Charge > 0 {
			myLasReady = true
			lasVal = las.Stats.Damage * b.dmgMul()
		}
	}
	type scored struct {
		s      float64
		ok     bool
		expTgt bool
		x, y   float64
	}
	sc := make([]scored, len(cands))
	anyExp, anyHid := false, false
	for ci, c := range cands {
		x, y := px+c.dx*c.r, py+c.dy*c.r
		if c.r > 0 && !c.free {
			if onNav {
				if !nv.WalkLine(px, py, x, y) {
					continue
				}
			} else if !b.w.InBounds(x, y) {
				continue
			}
		}
		if c.r == 0 && ctx.mustMove {
			continue
		}
		s := 0.0
		gz := b.w.Ground(x, y)
		if onNav {
			if cc := nv.Cell(x, y); cc >= 0 {
				gz = float64(nv.Hgt[cc])
				s -= float64(nv.Cost[cc]) * 2
				if b.P.OpenCost > 0 && nv.Open[cc] > 24 {
					s -= b.P.OpenCost * 4 * math.Min(float64(nv.Open[cc]-24)/30, 1)
				}
				if _, bad := b.blockedC[cc]; bad {
					s -= 40
				}
			}
		}
		_ = myH
		if hasCarrot {
			s -= ctx.wGoal * hyp(x, y, carrot[0], carrot[1])
		} else if ctx.goal != nil {

			gd := hyp(x, y, ctx.goal[0], ctx.goal[1])
			if b.P.OldArrive {
				s -= ctx.wGoal * 0.5 * gd
			} else if gd > ctx.arrive {
				s -= ctx.wGoal * 0.5 * (gd - ctx.arrive)
			}
			if ctx.keepOff > 0 && gd < ctx.keepOff {
				s -= 6 * (ctx.keepOff - gd)
			}
		}

		if !(c.free && ctx.mustMove) {
			if o := zNow.outside(x, y); o > -2 {
				s -= 400 + 25*(o+2)
			} else if o2 := zSoon.outside(x, y); o2 > -4 {
				s -= 60 + 8*(o2+4)
			}
		}

		for i := range ctx.threats {
			th := &ctx.threats[i]
			d := hyp(x, y, th.t.Pos[0], th.t.Pos[1])
			if d > 290 || (c.free && ctx.brave) {
				continue
			}
			exp := b.bodyExposed(th.eye, x, y, gz)
			if th.isTgt {
				sc[ci].expTgt = exp
			}
			if !exp && peek && th.t.has[2] && !th.t.NPC {

				e2 := [3]float64{th.eye[0] + th.t.Vel[0]*0.7, th.eye[1] + th.t.Vel[1]*0.7, th.eye[2]}
				if b.bodyExposed(e2, x, y, gz) {
					s -= 0.5 * 315 * math.Max(th.weight, 0.6) * boolf(b.laserIn(th.t) < 1.5)
				}
			}
			if exp {
				if peek && th.t.has[2] {

					s -= b.P.WExpo * b.enemyDpsNL(th.t, d) * th.weight
					if lt := b.laserIn(th.t); lt < 0.85 && d < 285 {
						s -= 1.25 * 315 * b.cubeMul(th.t.Cubes) * math.Max(th.weight, 0.6)
					} else if lt < 1.6 {
						s -= 40 * th.weight
					}
				} else {
					s -= b.P.WExpo * b.enemyDps(th.t, d) * th.weight
				}
				if ctx.hide {
					s -= 80 * th.weight
				}
				if th.isTgt && ctx.wantLOS != 0 {
					if peek && haveLas {
						s += ctx.wantLOS * b.P.WAtk * b.myDpsX(th.t, d, true)
						if myLasReady {
							s += ctx.wantLOS * lasVal
						}
					} else {
						s += ctx.wantLOS * b.P.WAtk * b.myDps(th.t, d)
					}
				}
			}

			if th.t.has[3] && d < 54 && !(th.isTgt && ctx.pref > 0 && ctx.pref < 50) {
				s -= (54 - d) * 7
			}

			if !th.t.NPC && !(th.isTgt && ctx.pref < 60) && d < 60 {
				s -= (60 - d) * 1.5
			}
			if d < 7 {
				s -= (7 - d) * 10
			}
		}
		if ctx.target != nil && ctx.pref > 0 {
			d := hyp(x, y, ctx.target.Pos[0], ctx.target.Pos[1])
			wr := b.P.WRange
			if peek && ctx.target.has[2] && b.coverNear {
				wr *= 0.15
			}
			s -= wr * math.Abs(d-ctx.pref)
		}
		if ctx.strafe && c.r > 0 {
			s += 5.0 * (c.dx*sdx + c.dy*sdy) * c.r / 5
		}

		if haveDodge {
			key := 0
			if c.r > 0 {
				k := math.Atan2(c.dy, c.dx) / (math.Pi / 8)
				kr := math.Round(k)
				key = 1 + (int(kr)+32)%16
				if math.Abs(k-kr) > 0.01 {

				}
			}
			s -= dodgePen[key]
		}

		if c.r > 0 {
			s += 1.2 * (c.dx*b.lastMove[0] + c.dy*b.lastMove[1])
		}
		sc[ci] = scored{s: s, ok: true, expTgt: sc[ci].expTgt, x: x, y: y}
		if sc[ci].expTgt {
			anyExp = true
		} else {
			anyHid = true
		}
	}
	b.coverNear = anyHid

	lurk := peek && ctx.target != nil && ctx.wantLOS > 0 && anyExp && anyHid && (haveLas || ctx.target.has[2])
	for ci, c := range cands {
		if !sc[ci].ok {
			continue
		}
		s := sc[ci].s
		if lurk && !sc[ci].expTgt {
			md := 14.0
			for cj := range cands {
				if sc[cj].ok && sc[cj].expTgt {
					if d := hyp(sc[ci].x, sc[ci].y, sc[cj].x, sc[cj].y); d < md {
						md = d
					}
				}
			}
			s -= 1.5 * md
		}
		if s > bestS {
			bestS = s
			bx, by = c.dx*boolf(c.r > 0), c.dy*boolf(c.r > 0)
			if c.r > 0 && c.r < 2 {
				bx, by = bx*c.r/2, by*c.r/2
			}
		}
	}
	if math.IsInf(bestS, -1) {

		if hasCarrot {
			d := hyp(carrot[0], carrot[1], px, py)
			if d > 0.1 {
				return (carrot[0] - px) / d, (carrot[1] - py) / d, ctx.boost
			}
		}
		return 0, 0, false
	}
	return bx, by, ctx.boost
}

type inc struct {
	p, v   [3]float64
	mortar bool
	ix, iy float64
	it     float64
}

const dodgeSteps = 20

var dirs16 = func() (d [17][2]float64) {
	for k := 0; k < 16; k++ {
		a := float64(k) * math.Pi / 8
		d[k+1] = [2]float64{math.Cos(a), math.Sin(a)}
	}
	return
}()

func (b *Bot) dodgePlan(incs []inc, speed float64) [17]float64 {
	me := b.me
	const dt = 0.05
	acc := me.Stats.GroundAccel
	if !me.Grounded {
		acc = me.Stats.AirAccel
	}
	R := me.Stats.Radius
	nv := b.w.Nav

	type rb struct {
		px, py, vx, vy float64
		z0, vz         float64
		mortar         bool
		ix, iy         float64
		ik             int
	}
	rs := make([]rb, 0, len(incs))
	for i := range incs {
		in := &incs[i]
		r := rb{px: in.p[0], py: in.p[1], vx: in.v[0], vy: in.v[1], z0: in.p[2], vz: in.v[2], mortar: in.mortar, ix: in.ix, iy: in.iy}
		r.ik = int(in.it/dt + 0.5)
		rs = append(rs, r)
	}
	if len(rs) > 14 {
		rs = rs[:14]
	}
	zLo, zHi := me.Pos[2]-1.0, me.Pos[2]+me.Stats.Height+1.0
	splash := 10.0

	stepCost := func(k int, x, y float64) float64 {
		c := 0.0
		t0 := float64(k) * dt
		for i := range rs {
			r := &rs[i]
			if r.mortar {
				if k == r.ik || (k == dodgeSteps-1 && r.ik >= dodgeSteps) {
					d := math.Hypot(x-r.ix, y-r.iy)
					if d < splash {
						c += 170 * (1 - d/splash)
					}
				}
				continue
			}
			ax, ay := r.px+r.vx*t0, r.py+r.vy*t0
			sx, sy := r.vx*dt, r.vy*dt
			l2 := sx*sx + sy*sy
			u := ((x-ax)*sx + (y-ay)*sy) / l2
			if u < -0.2 || u > 1.2 {
				continue
			}
			uc := clamp(u, 0, 1)
			d := math.Hypot(x-(ax+sx*uc), y-(ay+sy*uc))
			if d > R+1.6 {
				continue
			}
			bz := r.z0 + r.vz*(t0+uc*dt)
			if bz < zLo || bz > zHi {
				continue
			}
			if d < R+0.45 {
				c += 100
			} else {
				c += 14 * (1 - (d-R-0.45)/1.15)
			}
		}
		return c
	}
	type st struct{ x, y, vx, vy float64 }
	adv := func(s st, ux, uy float64) st {
		dvx, dvy := ux*speed-s.vx, uy*speed-s.vy
		dl := math.Hypot(dvx, dvy)
		if dl > acc*dt {
			dvx *= acc * dt / dl
			dvy *= acc * dt / dl
		}
		s.vx += dvx
		s.vy += dvy
		s.x += s.vx * dt
		s.y += s.vy * dt
		return s
	}
	const leg1, leg2 = 2, 5
	var out [17]float64
	s0 := st{me.Pos[0], me.Pos[1], me.Vel[0], me.Vel[1]}
	onNav := nv.Walkable(s0.x, s0.y)
	for d1 := 0; d1 < 17; d1++ {
		s1 := s0
		c1 := 0.0
		for k := 0; k < leg1; k++ {
			s1 = adv(s1, dirs16[d1][0], dirs16[d1][1])
			c1 += stepCost(k, s1.x, s1.y)
		}
		best := math.Inf(1)
		for d2 := 0; d2 < 17; d2 += 2 {
			if d2 > 0 {
				d2--
			}
			s2 := s1
			c2 := c1
			for k := leg1; k < leg2 && c2 < best; k++ {
				s2 = adv(s2, dirs16[d2][0], dirs16[d2][1])
				c2 += stepCost(k, s2.x, s2.y)
			}
			if c2 < best {
				for d3 := 0; d3 < 17; d3 += 2 {
					if d3 > 0 {
						d3--
					}
					s3 := s2
					c3 := c2
					for k := leg2; k < dodgeSteps && c3 < best; k++ {
						s3 = adv(s3, dirs16[d3][0], dirs16[d3][1])
						c3 += stepCost(k, s3.x, s3.y)
					}
					if c3 < best && onNav && !nv.Walkable(s3.x, s3.y) {
						c3 += 30
					}
					if c3 < best {
						best = c3
					}
					if d3 > 0 {
						d3++
					}
				}
			}
			if d2 > 0 {
				d2++
			}
		}
		out[d1] = best
	}

	mn := math.Inf(1)
	for _, v := range out {
		if v < mn {
			mn = v
		}
	}
	for i := range out {
		out[i] -= mn
	}
	return out
}

func boolf(v bool) float64 {
	if v {
		return 1
	}
	return 0
}

var _ = geo.NavStep
