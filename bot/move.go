package bot

import (
	"math"

	"mechbot/geo"
)

// threat is an enemy as the movement planner sees it.
type threat struct {
	t      *Track
	eye    [3]float64
	weight float64 // how hostile (0..1)
	isTgt  bool
}

type moveCtx struct {
	goal    *[2]float64 // nav goal (nil = hold area)
	threats []threat
	target  *Track
	wantLOS float64 // + seek line of fire to target, - avoid
	pref    float64 // preferred range to target (0 = none)
	strafe  bool
	hide    bool // avoid all sight lines
	boost   bool
	wGoal   float64
	arrive  float64 // stop within this distance of the goal
}

type zoneC struct {
	c [2]float64
	r float64
}

// zoneAt returns the zone circle at time t (within the current phase).
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
	// the observed circle is already partway through the shrink at b.t
	f := (t - b.t) / (z.ShrinkEnd - b.t)
	if b.t < z.ShrinkStart {
		f = (t - z.ShrinkStart) / (z.ShrinkEnd - z.ShrinkStart)
	}
	f = clamp(f, 0, 1)
	return zoneC{[2]float64{z.Center[0] + (z.NextCenter[0]-z.Center[0])*f, z.Center[1] + (z.NextCenter[1]-z.Center[1])*f}, z.Radius + (z.NextRadius-z.Radius)*f}
}

func (z zoneC) outside(x, y float64) float64 { return hyp(x, y, z.c[0], z.c[1]) - z.r }

// navTo returns a short-range steering point toward the goal.
func (b *Bot) navTo(gx, gy float64) (float64, float64, bool) {
	nv := b.w.Nav
	me := b.me
	if b.path == nil || hyp(gx, gy, b.pathGoal[0], b.pathGoal[1]) > 5 || b.t-b.pathT > 0.8 || !b.pathOK {
		if b.t-b.pathT > 0.25 || hyp(gx, gy, b.pathGoal[0], b.pathGoal[1]) > 5 {
			extra := func(c int) float32 {
				if _, bad := b.blockedC[c]; bad {
					return 30
				}
				return 0
			}
			var ex func(int) float32
			if len(b.blockedC) > 0 {
				ex = extra
			}
			b.path, b.pathCost = nv.Path(me.Pos[0], me.Pos[1], gx, gy, 40000, ex)
			b.pathGoal = [2]float64{gx, gy}
			b.pathT = b.t
			b.pathOK = b.path != nil
		}
	}
	if b.path == nil {
		return gx, gy, false
	}
	// closest path point
	bi, bd := 0, math.Inf(1)
	for i, p := range b.path {
		d := hyp(p[0], p[1], me.Pos[0], me.Pos[1])
		if d < bd {
			bd, bi = d, i
		}
	}
	// furthest point reachable in a straight walkable line within lookahead
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

// pathDist estimates walking distance to a point (cheap, straight-line based).
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

// steer picks the move direction by scoring sampled positions around us.
func (b *Bot) steer(ctx *moveCtx) (mx, my float64, boost bool) {
	me := b.me
	nv := b.w.Nav
	px, py := me.Pos[0], me.Pos[1]
	speed := me.Stats.WalkSpeed
	onNav := nv.Walkable(px, py)
	// off the walkable grid (roof, ledge, pocket): head for the nearest open ground
	if !onNav {
		c := nv.Nearest(px, py, 80)
		if c >= 0 {
			tx, ty := nv.Pos(c)
			if ctx.goal != nil {
				// prefer leaving in the goal's direction when several exits are similar
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
	// strafe side flips at random intervals
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

	// incoming projectiles
	var incs []inc
	var dodgeDone [17]bool
	var dodgePen [17]float64
	for i := range b.o.Projectiles {
		p := &b.o.Projectiles[i]
		rx, ry := px-p.Pos[0], py-p.Pos[1]
		if p.Kind == "mortar" {
			// integrate to impact
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
		// heading roughly at us?
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

	type cand struct{ dx, dy, r float64 }
	cands := make([]cand, 0, 40)
	cands = append(cands, cand{0, 0, 0})
	for k := 0; k < 16; k++ {
		a := float64(k) * math.Pi / 8
		cands = append(cands, cand{math.Cos(a), math.Sin(a), 5}, cand{math.Cos(a), math.Sin(a), 11})
	}
	if hasCarrot {
		d := hyp(carrot[0], carrot[1], px, py)
		if d > 0.5 {
			cands = append(cands, cand{(carrot[0] - px) / d, (carrot[1] - py) / d, math.Min(d, 11)})
		}
	}
	bestS := math.Inf(-1)
	var bx, by float64
	splash := 9.0
	for _, c := range cands {
		x, y := px+c.dx*c.r, py+c.dy*c.r
		if c.r > 0 {
			if onNav {
				if !nv.WalkLine(px, py, x, y) {
					continue
				}
			} else if !b.w.InBounds(x, y) {
				continue
			}
		}
		s := 0.0
		gz := b.w.Ground(x, y)
		if onNav {
			if cc := nv.Cell(x, y); cc >= 0 {
				gz = float64(nv.Hgt[cc])
				s -= float64(nv.Cost[cc]) * 2
				if _, bad := b.blockedC[cc]; bad {
					s -= 40
				}
			}
		}
		cz := gz + myH
		if hasCarrot {
			s -= ctx.wGoal * hyp(x, y, carrot[0], carrot[1])
		} else if ctx.goal != nil {
			s -= ctx.wGoal * 0.5 * hyp(x, y, ctx.goal[0], ctx.goal[1])
		}
		// zone
		if o := zNow.outside(x, y); o > -2 {
			s -= 400 + 25*(o+2)
		} else if o2 := zSoon.outside(x, y); o2 > -4 {
			s -= 60 + 8*(o2+4)
		}
		// threats and line of fire
		for i := range ctx.threats {
			th := &ctx.threats[i]
			d := hyp(x, y, th.t.Pos[0], th.t.Pos[1])
			if d > 290 {
				continue
			}
			exp := b.exposed(th.eye, x, y, cz) || b.exposed(th.eye, x, y, gz+me.Stats.Height-0.5)
			if exp {
				s -= b.P.WExpo * b.enemyDps(th.t, d) * th.weight
				if ctx.hide {
					s -= 80 * th.weight
				}
				if th.isTgt && ctx.wantLOS != 0 {
					s += ctx.wantLOS * b.P.WAtk * b.myDps(th.t, d)
				}
			}
			// never let lightning carriers close in for free
			if th.t.has[3] && d < 52 && th.weight > 0.3 {
				s -= (52 - d) * 6
			}
			if d < 7 {
				s -= (7 - d) * 10
			}
		}
		if ctx.target != nil && ctx.pref > 0 {
			d := hyp(x, y, ctx.target.Pos[0], ctx.target.Pos[1])
			s -= b.P.WRange * math.Abs(d-ctx.pref)
		}
		if ctx.strafe && c.r > 0 {
			s += 5.0 * (c.dx*sdx + c.dy*sdy) * c.r / 5
		}
		// dodge: follow our own accel-limited motion against each incoming round
		if len(incs) > 0 {
			key := 0
			if c.r > 0 {
				key = 1 + int(math.Round(math.Atan2(c.dy, c.dx)/(math.Pi/8)+16))%16
			}
			if c.r > 0 && math.Abs(math.Atan2(c.dy, c.dx)/(math.Pi/8)-math.Round(math.Atan2(c.dy, c.dx)/(math.Pi/8))) > 0.01 {
				key = -1
			}
			var pen float64
			if key >= 0 && dodgeDone[key] {
				pen = dodgePen[key]
			} else {
				pen = b.dodgeCost(c.dx*boolf(c.r > 0), c.dy*boolf(c.r > 0), speed, incs, splash)
				if key >= 0 {
					dodgeDone[key] = true
					dodgePen[key] = pen
				}
			}
			s -= pen
		}
		// inertia
		if c.r > 0 {
			s += 1.2 * (c.dx*b.lastMove[0] + c.dy*b.lastMove[1])
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
		// boxed in: head for the carrot / goal directly
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

// dodgeCost simulates moving in direction (ux,uy) for the next second and
// returns a penalty for every incoming round that would connect.
func (b *Bot) dodgeCost(ux, uy, speed float64, incs []inc, splash float64) float64 {
	me := b.me
	const dt = 0.05
	const steps = 24
	var traj [steps + 1][2]float64
	x, y := me.Pos[0], me.Pos[1]
	vx, vy := me.Vel[0], me.Vel[1]
	acc := me.Stats.GroundAccel
	if !me.Grounded {
		acc = me.Stats.AirAccel
	}
	traj[0] = [2]float64{x, y}
	for k := 1; k <= steps; k++ {
		dvx, dvy := ux*speed-vx, uy*speed-vy
		dl := math.Hypot(dvx, dvy)
		if dl > acc*dt {
			dvx *= acc * dt / dl
			dvy *= acc * dt / dl
		}
		vx += dvx
		vy += dvy
		x += vx * dt
		y += vy * dt
		traj[k] = [2]float64{x, y}
	}
	need := me.Stats.Radius + 0.9
	pen := 0.0
	for i := range incs {
		in := &incs[i]
		if in.mortar {
			k := int(in.it/dt + 0.5)
			if k > steps {
				k = steps
			}
			d := hyp(traj[k][0], traj[k][1], in.ix, in.iy)
			if d < splash+3 {
				pen += 170 * (1 - d/(splash+3))
			}
			continue
		}
		best := math.Inf(1)
		// the round sweeps a segment each step; test against our position then
		for k := 0; k < steps; k++ {
			t0 := float64(k)*dt + latency
			ax, ay := in.p[0]+in.v[0]*t0, in.p[1]+in.v[1]*t0
			bx, by := ax+in.v[0]*dt, ay+in.v[1]*dt
			// distance from our point to segment a-b
			cx, cy := traj[k][0], traj[k][1]
			sx, sy := bx-ax, by-ay
			l2 := sx*sx + sy*sy
			u := 0.0
			if l2 > 1e-9 {
				u = clamp(((cx-ax)*sx+(cy-ay)*sy)/l2, 0, 1)
			}
			d := math.Hypot(cx-(ax+sx*u), cy-(ay+sy*u))
			if d < best {
				bz := in.p[2] + in.v[2]*(t0+u*dt)
				if bz > me.Pos[2]-1.5 && bz < me.Pos[2]+me.Stats.Height+1.5 {
					best = d
				}
			}
			// stop once the round has passed us
			if (cx-bx)*sx+(cy-by)*sy < -need*math.Sqrt(l2)-20*math.Sqrt(l2) {
				break
			}
		}
		if best < need {
			pen += 50 * (1.2 - best/need)
		} else if best < need+1.5 {
			pen += 8 * (1 - (best-need)/1.5)
		}
	}
	return pen
}

func boolf(v bool) float64 {
	if v {
		return 1
	}
	return 0
}

var _ = geo.NavStep
