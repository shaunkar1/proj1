// Package house holds stand-in opponents for the local simulator: a squad
// that shares sightings and sends hunters, at several skill levels.
package house

import (
	"math"
	"math/rand"
	"sort"

	"mechbot/geo"
	"mechbot/proto"
)

type Sight struct {
	Pos, Vel [3]float64
	T        float64
	HP       float64
}

// Squad is the shared blackboard of the house bots in one match.
type Squad struct {
	Members  map[int]int     // id -> level (0 easy .. 3 expert)
	Alive    map[int]float64 // last report time
	Sight    map[int]*Sight
	LateT    float64
	Ramp     float64 // extra hunters per dead member
	Aggro    int     // 0: non-hunters never fight; 1: only when hit or crowded; 2: whenever they see the enemy
	T        float64
	hunters  map[int]bool
	huntCalc float64
}

func NewSquad() *Squad {
	return &Squad{Members: map[int]int{}, Alive: map[int]float64{}, Sight: map[int]*Sight{}, LateT: 290, Aggro: 1, Ramp: 1, hunters: map[int]bool{}, huntCalc: -1}
}

func (s *Squad) Report(o *proto.Obs) {
	s.T = o.T
	s.Alive[o.Self.ID] = o.T
	for i := range o.Mechs {
		m := &o.Mechs[i]
		if m.Kind != "mech" {
			continue
		}
		if _, ok := s.Members[m.ID]; ok {
			continue
		}
		s.Sight[m.ID] = &Sight{Pos: m.Pos, Vel: m.Vel, T: o.T, HP: m.HP}
	}
}

// Hunters: one at the start, more as members die, everyone late.
func (s *Squad) IsHunter(id int, aliveTotal int) bool {
	if s.T >= s.LateT {
		return true
	}
	if s.huntCalc != s.T {
		s.huntCalc = s.T
		dead := 0
		for mid := range s.Members {
			if s.T-s.Alive[mid] > 0.5 {
				dead++
			}
		}
		n := 1 + int(float64(dead)*s.Ramp)
		s.hunters = map[int]bool{}
		for lvl := 0; lvl <= 3 && n > 0; lvl++ {
			for mid := 0; mid < 64 && n > 0; mid++ {
				if l, ok := s.Members[mid]; ok && l == lvl && s.T-s.Alive[mid] <= 0.5 {
					s.hunters[mid] = true
					n--
				}
			}
		}
	}
	return s.hunters[id]
}

type Skill struct {
	AimNoise float64 // rad, correlated
	Lead     float64
	Strafe   float64
	Range    float64
	Retreat  float64
	Smart    bool
}

var Levels = []Skill{
	{AimNoise: 0.10, Lead: 0.0, Strafe: 0.0, Range: 45, Retreat: 0, Smart: false},
	{AimNoise: 0.05, Lead: 0.6, Strafe: 0.4, Range: 60, Retreat: 0, Smart: false},
	{AimNoise: 0.018, Lead: 1.0, Strafe: 0.8, Range: 75, Retreat: 0.25, Smart: true},
	{AimNoise: 0.008, Lead: 1.0, Strafe: 1.0, Range: 80, Retreat: 0.3, Smart: true},
}

type Bot struct {
	sq       *Squad
	sk       Skill
	lvl      int
	w        *geo.World
	cfg      proto.Config
	rng      *rand.Rand
	id       int
	path     [][2]float64
	pgoal    [2]float64
	pT       float64
	noise    [2]float64
	noiT     float64
	sSign    float64
	sT       float64
	roam     [2]float64
	roamT    float64
	cmd      [2]float64
	seenP    map[int]proto.PickupObs
	seenC    map[int]proto.ItemObs
	stuck    float64
	last     [3]float64
	unT      float64
	unD      [2]float64
	provoked float64
}

func New(h *proto.Hello, w *geo.World, sq *Squad, lvl int) *Bot {
	b := &Bot{sq: sq, sk: Levels[lvl], lvl: lvl, w: w, cfg: h.Config, id: h.Slot, sSign: 1}
	b.rng = rand.New(rand.NewSource(int64(h.Seed) ^ 0x5eed))
	b.seenP = map[int]proto.PickupObs{}
	b.seenC = map[int]proto.ItemObs{}
	if sq != nil {
		sq.Members[h.Slot] = lvl
		sq.Alive[h.Slot] = 0
	}
	return b
}

func ang(a, b float64) float64 {
	d := math.Mod(a-b, 2*math.Pi)
	if d > math.Pi {
		d -= 2 * math.Pi
	}
	if d < -math.Pi {
		d += 2 * math.Pi
	}
	return d
}

func hyp(ax, ay, bx, by float64) float64 { return math.Hypot(ax-bx, ay-by) }

func (b *Bot) nav(px, py, gx, gy, t float64) (float64, float64) {
	nv := b.w.Nav
	if b.path == nil || hyp(gx, gy, b.pgoal[0], b.pgoal[1]) > 6 || t-b.pT > 1.0 {
		b.path, _ = nv.Path(px, py, gx, gy, 30000, nil)
		b.pgoal = [2]float64{gx, gy}
		b.pT = t
	}
	if b.path == nil {
		return gx, gy
	}
	bi, bd := 0, math.Inf(1)
	for i, p := range b.path {
		if d := hyp(p[0], p[1], px, py); d < bd {
			bd, bi = d, i
		}
	}
	best := bi
	if bi+1 < len(b.path) {
		best = bi + 1
	}
	for i := bi + 1; i < len(b.path) && i < bi+10; i++ {
		if nv.WalkLine(px, py, b.path[i][0], b.path[i][1]) {
			best = i
		}
	}
	return b.path[best][0], b.path[best][1]
}

func (b *Bot) Act(o *proto.Obs) proto.Action {
	me := &o.Self
	act := proto.Action{Tick: o.Tick, Aim: b.cmd}
	px, py := me.Pos[0], me.Pos[1]
	if o.Offer != nil {
		pick := 0
		if b.lvl >= 2 {
			for i, c := range o.Offer.Choices {
				if c.ID == "plating" || c.ID == "armor" || c.ID == "gun_damage" || c.ID == "nanites" {
					pick = i
					break
				}
			}
		}
		act.Pick = &pick
	}
	if b.sq != nil {
		b.sq.Report(o)
	}
	for _, p := range o.Pickups {
		b.seenP[p.ID] = p
	}
	for _, c := range o.Crates {
		b.seenC[c.ID] = c
	}
	isFriend := func(id int) bool {
		if b.sq == nil {
			return false
		}
		_, ok := b.sq.Members[id]
		return ok
	}
	hunter := b.sq == nil || b.sq.IsHunter(me.ID, o.Alive)
	muz := [3]float64{px, py, me.Pos[2] + me.Stats.MuzzleHeight}
	// visible enemy
	var tgt *proto.ActorObs
	bd := 1e9
	for i := range o.Mechs {
		m := &o.Mechs[i]
		if m.Kind != "mech" || isFriend(m.ID) {
			continue
		}
		d := hyp(px, py, m.Pos[0], m.Pos[1])
		if d < bd {
			bd, tgt = d, m
		}
	}
	var npc *proto.ActorObs
	nd := 60.0
	for i := range o.Mechs {
		m := &o.Mechs[i]
		if m.Kind == "npc" {
			if d := hyp(px, py, m.Pos[0], m.Pos[1]); d < nd {
				nd, npc = d, m
			}
		}
	}

	z := o.Zone
	var goal [2]float64
	hasGoal := false
	boost := false
	dn := hyp(px, py, z.NextCenter[0], z.NextCenter[1])
	if dn > z.NextRadius*0.8 && (o.T > z.ShrinkStart-dn/9 || !me.InZone) {
		f := z.NextRadius * 0.6 / math.Max(dn, 1)
		goal = [2]float64{z.NextCenter[0] + (px-z.NextCenter[0])*f, z.NextCenter[1] + (py-z.NextCenter[1])*f}
		hasGoal = true
		boost = !me.InZone
	}
	for i := range o.Events {
		if e := &o.Events[i]; e.Type == "damaged" && e.By != nil && *e.By < 100 && !isFriend(*e.By) {
			b.provoked = o.T + 8
		}
	}
	fight := tgt != nil && hunter
	if tgt != nil && !hunter && b.sq != nil {
		switch b.sq.Aggro {
		case 1:
			fight = o.T < b.provoked || bd < 40
		case 2:
			fight = bd < 110 || o.T < b.provoked
		}
	}
	hpF := me.HP / me.MaxHP
	var mv [2]float64
	if b.t(o) < b.sT {
	} else {
		b.sSign = -b.sSign
		b.sT = o.T + 0.4 + b.rng.Float64()*0.8
	}
	if fight {
		los := b.w.Clear(muz[0], muz[1], muz[2], tgt.Pos[0], tgt.Pos[1], tgt.Pos[2]+4.5)
		pref := b.sk.Range
		if w := me.Weapons[1]; w != nil && w.Kind == "lightning" {
			pref = 28
		}
		dx, dy := tgt.Pos[0]-px, tgt.Pos[1]-py
		l := math.Max(math.Hypot(dx, dy), 0.1)
		ux, uy := dx/l, dy/l
		if !los && !hasGoal {
			cx, cy := b.nav(px, py, tgt.Pos[0], tgt.Pos[1], o.T)
			d := math.Max(hyp(cx, cy, px, py), 0.1)
			mv = [2]float64{(cx - px) / d, (cy - py) / d}
		} else if !hasGoal {
			rad := clampf((l-pref)/15, -1, 1)
			if b.sk.Retreat > 0 && hpF < b.sk.Retreat {
				rad = -1
			}
			st := b.sk.Strafe * b.sSign
			mv = [2]float64{ux*rad - uy*st, uy*rad + ux*st}

			if !b.w.Nav.Walkable(px+mv[0]*4, py+mv[1]*4) {
				mv = [2]float64{ux*rad + uy*st, uy*rad - ux*st}
				b.sSign = -b.sSign
			}
		}

		if o.T >= b.noiT {
			b.noiT = o.T + 0.35
			b.noise = [2]float64{b.rng.NormFloat64() * b.sk.AimNoise, b.rng.NormFloat64() * b.sk.AimNoise * 0.6}
		}
		gun := me.Weapons[0]
		speed := 252.0
		if gun != nil && gun.Kind == "gun" {
			speed = gun.Stats.Speed
		}
		d3 := math.Sqrt(l*l + (tgt.Pos[2]+4.5-muz[2])*(tgt.Pos[2]+4.5-muz[2]))
		tof := d3 / speed * b.sk.Lead
		ax, ay, az := tgt.Pos[0]+tgt.Vel[0]*tof, tgt.Pos[1]+tgt.Vel[1]*tof, tgt.Pos[2]+4.5
		yaw := math.Atan2(ay-py, ax-px)
		pitch := math.Atan2(az-muz[2], hyp(ax, ay, px, py))
		dyaw := math.Atan2(dy, dx)
		dpitch := math.Atan2(tgt.Pos[2]+4.5-muz[2], l)
		act.Aim = [2]float64{yaw + b.noise[0], pitch + b.noise[1]}
		onAim := math.Abs(ang(me.Aim[0], act.Aim[0])) < 0.08 && math.Abs(me.Aim[1]-act.Aim[1]) < 0.08
		for i, w := range me.Weapons {
			if w == nil {
				continue
			}
			switch w.Kind {
			case "gun":
				if los && onAim && l < w.Stats.Range {
					act.Fire[i] = true
				}
			case "lightning":
				if los && l < w.Stats.Range && math.Abs(ang(me.Aim[0], dyaw)) < 0.45 {
					act.Fire[i] = true
				}
			case "laser":
				if w.Charge > 0 {
					act.Fire[i] = true
					if w.Charge >= w.Stats.Charge-0.06 && los {
						act.Aim = [2]float64{dyaw + b.noise[0]*0.5, dpitch + b.noise[1]*0.5}
						if math.Abs(ang(me.Aim[0], act.Aim[0])) < 0.03 && math.Abs(me.Aim[1]-act.Aim[1]) < 0.04 {
							act.Fire[i] = false
						}
					}
				} else if los && w.Cooldown <= 0.1 && l < w.Stats.Range {
					act.Fire[i] = true
				}
			case "mortar":
				if los && w.Cooldown <= 0.1 && l > 14 && l < 80 && b.lvl >= 1 {
					v2 := w.Stats.Speed * w.Stats.Speed
					g := w.Stats.Gravity
					h := tgt.Pos[2] + 1 - (me.Pos[2] + me.Stats.Height + 0.5)
					tm := l / w.Stats.Speed * b.sk.Lead
					mx, my := tgt.Pos[0]+tgt.Vel[0]*tm, tgt.Pos[1]+tgt.Vel[1]*tm
					dd := hyp(mx, my, px, py)
					disc := v2*v2 - g*(g*dd*dd+2*h*v2)
					if disc > 0 {
						th := math.Atan((v2 - math.Sqrt(disc)) / (g * dd))
						act.Aim = [2]float64{math.Atan2(my-py, mx-px) + b.noise[0], th}
						if math.Abs(ang(me.Aim[0], act.Aim[0])) < 0.05 && math.Abs(me.Aim[1]-th) < 0.04 {
							act.Fire[i] = true
						}
					}
				}
			}
		}
	}
	if hasGoal {
		cx, cy := b.nav(px, py, goal[0], goal[1], o.T)
		d := math.Max(hyp(cx, cy, px, py), 0.1)
		g := [2]float64{(cx - px) / d, (cy - py) / d}
		if fight {
			mv = [2]float64{g[0] + mv[0]*0.3, g[1] + mv[1]*0.3}
		} else {
			mv = g
		}
	} else if !fight {
		// hunt the shared sighting, else loot, else roam
		var obj *[2]float64
		arrive := 2.0
		if hunter && b.sq != nil {
			var s *Sight
			for id := 0; id < 16; id++ {
				if si := b.sq.Sight[id]; si != nil && o.T-si.T < 12 && (s == nil || si.T > s.T) {
					s = si
				}
			}
			if s != nil {
				p := [2]float64{s.Pos[0], s.Pos[1]}
				obj = &p
				arrive = 15
				boost = me.Energy > 50
			}
		}
		if obj == nil {
			best := 150.0
			for _, c := range o.Cubes {
				if d := hyp(c.Pos[0], c.Pos[1], px, py); d < best && math.Abs(c.Pos[2]-me.Pos[2]) < 4 {
					best = d
					p := [2]float64{c.Pos[0], c.Pos[1]}
					obj = &p
					arrive = 0.5
				}
			}
			if obj == nil && me.Weapons[1] == nil {
				for _, id := range sortedKeysP(b.seenP) {
					p := b.seenP[id]
					if p.Pos[2] > b.w.Ground(p.Pos[0], p.Pos[1])+2.5 {
						continue
					}
					d := hyp(p.Pos[0], p.Pos[1], px, py)
					if d < 4 {
						tid := id
						act.Take = &tid
						delete(b.seenP, id)
					}
					if d < best {
						best = d
						q := [2]float64{p.Pos[0], p.Pos[1]}
						obj = &q
						arrive = 0.5
					}
				}
			}
			if obj == nil {
				for _, id := range sortedKeysC(b.seenC) {
					c := b.seenC[id]
					if c.Pos[2] > b.w.Ground(c.Pos[0], c.Pos[1])+2.5 {
						continue
					}
					d := hyp(c.Pos[0], c.Pos[1], px, py)
					if d < 45 {
						found := false
						for _, cc := range o.Crates {
							if cc.ID == id {
								found = true
							}
						}
						if !found {
							delete(b.seenC, id)
							continue
						}
					}
					if d < best {
						best = d
						q := [2]float64{c.Pos[0], c.Pos[1]}
						obj = &q
						arrive = 12

						if d < 60 {
							yaw := math.Atan2(c.Pos[1]-py, c.Pos[0]-px)
							pitch := math.Atan2(c.Pos[2]+1.2-muz[2], d)
							act.Aim = [2]float64{yaw, pitch}
							if math.Abs(ang(me.Aim[0], yaw)) < 0.03 && math.Abs(me.Aim[1]-pitch) < 0.03 && b.w.Clear(muz[0], muz[1], muz[2], c.Pos[0], c.Pos[1], c.Pos[2]+1.2) {
								act.Fire[0] = true
							}
						}
					}
				}
			}
		}
		if obj == nil && npc != nil && b.lvl >= 1 {
			p := [2]float64{npc.Pos[0], npc.Pos[1]}
			obj = &p
			arrive = 25
			yaw := math.Atan2(npc.Pos[1]-py, npc.Pos[0]-px)
			pitch := math.Atan2(npc.Pos[2]+1.7-muz[2], nd)
			act.Aim = [2]float64{yaw, pitch}
			if math.Abs(ang(me.Aim[0], yaw)) < 0.03 && math.Abs(me.Aim[1]-pitch) < 0.04 {
				act.Fire[0] = true
			}
		}
		if obj == nil {
			if o.T > b.roamT || hyp(b.roam[0], b.roam[1], px, py) < 8 {
				a := b.rng.Float64() * 2 * math.Pi
				r := z.NextRadius * 0.7 * math.Sqrt(b.rng.Float64())
				b.roam = [2]float64{z.NextCenter[0] + r*math.Cos(a), z.NextCenter[1] + r*math.Sin(a)}
				b.roamT = o.T + 25
			}
			obj = &b.roam
			arrive = 6
		}
		if d := hyp(obj[0], obj[1], px, py); d > arrive {
			cx, cy := b.nav(px, py, obj[0], obj[1], o.T)
			dd := math.Max(hyp(cx, cy, px, py), 0.1)
			mv = [2]float64{(cx - px) / dd, (cy - py) / dd}
		}
		if !act.Fire[0] && act.Aim == b.cmd && math.Hypot(mv[0], mv[1]) > 0.1 {
			act.Aim = [2]float64{math.Atan2(mv[1], mv[0]) + math.Sin(o.T*0.9+float64(b.id))*1.2, 0}
		}
	}

	if math.Hypot(mv[0], mv[1]) > 0.3 && hyp(px, py, b.last[0], b.last[1]) < 0.08 {
		b.stuck += 0.1
	} else {
		b.stuck = 0
	}
	if o.T < b.unT {
		mv = b.unD
		act.Jump = me.Grounded
		boost = true
	} else if b.stuck > 0.6 {
		a := b.rng.Float64() * 2 * math.Pi
		b.unD = [2]float64{math.Cos(a), math.Sin(a)}
		b.unT = o.T + 0.8
		b.stuck = 0
	}
	b.last = me.Pos
	act.Move = mv
	act.Boost = boost && (me.Grounded || o.T < b.unT)
	if gun := me.Weapons[0]; gun != nil && gun.Kind == "gun" && !fight && gun.Ammo < gun.Stats.Mag*0.5 && gun.Reload <= 0 {
		act.Reload = true
	}
	b.cmd = act.Aim
	return act
}

func (b *Bot) t(o *proto.Obs) float64 { return o.T }

func clampf(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func sortedKeysP(m map[int]proto.PickupObs) []int {
	k := make([]int, 0, len(m))
	for id := range m {
		k = append(k, id)
	}
	sort.Ints(k)
	return k
}

func sortedKeysC(m map[int]proto.ItemObs) []int {
	k := make([]int, 0, len(m))
	for id := range m {
		k = append(k, id)
	}
	sort.Ints(k)
	return k
}
