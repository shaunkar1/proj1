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

type Params struct {
	AggroD     float64
	AggroAfter float64
	AggroHP    float64
	Camp       bool
	CampD      float64
	CampDuty   float64
	EarP       float64
	EarWalk    float64
	EarAfter   float64

	Name            string
	EngageRatio     float64
	RetreatHP       float64
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
	Aggressive      bool
	HuntPassive     bool
	HuntAfter       float64
	FarmNPC         bool
	UseDrops        bool
	DropGunForLas   bool
	DropGunForLtn   bool
	UpgradePrio     []string
	WeaponRank      map[string]float64
	LeadScale       float64
	Roof            bool
	NoDodge         bool
	NoAdapt         bool
	PracNoPeek      bool
	PracNearFarm    bool
	PracAggro       bool
	PracRetreatHP   float64
	PracBerserk     bool
	PracFarmCubes   int
	PracFarmT       float64
	PracAggroD      float64
	PracAggroHP     float64
	NoChaserRule    bool
	NoRetreatCubes  bool
	OpenCost        float64
	NPCValue        float64
	NPCMemory       bool
	OldArrive       bool
	OldReload       bool
	PreyRatio       float64
	EarlyEngage     float64
	EarlyAlive      int
	NoFarCover      bool
	SpinScan        bool
	Periscope       float64
	BossRatio       float64
	NoLaserRush     bool
	NoExplore       bool
	NoLasCheck      bool
	ChaseHP         float64
	ChaseD          float64
	PracStopKills   int
	Probe           string
	Calm            bool
	NoWatchdog      bool
	ObjHide         bool
	Open            bool
	OpenRatio       float64
	OpenHP          float64
	OpenBusyD       float64
	NoOpenLas       bool
	KeepAwayD       float64
	KeepAwayRatio   float64
	KeepAwayAlive   int
	QuietT          float64
	QuietAlways     bool
	QuietHide       bool
	QuietObjD       float64
	ShyD            float64
	ShyFarD         float64
	ShyNear0        float64
	ShyLaserD       float64
	ShyWeapon0      bool
	ShySoft0        float64
	ShyT0           float64
	ShyMove0        bool
	NoShyRoam0      bool
	QuietNoAim      bool
	BossFarD        float64
	NoBreak         bool
	QuietCap        float64
	QuietEngage     float64
	QuietAdvD       float64
	FFAAdvD         float64
	ShyInside       float64
	ZoneLead        float64
	CrossAlive      int
	CrossD          float64
	ShyMem          float64
	FleeLaser       int
	QuietNoClose    bool
	SafeAway        float64
	HurtHP          float64
	HitMem          float64
	LtnObjD         float64
	LateApproachR   float64
	LateApproach    float64
	PatientFinish   float64
	ShyHold         float64
	HideCover       float64
	HideGeo         float64
	HideStick       float64
	HideStill       bool
	HideCorner      float64
	HideCtr         float64
	HideRoof        float64
	LateStrike      int
	LateStrikeD     float64
	LateStrikeHP    float64
	LateStrikeRatio float64
	QuietNoChase    bool
	EndLow          int
	EndLowZoneR     float64
	EndLowD         float64
	EndLowFarm      bool
	FFAPassive      bool
	QuietWalk       float64
	QuietWalkT      float64
	QuietWalkQ      bool
	QuietWalkZone   bool
	ZoneGoalIn      bool
	ZoneStall       float64
	ZoneStallE      float64
	HushT           float64
	HushD           float64
	HushLaser       bool
	FleeD           float64
	FleeLasD        float64
	FleeV           float64
	FleeT           float64
	FleeHold        float64
	FleeRatio       float64
	FleeMem         float64
	ScanAhead       float64
	EndScan         int
	EndgameFFA      int
	QuietAlive      int
	FFAAfter        int64
	LtnKite         bool
	Mix             string
	ShyRoam         bool
	QuietAvoidD     float64
	PracDrop        float64
	PracDropD       float64

	PracKS          float64
	PracStickyOn    bool
	PracStickyT     float64
	PracCubeMax     int
	PracDropWise    float64
	PracStand       float64
	FFADropWait     float64
	FFADrop         float64
	RngSalt         int
	MoveJit         int
	FFA0            string
	Grab            float64
	GrabCube        float64
	GrabAlways      bool
	ScanAlways      bool
	ScanFirst       bool
	PracLtnRank     float64
	Ambush          float64
	AmbushAng       float64
	AmbushRatio     float64
	Snipe           float64
	Preempt         float64
	NoBiter         bool
	NoLegacy        bool
	PileBonus       float64
	BossObjD        float64
	BossAvoidD      float64
	BossSeenD       float64
	ExploreOld      bool
	EvadeSniper     bool
	HardRatio       float64
	PeerAvoidD      float64
	PeerObjD        float64
	RoofBold        bool
	BossHot         float64
	IdleXUnarmed    bool
	Sites           bool
	SiteFar         float64
	SiteFarVal      float64
	SiteTTL         float64
	FixRange        float64
	PracRange       float64
	Endgame         int
	PracLasLtn      bool
	PracUpgradePrio []string
	NoPeek          bool
	Still           bool
	NoFire          bool
	FFADropFree     float64
	FFADropFreeD    float64
	RoofLoot        bool
	RoofLootClear   float64
	QuietScout      float64
	QuietScoutClear float64

	EndPounce  float64
	EndPounceT float64
	EndRushD   float64
	EndRefuse  float64
	EndStalk   float64
	EndThird   float64
	EndDrop    int
	EndDropD   float64
	EndDropR   float64

	CubeCap        int
	HotR           float64
	HotT           float64
	ThreatMem      float64
	ThreatMemRatio float64
	ZoneBrave      float64
	ZoneCommit     float64
	HushMem        float64
	HushLook       bool
	FarmClear      float64
	FarmClearT     float64
	FarmScout      float64
	PileAge        float64
	SilentCollect  float64
	SilentClear    float64
	LaserErrand    float64
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
		WeaponRank: map[string]float64{"lightning": 2.6, "laser": 3, "mortar": 0, "gun": 0.5},
		LeadScale:  1.0,
		PracNoPeek: true, PracNearFarm: true, NoExplore: false, BossRatio: 1.5, HardRatio: 0, Sites: true, RoofBold: true, PeerAvoidD: 60, PeerObjD: 45, BossObjD: 100, BossAvoidD: 90, BossSeenD: 220, PileBonus: 30, Grab: 45, GrabCube: 18, PracDropD: 200, KeepAwayD: 90, KeepAwayRatio: 0.9, KeepAwayAlive: 6, QuietT: 90, QuietNoAim: true, QuietAlive: 6, QuietCap: 300, EndgameFFA: 16, QuietAdvD: 50, FleeLaser: 1, QuietNoClose: true, QuietNoChase: true, SafeAway: 1.0, LateStrikeD: 130, LateStrikeHP: 0.8, LateStrikeRatio: 1.0, EndLow: 10, EndLowD: 40, ZoneGoalIn: true, FFAPassive: true, HideStick: 40, HideStill: true, HideCover: 120, EndPounce: 0.85, EndRushD: 110, EndPounceT: 12, EndRefuse: 1.25, EndScan: 6, HushT: 45, HushD: 200, FleeV: 3.5, FleeT: 2, FleeHold: 6, FleeRatio: 0.3, FleeMem: 20, QuietObjD: 120, BossFarD: 150, ShyD: 150, ShyFarD: 190, ShyNear0: 40, ShyLaserD: 70, ShyRoam: true, QuietAvoidD: 150, FFADrop: 0.6, FFADropWait: 10, ScanFirst: true, Calm: false, ObjHide: true, EvadeSniper: true, Open: true, NoOpenLas: true, OpenRatio: 1.6, OpenHP: 0.6, OpenBusyD: 180, NPCValue: 170, NPCMemory: true, PracRetreatHP: -1, PracAggroD: 160, PracAggroHP: 0.6, SiteFar: 240, SiteFarVal: 0, SiteTTL: 150,
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
	Seen        float64
	First       float64
	LOS         bool
	HitMe       float64
	DmgToMe     float64
	IHit        float64
	AvgVel      [2]float64
	Scale       float64
	hist        [16]histRec
	hn          int
	pred        int
	predHit     [nPred]float64
	shots       float64
	hits        float64
	lasChargeT  float64
	lasFireT    float64
	wasCharging bool
	Ghost       bool
	aimOff      [3]float64
	Hostile     float64
	aimedAt     float64
	has         [4]bool
	busyT       float64
	closeT      float64
	fledT       float64
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
	Bad  float64
	wdT  float64
	wdD  float64
	wdHP float64
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

	path     [][2]float64
	pathGoal [2]float64
	pathT    float64
	pathOK   bool
	pathCost float64

	strafeSign float64
	strafeT    float64
	lastMove   [2]float64
	stuckT     float64
	unstickT   float64
	unstickDir [2]float64
	lastPos    [3]float64
	wantMove   bool

	targetID      int
	retreating    bool
	lastDmgT      float64
	quietWalkN    int
	lastDmgBy     int
	lastDmgDir    float64
	dmgRecent     float64
	scanPhase     float64
	lasHold       bool
	cmdAim        [2]float64
	campHold      bool
	earT          float64
	earHeardT     float64
	earDir        float64
	earPos        [2]float64
	earRolls      []noiseSrc
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
	ltnClose      bool
	dithAt        [2]float64
	dithT         float64
	dithPath      float64
	dithPrev      [2]float64
	breakT        float64
	breakG        [2]float64
	breakC        [2]float64
	breakTried    []float64
	cfgFirst      int
	cfgN          int
	wallOn        bool
	wallAt        []time.Time
	wallTick      []int
	wallMed       float64
	wallMin       float64
	readyAt       time.Time
	readyGap      float64
	shyMoveT      float64
	shyMoveG      [2]float64
	hideExpo      map[int]float32
	hideCornerG   [2]float64
	hideSurvC     [2]float64
	hideSurvI     int
	hideCornerT   float64
	hideN         [4]int
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
	searchTick    int
	lasFree       bool
	lasCrate      bool
	coverNear     bool
	practice      bool
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
	ffaSure       bool
	seed          int64
	peer          *Track
	sawOtherDeath bool
	ksN           int
	ksPrevT       float64
	pracLatch     bool
	standT        float64
	standOn       bool
	quiet         bool
	endLow        bool
	hush          bool
	fleeG         [2]float64
	fleeUntil     float64
	fleeID        int
	dropSeen      map[int]float64
	e8T           *Track
	e8Go          bool
	e8Site        [2]float64
	e8SiteT       float64
	e8Las         *Track
	e8Snd         []e8snd
	e8Fight       [2]float64
	e8FightT      float64
	e8FightSeen   bool
	e8FightFar    bool
	e8Hold        [2]float64
	e8HoldT       float64
	dbgObj        int
	DbgGoal       [2]float64
	DbgHasGoal    bool
	grabID        int
	grabT         float64
	hop           *geo.Exit
	hopFly        bool
	hopT          float64
	hopFail       float64
	climbID       int
	climbT        float64
	climbPos      [3]float64
	climbFrom     [2]float64
	climbCube     bool
	spawnYaw      float64
	zonePathT     float64
	zonePathG     [2]float64
	zonePathC     float64
	zoneAltG      [2]float64
	zoneAltT      float64
	zoneAltC      zoneC
	zoneAltBad    [][3]float64
	zsBest        float64
	zsBestC       float64
	zsT           float64
	zsUrgT        float64
	zsN           int
	boltT         float64
	boltDir       [2]float64
	boltZ         float64
	zsCirc        zoneC
	dbg           string
	x10           x10state
}

func New(h *proto.Hello, w *geo.World, p Params) *Bot {
	if p.Mix != "" {
		if prof := mixPick(p.Mix, int64(h.Seed), h.Slot); prof != "" {
			name := p.Name
			p = baseParams()
			p.Name = name
			ApplyProfile(&p, prof)
		}
	}
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
			t.busyT = o.T
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
			t.hn = 0
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
				t.busyT = o.T
			}
			if t.aimedAt >= 0.6 && !(b.P.QuietNoAim && b.quiet && b.ffaSure && !a.Charging && d > 60) {
				t.Hostile = math.Max(t.Hostile, o.T+8)
			}
			if d < 30 && t.LOS {
				t.Hostile = math.Max(t.Hostile, o.T+4)
			}
		}
	}

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
			if b.P.RoofLoot && b.ffaSure && !b.practice && it.Roof && o.T-it.Seen < 120 && !b.w.Clear(b.eye[0], b.eye[1], b.eye[2], it.Pos[0], it.Pos[1], it.Pos[2]+0.5) {
				continue
			}
			delete(b.cubes, id)
		}
	}

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
			delete(b.x10.mem, e.Target)
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
	if !b.ffaSure && b.wallOn && b.P.FFA0 != "" && o.T < 2 && b.ffaByClock() {
		b.ffaSure, b.sawOtherDeath = true, true
	}
	b.othersDied = int(b.cfg.Match.Mechs) - o.Alive - b.kills - b.ksN
	if b.P.PracKS > 0 && b.othersDied > 0 && !b.sawOtherDeath && !b.ffaSure && o.T >= 40 {

		present := map[int]bool{}
		for i := range o.Mechs {
			present[o.Mechs[i].ID] = true
		}
		ids := make([]int, 0, len(b.tracks))
		for id := range b.tracks {
			ids = append(ids, id)
		}
		sort.Ints(ids)
		for _, id := range ids {
			t := b.tracks[id]
			if b.othersDied <= 0 {
				break
			}
			if t.NPC || t.Ghost || present[id] || o.T-t.IHit > b.P.PracKS || t.Seen < b.ksPrevT-0.01 {
				continue
			}
			b.ksN++
			b.othersDied--
			delete(b.tracks, id)
			if b.targetID == id {
				b.targetID = -1
			}
		}
	}
	b.ksPrevT = o.T
	if b.P.PracStickyOn && !b.ffaSure && !b.sawOtherDeath && o.T >= b.P.PracStickyT {
		b.pracLatch = true
	}
	if b.pracLatch {
		b.othersDied = 0
	}
	if b.othersDied > 0 {
		b.sawOtherDeath = true
		if o.T < 40 {
			b.ffaSure = true
		}
	}

	b.practice = !b.sawOtherDeath && o.T > 20
	for _, s := range o.Sounds {
		if s.Kind == "death" && b.P.Sites && !b.practice {
			b.heardDeath(s.Bearing, s.Dist, o.T-s.Age)
		}
		switch s.Kind {
		case "gun", "laser", "laser_charge", "lightning", "mortar", "reload", "boost", "jet", "step":
			b.heardMech(s.Kind, s.Bearing, s.Dist, o.T-s.Age)
		}
		if b.e8Any() {
			b.e8Hear(s.Kind, s.Bearing, s.Dist, o.T-s.Age)
		}
		if b.hotOn() && (s.Kind == "gun" || s.Kind == "laser" || s.Kind == "lightning" || s.Kind == "mortar") {
			b.hotHear(s.Kind, s.Bearing, s.Dist, o.T-s.Age)
		}
		if b.P.EarP > 0 && s.Kind == "gun" && s.Dist != "far" {
			b.earHear(s.Bearing, s.Dist, o.T-s.Age)
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
	b.campHold = false
	b.decide(&act)
	if b.P.MoveJit != 0 && !b.ffaSure && (act.Move[0] != 0 || act.Move[1] != 0) {

		x := uint64(b.P.MoveJit)*0x9E3779B97F4A7C15 + uint64(o.Tick)*0xBF58476D1CE4E5B9 + uint64(int64(b.seed))*0x94D049BB133111EB + uint64(b.slot)
		x ^= x >> 30
		x *= 0xBF58476D1CE4E5B9
		x ^= x >> 27
		x *= 0x94D049BB133111EB
		x ^= x >> 31
		a := (float64(x%2001)/1000 - 1) * math.Pi / 180
		c, s := math.Cos(a), math.Sin(a)
		act.Move = proto.Vec2{act.Move[0]*c - act.Move[1]*s, act.Move[0]*s + act.Move[1]*c}
	}
	if b.P.Still || (b.campHold && b.P.Camp) {
		act.Move = proto.Vec2{}
		act.Jump = false
		act.Boost = false
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

func (b *Bot) DbgMode() string { return b.dbg }

func (b *Bot) DbgTarget() int { return b.targetID }

func (b *Bot) ffaAtStart() bool {
	if b.P.FFAAfter > 0 && b.wallOn {
		if now := time.Now().Unix(); now > b.P.FFAAfter && now < b.P.FFAAfter+40*86400 {
			return true
		}
	}
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

func (b *Bot) ffaByClock() bool {
	if len(b.P.FFA0) > 4 && b.P.FFA0[:4] == "wall" {
		if v, err := strconv.Atoi(b.P.FFA0[4:]); err == nil {
			return b.wallMed > float64(v)
		}
	}
	if len(b.P.FFA0) > 3 && b.P.FFA0[:3] == "gap" {

		if v, err := strconv.Atoi(b.P.FFA0[3:]); err == nil {
			return b.readyGap > float64(v)
		}
	}
	return false
}

func (b *Bot) probeStop() bool {
	if len(b.P.Probe) > 1 && b.P.Probe[0] == 't' {

		if v, err := strconv.Atoi(b.P.Probe[1:]); err == nil {
			return b.t >= float64(v)
		}
	}
	if len(b.P.Probe) > 2 && b.P.Probe[:2] == "dt" {

		if v, err := strconv.Atoi(b.P.Probe[2:]); err == nil {
			return b.wallMed > float64(v)
		}
	}
	if b.P.Probe == "tm" {

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

type effP struct {
	BossRatio, HardRatio, BossObjD, BossAvoidD, BossSeenD, PileBonus float64
	ExploreOld                                                       bool
	Calm                                                             bool
	ObjHide                                                          bool
	Watchdog                                                         bool
}

type noiseSrc struct {
	pos [2]float64
	t   float64
	far bool
}

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
			continue
		}
		keep = append(keep, n)
	}
	b.noise = append(keep, noiseSrc{pos: [2]float64{x, y}, t: at, far: far})
}

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

const Build = 4

func gapCode(ms float64) float64 {
	if ms <= 0 {
		return 0
	}
	k := math.Round(math.Log2(ms))
	return 0.03 * (math.Max(-4, math.Min(15, k)) + 5)
}

func wallCode(ms float64) float64 {
	if ms <= 0 {
		return 0.6
	}
	k := math.Round(2 * math.Log2(ms))
	return 0.03 * (math.Max(-4, math.Min(14, k)) + 5)
}
