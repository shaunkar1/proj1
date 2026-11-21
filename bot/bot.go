// Package bot is the mech pilot: perception, aiming, movement and strategy.
package bot

import (
	"math"
	"math/rand"
	"sort"
	"strconv"
	"time"

	"mechbot/geo"
	"mechbot/proto"
)

// Params are the tunable knobs (so variants can be compared in the simulator).
type Params struct {
	Name            string
	EngageRatio     float64 // fight when myTTK < ratio * theirTTK
	RetreatHP       float64 // fraction of max HP below which we break off
	ReengageHP      float64
	PrefRangeGun    float64
	PrefRangeLas    float64
	PrefRangeLtn    float64
	JukeMin         float64
	JukeMax         float64
	WExpo           float64
	WAtk            float64
	WGoal           float64
	WRange          float64
	Aggressive      bool    // attack any mech on sight
	HuntPassive     bool    // seek out enemies when healthy
	HuntAfter       float64 // start hunting only after this match time
	FarmNPC         bool
	UseDrops        bool
	DropGunForLas   bool
	DropGunForLtn   bool // with a laser in hand, trade the gun for lightning
	UpgradePrio     []string
	WeaponRank      map[string]float64
	LeadScale       float64
	Roof            bool
	NoDodge         bool
	NoAdapt         bool
	PracNoPeek      bool    // in practice (squad) matches, skip the laser-timing caution
	PracNearFarm    bool    // in practice matches, only farm what is close
	PracAggro       bool    // in practice matches, attack any mech on sight while healthy
	PracRetreatHP   float64 // retreat threshold in practice matches (<0: same as RetreatHP)
	PracBerserk     bool    // in practice matches always take the fight
	PracFarmCubes   int     // in practice matches avoid mechs until we hold this many cubes ...
	PracFarmT       float64 // ... or this much time has passed
	PracAggroD      float64 // range of the practice attack-on-sight rule
	PracAggroHP     float64 // health needed for it
	NoChaserRule    bool    // old behaviour: charge the laser even with a lightning carrier closing
	NoRetreatCubes  bool    // old behaviour: ignore cubes while retreating
	OpenCost        float64 // extra path cost for open ground far from buildings (0: off)
	NPCValue        float64 // objective value of a drone (two cubes)
	NPCMemory       bool    // go back to drones seen earlier
	OldArrive       bool    // old behaviour: keep closing on the goal point inside the arrival ring
	OldReload       bool    // old behaviour: top up the magazine whenever quiet
	PreyRatio       float64 // free-for-all: attack on sight mechs whose danger ratio is below this (0: off)
	EarlyEngage     float64 // free-for-all: engage-ratio multiplier while more than EarlyAlive mechs live (0: off)
	EarlyAlive      int
	NoFarCover      bool    // old behaviour: only look for cover within 78 m
	SpinScan        bool    // look around by turning continuously instead of sweeping
	Periscope       float64 // opening jet hop of this many metres to look over rooftops (0: off)
	BossRatio       float64 // free-for-all: avoid mechs this many times stronger than us (0: off)
	NoLaserRush     bool    // old behaviour: no extra pull toward lasers and unseen ground
	NoExplore       bool    // old behaviour: roam at random instead of toward unseen ground
	NoLasCheck      bool    // old behaviour: release the laser without checking the muzzle line
	ChaseHP         float64 // close in on a lone target below this health fraction (0: off)
	ChaseD          float64 // range held while chasing
	PracStopKills   int     // measurement: go idle in practice after this many kills
	Probe           string  // measurement runs (see probeStop)
	Calm            bool    // a stronger mech in view that is not aiming at us and has not hit us lately does not send us to cover
	NoWatchdog      bool    // keep walking at a weapon even when the distance to it stops shrinking
	ObjHide         bool    // with a stronger mech around, only take on work it cannot see
	Open            bool    // strike first with a ready laser, or at a mech busy with someone else
	OpenRatio       float64 // ... unless it is this much stronger than us
	OpenHP          float64
	OpenBusyD       float64
	NoOpenLas       bool
	KeepAwayD       float64 // free-for-all: skip errands this close to a mech at least KeepAwayRatio as strong as us (0 = off)
	KeepAwayRatio   float64
	KeepAwayAlive   int     // ... while more than this many are alive
	QuietT          float64 // free-for-all: for this long treat every mech as one to keep away from (0 = off)
	QuietAlways     bool    // ... also before the match is known to be a free-for-all
	QuietHide       bool    // quiet time also sends us to cover from anyone in view (else it only filters what we walk to)
	QuietObjD       float64
	ShyD            float64 // quiet time: no errand that closes to within this of a mech heard at short or middle range (0 = off)
	ShyFarD         float64 // ... of one heard far away
	ShyNear0        float64 // before the match is known to be a free-for-all, errands shorter than this are always allowed
	ShyLaserD       float64 // a laser this near is fetched whoever was heard (never past a mech in view)
	ShyWeapon0      bool    // ... and so is every walk to a weapon
	ShySoft0        float64 // > 0: before the free-for-all is known the filter only costs points (this many per metre closed) instead of forbidding
	ShyT0           float64 // > 0: quiet time before the free-for-all is known ends this early
	ShyMove0        bool    // ... and with nothing to do we walk on, away from everybody, over unseen ground, instead of standing
	NoShyRoam0      bool    // ... and there is no idling at a quiet spot yet
	QuietNoAim      bool    // quiet time (free-for-all known): being looked at is no reason to open fire, only a charging laser or a hit is
	BossFarD        float64 // free-for-all: an errand this far from the strongest mech is fine even if it is nearer to it than we are
	NoBreak         bool    // no walking away from a spot we keep circling
	ShyRoam         bool    // quiet time with nothing to do: drift to where nobody was seen or heard
	QuietAvoidD     float64
	PracDrop        float64 // squad matches: below this health fraction walk to a supply drop and stand in it (0 = off)
	PracDropD       float64
	FFADropWait     float64  // ... once it has lain there this long
	FFADrop         float64  // free-for-all: same below this health fraction, when nobody else is at the drop
	RngSalt         int      // shifts the random stream (juke timing and the like)
	FFA0            string   // start-of-match free-for-all test (see ffaAtStart)
	Grab            float64  // weapon within this many metres is fetched before anything else (0 = off)
	GrabCube        float64  // same for a cube this close
	GrabAlways      bool     // also before the match is known to be a free-for-all
	ScanAlways      bool     // ... also before the match is known to be a free-for-all
	ScanFirst       bool     // finish the first look around before shooting crates
	PracLtnRank     float64  // squad matches: value of lightning as a second weapon (0: the usual rank; laser is 3)
	Ambush          float64  // open on mechs looking away from us inside this range when we hold a laser (0: off)
	AmbushAng       float64  // how far off us its aim must be (rad)
	AmbushRatio     float64  // skip targets more dangerous than this ratio
	Snipe           float64  // open on laserless mechs between 110 m and this range when we hold a laser (0: off)
	Preempt         float64  // open on any near-equal mech inside this range when we hold a laser (0: off)
	NoBiter         bool     // test: old behaviour, a drone hitting us does not come before pre-aiming at an unseen mech
	NoLegacy        bool     // test: apply the current settings from the first tick in every match
	PileBonus       float64  // extra objective value per other cube lying next to a cube (up to six)
	BossObjD        float64  // objectives this close to a much stronger mech are left alone
	BossAvoidD      float64  // hide from a much stronger mech inside this distance even without a line of sight
	BossSeenD       float64  // ... and inside this distance when it has one
	ExploreOld      bool     // test: search for a laser only when no mech at all is known and after 20 s
	EvadeSniper     bool     // a laser from someone we cannot see breaks off the fight we are in (unless it is nearly won)
	HardRatio       float64  // above this ratio a mech is avoided outright; between BossRatio and this it is only given room (0: everything above BossRatio is avoided outright)
	PeerAvoidD      float64  // step out of an equal\'s line of sight inside this distance
	PeerObjD        float64  // leave alone objectives this close to an equal
	RoofBold        bool     // climb for a weapon on a roof unless an enemy is close; take a near weapon even while repairing
	BossHot         float64  // a mech that is hitting us only counts as one to avoid above this ratio (0: same as BossRatio)
	IdleXUnarmed    bool     // with exploring on: only head for unseen ground while we hold no laser
	Sites           bool     // walk to where mechs died (heard) to collect the cubes they dropped
	SiteFar         float64  // distance assumed for a death heard in the far band
	SiteFarVal      float64  // objective value of such a site
	SiteTTL         float64  // forget a site after this many seconds
	FixRange        float64  // fixed fighting distance in every mode (test stand-ins)
	PracRange       float64  // fixed fighting distance in practice matches (0: choose by damage model)
	Endgame         int      // free-for-all: with this many mechs left (but more than two), only fight back
	PracLasLtn      bool     // in practice matches trade the gun for lightning once we hold a laser
	PracUpgradePrio []string // upgrade order in practice matches (empty: UpgradePrio)
	NoPeek          bool     // ignore enemy laser timing (old behaviour)
	Still           bool     // test: never move
	NoFire          bool     // test: never shoot
}

func DefaultParams() Params {
	p := baseParams()
	ApplyProfile(&p, Profile)
	return p
}

func baseParams() Params {
	return Params{
		Name: "paw", EngageRatio: 1.0, RetreatHP: 0.35, ReengageHP: 0.8,
		PrefRangeGun: 130, PrefRangeLas: 140, PrefRangeLtn: 30,
		JukeMin: 0.35, JukeMax: 0.9, WExpo: 1.0, WAtk: 1.0, WGoal: 1.0, WRange: 1.2,
		HuntPassive: true, HuntAfter: 20, FarmNPC: true, UseDrops: true,
		UpgradePrio: []string{"plating", "armor", "laser_damage", "laser_charge", "nanites", "gun_damage", "gun_rate", "lightning_efficiency",
			"servos", "lightning_chain", "capacitor", "reactor", "gyros", "gun_mag", "lightning_range", "mortar_reload", "mortar_splash",
			"gun_velocity", "mortar_velocity", "sensors", "jets", "laser_pierce"},
		WeaponRank: map[string]float64{"lightning": 2.6, "laser": 3, "mortar": 0, "gun": 0.5}, // no mortar: carrying one makes every laser hit 60% worse
		LeadScale:  1.0,
		PracNoPeek: true, PracNearFarm: true, NoExplore: false, BossRatio: 1.5, HardRatio: 0, Sites: true, RoofBold: true, PeerAvoidD: 60, PeerObjD: 45, BossObjD: 100, BossAvoidD: 90, BossSeenD: 220, PileBonus: 30, Grab: 45, GrabCube: 18, PracDropD: 200, KeepAwayD: 90, KeepAwayRatio: 0.9, KeepAwayAlive: 6, QuietT: 45, QuietObjD: 120, BossFarD: 150, ShyD: 150, ShyFarD: 190, ShyNear0: 40, ShyLaserD: 70, ShyRoam: true, QuietAvoidD: 150, FFADrop: 0.6, FFADropWait: 10, ScanFirst: true, Calm: false, ObjHide: true, EvadeSniper: true, Open: true, NoOpenLas: true, OpenRatio: 1.6, OpenHP: 0.6, OpenBusyD: 180, NPCValue: 170, NPCMemory: true, PracRetreatHP: -1, PracAggroD: 160, PracAggroHP: 0.6, SiteFar: 240, SiteFarVal: 0, SiteTTL: 150,
		Roof: true,
	}
}

type Track struct {
	ID          int
	NPC         bool
	Pos, Vel    [3]float64
	PrevVel     [3]float64
	Aim         [2]float64
	HP          float64
	MaxHP       float64
	Cubes       float64
	Weapons     []string
	Charging    bool
	Seen        float64 // last time present in obs
	First       float64
	LOS         bool    // I can shoot it right now
	HitMe       float64 // last time it damaged me
	DmgToMe     float64 // decayed damage it has done to me
	IHit        float64 // last time I damaged it
	AvgVel      [2]float64
	Scale       float64
	hist        [16]histRec // ring of recent states (10 Hz)
	hn          int
	pred        int // which predictor currently fits this target best
	predHit     [nPred]float64
	shots       float64 // my gun shots at it (decayed)
	hits        float64 // my gun hits on it (decayed)
	lasChargeT  float64 // when its current laser charge began
	lasFireT    float64 // when its laser last fired (or lost its charge)
	wasCharging bool
	Ghost       bool       // position is a guess from being shot, not a sighting
	aimOff      [3]float64 // offset from its centre to the part we have a line on
	Hostile     float64    // considered hostile until this time
	aimedAt     float64    // accumulated seconds spent aiming at us
	has         [4]bool    // gun mortar laser lightning
	busyT       float64    // last time it was seen fighting someone else
}

func (t *Track) Center() [3]float64 {
	h := 9.0 * t.Scale
	if t.NPC {
		h = 3.5
	}
	return [3]float64{t.Pos[0], t.Pos[1], t.Pos[2] + h/2}
}
func (t *Track) Top() [3]float64 {
	h := 9.0 * t.Scale
	if t.NPC {
		h = 3.5
	}
	return [3]float64{t.Pos[0], t.Pos[1], t.Pos[2] + h - 0.3}
}
func (t *Track) Radius() float64 {
	if t.NPC {
		return 1.8
	}
	return 2.5 * t.Scale
}

type histRec struct {
	t        float64
	pos, vel [3]float64
}

const nPred = 5

type Item struct {
	ID   int
	Kind string
	Pos  [3]float64
	HP   float64
	Seen float64
	Roof bool
	Bad  float64 // time until which we ignore it (unreachable)
	wdT  float64 // last time we got closer to it while heading there (0: not yet)
	wdD  float64 // closest we have been
	wdHP float64 // its hit points then (crates)
}

type Bot struct {
	P    Params
	w    *geo.World
	cfg  proto.Config
	slot int
	rng  *rand.Rand

	o   *proto.Obs
	me  *proto.SelfObs
	t   float64
	eye [3]float64
	muz [3]float64

	tracks  map[int]*Track
	npcMem  map[int]*Item
	pickups map[int]*Item
	crates  map[int]*Item
	cubes   map[int]*Item
	trackL  []*Track
	pickupL []*Item
	crateL  []*Item
	cubeL   []*Item
	sites   []*deathSite

	// nav
	path     [][2]float64
	pathGoal [2]float64
	pathT    float64
	pathOK   bool
	pathCost float64

	// movement state
	strafeSign float64
	strafeT    float64
	lastMove   [2]float64
	stuckT     float64
	unstickT   float64
	unstickDir [2]float64
	lastPos    [3]float64
	wantMove   bool

	// combat state
	targetID      int
	retreating    bool
	lastDmgT      float64 // last time I took weapon damage
	lastDmgBy     int
	lastDmgDir    float64
	dmgRecent     float64 // decayed incoming dps estimate
	scanPhase     float64
	lasHold       bool
	cmdAim        [2]float64
	kills         int
	othersDied    int
	prevAlive     int
	mode          string
	goalKind      string
	takeTryT      float64
	hearT         float64
	hearDir       float64
	noise         []noiseSrc
	shyGoal       [2]float64
	shyT          float64
	dithAt        [2]float64 // where the current 12 m circle was entered, when, and the ground covered inside it
	dithT         float64
	dithPath      float64
	dithPrev      [2]float64
	breakT        float64 // walking out of a spot we kept circling
	breakG        [2]float64
	cfgFirst      int // hello differs from the kit's: first section, and how many (see cfgDiff)
	cfgN          int
	wallOn        bool // running under the real server: observation arrival times are kept
	wallAt        []time.Time
	wallTick      []int
	wallMed       float64 // ms between observations early in the match (median, minimum); 0 = not measured
	wallMin       float64
	readyAt       time.Time // when we were ready; and how long the first observation then took (ms)
	readyGap      float64
	shyMoveT      float64
	shyMoveG      [2]float64
	safeSpot      [2]float64
	safeT         float64
	roamGoal      [2]float64
	roamT         float64
	seen          []bool
	unseenG       [2]float64
	unseenT       float64
	seenW, seenH  int
	blockedC      map[int]float64
	lastObjKind   string
	lastObjPos    [3]float64
	objID         int
	objT          float64
	searchTick    int  // tick of the last path search (at most one per decision)
	lasFree       bool // no mech around: the laser may be spent on crates
	lasCrate      bool
	coverNear     bool // last steer found a spot hidden from the target close by
	practice      bool // looks like a squad match (see perceive)
	farmPhase     bool
	patient       bool
	farCover      float64
	nearOnly      bool
	backOff       [2]float64
	backOffT      float64
	quietT        float64
	objNearT      float64
	farTry        float64
	periZ         float64
	boss          *Track
	E             effP
	ffaSure       bool // another mech died early: this is a free-for-all
	seed          int64
	peer          *Track
	sawOtherDeath bool
	quiet         bool
	dropSeen      map[int]float64
	dbgObj        int
	grabID        int
	grabT         float64
	hop           *geo.Exit // jet hop in progress (or being walked to)
	hopFly        bool
	hopT          float64
	hopFail       float64
	climbID       int
	climbT        float64
	climbPos      [3]float64
	climbFrom     [2]float64
	spawnYaw      float64
	zonePathT     float64
	zonePathG     [2]float64
	zonePathC     float64
	dbg           string
}

func New(h *proto.Hello, w *geo.World, p Params) *Bot {
	b := &Bot{P: p, w: w, cfg: h.Config, slot: h.Slot}
	b.seed = int64(h.Seed)
	b.cfgFirst, b.cfgN = cfgDiff(h.Raw)
	b.wallOn = len(h.Raw) > 0
	if b.ffaAtStart() {
		b.ffaSure, b.sawOtherDeath = true, true
	}
	b.rng = rand.New(rand.NewSource(int64(h.Seed) + 7919*int64(h.Slot) + 104729*int64(p.RngSalt)))
	b.tracks = map[int]*Track{}
	b.pickups = map[int]*Item{}
	b.crates = map[int]*Item{}
	b.cubes = map[int]*Item{}
	b.blockedC = map[int]float64{}
	b.strafeSign = 1
	b.targetID = -1
	b.lastDmgBy = -1
	b.lastDmgT = -100
	b.climbID = -1
	b.zonePathT = -100
	b.prevAlive = int(h.Config.Match.Mechs)
	if b.prevAlive == 0 {
		b.prevAlive = 16
	}
	b.readyAt = time.Now()
	return b
}

func sortItems(dst []*Item, m map[int]*Item) []*Item {
	for _, it := range m {
		dst = append(dst, it)
	}
	sort.Slice(dst, func(i, j int) bool { return dst[i].ID < dst[j].ID })
	return dst
}

func (b *Bot) Name() string { return b.P.Name }

func (b *Bot) weapon(kind string) (*proto.WeaponObs, int) {
	for i, w := range b.me.Weapons {
		if w != nil && w.Kind == kind {
			return w, i
		}
	}
	return nil, -1
}

func (b *Bot) dmgMul() float64 {
	c := math.Min(b.me.Cubes, b.cfg.Cubes.StatCap)
	return 1 + b.cfg.Cubes.DamagePerCube*c
}

func (b *Bot) perceive(o *proto.Obs) {
	b.o = o
	b.me = &o.Self
	b.t = o.T
	me := b.me
	b.eye = [3]float64{me.Pos[0], me.Pos[1], me.Pos[2] + me.Stats.EyeHeight}
	b.muz = [3]float64{me.Pos[0], me.Pos[1], me.Pos[2] + me.Stats.MuzzleHeight}
	for _, t := range b.tracks {
		t.LOS = false
	}
	for i := range o.Mechs {
		a := &o.Mechs[i]
		t := b.tracks[a.ID]
		if t == nil {
			t = &Track{ID: a.ID, NPC: a.Kind == "npc", First: o.T, HitMe: -100, IHit: -100, busyT: -100, Scale: 1, lasFireT: -100, lasChargeT: -100}
			b.tracks[a.ID] = t
		}
		if !t.NPC && !t.Ghost && o.T-t.Seen < 0.35 && a.HP < t.HP-20 && o.T-t.IHit > 0.45 {
			t.busyT = o.T // someone else is hurting it
		}
		t.Ghost = false
		t.PrevVel = t.Vel
		t.Pos, t.Vel, t.Aim = a.Pos, a.Vel, a.Aim
		t.HP, t.MaxHP, t.Cubes, t.Weapons, t.Charging = a.HP, a.MaxHP, a.Cubes, a.Weapons, a.Charging
		if a.Charging && !t.wasCharging {
			t.lasChargeT = o.T
		}
		if !a.Charging && t.wasCharging {
			t.lasFireT = o.T
		}
		t.wasCharging = a.Charging
		t.AvgVel[0] = 0.85*t.AvgVel[0] + 0.15*a.Vel[0]
		t.AvgVel[1] = 0.85*t.AvgVel[1] + 0.15*a.Vel[1]
		if o.T-t.Seen > 0.25 {
			t.hn = 0 // gap in the data: restart the history
		}
		t.hist[t.hn%len(t.hist)] = histRec{o.T, a.Pos, a.Vel}
		t.hn++
		t.Seen = o.T
		if !t.NPC {
			b.scorePredictors(t)
		}
		if !t.NPC {
			t.Scale = 1 + b.cfg.Cubes.SizePerCube*a.Cubes
			t.has = [4]bool{}
			for _, w := range a.Weapons {
				switch w {
				case "gun":
					t.has[0] = true
				case "mortar":
					t.has[1] = true
				case "laser":
					t.has[2] = true
				case "lightning":
					t.has[3] = true
				}
			}
		}
		c := t.Center()
		tp := t.Top()
		// which part of it can we hit? centre, top, or either flank
		t.aimOff = [3]float64{}
		t.LOS = b.w.Clear(b.muz[0], b.muz[1], b.muz[2], c[0], c[1], c[2])
		if !t.LOS && b.w.Clear(b.muz[0], b.muz[1], b.muz[2], tp[0], tp[1], tp[2]) {
			t.LOS = true
			t.aimOff = [3]float64{0, 0, tp[2] - c[2] - 0.6}
		}
		if !t.LOS {
			dx, dy := c[0]-b.muz[0], c[1]-b.muz[1]
			if l := math.Hypot(dx, dy); l > 3 {
				r := t.Radius() * 0.75
				ox, oy := -dy/l*r, dx/l*r
				if b.w.Clear(b.muz[0], b.muz[1], b.muz[2], c[0]+ox, c[1]+oy, c[2]) {
					t.LOS = true
					t.aimOff = [3]float64{ox, oy, 0}
				} else if b.w.Clear(b.muz[0], b.muz[1], b.muz[2], c[0]-ox, c[1]-oy, c[2]) {
					t.LOS = true
					t.aimOff = [3]float64{-ox, -oy, 0}
				}
			}
		}
		if !t.NPC {
			d := hyp(a.Pos[0], a.Pos[1], me.Pos[0], me.Pos[1])
			brg := math.Atan2(me.Pos[1]-a.Pos[1], me.Pos[0]-a.Pos[0])
			off := math.Abs(angDiff(a.Aim[0], brg))
			tol := math.Max(0.12, math.Atan2(6, d))
			if t.LOS && off < tol && d < 260 {
				t.aimedAt += 0.1
				if a.Charging {
					t.aimedAt += 0.3
				}
			} else {
				t.aimedAt = math.Max(0, t.aimedAt-0.05)
			}
			if a.Charging && off > 0.6 {
				t.busyT = o.T // lining up on somebody else
			}
			if t.aimedAt >= 0.6 && !(b.P.QuietNoAim && b.quiet && b.ffaSure && !a.Charging && d > 60) {
				t.Hostile = math.Max(t.Hostile, o.T+8)
			}
			if d < 30 && t.LOS {
				t.Hostile = math.Max(t.Hostile, o.T+4)
			}
		}
	}
	// drones stay near home: remember where we saw them (two cubes each)
	if b.npcMem == nil {
		b.npcMem = map[int]*Item{}
	}
	for id, t := range b.tracks {
		if t.NPC && o.T-t.Seen < 0.25 {
			it := b.npcMem[id]
			if it == nil {
				it = &Item{ID: id}
				b.npcMem[id] = it
			}
			it.Pos, it.Seen = t.Pos, o.T
		}
	}
	for id, it := range b.npcMem {
		tr := b.tracks[id]
		gone := (tr == nil || o.T-tr.Seen > 1.0) && hyp(it.Pos[0], it.Pos[1], me.Pos[0], me.Pos[1]) < 40 &&
			b.w.Clear(b.eye[0], b.eye[1], b.eye[2], it.Pos[0], it.Pos[1], it.Pos[2]+2)
		if gone || o.T-it.Seen > 120 {
			delete(b.npcMem, id)
		}
	}
	for id, t := range b.tracks {
		if t.NPC && o.T-t.Seen > 6 {
			delete(b.tracks, id)
		} else if o.T-t.Seen > 25 {
			delete(b.tracks, id)
		}
		t.DmgToMe *= 0.97
	}
	// items: refresh seen ones; forget ones that should be visible but are not
	seenP := map[int]bool{}
	for _, p := range o.Pickups {
		seenP[p.ID] = true
		it := b.pickups[p.ID]
		if it == nil {
			it = &Item{ID: p.ID}
			b.pickups[p.ID] = it
			it.Roof = p.Pos[2] > b.w.Ground(p.Pos[0], p.Pos[1])+2.5
		}
		it.Kind, it.Pos, it.Seen = p.Kind, p.Pos, o.T
	}
	near := b.cfg.Vision.NearRange + me.Stats.NearRangeBonus - 3
	for id, it := range b.pickups {
		if !seenP[id] && hyp(it.Pos[0], it.Pos[1], me.Pos[0], me.Pos[1]) < near && b.w.Clear(b.eye[0], b.eye[1], b.eye[2], it.Pos[0], it.Pos[1], it.Pos[2]+0.5) {
			delete(b.pickups, id)
		}
	}
	seenC := map[int]bool{}
	for _, c := range o.Crates {
		seenC[c.ID] = true
		it := b.crates[c.ID]
		if it == nil {
			it = &Item{ID: c.ID}
			b.crates[c.ID] = it
			it.Roof = c.Pos[2] > b.w.Ground(c.Pos[0], c.Pos[1])+2.5
		}
		it.Pos, it.HP, it.Seen = c.Pos, c.HP, o.T
	}
	for id, it := range b.crates {
		if !seenC[id] && hyp(it.Pos[0], it.Pos[1], me.Pos[0], me.Pos[1]) < near && b.w.Clear(b.eye[0], b.eye[1], b.eye[2], it.Pos[0], it.Pos[1], it.Pos[2]+1.2) {
			delete(b.crates, id)
		}
	}
	seenQ := map[int]bool{}
	for _, c := range o.Cubes {
		seenQ[c.ID] = true
		it := b.cubes[c.ID]
		if it == nil {
			it = &Item{ID: c.ID}
			b.cubes[c.ID] = it
			it.Roof = c.Pos[2] > b.w.Ground(c.Pos[0], c.Pos[1])+2.5
		}
		it.Pos, it.Seen = c.Pos, o.T
	}
	for id, it := range b.cubes {
		if !seenQ[id] && (hyp(it.Pos[0], it.Pos[1], me.Pos[0], me.Pos[1]) < near || o.T-it.Seen > 40) {
			delete(b.cubes, id)
		}
	}
	// events
	for i := range o.Events {
		e := &o.Events[i]
		switch e.Type {
		case "damaged":
			if e.Weapon == "zone" {
				continue
			}
			b.lastDmgT = o.T
			b.dmgRecent += e.Amount
			if e.Bearing != nil {
				b.lastDmgDir = *e.Bearing
			}
			if e.By != nil {
				b.lastDmgBy = *e.By
				t := b.tracks[*e.By]
				if *e.By < 100 && e.Bearing != nil && (t == nil || o.T-t.Seen > 0.35 || t.Ghost) {
					// shot by something we cannot see: place a ghost along the bearing
					d := 110.0
					switch e.Weapon {
					case "lightning":
						d = 35
					case "mortar":
						d = 60
					}
					if t != nil && !t.Ghost && o.T-t.Seen < 8 {
						d = hyp(t.Pos[0], t.Pos[1], me.Pos[0], me.Pos[1])
					}
					for i := range o.Beams {
						if bm := &o.Beams[i]; bm.By == *e.By && len(bm.Path) > 0 {
							d = hyp(bm.Path[0][0], bm.Path[0][1], me.Pos[0], me.Pos[1])
						}
					}
					if t == nil {
						t = &Track{ID: *e.By, First: o.T, IHit: -100, Scale: 1, lasFireT: -100, lasChargeT: -100, MaxHP: 1400, HP: 1400}
						b.tracks[*e.By] = t
					}
					gx, gy := me.Pos[0]+math.Cos(*e.Bearing)*d, me.Pos[1]+math.Sin(*e.Bearing)*d
					t.Pos = [3]float64{gx, gy, b.w.Ground(gx, gy)}
					t.Vel = [3]float64{}
					t.Ghost = true
					t.Seen = o.T
					t.LOS = false
					switch e.Weapon {
					case "gun":
						t.has[0] = true
					case "mortar":
						t.has[1] = true
					case "laser":
						t.has[2] = true
						t.lasFireT = o.T
					case "lightning":
						t.has[3] = true
					}
				}
				if t != nil {
					t.HitMe = o.T
					t.DmgToMe += e.Amount
					t.Hostile = math.Max(t.Hostile, o.T+25)
				}
			}
		case "hit":
			if t := b.tracks[e.Target]; t != nil {
				t.IHit = o.T
				if e.Weapon == "gun" {
					t.hits++
				}
			}
		case "kill":
			b.kills++
			delete(b.tracks, e.Target)
			if b.targetID == e.Target {
				b.targetID = -1
			}
		}
	}
	for i := range o.Beams {
		if bm := &o.Beams[i]; bm.Kind == "laser" {
			if t := b.tracks[bm.By]; t != nil {
				t.lasFireT = o.T - bm.Age
			}
		}
	}
	b.dmgRecent *= 0.9
	if o.Alive < b.prevAlive {
		b.prevAlive = o.Alive
	}
	if !b.ffaSure && b.wallMed > 0 && b.ffaByClock() {
		b.ffaSure, b.sawOtherDeath = true, true
	}
	b.othersDied = int(b.cfg.Match.Mechs) - o.Alive - b.kills
	if b.othersDied > 0 {
		b.sawOtherDeath = true
		if o.T < 40 {
			b.ffaSure = true
		}
	}
	// practice = everyone else is one squad: nobody dies unless we kill them
	b.practice = !b.sawOtherDeath && o.T > 20
	for _, s := range o.Sounds {
		if s.Kind == "death" && b.P.Sites && !b.practice {
			b.heardDeath(s.Bearing, s.Dist, o.T-s.Age)
		}
		switch s.Kind {
		case "gun", "laser", "laser_charge", "lightning", "mortar", "reload", "boost", "jet", "step":
			b.heardMech(s.Kind, s.Bearing, s.Dist, o.T-s.Age)
		}
		if s.Dist == "near" && (s.Kind == "step" || s.Kind == "boost" || s.Kind == "jet" || s.Kind == "reload" || s.Kind == "laser_charge" || s.Kind == "lightning" || s.Kind == "gun") {
			b.hearT = o.T - s.Age
			b.hearDir = s.Bearing
		}
	}
	b.trackL = b.trackL[:0]
	for _, t := range b.tracks {
		b.trackL = append(b.trackL, t)
	}
	sort.Slice(b.trackL, func(i, j int) bool { return b.trackL[i].ID < b.trackL[j].ID })
	b.pickupL = sortItems(b.pickupL[:0], b.pickups)
	b.crateL = sortItems(b.crateL[:0], b.crates)
	b.cubeL = sortItems(b.cubeL[:0], b.cubes)
	for c, until := range b.blockedC {
		if o.T > until {
			delete(b.blockedC, c)
		}
	}
}

// Act is the per-decision entry point.
func (b *Bot) Act(o *proto.Obs) proto.Action {
	if b.wallOn && b.readyGap == 0 {
		b.readyGap = math.Max(float64(time.Since(b.readyAt).Microseconds())/1000, 0.01)
	}
	if b.wallOn && b.wallMed == 0 {
		b.wallClock(o)
	}
	b.perceive(o)
	if !b.P.NoExplore {
		b.markSeen()
	}
	if o.Tick <= 1 {
		b.spawnYaw = o.Self.Aim[0]
	}
	act := proto.Action{Tick: o.Tick}
	act.Aim = b.cmdAim
	if o.Tick <= 1 && b.cmdAim == ([2]float64{}) {
		act.Aim = o.Self.Aim
	}
	b.decide(&act)
	if b.P.Still {
		act.Move = proto.Vec2{}
		act.Jump = false
	}
	if b.P.NoFire {
		act.Fire = [2]bool{}
	}
	if b.P.Probe != "" && b.probeStop() {
		act.Move = proto.Vec2{}
		act.Jump = false
		act.Boost = false
		act.Fire = [2]bool{}
	}
	if b.P.PracStopKills > 0 && b.practice && b.kills >= b.P.PracStopKills {
		// measurement runs: stop playing after this many kills
		act.Move = proto.Vec2{}
		act.Jump = false
		act.Boost = false
		act.Fire = [2]bool{}
	}
	if !finite(act.Aim[0]) || !finite(act.Aim[1]) {
		act.Aim = o.Self.Aim
	}
	if !finite(act.Move[0]) || !finite(act.Move[1]) {
		act.Move = proto.Vec2{}
	}
	act.Aim[1] = clamp(act.Aim[1], b.me.Stats.MinPitch, b.me.Stats.MaxPitch)
	b.cmdAim = act.Aim
	b.lastPos = o.Self.Pos
	b.trace(act)
	return act
}

// DbgMode returns the last decision's mode label (for tooling).
func (b *Bot) DbgMode() string { return b.dbg }

// ffaAtStart: rules that tell a free-for-all from a squad match before the first tick.
func (b *Bot) ffaAtStart() bool {
	switch b.P.FFA0 {
	case "always":
		return true
	case "slot":
		return b.slot != 0
	case "seed":
		return b.seed >= 1000000
	case "cfg":
		return b.cfgN > 0
	}
	return false
}

// ffaByClock: the match is a free-for-all because the server waits on other teams' bots between ticks.
func (b *Bot) ffaByClock() bool {
	if len(b.P.FFA0) > 4 && b.P.FFA0[:4] == "wall" {
		if v, err := strconv.Atoi(b.P.FFA0[4:]); err == nil {
			return b.wallMed > float64(v)
		}
	}
	return false
}

// probeStop: measurement builds go idle under a condition, so the score reveals
// something about the matches the scorer runs.
func (b *Bot) probeStop() bool {
	if len(b.P.Probe) > 1 && b.P.Probe[0] == 't' {
		// stop at a set time: the score is then the kills made by that time
		if v, err := strconv.Atoi(b.P.Probe[1:]); err == nil {
			return b.t >= float64(v)
		}
	}
	if len(b.P.Probe) > 2 && b.P.Probe[:2] == "dt" {
		// stop when the server ticks slower than this many milliseconds
		if v, err := strconv.Atoi(b.P.Probe[2:]); err == nil {
			return b.wallMed > float64(v)
		}
	}
	if b.P.Probe == "tm" {
		// ... or when either clock looks unlike a quiet server: slow ticks, or a long wait for the first one
		return b.wallMed > 4 || b.readyGap > 150
	}
	switch b.P.Probe {
	case "slot":
		return b.kills >= b.slot
	case "slotr":
		return b.kills >= 15-b.slot
	case "seedlow":
		return b.seed < 1000000
	case "seedhi":
		return b.seed >= 1<<31
	case "seedodd":
		return b.seed%2 == 1
	case "idle":
		return true
	}
	return false
}

// effP are the settings in force this tick (see decide).
type effP struct {
	BossRatio, HardRatio, BossObjD, BossAvoidD, BossSeenD, PileBonus float64
	ExploreOld                                                       bool
	Calm                                                             bool
	ObjHide                                                          bool
	Watchdog                                                         bool
}

// noiseSrc is a place where a mech was heard (never seen): bearing is good to a few
// degrees, distance only to a band.
type noiseSrc struct {
	pos [2]float64
	t   float64
	far bool
}

// heardMech notes where an unseen mech made a sound.
func (b *Bot) heardMech(kind string, brg float64, band string, at float64) {
	px, py := b.me.Pos[0], b.me.Pos[1]
	rng := b.cfg.Sound.Range[kind]
	if rng <= 0 {
		rng = 200
	}
	near, mid := b.cfg.Sound.Near, b.cfg.Sound.Mid
	if near <= 0 || mid <= near {
		near, mid = 60, 160
	}
	var d float64
	far := false
	switch band {
	case "near":
		d = math.Min(near, rng) * 0.6
	case "mid":
		d = (near + math.Min(mid, math.Max(rng, near))) / 2
	default:
		d = (mid + math.Max(rng, mid)) / 2
		far = true
	}
	// somebody we are looking at anyway?
	for _, t := range b.tracks {
		if t.NPC || t.Ghost || at-t.Seen > 2 {
			continue
		}
		td := hyp(px, py, t.Pos[0], t.Pos[1])
		if math.Abs(angDiff(math.Atan2(t.Pos[1]-py, t.Pos[0]-px), brg)) < 0.3 && td > d*0.5-20 && td < d*1.6+30 {
			return
		}
	}
	x, y := px+math.Cos(brg)*d, py+math.Sin(brg)*d
	keep := b.noise[:0]
	for _, n := range b.noise {
		if at-n.t > 20 {
			continue
		}
		if hyp(n.pos[0], n.pos[1], x, y) < 45 || (n.far && far && math.Abs(angDiff(math.Atan2(n.pos[1]-py, n.pos[0]-px), brg)) < 0.25) {
			continue // the same one again: keep the newer fix
		}
		keep = append(keep, n)
	}
	b.noise = append(keep, noiseSrc{pos: [2]float64{x, y}, t: at, far: far})
}

// wallClock measures how long the server takes between observations in the first second.
func (b *Bot) wallClock(o *proto.Obs) {
	if o.T < 0.05 {
		return
	}
	if o.T <= 0.82 {
		b.wallAt = append(b.wallAt, time.Now())
		b.wallTick = append(b.wallTick, o.Tick)
		return
	}
	var d []float64
	for i := 1; i < len(b.wallAt); i++ {
		if b.wallTick[i]-b.wallTick[i-1] == 2 {
			d = append(d, float64(b.wallAt[i].Sub(b.wallAt[i-1]).Microseconds())/1000)
		}
	}
	if len(d) < 3 {
		b.wallMed, b.wallMin = -1, -1
		return
	}
	sort.Float64s(d)
	b.wallMed, b.wallMin = math.Max(d[len(d)/2], 0.01), math.Max(d[0], 0.01)
}

// gapCode is wallCode in whole octaves (1/16 ms to 32 s).
func gapCode(ms float64) float64 {
	if ms <= 0 {
		return 0
	}
	k := math.Round(math.Log2(ms))
	return 0.03 * (math.Max(-4, math.Min(15, k)) + 5)
}

// wallCode turns milliseconds into a torso pitch: 0.03 rad per half octave, 0.15 rad = 1 ms.
func wallCode(ms float64) float64 {
	if ms <= 0 {
		return 0.6
	}
	k := math.Round(2 * math.Log2(ms))
	return 0.03 * (math.Max(-4, math.Min(14, k)) + 5)
}
