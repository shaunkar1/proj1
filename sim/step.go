package sim

import (
	"encoding/json"
	"math"

	"mechbot/proto"
)

func segCyl(a, b [3]float64, cx, cy, r, z0, z1 float64) (float64, bool) {
	dx, dy, dz := b[0]-a[0], b[1]-a[1], b[2]-a[2]
	fx, fy := a[0]-cx, a[1]-cy
	t0, t1 := 0.0, 1.0
	A := dx*dx + dy*dy
	C := fx*fx + fy*fy - r*r
	if A > 1e-12 {
		B := 2 * (fx*dx + fy*dy)
		disc := B*B - 4*A*C
		if disc < 0 {
			return 0, false
		}
		sq := math.Sqrt(disc)
		ta := (-B - sq) / (2 * A)
		tb := (-B + sq) / (2 * A)
		if ta > t0 {
			t0 = ta
		}
		if tb < t1 {
			t1 = tb
		}
	} else if C > 0 {
		return 0, false
	}
	if math.Abs(dz) > 1e-12 {
		ta := (z0 - a[2]) / dz
		tb := (z1 - a[2]) / dz
		if ta > tb {
			ta, tb = tb, ta
		}
		if ta > t0 {
			t0 = ta
		}
		if tb < t1 {
			t1 = tb
		}
	} else if a[2] < z0 || a[2] > z1 {
		return 0, false
	}
	if t0 > t1 {
		return 0, false
	}
	return t0, true
}

type target struct {
	kind int // 0 mech 1 npc 2 crate
	idx  int
	t    float64
}

// firstHits returns entity hits along a→b sorted by t (only the nearest unless all).
func (s *Sim) entityHit(a, b [3]float64, owner int, ownerNPC bool, skip map[int]bool) (target, bool) {
	best := target{t: 2}
	found := false
	for i, m := range s.Mechs {
		if !m.Alive || (!ownerNPC && m.ID == owner) || (skip != nil && skip[m.ID]) {
			continue
		}
		sc := m.Scale(&s.Cfg)
		if t, ok := segCyl(a, b, m.Pos[0], m.Pos[1], m.Stats.Radius*sc, m.Pos[2], m.Pos[2]+m.Stats.Height*sc); ok && t < best.t {
			best = target{0, i, t}
			found = true
		}
	}
	for i, n := range s.NPCs {
		if !n.Alive || (ownerNPC && n.ID == owner) || (skip != nil && skip[n.ID]) || ownerNPC {
			continue
		}
		if t, ok := segCyl(a, b, n.Pos[0], n.Pos[1], s.Cfg.NPC.Radius, n.Pos[2], n.Pos[2]+s.Cfg.NPC.Height); ok && t < best.t {
			best = target{1, i, t}
			found = true
		}
	}
	if !ownerNPC {
		for i, c := range s.Crates {
			if skip != nil && skip[c.ID] {
				continue
			}
			if t, ok := segCyl(a, b, c.Pos[0], c.Pos[1], s.Cfg.Crates.Radius, c.Pos[2], c.Pos[2]+s.Cfg.Crates.Height); ok && t < best.t {
				best = target{2, i, t}
				found = true
			}
		}
	}
	return best, found
}

func (s *Sim) sound(kind string, x, y float64, src int) {
	rng := s.Cfg.Sound.Range[kind]
	for _, m := range s.Mechs {
		if !m.Alive || m.ID == src {
			continue
		}
		d := hyp(x, y, m.Pos[0], m.Pos[1])
		if d > rng {
			continue
		}
		b := math.Atan2(y-m.Pos[1], x-m.Pos[0]) + s.Rng.NormFloat64()*s.Cfg.Sound.BearingNoiseDeg*m.Stats.Hearing*math.Pi/180
		ds := "far"
		if d < s.Cfg.Sound.Near {
			ds = "near"
		} else if d < s.Cfg.Sound.Mid {
			ds = "mid"
		}
		m.sounds = append(m.sounds, sndRec{kind, angDiff(b, 0), ds, s.T})
	}
}

func (s *Sim) beam(by int, kind string, path []proto.Vec3) {
	for _, m := range s.Mechs {
		if !m.Alive || m.ID == by {
			continue
		}
		see := false
		for i := 0; i < len(path) && !see; i++ {
			p := path[i]
			if s.canSee(m, p[0], p[1], p[2], p[2]) {
				see = true
			}
			if i+1 < len(path) && !see {
				q := path[i+1]
				if s.canSee(m, (p[0]+q[0])/2, (p[1]+q[1])/2, (p[2]+q[2])/2, (p[2]+q[2])/2) {
					see = true
				}
			}
		}
		if see {
			m.beams = append(m.beams, beamRec{by, kind, path, s.T})
		}
	}
}

func (s *Sim) canSee(m *Mech, x, y, zc, ztop float64) bool {
	sc := m.Scale(&s.Cfg)
	ex, ey, ez := m.Pos[0], m.Pos[1], m.Pos[2]+m.Stats.EyeHeight*sc
	d := hyp(x, y, ex, ey)
	near := s.Cfg.Vision.NearRange + m.Stats.NearRangeBonus
	if d <= near {
		return s.W.Clear(ex, ey, ez, x, y, zc) || s.W.Clear(ex, ey, ez, x, y, ztop)
	}
	if d > s.Cfg.Vision.Range+m.Stats.VisionBonus {
		return false
	}
	if math.Abs(angDiff(math.Atan2(y-ey, x-ex), m.Aim[0])) > s.Cfg.Vision.FovDeg*math.Pi/360 {
		return false
	}
	return s.W.Sees(ex, ey, ez, x, y, zc) || s.W.Sees(ex, ey, ez, x, y, ztop)
}

func intp(i int) *int         { return &i }
func fltp(f float64) *float64 { return &f }
func rawID(i int) json.RawMessage {
	b, _ := json.Marshal(i)
	return b
}
func rawStr(v string) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

// hurt applies weapon damage to a mech. from is the point the hit came from.
func (s *Sim) hurt(m *Mech, amount float64, weapon string, by int, byNPC bool, from [3]float64, effect string) {
	if !m.Alive {
		return
	}
	bearing := math.Atan2(from[1]-m.Pos[1], from[0]-m.Pos[0])
	rear := math.Abs(angDiff(bearing, m.Aim[0])) > m.Stats.FrontArcDeg*math.Pi/360
	if rear {
		amount *= m.Stats.RearMult
	}
	amount *= 1 - m.Stats.Armor
	m.HP -= amount
	m.DmgTaken += amount
	m.lastDmg = s.T
	{
		by2 := by
		if byNPC {
			by2 = 100
		}
		m.DmgLog = append(m.DmgLog, DmgRec{s.T, by2, weapon, amount, hyp(from[0], from[1], m.Pos[0], m.Pos[1])})
	}
	ev := proto.Event{Type: "damaged", Amount: amount, Weapon: weapon, Bearing: fltp(bearing), Rear: rear}
	if !byNPC {
		ev.By = intp(by)
	} else {
		ev.By = intp(by)
	}
	m.events = append(m.events, ev)
	if !byNPC && by >= 0 && by != m.ID {
		a := s.Mechs[by]
		a.lastDmg = s.T
		a.DmgDealt += amount
		a.events = append(a.events, proto.Event{Type: "hit", Target: m.ID, Amount: amount, Weapon: weapon, Effect: effect, Rear: rear})
		m.lastHit = by
	}
	if effect != "" {
		m.events = append(m.events, proto.Event{Type: "effect", Effect: effect, By: intp(by)})
	}
}

func (s *Sim) hurtNPC(n *NPC, amount float64, weapon string, by int) {
	if !n.Alive {
		return
	}
	n.HP -= amount
	if by >= 0 {
		a := s.Mechs[by]
		a.lastDmg = s.T
		a.events = append(a.events, proto.Event{Type: "hit", Target: n.ID, Amount: amount, Weapon: weapon})
		n.Target = by
		n.back = false
	}
	if n.HP <= 0 {
		n.Alive = false
		for k := 0; k < int(s.Cfg.NPC.Cubes); k++ {
			s.spawnCube(n.Pos)
		}
	}
}

func (s *Sim) hurtCrate(ci int, amount float64, weapon string, by int) {
	c := s.Crates[ci]
	c.HP -= amount
	if by >= 0 {
		a := s.Mechs[by]
		a.lastDmg = s.T
		a.events = append(a.events, proto.Event{Type: "hit", Target: c.ID, Amount: amount, Weapon: weapon})
	}
}

func (s *Sim) reapCrates() {
	out := s.Crates[:0]
	for _, c := range s.Crates {
		if c.HP > 0 {
			out = append(out, c)
			continue
		}
		s.sound("crate", c.Pos[0], c.Pos[1], -1)
		for k := 0; k < int(s.Cfg.Crates.Cubes); k++ {
			s.spawnCube(c.Pos)
		}
		if s.Rng.Float64() < s.Cfg.Loot.CrateWeaponChance {
			s.Picks = append(s.Picks, &Pickup{ID: s.nextPick, Kind: s.pickKind(), Pos: c.Pos})
			s.nextPick++
		}
	}
	s.Crates = out
}

func (s *Sim) spawnCube(p [3]float64) {
	a := s.Rng.Float64() * 2 * math.Pi
	d := s.Rng.Float64() * 1.5
	x, y := p[0]+d*math.Cos(a), p[1]+d*math.Sin(a)
	z, _ := s.W.Surface(x, y, p[2]+1.5)
	s.Cubes = append(s.Cubes, &Cube{ID: s.nextCube, Pos: [3]float64{x, y, z}})
	s.nextCube++
}

func (s *Sim) muzzle(m *Mech, h float64) (origin [3]float64, dir [3]float64, ok bool) {
	sc := m.Scale(&s.Cfg)
	dir = dirVec(m.Aim[0], m.Aim[1])
	mz := [3]float64{m.Pos[0], m.Pos[1], m.Pos[2] + h}
	l := m.Stats.Radius*sc + 0.5
	origin = [3]float64{mz[0] + dir[0]*l, mz[1] + dir[1]*l, mz[2] + dir[2]*l}
	if _, hit := s.W.RayHit(mz[0], mz[1], mz[2], origin[0], origin[1], origin[2]); hit {
		return origin, dir, false
	}
	return origin, dir, true
}

func (s *Sim) explode(p [3]float64, pr *Proj, direct target, hasDirect bool) {
	s.sound("explosion", p[0], p[1], -1)
	st := pr.Stats
	mul := pr.Damage / st.Damage
	if hasDirect {
		switch direct.kind {
		case 0:
			s.hurt(s.Mechs[direct.idx], pr.Damage, "mortar", pr.Owner, false, p, "")
		case 1:
			s.hurtNPC(s.NPCs[direct.idx], pr.Damage, "mortar", pr.Owner)
		case 2:
			s.hurtCrate(direct.idx, pr.Damage, "mortar", pr.Owner)
		}
	}
	fall := func(d float64) float64 {
		if d > st.SplashRadius {
			return 0
		}
		return 1 - (1-st.SplashEdge)*d/st.SplashRadius
	}
	for i, m := range s.Mechs {
		if !m.Alive {
			continue
		}
		sc := m.Scale(&s.Cfg)
		cz := math.Max(m.Pos[2], math.Min(p[2], m.Pos[2]+m.Stats.Height*sc))
		d := math.Max(0, math.Sqrt((p[0]-m.Pos[0])*(p[0]-m.Pos[0])+(p[1]-m.Pos[1])*(p[1]-m.Pos[1])+(p[2]-cz)*(p[2]-cz))-m.Stats.Radius*sc)
		if hasDirect && direct.kind == 0 && direct.idx == i {
			d = 0
		}
		f := fall(d)
		if f <= 0 {
			continue
		}
		if !(hasDirect && direct.kind == 0 && direct.idx == i) && !s.W.Clear(p[0], p[1], p[2]+0.3, m.Pos[0], m.Pos[1], m.Pos[2]+m.Stats.Height*sc*0.5) {
			continue
		}
		dmg := st.Splash * mul * f
		if m.ID == pr.Owner {
			dmg *= m.Stats.SelfDamageMult
		}
		s.hurt(m, dmg, "mortar", pr.Owner, false, p, "concuss")
		hx, hy := m.Pos[0]-p[0], m.Pos[1]-p[1]
		hl := math.Hypot(hx, hy)
		if hl < 0.1 {
			hx, hy, hl = 1, 0, 1
		}
		m.Vel[0] += hx / hl * st.Knock * f
		m.Vel[1] += hy / hl * st.Knock * f
		m.Vel[2] += st.KnockUp * f
		m.Grounded = false
		m.Energy = math.Max(0, m.Energy-st.Drain*f)
	}
	for _, n := range s.NPCs {
		if !n.Alive {
			continue
		}
		d := math.Max(0, math.Hypot(p[0]-n.Pos[0], p[1]-n.Pos[1])-s.Cfg.NPC.Radius)
		if f := fall(d); f > 0 {
			s.hurtNPC(n, st.Splash*mul*f, "mortar", pr.Owner)
		}
	}
	for i, c := range s.Crates {
		d := math.Max(0, math.Hypot(p[0]-c.Pos[0], p[1]-c.Pos[1])-s.Cfg.Crates.Radius)
		if f := fall(d); f > 0 && math.Abs(p[2]-c.Pos[2]) < st.SplashRadius {
			s.hurtCrate(i, st.Splash*mul*f, "mortar", pr.Owner)
		}
	}
}

func (s *Sim) fireWeapons(m *Mech) {
	sc := m.Scale(&s.Cfg)
	for slot := 0; slot < 2; slot++ {
		w := m.W[slot]
		if w == nil {
			continue
		}
		if w.Cooldown > 0 {
			w.Cooldown -= s.DT
		}
		trig := m.act.Fire[slot] && m.coast <= 0
		switch w.Kind {
		case "gun":
			if w.Reload > 0 {
				w.Reload -= s.DT
				if w.Reload <= 0 {
					w.Reload = 0
					w.Ammo = int(w.Stats.Mag)
				}
				continue
			}
			if m.act.Reload && m.fresh && w.Ammo < int(w.Stats.Mag) {
				w.Reload = w.Stats.Reload
				s.sound("reload", m.Pos[0], m.Pos[1], m.ID)
				continue
			}
			if trig && w.Cooldown <= 1e-9 && w.Ammo > 0 {
				if w.Cooldown < -s.DT {
					w.Cooldown = 0
				}
				w.Cooldown += w.Stats.Cooldown
				w.Ammo--
				if w.Ammo == 0 {
					w.Reload = w.Stats.Reload
					s.sound("reload", m.Pos[0], m.Pos[1], m.ID)
				}
				s.sound("gun", m.Pos[0], m.Pos[1], m.ID)
				m.Shots++
				o, _, ok := s.muzzle(m, m.Stats.MuzzleHeight*sc)
				if !ok {
					continue
				}
				sp := w.Stats.SpreadDeg * math.Pi / 180
				d := dirVec(m.Aim[0]+s.Rng.NormFloat64()*sp, m.Aim[1]+s.Rng.NormFloat64()*sp)
				s.Projs = append(s.Projs, &Proj{ID: s.nextProj, Kind: "gun", Owner: m.ID, Pos: o,
					Vel: [3]float64{d[0] * w.Stats.Speed, d[1] * w.Stats.Speed, d[2] * w.Stats.Speed}, Left: w.Stats.Range, Damage: w.Stats.Damage * s.dmgMul(m), Stats: w.Stats})
				s.nextProj++
			}
		case "mortar":
			if trig && w.Cooldown <= 1e-9 {
				w.Cooldown = w.Stats.Cooldown
				s.sound("mortar", m.Pos[0], m.Pos[1], m.ID)
				o, d, ok := s.muzzle(m, m.Stats.Height*sc+0.5)
				if !ok {
					continue
				}
				s.Projs = append(s.Projs, &Proj{ID: s.nextProj, Kind: "mortar", Owner: m.ID, Pos: o,
					Vel: [3]float64{d[0] * w.Stats.Speed, d[1] * w.Stats.Speed, d[2] * w.Stats.Speed}, Left: w.Stats.TTL, Damage: w.Stats.Damage * s.dmgMul(m), Stats: w.Stats})
				s.nextProj++
			}
		case "laser":
			if s.T < m.shortTil {
				w.Charging, w.Charge, w.Held, w.Release = false, 0, 0, false
				continue
			}
			fire := false
			if w.Charging {
				if trig && !w.Release {
					if w.Charge < w.Stats.Charge {
						w.Charge = math.Min(w.Stats.Charge, w.Charge+s.DT)
					} else {
						w.Held += s.DT
						if w.Held >= w.Stats.HoldS {
							fire = true
						}
					}
				} else {
					w.Release = true
					if w.Charge >= w.Stats.MinCharge {
						fire = true
					} else {
						w.Charge += s.DT
					}
				}
			} else if trig && w.Cooldown <= 1e-9 {
				w.Charging, w.Charge, w.Held, w.Release = true, 0, 0, false
				s.sound("laser_charge", m.Pos[0], m.Pos[1], m.ID)
			}
			if fire {
				frac := w.Stats.MinFrac + (1-w.Stats.MinFrac)*math.Min(1, w.Charge/w.Stats.Charge)
				dmg := w.Stats.Damage * frac * s.dmgMul(m)
				w.Charging, w.Charge, w.Held, w.Release = false, 0, 0, false
				w.Cooldown = w.Stats.Cooldown
				s.sound("laser", m.Pos[0], m.Pos[1], m.ID)
				o, d, ok := s.muzzle(m, m.Stats.MuzzleHeight*sc)
				if !ok {
					continue
				}
				end := [3]float64{o[0] + d[0]*w.Stats.Range, o[1] + d[1]*w.Stats.Range, o[2] + d[2]*w.Stats.Range}
				tw, _ := s.W.RayHit(o[0], o[1], o[2], end[0], end[1], end[2])
				skip := map[int]bool{}
				lastT := tw
				for n := 0; n <= int(w.Stats.Pierce); n++ {
					tg, ok := s.entityHit(o, end, m.ID, false, skip)
					if !ok || tg.t > tw {
						lastT = tw
						break
					}
					lastT = tg.t
					switch tg.kind {
					case 0:
						t := s.Mechs[tg.idx]
						skip[t.ID] = true
						dd := dmg
						eff := ""
						if (t.W[0] != nil && t.W[0].Kind == "mortar") || (t.W[1] != nil && t.W[1].Kind == "mortar") {
							dd *= 1 + w.Stats.Cookoff
							eff = "cookoff"
						}
						s.hurt(t, dd, "laser", m.ID, false, o, eff)
					case 1:
						skip[s.NPCs[tg.idx].ID] = true
						s.hurtNPC(s.NPCs[tg.idx], dmg, "laser", m.ID)
					case 2:
						skip[s.Crates[tg.idx].ID] = true
						s.hurtCrate(tg.idx, dmg, "laser", m.ID)
					}
				}
				hit := proto.Vec3{o[0] + (end[0]-o[0])*lastT, o[1] + (end[1]-o[1])*lastT, o[2] + (end[2]-o[2])*lastT}
				s.beam(m.ID, "laser", []proto.Vec3{o, hit})
			}
		case "lightning":
			if trig && w.Cooldown <= 1e-9 && m.Energy >= w.Stats.Energy {
				w.Cooldown = w.Stats.Cooldown
				m.Energy -= w.Stats.Energy
				m.lastSpnd = s.T
				s.sound("lightning", m.Pos[0], m.Pos[1], m.ID)
				o := [3]float64{m.Pos[0], m.Pos[1], m.Pos[2] + m.Stats.MuzzleHeight*sc}
				path := []proto.Vec3{o}
				hitIDs := map[int]bool{m.ID: true}
				cur := o
				dmg := w.Stats.Damage * s.dmgMul(m)
				rng := w.Stats.Range
				cone := w.Stats.ConeDeg * math.Pi / 180
				for jump := 0; jump <= int(w.Stats.Chain); jump++ {
					bk, bi, bd := -1, -1, rng
					var bp [3]float64
					consider := func(kind, idx, id int, c [3]float64) {
						if hitIDs[id] {
							return
						}
						d := math.Sqrt((c[0]-cur[0])*(c[0]-cur[0]) + (c[1]-cur[1])*(c[1]-cur[1]) + (c[2]-cur[2])*(c[2]-cur[2]))
						if d >= bd {
							return
						}
						if jump == 0 && math.Abs(angDiff(math.Atan2(c[1]-cur[1], c[0]-cur[0]), m.Aim[0])) > cone {
							return
						}
						if !s.W.Clear(cur[0], cur[1], cur[2], c[0], c[1], c[2]) {
							return
						}
						bk, bi, bd, bp = kind, idx, d, c
					}
					for i, t := range s.Mechs {
						if t.Alive {
							consider(0, i, t.ID, [3]float64{t.Pos[0], t.Pos[1], t.Pos[2] + t.Stats.Height*t.Scale(&s.Cfg)/2})
						}
					}
					for i, n := range s.NPCs {
						if n.Alive {
							consider(1, i, n.ID, [3]float64{n.Pos[0], n.Pos[1], n.Pos[2] + s.Cfg.NPC.Height/2})
						}
					}
					for i, c := range s.Crates {
						if c.HP > 0 {
							consider(2, i, c.ID, [3]float64{c.Pos[0], c.Pos[1], c.Pos[2] + s.Cfg.Crates.Height/2})
						}
					}
					if bk < 0 {
						break
					}
					switch bk {
					case 0:
						t := s.Mechs[bi]
						hitIDs[t.ID] = true
						eff := ""
						for _, tw := range t.W {
							if tw != nil && tw.Kind == "laser" {
								eff = "short"
								tw.Charging, tw.Charge, tw.Held, tw.Release = false, 0, 0, false
								t.shortTil = s.T + w.Stats.ShortS
							}
						}
						s.hurt(t, dmg, "lightning", m.ID, false, o, eff)
					case 1:
						hitIDs[s.NPCs[bi].ID] = true
						s.hurtNPC(s.NPCs[bi], dmg, "lightning", m.ID)
					case 2:
						hitIDs[s.Crates[bi].ID] = true
						s.hurtCrate(bi, dmg, "lightning", m.ID)
					}
					path = append(path, bp)
					cur = bp
					dmg *= w.Stats.ChainFalloff
					rng = w.Stats.ChainRange
				}
				if len(path) == 1 {
					d := dirVec(m.Aim[0], m.Aim[1])
					e := [3]float64{o[0] + d[0]*w.Stats.Range, o[1] + d[1]*w.Stats.Range, o[2] + d[2]*w.Stats.Range}
					t, _ := s.W.RayHit(o[0], o[1], o[2], e[0], e[1], e[2])
					path = append(path, proto.Vec3{o[0] + (e[0]-o[0])*t, o[1] + (e[1]-o[1])*t, o[2] + (e[2]-o[2])*t})
				}
				s.beam(m.ID, "lightning", path)
			}
		}
	}
}

func (s *Sim) bodyBlocked(x, y, z, r, h, step float64) bool {
	if s.W.BodyBlocked(x, y, z+step, z+h) {
		return true
	}
	for k := 0; k < 8; k++ {
		a := float64(k) * math.Pi / 4
		if s.W.BodyBlocked(x+r*math.Cos(a), y+r*math.Sin(a), z+step, z+h) {
			return true
		}
	}
	return false
}

func (s *Sim) moveMech(m *Mech) {
	dt := s.DT
	st := &m.Stats
	sc := m.Scale(&s.Cfg)
	// aim
	dy := angDiff(m.CmdAim[0], m.Aim[0])
	maxY := st.TurnRate * dt
	if dy > maxY {
		dy = maxY
	} else if dy < -maxY {
		dy = -maxY
	}
	m.Aim[0] = angDiff(m.Aim[0]+dy, 0)
	dp := m.CmdAim[1] - m.Aim[1]
	maxP := st.PitchRate * dt
	if dp > maxP {
		dp = maxP
	} else if dp < -maxP {
		dp = -maxP
	}
	m.Aim[1] += dp
	mv := m.act.Move
	if m.coast > s.Cfg.Bot.CoastS {
		mv = proto.Vec2{}
	}
	ml := math.Hypot(mv[0], mv[1])
	if ml > 1 {
		mv[0] /= ml
		mv[1] /= ml
		ml = 1
	}
	speed := st.WalkSpeed
	boost := m.act.Boost && m.coast <= s.Cfg.Bot.CoastS
	if boost && m.Grounded && ml > 0.01 && m.Energy > 0 {
		speed = st.BoostSpeed
		m.Energy -= st.BoostDrain * dt
		m.lastSpnd = s.T
	}
	for _, w := range m.W {
		if w != nil && w.Charging {
			speed *= st.ChargeMoveMult
		}
	}
	acc := st.AirAccel
	if m.Grounded {
		acc = st.GroundAccel
	}
	dvx, dvy := mv[0]*speed-m.Vel[0], mv[1]*speed-m.Vel[1]
	dl := math.Hypot(dvx, dvy)
	if dl > acc*dt {
		dvx *= acc * dt / dl
		dvy *= acc * dt / dl
	}
	m.Vel[0] += dvx
	m.Vel[1] += dvy
	if m.Grounded && m.act.Jump && m.fresh && m.coast <= 0 {
		m.Vel[2] = st.JumpSpeed
		m.Grounded = false
	}
	if !m.Grounded {
		if boost && m.Energy > 0 {
			m.Energy -= st.JetDrain * dt
			m.lastSpnd = s.T
			if m.Vel[2] < st.MaxRiseSpeed {
				m.Vel[2] = math.Min(st.MaxRiseSpeed, m.Vel[2]+(st.JetAccel-st.Gravity)*dt)
			} else {
				m.Vel[2] -= st.Gravity * dt
			}
			if s.Tick%6 == 0 {
				s.sound("jet", m.Pos[0], m.Pos[1], m.ID)
			}
		} else {
			m.Vel[2] -= st.Gravity * dt
		}
	}
	if m.Energy < 0 {
		m.Energy = 0
	}
	r, h := st.Radius*sc, st.Height*sc
	x, y, z := m.Pos[0], m.Pos[1], m.Pos[2]
	nx, ny := x+m.Vel[0]*dt, y+m.Vel[1]*dt
	try := func(tx, ty float64) bool {
		if s.bodyBlocked(tx, ty, z, r, h, st.StepUp) {
			return false
		}
		g := s.W.Ground(tx, ty)
		if g > z+st.StepUp {
			return false
		}
		if m.Grounded {
			d := math.Hypot(tx-x, ty-y)
			if g-z > 0.02 && g-s.W.Ground(x, y) > math.Tan(st.MaxSlopeDeg*math.Pi/180)*d+0.05 {
				return false
			}
		}
		return true
	}
	if try(nx, ny) {
		x, y = nx, ny
	} else if try(nx, y) {
		x = nx
		m.Vel[1] = 0
	} else if try(x, ny) {
		y = ny
		m.Vel[0] = 0
	} else {
		m.Vel[0], m.Vel[1] = 0, 0
	}
	surf, _ := s.W.Surface(x, y, z+st.StepUp)
	if m.Grounded {
		if surf >= z-0.6 {
			z = surf
		} else {
			m.Grounded = false
			m.Vel[2] = 0
		}
	}
	if !m.Grounded {
		nz := z + m.Vel[2]*dt
		if m.Vel[2] > 0 && s.W.BodyBlocked(x, y, nz+h-0.5, nz+h) {
			m.Vel[2] = 0
			nz = z
		}
		if nz <= surf && m.Vel[2] <= 0 {
			nz = surf
			m.Vel[2] = 0
			m.Grounded = true
		}
		z = nz
	}
	m.Pos = [3]float64{x, y, z}
	// footsteps
	if m.Grounded {
		sp := math.Hypot(m.Vel[0], m.Vel[1])
		if sp > s.Cfg.Sound.StepSpeed {
			m.stepT += dt
			if m.stepT >= s.Cfg.Sound.StepEvery {
				m.stepT = 0
				if speed > st.WalkSpeed+0.1 {
					s.sound("boost", x, y, m.ID)
				} else {
					s.sound("step", x, y, m.ID)
				}
			}
		}
	}
}

func (s *Sim) handleItems(m *Mech) {
	a := &m.act
	// offers
	if len(m.offers) > 0 {
		o := m.offers[0]
		if !m.offerSet {
			o.Expires = s.T + s.Cfg.Bot.OfferTimeoutS
			m.offerSet = true
		}
		pick := -1
		if a.Pick != nil && m.fresh && *a.Pick >= 0 && *a.Pick < len(o.Choices) {
			pick = *a.Pick
		} else if s.T >= o.Expires {
			pick = 0
		}
		if pick >= 0 {
			id := o.Choices[pick].ID
			m.upCount[id]++
			m.Upgrades = append(m.Upgrades, id)
			s.recompute(m)
			m.events = append(m.events, proto.Event{Type: "upgrade", ID: rawStr(id)})
			m.offers = m.offers[1:]
			m.offerSet = false
		}
	}
	if !m.fresh || m.coast > 0 {
		return
	}
	dropSlot := func(slot int) {
		w := m.W[slot]
		if w == nil {
			return
		}
		p := &Pickup{ID: s.nextPick, Kind: w.Kind, Pos: m.Pos, Ammo: w.Ammo}
		s.nextPick++
		s.Picks = append(s.Picks, p)
		m.events = append(m.events, proto.Event{Type: "dropped", ID: rawID(p.ID), Kind: w.Kind, Slot: slot})
		m.W[slot] = nil
	}
	if a.Take != nil {
		id := *a.Take
		fail := func(r string) {
			m.events = append(m.events, proto.Event{Type: "take_failed", ID: rawID(id), Reason: r})
		}
		pi := -1
		for i, p := range s.Picks {
			if p.ID == id {
				pi = i
			}
		}
		switch {
		case pi < 0:
			fail("gone")
		case hyp(s.Picks[pi].Pos[0], s.Picks[pi].Pos[1], m.Pos[0], m.Pos[1]) > s.Cfg.Loot.PickupRadius || math.Abs(s.Picks[pi].Pos[2]-m.Pos[2]) > 4:
			fail("range")
		default:
			p := s.Picks[pi]
			slot := -1
			if a.Drop != nil && (*a.Drop == 0 || *a.Drop == 1) {
				slot = *a.Drop
			} else if m.W[0] == nil {
				slot = 0
			} else if m.W[1] == nil {
				slot = 1
			}
			other := 1 - slot
			if slot < 0 {
				fail("full")
			} else if m.W[other] != nil && m.W[other].Kind == p.Kind {
				fail("have")
			} else if m.W[slot] != nil && m.W[slot].Kind == p.Kind {
				fail("have")
			} else if m.W[slot] != nil && m.W[slot].Charging {
				fail("busy")
			} else {
				kind, ammo := p.Kind, p.Ammo
				s.Picks = append(s.Picks[:pi], s.Picks[pi+1:]...)
				dropSlot(slot)
				w := &Weapon{Kind: kind, Stats: s.weaponStats(m, kind)}
				w.Cooldown = math.Max(s.Cfg.Loot.EquipS, w.Stats.Cooldown)
				if kind == "gun" {
					w.Ammo = ammo
					if ammo == 0 {
						w.Reload = w.Stats.Reload
					}
				}
				m.W[slot] = w
				if kind != "gun" && m.FirstW < 0 {
					m.FirstW = s.T
				}
				if kind == "laser" && m.FirstLas < 0 {
					m.FirstLas = s.T
				}
				m.events = append(m.events, proto.Event{Type: "took", ID: rawID(id), Kind: kind, Slot: slot})
			}
		}
	} else if a.Drop != nil && (*a.Drop == 0 || *a.Drop == 1) {
		if w := m.W[*a.Drop]; w != nil && !w.Charging {
			dropSlot(*a.Drop)
		}
	}
}

func (s *Sim) stepNPC(n *NPC) {
	dt := s.DT
	c := &s.Cfg.NPC
	dHome := hyp(n.Pos[0], n.Pos[1], n.Home[0], n.Home[1])
	if n.Target >= 0 {
		t := s.Mechs[n.Target]
		d := hyp(n.Pos[0], n.Pos[1], t.Pos[0], t.Pos[1])
		if !t.Alive || d > c.DeaggroRange || dHome > c.ChaseLeash {
			n.Target = -1
			n.back = true
		}
	}
	if n.Target < 0 && dHome <= c.Leash {
		n.back = false
		best := c.AggroRange
		for _, m := range s.Mechs {
			if !m.Alive {
				continue
			}
			d := hyp(n.Pos[0], n.Pos[1], m.Pos[0], m.Pos[1])
			if d < best && s.W.Clear(n.Pos[0], n.Pos[1], n.Pos[2]+2.5, m.Pos[0], m.Pos[1], m.Pos[2]+4.5) {
				best = d
				n.Target = m.ID
			}
		}
	}
	var gx, gy float64
	move := true
	if n.Target >= 0 {
		t := s.Mechs[n.Target]
		gx, gy = t.Pos[0], t.Pos[1]
		d := hyp(n.Pos[0], n.Pos[1], gx, gy)
		if d < 25 {
			move = false
		}
		n.Aim[0] = math.Atan2(gy-n.Pos[1], gx-n.Pos[0])
		n.Cooldown -= dt
		if n.Cooldown <= 0 && s.W.Clear(n.Pos[0], n.Pos[1], n.Pos[2]+2.5, t.Pos[0], t.Pos[1], t.Pos[2]+4.5) {
			n.Cooldown = c.Cooldown
			tof := d / c.ProjectileSpeed
			tx, ty, tz := t.Pos[0]+t.Vel[0]*tof, t.Pos[1]+t.Vel[1]*tof, t.Pos[2]+4.5
			o := [3]float64{n.Pos[0], n.Pos[1], n.Pos[2] + 2.5}
			yaw := math.Atan2(ty-o[1], tx-o[0]) + s.Rng.NormFloat64()*c.SpreadDeg*math.Pi/180
			pitch := math.Atan2(tz-o[2], math.Hypot(tx-o[0], ty-o[1])) + s.Rng.NormFloat64()*c.SpreadDeg*math.Pi/180
			dv := dirVec(yaw, pitch)
			s.Projs = append(s.Projs, &Proj{ID: s.nextProj, Kind: "npc", Owner: n.ID, OwnerNPC: true, Pos: o,
				Vel: [3]float64{dv[0] * c.ProjectileSpeed, dv[1] * c.ProjectileSpeed, dv[2] * c.ProjectileSpeed}, Left: 200, Damage: c.Damage})
			s.nextProj++
			s.sound("npc_gun", n.Pos[0], n.Pos[1], -1)
		}
	} else if n.back {
		gx, gy = n.Home[0], n.Home[1]
		if dHome < 3 {
			n.back = false
		}
	} else {
		n.wanderT -= dt
		if n.wanderT <= 0 {
			n.wanderT = 3 + s.Rng.Float64()*4
			a := s.Rng.Float64() * 2 * math.Pi
			d := s.Rng.Float64() * c.Leash * 0.8
			n.wander = [2]float64{n.Home[0] + d*math.Cos(a), n.Home[1] + d*math.Sin(a)}
		}
		gx, gy = n.wander[0], n.wander[1]
		if hyp(n.Pos[0], n.Pos[1], gx, gy) < 1 {
			move = false
		}
	}
	n.Vel = [3]float64{}
	if move {
		dx, dy := gx-n.Pos[0], gy-n.Pos[1]
		l := math.Hypot(dx, dy)
		if l > 0.01 {
			vx, vy := dx/l*c.Speed, dy/l*c.Speed
			nx, ny := n.Pos[0]+vx*dt, n.Pos[1]+vy*dt
			if n.Pos[2] > s.W.Ground(n.Pos[0], n.Pos[1])+2 || s.W.Nav.Walkable(nx, ny) {
				if !s.W.BodyBlocked(nx, ny, n.Pos[2]+1.2, n.Pos[2]+3.5) {
					n.Pos[0], n.Pos[1] = nx, ny
					n.Vel = [3]float64{vx, vy, 0}
					if z, ok := s.W.Surface(nx, ny, n.Pos[2]+1.2); ok {
						n.Pos[2] = z
					}
				}
			}
			if n.Target < 0 {
				n.Aim[0] = math.Atan2(dy, dx)
			}
		}
	}
}

func (s *Sim) stepProjs() {
	dt := s.DT
	out := s.Projs[:0]
	for _, p := range s.Projs {
		if p.Kind == "mortar" {
			p.Vel[2] -= p.Stats.Gravity * dt
		}
		a := p.Pos
		b := [3]float64{a[0] + p.Vel[0]*dt, a[1] + p.Vel[1]*dt, a[2] + p.Vel[2]*dt}
		tw, wh := s.W.RayHit(a[0], a[1], a[2], b[0], b[1], b[2])
		tg, eh := s.entityHit(a, b, p.Owner, p.OwnerNPC, nil)
		if eh && tg.t <= tw {
			hp := [3]float64{a[0] + (b[0]-a[0])*tg.t, a[1] + (b[1]-a[1])*tg.t, a[2] + (b[2]-a[2])*tg.t}
			switch p.Kind {
			case "mortar":
				s.explode(hp, p, tg, true)
			default:
				from := [3]float64{hp[0] - p.Vel[0], hp[1] - p.Vel[1], hp[2] - p.Vel[2]}
				switch tg.kind {
				case 0:
					if p.OwnerNPC {
						s.hurt(s.Mechs[tg.idx], p.Damage, "npc", p.Owner, true, from, "")
					} else {
						s.hurt(s.Mechs[tg.idx], p.Damage, "gun", p.Owner, false, from, "")
						s.Mechs[p.Owner].Hits++
					}
				case 1:
					s.hurtNPC(s.NPCs[tg.idx], p.Damage, "gun", p.Owner)
				case 2:
					s.hurtCrate(tg.idx, p.Damage, "gun", p.Owner)
				}
			}
			continue
		}
		if wh {
			if p.Kind == "mortar" {
				hp := [3]float64{a[0] + (b[0]-a[0])*tw, a[1] + (b[1]-a[1])*tw, a[2] + (b[2]-a[2])*tw}
				// back off slightly so the blast is outside geometry
				hp[2] += 0.2
				s.explode(hp, p, target{}, false)
			}
			continue
		}
		p.Pos = b
		if p.Kind == "mortar" {
			p.Left -= dt
		} else {
			p.Left -= math.Sqrt(p.Vel[0]*p.Vel[0]+p.Vel[1]*p.Vel[1]+p.Vel[2]*p.Vel[2]) * dt
		}
		if p.Left <= 0 {
			if p.Kind == "mortar" {
				s.explode(b, p, target{}, false)
			}
			continue
		}
		out = append(out, p)
	}
	s.Projs = out
}

func (s *Sim) giveCube(m *Mech, n int) {
	for k := 0; k < n; k++ {
		m.Cubes++
		for _, th := range s.Cfg.Cubes.UpgradeThresholds {
			if float64(m.Cubes) == th {
				s.makeOffer(m)
			}
		}
	}
	s.recompute(m)
}

func (s *Sim) makeOffer(m *Mech) {
	var avail []proto.UpgradeDef
	for _, u := range s.Cfg.Upgrades {
		if float64(m.upCount[u.ID]) >= u.Max {
			continue
		}
		if u.Requires != "" {
			has := false
			for _, w := range m.W {
				if w != nil && w.Kind == u.Requires {
					has = true
				}
			}
			if !has {
				continue
			}
		}
		avail = append(avail, u)
	}
	s.Rng.Shuffle(len(avail), func(i, j int) { avail[i], avail[j] = avail[j], avail[i] })
	if len(avail) > 3 {
		avail = avail[:3]
	}
	o := &proto.Offer{ID: len(m.Upgrades) + len(m.offers) + 1}
	for _, u := range avail {
		o.Choices = append(o.Choices, proto.OfferChoice{ID: u.ID, Name: u.Name, Desc: u.Desc})
	}
	if len(o.Choices) > 0 {
		m.offers = append(m.offers, o)
	}
}

func (s *Sim) Step() {
	dt := s.DT
	cfg := &s.Cfg
	decide := s.Tick%int(cfg.Match.DecideEvery) == 0
	if decide {
		for _, m := range s.Mechs {
			if !m.Alive {
				continue
			}
			obs := s.BuildObs(m)
			if m.Ctl != nil {
				a := m.Ctl.Act(obs)
				if a.Tick == obs.Tick {
					prevAim := m.CmdAim
					m.act = a
					m.CmdAim = a.Aim
					if math.IsNaN(a.Aim[0]) || math.IsNaN(a.Aim[1]) {
						m.CmdAim = prevAim
					}
					if m.CmdAim[1] < m.Stats.MinPitch {
						m.CmdAim[1] = m.Stats.MinPitch
					}
					if m.CmdAim[1] > m.Stats.MaxPitch {
						m.CmdAim[1] = m.Stats.MaxPitch
					}
					m.coast = 0
				} else {
					m.coast += dt * cfg.Match.DecideEvery
				}
			}
			m.fresh = true
		}
	}
	// supply drops
	if s.dropIdx < len(cfg.Drops.Times) && s.T >= cfg.Drops.Times[s.dropIdx] {
		s.dropIdx++
		for tries := 0; tries < 500; tries++ {
			if p, ok := s.randGround(s.znc, math.Max(5, s.znr*cfg.Drops.Spread)); ok {
				s.Drops = append(s.Drops, &Drop{ID: s.nextDrop, Pos: [3]float64{p[0], p[1], s.W.Ground(p[0], p[1])}, LandT: s.T + cfg.Drops.FallS, Holder: -1})
				s.nextDrop++
				break
			}
		}
	}
	for _, m := range s.Mechs {
		if !m.Alive {
			continue
		}
		s.handleItems(m)
		s.moveMech(m)
	}
	// push overlapping mechs apart
	for i, a := range s.Mechs {
		if !a.Alive {
			continue
		}
		for _, b := range s.Mechs[i+1:] {
			if !b.Alive || math.Abs(a.Pos[2]-b.Pos[2]) > 9 {
				continue
			}
			rr := a.Stats.Radius*a.Scale(cfg) + b.Stats.Radius*b.Scale(cfg)
			dx, dy := b.Pos[0]-a.Pos[0], b.Pos[1]-a.Pos[1]
			d := math.Hypot(dx, dy)
			if d < rr && d > 1e-6 {
				push := (rr - d) / 2
				ux, uy := dx/d, dy/d
				if !s.bodyBlocked(a.Pos[0]-ux*push, a.Pos[1]-uy*push, a.Pos[2], a.Stats.Radius, a.Stats.Height, a.Stats.StepUp) {
					a.Pos[0] -= ux * push
					a.Pos[1] -= uy * push
				}
				if !s.bodyBlocked(b.Pos[0]+ux*push, b.Pos[1]+uy*push, b.Pos[2], b.Stats.Radius, b.Stats.Height, b.Stats.StepUp) {
					b.Pos[0] += ux * push
					b.Pos[1] += uy * push
				}
			}
		}
	}
	for _, m := range s.Mechs {
		if m.Alive {
			s.fireWeapons(m)
			m.fresh = false
		}
	}
	for _, n := range s.NPCs {
		if n.Alive {
			s.stepNPC(n)
		}
	}
	s.stepProjs()
	s.reapCrates()
	s.stepZone()
	// cubes
	if len(s.Cubes) > 0 {
		out := s.Cubes[:0]
		for _, c := range s.Cubes {
			var best *Mech
			bd := cfg.Cubes.PickupRadius
			for _, m := range s.Mechs {
				if !m.Alive || math.Abs(m.Pos[2]-c.Pos[2]) > 4 {
					continue
				}
				d := hyp(m.Pos[0], m.Pos[1], c.Pos[0], c.Pos[1])
				if d < bd {
					bd = d
					best = m
				}
			}
			if best == nil {
				out = append(out, c)
				continue
			}
			s.giveCube(best, 1)
			best.events = append(best.events, proto.Event{Type: "pickup", Cubes: float64(best.Cubes)})
		}
		s.Cubes = out
	}
	// drops
	if len(s.Drops) > 0 {
		out := s.Drops[:0]
		for _, d := range s.Drops {
			if s.T < d.LandT {
				out = append(out, d)
				continue
			}
			if !d.Landed {
				d.Landed = true
				s.sound("drop", d.Pos[0], d.Pos[1], -1)
			}
			in := -1
			cnt := 0
			for _, m := range s.Mechs {
				if m.Alive && hyp(m.Pos[0], m.Pos[1], d.Pos[0], d.Pos[1]) <= cfg.Drops.Radius && math.Abs(m.Pos[2]-d.Pos[2]) < 6 {
					in = m.ID
					cnt++
				}
			}
			if cnt == 0 {
				d.Holder, d.Progress = -1, 0
			} else if cnt == 1 {
				if d.Holder != in {
					d.Holder, d.Progress = in, 0
				}
				d.Progress += dt
				if d.Progress >= cfg.Drops.CaptureS {
					m := s.Mechs[in]
					s.giveCube(m, int(cfg.Drops.Cubes))
					m.HP = m.MaxHP
					m.Energy = m.Stats.Energy
					m.events = append(m.events, proto.Event{Type: "drop", ID: rawID(d.ID), Cubes: float64(m.Cubes)})
					continue
				}
			}
			out = append(out, d)
		}
		s.Drops = out
	}
	// zone, regen, energy
	dps := s.zoneDPS()
	for _, m := range s.Mechs {
		if !m.Alive {
			continue
		}
		in := s.inZone(m.Pos[0], m.Pos[1])
		if !in {
			m.HP -= dps * dt
			m.ZoneDmg += dps * dt
			m.lastDmg = s.T
			if s.Tick%10 == 0 {
				m.events = append(m.events, proto.Event{Type: "damaged", Amount: dps * dt * 10, Weapon: "zone"})
			}
		} else if s.T-m.lastDmg >= m.Stats.RegenDelay && m.HP < m.MaxHP {
			m.HP = math.Min(m.MaxHP, m.HP+m.Stats.RegenPerS*dt)
		}
		if s.T-m.lastSpnd >= m.Stats.EnergyRegenDelay && m.Energy < m.Stats.Energy {
			m.Energy = math.Min(m.Stats.Energy, m.Energy+m.Stats.EnergyRegen*dt)
		}
	}
	if s.Tick == 1200 || s.Tick == 2400 {
		for _, m := range s.Mechs {
			if s.Tick == 1200 {
				m.Cubes60 = m.Cubes
			} else {
				m.Cubes120 = m.Cubes
			}
		}
	}
	// deaths
	dead := 0
	for _, m := range s.Mechs {
		if m.Alive && m.HP <= 0 {
			dead++
		}
	}
	if dead > 0 {
		place := s.Alive - dead + 1
		for _, m := range s.Mechs {
			if !m.Alive || m.HP > 0 {
				continue
			}
			m.Alive = false
			m.Place = place
			m.DeathT = s.T
			s.sound("death", m.Pos[0], m.Pos[1], m.ID)
			if m.lastHit >= 0 && s.Mechs[m.lastHit].Alive && s.Mechs[m.lastHit].HP > 0 {
				k := s.Mechs[m.lastHit]
				k.Kills++
				k.events = append(k.events, proto.Event{Type: "kill", Target: m.ID})
			}
			for k := 0; k < m.Cubes+int(cfg.Cubes.DeathDropExtra); k++ {
				s.spawnCube(m.Pos)
			}
			for _, w := range m.W {
				if w != nil && w.Kind != "gun" {
					s.Picks = append(s.Picks, &Pickup{ID: s.nextPick, Kind: w.Kind, Pos: m.Pos})
					s.nextPick++
				}
			}
			if s.Log != nil {
				s.Log("t=%.1f death %d(%s) place %d by %d cubes %d", s.T, m.ID, m.Name, place, m.lastHit, m.Cubes)
			}
		}
		s.Alive -= dead
	}
	if s.Rec != nil && s.Tick%4 == 0 {
		s.Rec.frame(s)
	}
	s.Tick++
	s.T = float64(s.Tick) * dt
	if s.Alive <= 1 || s.T >= cfg.Match.DurationS {
		s.Done = true
		// rank survivors by HP
		var surv []*Mech
		for _, m := range s.Mechs {
			if m.Alive {
				surv = append(surv, m)
			}
		}
		for i := range surv {
			rank := 1
			for j := range surv {
				if surv[j].HP > surv[i].HP {
					rank++
				}
			}
			surv[i].Place = rank
			surv[i].DeathT = s.T
		}
	}
}

func (s *Sim) Run() {
	for !s.Done {
		s.Step()
	}
}

func (s *Sim) BuildObs(m *Mech) *proto.Obs {
	cfg := &s.Cfg
	o := &proto.Obs{Type: "obs", Tick: s.Tick, T: s.T, Alive: s.Alive}
	o.Self = proto.SelfObs{ID: m.ID, Pos: m.Pos, Vel: m.Vel, Aim: m.Aim, HP: m.HP, MaxHP: m.MaxHP, Energy: m.Energy, MaxEnergy: m.Stats.Energy,
		Grounded: m.Grounded, Cubes: float64(m.Cubes), Upgrades: append([]string{}, m.Upgrades...), InZone: s.inZone(m.Pos[0], m.Pos[1]), Stats: m.Stats}
	sc := m.Scale(cfg)
	o.Self.Stats.Radius *= sc
	o.Self.Stats.Height *= sc
	o.Self.Stats.EyeHeight *= sc
	o.Self.Stats.MuzzleHeight *= sc
	o.Self.Weapons = make([]*proto.WeaponObs, 2)
	for i, w := range m.W {
		if w != nil {
			o.Self.Weapons[i] = &proto.WeaponObs{Kind: w.Kind, Cooldown: math.Max(0, w.Cooldown), Ammo: float64(w.Ammo), Reload: math.Max(0, w.Reload), Charge: w.Charge, Stats: w.Stats}
			if w.Charging && w.Charge == 0 {
				o.Self.Weapons[i].Charge = 0.001
			}
		}
	}
	mem := cfg.Vision.MemoryS
	for _, t := range s.Mechs {
		if !t.Alive || t.ID == m.ID {
			continue
		}
		ts := t.Scale(cfg)
		if s.canSee(m, t.Pos[0], t.Pos[1], t.Pos[2]+t.Stats.Height*ts/2, t.Pos[2]+t.Stats.Height*ts) {
			m.memory[t.ID] = s.T
		}
		if last, ok := m.memory[t.ID]; ok && s.T-last <= mem {
			a := proto.ActorObs{ID: t.ID, Kind: "mech", Pos: t.Pos, Vel: t.Vel, Aim: t.Aim, HP: t.HP, MaxHP: t.MaxHP, Cubes: float64(t.Cubes)}
			for _, w := range t.W {
				if w != nil {
					a.Weapons = append(a.Weapons, w.Kind)
					if w.Charging {
						a.Charging = true
					}
				}
			}
			o.Mechs = append(o.Mechs, a)
		}
	}
	for _, n := range s.NPCs {
		if !n.Alive {
			continue
		}
		if s.canSee(m, n.Pos[0], n.Pos[1], n.Pos[2]+cfg.NPC.Height/2, n.Pos[2]+cfg.NPC.Height) {
			m.memory[n.ID] = s.T
		}
		if last, ok := m.memory[n.ID]; ok && s.T-last <= mem {
			o.Mechs = append(o.Mechs, proto.ActorObs{ID: n.ID, Kind: "npc", Pos: n.Pos, Vel: n.Vel, Aim: n.Aim, HP: n.HP, MaxHP: cfg.NPC.HP})
		}
	}
	for _, p := range s.Projs {
		if !p.OwnerNPC && p.Owner == m.ID {
			continue
		}
		if s.canSee(m, p.Pos[0], p.Pos[1], p.Pos[2], p.Pos[2]) {
			o.Projectiles = append(o.Projectiles, proto.ProjObs{ID: p.ID, Kind: p.Kind, Pos: p.Pos, Vel: p.Vel})
		}
	}
	for _, c := range s.Crates {
		if s.canSee(m, c.Pos[0], c.Pos[1], c.Pos[2]+cfg.Crates.Height/2, c.Pos[2]+cfg.Crates.Height) {
			o.Crates = append(o.Crates, proto.ItemObs{ID: c.ID, Pos: c.Pos, HP: c.HP})
		}
	}
	for _, c := range s.Cubes {
		if s.canSee(m, c.Pos[0], c.Pos[1], c.Pos[2]+0.5, c.Pos[2]+1) {
			o.Cubes = append(o.Cubes, proto.ItemObs{ID: c.ID, Pos: c.Pos})
		}
	}
	for _, p := range s.Picks {
		if s.canSee(m, p.Pos[0], p.Pos[1], p.Pos[2]+0.5, p.Pos[2]+1) {
			o.Pickups = append(o.Pickups, proto.PickupObs{ID: p.ID, Kind: p.Kind, Pos: p.Pos})
		}
	}
	for _, sr := range m.sounds {
		o.Sounds = append(o.Sounds, proto.SoundObs{Kind: sr.kind, Bearing: sr.bearing, Dist: sr.dist, Age: s.T - sr.t})
	}
	m.sounds = m.sounds[:0]
	for _, b := range m.beams {
		o.Beams = append(o.Beams, proto.BeamObs{By: b.by, Kind: b.kind, Path: b.path, Age: s.T - b.t})
	}
	m.beams = m.beams[:0]
	o.Events = m.events
	m.events = nil
	for _, d := range s.Drops {
		do := proto.DropObs{ID: d.ID, Pos: d.Pos, Radius: cfg.Drops.Radius, LandsIn: math.Max(0, d.LandT-s.T), CaptureS: cfg.Drops.CaptureS, Progress: d.Progress}
		if d.Holder >= 0 {
			do.Holder = intp(d.Holder)
		}
		o.Drops = append(o.Drops, do)
	}
	o.Zone = proto.ZoneObs{Center: s.zc, Radius: s.zr, NextCenter: s.znc, NextRadius: s.znr, ShrinkStart: s.zstart, ShrinkEnd: s.zend, DPS: s.zoneDPS()}
	if s.zphase >= len(cfg.Zone.Phases) {
		o.Zone.ShrinkStart, o.Zone.ShrinkEnd = s.T, s.T
	}
	if len(m.offers) > 0 && m.offerSet {
		o.Offer = m.offers[0]
	}
	return o
}
