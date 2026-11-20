package bot

import (
	"math"

	"mechbot/proto"
)

const latency = 0.06 // s between observing and the shot leaving

// predict returns the target's centre after dt seconds.
func (b *Bot) predict(t *Track, dt float64) [3]float64 {
	c := t.Center()
	ls := b.P.LeadScale
	x := c[0] + t.Vel[0]*dt*ls
	y := c[1] + t.Vel[1]*dt*ls
	z := c[2]
	if math.Abs(t.Vel[2]) > 0.5 {
		// airborne: ballistic, but never below the ground
		z += t.Vel[2]*dt - 0.5*b.cfg.Mech.Gravity*dt*dt
		g := b.w.Ground(x, y) + (c[2] - t.Pos[2])
		if z < g {
			z = g
		}
	}
	return [3]float64{x, y, z}
}

func (b *Bot) aimAt(p [3]float64) (yaw, pitch, d float64) {
	dx, dy, dz := p[0]-b.muz[0], p[1]-b.muz[1], p[2]-b.muz[2]
	h := math.Hypot(dx, dy)
	return math.Atan2(dy, dx), math.Atan2(dz, h), math.Sqrt(h*h + dz*dz)
}

// gunSolve leads the target with a constant-speed projectile.
func (b *Bot) gunSolve(t *Track, speed float64) (yaw, pitch, tof float64, pt [3]float64) {
	tof = dist3(b.muz, t.Center()) / speed
	for i := 0; i < 3; i++ {
		pt = b.predict(t, tof+latency)
		_, _, d := b.aimAt(pt)
		tof = math.Max(0, d-3) / speed
	}
	yaw, pitch, _ = b.aimAt(pt)
	return
}

func ballistic(v, g, d, h float64) (theta, tof float64, ok bool) {
	if d < 0.5 {
		return 0, 0, false
	}
	v2 := v * v
	disc := v2*v2 - g*(g*d*d+2*h*v2)
	if disc < 0 {
		return 0, 0, false
	}
	theta = math.Atan((v2 - math.Sqrt(disc)) / (g * d))
	tof = d / (v * math.Cos(theta))
	return theta, tof, true
}

// mortarSolve aims a shell at the target's predicted feet.
func (b *Bot) mortarSolve(t *Track, st *proto.WeaponStats) (yaw, pitch, tof float64, ok bool) {
	tube := [3]float64{b.me.Pos[0], b.me.Pos[1], b.me.Pos[2] + b.me.Stats.Height + 0.5}
	tof = hyp(tube[0], tube[1], t.Pos[0], t.Pos[1]) / st.Speed
	var th float64
	var px, py, pz float64
	for i := 0; i < 3; i++ {
		lead := (tof + latency) * 0.85
		px, py = t.Pos[0]+t.Vel[0]*lead, t.Pos[1]+t.Vel[1]*lead
		pz = t.Pos[2] + 1.0
		d := hyp(tube[0], tube[1], px, py)
		th, tof, ok = ballistic(st.Speed, st.Gravity, d, pz-tube[2])
		if !ok {
			return
		}
	}
	yaw = math.Atan2(py-tube[1], px-tube[0])
	pitch = th
	if pitch < b.me.Stats.MinPitch || pitch > b.me.Stats.MaxPitch {
		return yaw, pitch, tof, false
	}
	// check the arc is clear (sampled)
	vx, vy, vz := math.Cos(th)*math.Cos(yaw)*st.Speed, math.Cos(th)*math.Sin(yaw)*st.Speed, math.Sin(th)*st.Speed
	n := 8
	for i := 1; i < n; i++ {
		tt := tof * float64(i) / float64(n)
		if b.w.SolidAt(tube[0]+vx*tt, tube[1]+vy*tt, tube[2]+vz*tt-0.5*st.Gravity*tt*tt) {
			return yaw, pitch, tof, false
		}
	}
	return yaw, pitch, tof, true
}

// aimNext estimates the torso aim one tick after commanding cmd.
func (b *Bot) aimNext(cmd [2]float64, ticks float64) [2]float64 {
	a := b.me.Aim
	dt := ticks / b.cfg.Match.TickHz
	dy := clamp(angDiff(cmd[0], a[0]), -b.me.Stats.TurnRate*dt, b.me.Stats.TurnRate*dt)
	dp := clamp(cmd[1]-a[1], -b.me.Stats.PitchRate*dt, b.me.Stats.PitchRate*dt)
	return [2]float64{a[0] + dy, a[1] + dp}
}

func aimErr(a [2]float64, yaw, pitch float64) (float64, float64) {
	return math.Abs(angDiff(a[0], yaw)), math.Abs(a[1] - pitch)
}

// shootMech commands aim and triggers against a mech (or NPC) target.
// Returns true if any weapon has a shot (now or imminently).
func (b *Bot) shootMech(act *proto.Action, t *Track) bool {
	me := b.me
	c := t.Center()
	d := dist3(b.muz, c)
	rad := t.Radius()
	hgt := 9.0 * t.Scale
	if t.NPC {
		hgt = 3.5
	}
	tolY := math.Atan2(rad*0.85, d)
	tolP := math.Atan2(hgt*0.42, d)
	var cmd [2]float64
	have := false
	shot := false

	gun, gi := b.weapon("gun")
	las, li := b.weapon("laser")
	mor, mi := b.weapon("mortar")
	ltn, ni := b.weapon("lightning")

	// gun solution (also the default aim)
	if gun != nil {
		y, p, _, pt := b.gunSolve(t, gun.Stats.Speed)
		if d < gun.Stats.Range+10 {
			cmd = [2]float64{y, p}
			have = true
			an := b.aimNext(cmd, 1)
			ey, ep := aimErr(an, y, p)
			clear := t.LOS && b.w.Clear(b.muz[0], b.muz[1], b.muz[2], pt[0], pt[1], pt[2])
			if clear && gun.Reload <= 0 && gun.Ammo > 0 && ey < tolY && ep < tolP && d < gun.Stats.Range {
				act.Fire[gi] = true
				shot = true
			}
			if clear {
				shot = true
			}
		}
	}
	if !have {
		y, p, _ := b.aimAt(b.predict(t, latency))
		cmd = [2]float64{y, p}
	}
	// mortar: when ready and a firing solution exists
	if mor != nil && !t.NPC && mor.Cooldown <= 0.11 {
		splash := mor.Stats.SplashRadius
		hd := hyp(me.Pos[0], me.Pos[1], t.Pos[0], t.Pos[1])
		y, p, tof, ok := b.mortarSolve(t, &mor.Stats)
		slow := math.Hypot(t.Vel[0], t.Vel[1]) < 4
		if ok && hd > splash+4 && (tof < 1.25 || (slow && tof < 3.5)) {
			cmd = [2]float64{y, p}
			an := b.aimNext(cmd, 1)
			ey, ep := aimErr(an, y, p)
			if ey < math.Atan2(splash*0.4, hd) && ep < 0.03 {
				act.Fire[mi] = true
			}
			shot = true
		}
	}
	// laser: charge when a shot is likely, release on target
	if las != nil {
		full := las.Stats.Charge
		charging := las.Charge > 0
		y, p, _ := b.aimAt(b.predict(t, latency))
		inRange := d < las.Stats.Range-5
		if charging {
			act.Fire[li] = true
			ready := las.Charge >= full-0.11
			dmgNow := las.Stats.Damage * b.dmgMul() * (las.Stats.MinFrac + (1-las.Stats.MinFrac)*math.Min(1, las.Charge/full))
			if (ready || (dmgNow >= t.HP && las.Charge >= las.Stats.MinCharge)) && t.LOS && inRange {
				cmd = [2]float64{y, p}
				an := b.aimNext(cmd, 1)
				ey, ep := aimErr(an, y, p)
				if ey < tolY*0.8 && ep < tolP*0.8 {
					act.Fire[li] = false // release: fires
				}
				shot = true
			}
		} else if las.Cooldown <= 0.11 && inRange && (t.LOS || b.t-t.Seen < 1.0) && !t.NPC {
			act.Fire[li] = true
			shot = shot || t.LOS
		} else if las.Cooldown <= 0.11 && inRange && t.LOS && t.NPC && t.HP > 150 {
			act.Fire[li] = true
		}
	}
	// lightning: instant, auto-targets the nearest thing in the cone
	if ltn != nil && ltn.Cooldown <= 0.11 && me.Energy >= ltn.Stats.Energy {
		hd := dist3(b.muz, c)
		cone := ltn.Stats.ConeDeg * math.Pi / 180
		an := b.aimNext(cmd, 1)
		yawTo := math.Atan2(c[1]-b.muz[1], c[0]-b.muz[0])
		if hd < ltn.Stats.Range-1.0 && t.LOS && math.Abs(angDiff(an[0], yawTo)) < cone*0.9 {
			if !t.NPC || me.Energy > 0.55*me.MaxEnergy {
				act.Fire[ni] = true
				shot = true
			}
		}
	}
	act.Aim = cmd
	return shot
}

// shootCrate fires the gun at a crate.
func (b *Bot) shootCrate(act *proto.Action, it *Item) bool {
	p := [3]float64{it.Pos[0], it.Pos[1], it.Pos[2] + 1.2}
	y, pi, d := b.aimAt(p)
	if !b.w.Clear(b.muz[0], b.muz[1], b.muz[2], p[0], p[1], p[2]) {
		return false
	}
	act.Aim = [2]float64{y, pi}
	an := b.aimNext(act.Aim, 1)
	ey, ep := aimErr(an, y, pi)
	if gun, gi := b.weapon("gun"); gun != nil && d < gun.Stats.Range && gun.Reload <= 0 && gun.Ammo > 0 && ey < math.Atan2(1.1, d) && ep < math.Atan2(0.9, d) {
		act.Fire[gi] = true
	}
	if ltn, ni := b.weapon("lightning"); ltn != nil && d < ltn.Stats.Range-1 && b.me.Energy > 0.7*b.me.MaxEnergy && ey < 0.3 {
		act.Fire[ni] = true
	}
	return true
}
