package sim

import (
	"encoding/json"
	"math"
	"math/rand"
	"sort"

	"mechbot/geo"
	"mechbot/proto"
)

type Controller interface {
	Act(obs *proto.Obs) proto.Action
}

type Weapon struct {
	Kind     string
	Cooldown float64
	Ammo     int
	Reload   float64
	Charge   float64
	Charging bool
	Held     float64
	Release  bool
	Stats    proto.WeaponStats
}

type Mech struct {
	ID       int
	Alive    bool
	Pos, Vel [3]float64
	Aim      [2]float64
	CmdAim   [2]float64
	HP       float64
	MaxHP    float64
	Energy   float64
	Grounded bool
	Cubes    int
	Upgrades []string
	upCount  map[string]int
	Stats    proto.MechStats
	W        [2]*Weapon
	lastDmg  float64
	lastSpnd float64
	shortTil float64
	act      proto.Action
	fresh    bool
	coast    float64
	offers   []*proto.Offer
	offerSet bool
	events   []proto.Event
	sounds   []sndRec
	beams    []beamRec
	memory   map[int]float64
	Place    int
	Kills    int
	DeathT   float64
	lastHit  int
	stepT    float64
	Ctl      Controller
	Name     string
	DmgDealt float64
	DmgTaken float64
	ZoneDmg  float64
	DmgLog   []DmgRec
	Shots    int
	LasShots int
	LasHits  int
	Hits     int
	FirstW   float64
	FirstLas float64
	Cubes60  int
	Cubes120 int
}

type DmgRec struct {
	T      float64
	By     int
	Weapon string
	Amount float64
	Dist   float64
}

func (m *Mech) Scale(cfg *proto.Config) float64 {
	return 1 + cfg.Cubes.SizePerCube*float64(m.Cubes)
}

type sndRec struct {
	kind    string
	bearing float64
	dist    string
	t       float64
}
type beamRec struct {
	by   int
	kind string
	path []proto.Vec3
	t    float64
}

type NPC struct {
	ID       int
	Alive    bool
	Pos, Vel [3]float64
	Home     [2]float64
	Aim      [2]float64
	HP       float64
	Target   int
	Cooldown float64
	wander   [2]float64
	wanderT  float64
	back     bool
}

type Crate struct {
	ID  int
	Pos [3]float64
	HP  float64
}
type Cube struct {
	ID  int
	Pos [3]float64
}
type Pickup struct {
	ID   int
	Kind string
	Pos  [3]float64
	Ammo int
}
type Drop struct {
	ID       int
	Pos      [3]float64
	LandT    float64
	Holder   int
	Progress float64
	Landed   bool
}
type Proj struct {
	ID       int
	Kind     string
	Owner    int
	Pos, Vel [3]float64
	Left     float64
	Damage   float64
	Stats    proto.WeaponStats
	OwnerNPC bool
}

type Sim struct {
	W                                      *geo.World
	Cfg                                    proto.Config
	Rng                                    *rand.Rand
	Tick                                   int
	T                                      float64
	DT                                     float64
	Mechs                                  []*Mech
	NPCs                                   []*NPC
	Crates                                 []*Crate
	Cubes                                  []*Cube
	Picks                                  []*Pickup
	Drops                                  []*Drop
	Projs                                  []*Proj
	nextCube, nextPick, nextProj, nextDrop int

	zc       [2]float64
	zr       float64
	znc      [2]float64
	znr      float64
	zstart   float64
	zend     float64
	zphase   int
	zfrom    [2]float64
	zfromR   float64
	dropIdx  int
	Alive    int
	Done     bool
	Log      func(format string, a ...interface{})
	pickKind func() string
	Rec      *Recorder
}

func cloneCfg(h *proto.Hello) proto.Config {
	b, _ := json.Marshal(h.Config)
	var c proto.Config
	json.Unmarshal(b, &c)
	return c
}

func angDiff(a, b float64) float64 {
	d := math.Mod(a-b, 2*math.Pi)
	if d > math.Pi {
		d -= 2 * math.Pi
	}
	if d < -math.Pi {
		d += 2 * math.Pi
	}
	return d
}

func dirVec(yaw, pitch float64) [3]float64 {
	cp := math.Cos(pitch)
	return [3]float64{cp * math.Cos(yaw), cp * math.Sin(yaw), math.Sin(pitch)}
}

func hyp(ax, ay, bx, by float64) float64 { return math.Hypot(ax-bx, ay-by) }

func (s *Sim) dmgMul(m *Mech) float64 {
	c := float64(m.Cubes)
	if c > s.Cfg.Cubes.StatCap {
		c = s.Cfg.Cubes.StatCap
	}
	return 1 + s.Cfg.Cubes.DamagePerCube*c
}

func (s *Sim) recompute(m *Mech) {
	st := s.Cfg.Mech
	for _, u := range s.Cfg.Upgrades {
		n := m.upCount[u.ID]
		for k := 0; k < n; k++ {
			for _, e := range u.Effects {
				applyMech(&st, e)
			}
		}
	}
	m.Stats = st
	for _, w := range m.W {
		if w != nil {
			w.Stats = s.weaponStats(m, w.Kind)
		}
	}
	c := float64(m.Cubes)
	if c > s.Cfg.Cubes.StatCap {
		c = s.Cfg.Cubes.StatCap
	}
	nm := st.HP * (1 + s.Cfg.Cubes.HPPerCube*c)
	if nm > m.MaxHP {
		m.HP += nm - m.MaxHP
	}
	m.MaxHP = nm
	if m.HP > m.MaxHP {
		m.HP = m.MaxHP
	}
}

func (s *Sim) weaponStats(m *Mech, kind string) proto.WeaponStats {
	st := s.Cfg.Weapons[kind]
	for _, u := range s.Cfg.Upgrades {
		if u.Requires != kind {
			continue
		}
		n := m.upCount[u.ID]
		for k := 0; k < n; k++ {
			for _, e := range u.Effects {
				applyWeapon(&st, e)
			}
		}
	}
	return st
}

func fx(v *float64, e proto.Effect) {
	if e.Mul != nil {
		*v *= *e.Mul
	}
	if e.Add != nil {
		*v += *e.Add
	}
}

func applyMech(st *proto.MechStats, e proto.Effect) {
	switch e.Path {
	case "mech.hp":
		fx(&st.HP, e)
	case "mech.armor":
		fx(&st.Armor, e)
	case "mech.walk_speed":
		fx(&st.WalkSpeed, e)
	case "mech.boost_speed":
		fx(&st.BoostSpeed, e)
	case "mech.energy":
		fx(&st.Energy, e)
	case "mech.energy_regen":
		fx(&st.EnergyRegen, e)
	case "mech.boost_drain":
		fx(&st.BoostDrain, e)
	case "mech.jet_drain":
		fx(&st.JetDrain, e)
	case "mech.regen_per_s":
		fx(&st.RegenPerS, e)
	case "mech.regen_delay":
		fx(&st.RegenDelay, e)
	case "mech.turn_rate":
		fx(&st.TurnRate, e)
	case "mech.pitch_rate":
		fx(&st.PitchRate, e)
	case "mech.near_range_bonus":
		fx(&st.NearRangeBonus, e)
	case "mech.vision_bonus":
		fx(&st.VisionBonus, e)
	case "mech.hearing":
		fx(&st.Hearing, e)
	}
}

func applyWeapon(st *proto.WeaponStats, e proto.Effect) {
	i := len(e.Path) - 1
	for i >= 0 && e.Path[i] != '.' {
		i--
	}
	switch e.Path[i+1:] {
	case "damage":
		fx(&st.Damage, e)
	case "cooldown":
		fx(&st.Cooldown, e)
	case "speed":
		fx(&st.Speed, e)
	case "range":
		fx(&st.Range, e)
	case "mag":
		fx(&st.Mag, e)
	case "reload":
		fx(&st.Reload, e)
	case "splash_radius":
		fx(&st.SplashRadius, e)
	case "charge":
		fx(&st.Charge, e)
	case "pierce":
		fx(&st.Pierce, e)
	case "chain":
		fx(&st.Chain, e)
	case "cone_deg":
		fx(&st.ConeDeg, e)
	case "energy":
		fx(&st.Energy, e)
	}
}

func New(w *geo.World, base *proto.Hello, seed int64, mk func(slot int, hello *proto.Hello) (Controller, string)) *Sim {
	s := &Sim{W: w, Cfg: cloneCfg(base), Rng: rand.New(rand.NewSource(seed))}
	s.DT = 1 / s.Cfg.Match.TickHz
	s.nextCube, s.nextPick, s.nextProj, s.nextDrop = 5000, 8000, 1, 20000
	s.zc = [2]float64{s.Cfg.Zone.StartCenter[0], s.Cfg.Zone.StartCenter[1]}
	s.zr = s.Cfg.Zone.StartRadius
	s.zphase = -1
	s.advanceZone()
	n := int(s.Cfg.Match.Mechs)

	var pts [][2]float64
	sep := s.Cfg.Match.SpawnMinSeparation
	for len(pts) < n {
		tries := 0
		for {
			tries++
			if tries > 2000 {
				sep *= 0.9
				tries = 0
			}
			p, ok := s.randGround(s.zc, s.zr*s.Cfg.Match.SpawnZoneFraction)
			if !ok {
				continue
			}
			good := true
			for _, q := range pts {
				if hyp(p[0], p[1], q[0], q[1]) < sep {
					good = false
					break
				}
			}
			if good {
				pts = append(pts, p)
				break
			}
		}
	}
	for i := 0; i < n; i++ {
		m := &Mech{ID: i, Alive: true, upCount: map[string]int{}, memory: map[int]float64{}, lastDmg: -100, lastSpnd: -100, lastHit: -1, FirstW: -1, FirstLas: -1}
		m.Pos = [3]float64{pts[i][0], pts[i][1], w.Ground(pts[i][0], pts[i][1])}
		m.Grounded = true
		m.Aim[0] = math.Atan2(s.zc[1]-m.Pos[1], s.zc[0]-m.Pos[0])
		m.CmdAim = m.Aim
		m.W[0] = &Weapon{Kind: "gun"}
		s.recompute(m)
		m.W[0].Ammo = int(m.W[0].Stats.Mag)
		m.HP = m.MaxHP
		m.Energy = m.Stats.Energy
		s.Mechs = append(s.Mechs, m)
	}
	s.Alive = n

	kinds := []string{}
	for k, wt := range s.Cfg.Loot.Weights {
		if wt > 0 {
			kinds = append(kinds, k)
		}
	}
	sort.Strings(kinds)
	pickKind := func() string {
		tot := 0.0
		for _, k := range kinds {
			tot += s.Cfg.Loot.Weights[k]
		}
		r := s.Rng.Float64() * tot
		for _, k := range kinds {
			r -= s.Cfg.Loot.Weights[k]
			if r <= 0 {
				return k
			}
		}
		return kinds[len(kinds)-1]
	}
	s.pickKind = pickKind
	for i := 0; i < int(s.Cfg.Loot.FloorCount); i++ {
		p := s.randSpot(s.Rng.Float64() < s.Cfg.Loot.FloorRoofFraction)
		s.Picks = append(s.Picks, &Pickup{ID: s.nextPick, Kind: pickKind(), Pos: p})
		s.nextPick++
	}
	for i := 0; i < int(s.Cfg.Crates.Count); i++ {
		p := s.randSpot(s.Rng.Float64() < s.Cfg.Crates.RoofFraction)
		s.Crates = append(s.Crates, &Crate{ID: 1000 + i, Pos: p, HP: s.Cfg.Crates.HP})
	}
	for i := 0; i < int(s.Cfg.NPC.Count); i++ {
		p := s.randSpot(false)
		s.NPCs = append(s.NPCs, &NPC{ID: 100 + i, Alive: true, Pos: p, Home: [2]float64{p[0], p[1]}, HP: s.Cfg.NPC.HP, Target: -1})
	}
	for i := 0; i < n; i++ {
		h := *base
		h.Slot = i
		h.Seed = float64(s.Rng.Int63n(1 << 31))
		h.Config = cloneCfg(base)
		c, name := mk(i, &h)
		s.Mechs[i].Ctl = c
		s.Mechs[i].Name = name
	}
	return s
}

func (s *Sim) randGround(c [2]float64, r float64) ([2]float64, bool) {
	a := s.Rng.Float64() * 2 * math.Pi
	d := r * math.Sqrt(s.Rng.Float64())
	x, y := c[0]+d*math.Cos(a), c[1]+d*math.Sin(a)
	if !s.W.Nav.WalkableMain(x, y) {
		return [2]float64{}, false
	}
	return [2]float64{x, y}, true
}

func (s *Sim) randSpot(roof bool) [3]float64 {
	for {
		a := s.Rng.Float64() * 2 * math.Pi
		d := s.Cfg.Zone.StartRadius * 0.85 * math.Sqrt(s.Rng.Float64())
		x, y := s.Cfg.Zone.StartCenter[0]+d*math.Cos(a), s.Cfg.Zone.StartCenter[1]+d*math.Sin(a)
		if !s.W.InBounds(x, y) {
			continue
		}
		g := s.W.Ground(x, y)
		top := s.W.TopAt(x, y)
		if roof {
			if top > g+4 && top < g+45 && s.W.TopAt(x+2, y) == top && s.W.TopAt(x-2, y) == top && s.W.TopAt(x, y+2) == top && s.W.TopAt(x, y-2) == top {
				return [3]float64{x, y, top}
			}
			continue
		}
		if s.W.Nav.WalkableMain(x, y) {
			return [3]float64{x, y, g}
		}
	}
}

func (s *Sim) advanceZone() {
	s.zphase++
	ph := s.Cfg.Zone.Phases
	if s.zphase >= len(ph) {
		s.znc, s.znr = s.zc, 0
		s.zstart, s.zend = 1e9, 1e9
		return
	}
	p := ph[s.zphase]
	s.zfrom, s.zfromR = s.zc, s.zr
	base := s.T
	if s.zphase == 0 {
		base = 0
	}
	s.zstart = base + p.HoldS
	s.zend = s.zstart + p.ShrinkS

	for tries := 0; ; tries++ {
		a := s.Rng.Float64() * 2 * math.Pi
		d := (s.zr - p.Radius) * math.Sqrt(s.Rng.Float64())
		x, y := s.zc[0]+d*math.Cos(a), s.zc[1]+d*math.Sin(a)
		if tries > 200 || (s.W.InBounds(x, y) && s.W.Nav.WalkableMain(x, y)) {
			s.znc = [2]float64{x, y}
			break
		}
	}
	s.znr = p.Radius
}

func (s *Sim) zoneDPS() float64 {
	ph := s.Cfg.Zone.Phases
	if s.zphase >= len(ph) {
		return ph[len(ph)-1].DPS
	}
	return ph[s.zphase].DPS
}

func (s *Sim) stepZone() {
	if s.zphase >= len(s.Cfg.Zone.Phases) {
		return
	}
	if s.T >= s.zend {
		s.zc, s.zr = s.znc, s.znr
		s.advanceZone()
		return
	}
	if s.T > s.zstart {
		f := (s.T - s.zstart) / (s.zend - s.zstart)
		s.zc[0] = s.zfrom[0] + (s.znc[0]-s.zfrom[0])*f
		s.zc[1] = s.zfrom[1] + (s.znc[1]-s.zfrom[1])*f
		s.zr = s.zfromR + (s.znr-s.zfromR)*f
	}
}

func (s *Sim) inZone(x, y float64) bool { return hyp(x, y, s.zc[0], s.zc[1]) <= s.zr }

func (s *Sim) Give(m *Mech, kind string) {
	m.W[1] = &Weapon{Kind: kind, Stats: s.weaponStats(m, kind)}
}
