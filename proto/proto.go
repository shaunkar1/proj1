package proto

import "encoding/json"

type Vec2 = [2]float64
type Vec3 = [3]float64

type WeaponStats struct {
	Damage       float64 `json:"damage"`
	Cooldown     float64 `json:"cooldown"`
	Speed        float64 `json:"speed"`
	Range        float64 `json:"range"`
	SpreadDeg    float64 `json:"spread_deg"`
	Mag          float64 `json:"mag"`
	Reload       float64 `json:"reload"`
	Splash       float64 `json:"splash"`
	SplashRadius float64 `json:"splash_radius"`
	SplashEdge   float64 `json:"splash_edge"`
	Gravity      float64 `json:"gravity"`
	TTL          float64 `json:"ttl"`
	Knock        float64 `json:"knock"`
	KnockUp      float64 `json:"knock_up"`
	Drain        float64 `json:"drain"`
	Charge       float64 `json:"charge"`
	MinCharge    float64 `json:"min_charge"`
	MinFrac      float64 `json:"min_frac"`
	HoldS        float64 `json:"hold_s"`
	Pierce       float64 `json:"pierce"`
	ChargeTurn   float64 `json:"charge_turn_mult"`
	Cookoff      float64 `json:"cookoff"`
	ConeDeg      float64 `json:"cone_deg"`
	Chain        float64 `json:"chain"`
	ChainRange   float64 `json:"chain_range"`
	ChainFalloff float64 `json:"chain_falloff"`
	Energy       float64 `json:"energy"`
	ShortS       float64 `json:"short_s"`
}

type MechStats struct {
	Radius           float64 `json:"radius"`
	Height           float64 `json:"height"`
	EyeHeight        float64 `json:"eye_height"`
	MuzzleHeight     float64 `json:"muzzle_height"`
	HP               float64 `json:"hp"`
	WalkSpeed        float64 `json:"walk_speed"`
	BoostSpeed       float64 `json:"boost_speed"`
	GroundAccel      float64 `json:"ground_accel"`
	AirAccel         float64 `json:"air_accel"`
	Gravity          float64 `json:"gravity"`
	JumpSpeed        float64 `json:"jump_speed"`
	JetAccel         float64 `json:"jet_accel"`
	MaxRiseSpeed     float64 `json:"max_rise_speed"`
	StepUp           float64 `json:"step_up"`
	MaxSlopeDeg      float64 `json:"max_slope_deg"`
	Energy           float64 `json:"energy"`
	EnergyRegen      float64 `json:"energy_regen"`
	EnergyRegenDelay float64 `json:"energy_regen_delay"`
	BoostDrain       float64 `json:"boost_drain"`
	JetDrain         float64 `json:"jet_drain"`
	TurnRate         float64 `json:"turn_rate"`
	PitchRate        float64 `json:"pitch_rate"`
	MinPitch         float64 `json:"min_pitch"`
	MaxPitch         float64 `json:"max_pitch"`
	RegenDelay       float64 `json:"regen_delay"`
	RegenPerS        float64 `json:"regen_per_s"`
	ChargeMoveMult   float64 `json:"charge_move_mult"`
	SelfDamageMult   float64 `json:"self_damage_mult"`
	Armor            float64 `json:"armor"`
	RearMult         float64 `json:"rear_mult"`
	FrontArcDeg      float64 `json:"front_arc_deg"`
	NearRangeBonus   float64 `json:"near_range_bonus"`
	VisionBonus      float64 `json:"vision_bonus"`
	Hearing          float64 `json:"hearing"`
}

type ZonePhase struct {
	HoldS   float64 `json:"hold_s"`
	ShrinkS float64 `json:"shrink_s"`
	Radius  float64 `json:"radius"`
	DPS     float64 `json:"dps"`
}

type Effect struct {
	Path string   `json:"path"`
	Mul  *float64 `json:"mul,omitempty"`
	Add  *float64 `json:"add,omitempty"`
}

type UpgradeDef struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Desc     string   `json:"desc"`
	Requires string   `json:"requires,omitempty"`
	Max      float64  `json:"max"`
	Effects  []Effect `json:"effects"`
}

type Config struct {
	Match struct {
		TickHz             float64 `json:"tick_hz"`
		DecideEvery        float64 `json:"decide_every"`
		DurationS          float64 `json:"duration_s"`
		Mechs              float64 `json:"mechs"`
		SpawnMinSeparation float64 `json:"spawn_min_separation"`
		SpawnZoneFraction  float64 `json:"spawn_zone_fraction"`
		ArenaMargin        float64 `json:"arena_margin"`
	} `json:"match"`
	Bot struct {
		TickBudgetMs  float64 `json:"tick_budget_ms"`
		HelloBudgetMs float64 `json:"hello_budget_ms"`
		CoastS        float64 `json:"coast_s"`
		OfferTimeoutS float64 `json:"offer_timeout_s"`
		MaxLineBytes  float64 `json:"max_line_bytes"`
	} `json:"bot"`
	Mech   MechStats `json:"mech"`
	Vision struct {
		Range        float64 `json:"range"`
		FovDeg       float64 `json:"fov_deg"`
		NearRange    float64 `json:"near_range"`
		CanopyBlocks bool    `json:"canopy_blocks"`
		ShowEnemyHP  bool    `json:"show_enemy_hp"`
		MemoryS      float64 `json:"memory_s"`
	} `json:"vision"`
	Sound struct {
		BearingNoiseDeg float64            `json:"bearing_noise_deg"`
		Near            float64            `json:"near"`
		Mid             float64            `json:"mid"`
		StepSpeed       float64            `json:"step_speed"`
		StepEvery       float64            `json:"step_every"`
		Range           map[string]float64 `json:"range"`
	} `json:"sound"`
	Weapons    map[string]WeaponStats `json:"weapons"`
	Projectile struct {
		GunRadius float64 `json:"gun_radius"`
		NpcRadius float64 `json:"npc_radius"`
	} `json:"projectile"`
	Cubes struct {
		PickupRadius      float64   `json:"pickup_radius"`
		HPPerCube         float64   `json:"hp_per_cube"`
		DamagePerCube     float64   `json:"damage_per_cube"`
		StatCap           float64   `json:"stat_cap"`
		DeathDropExtra    float64   `json:"death_drop_extra"`
		SizePerCube       float64   `json:"size_per_cube"`
		UpgradeThresholds []float64 `json:"upgrade_thresholds"`
	} `json:"cubes"`
	Crates struct {
		Count        float64 `json:"count"`
		RoofFraction float64 `json:"roof_fraction"`
		HP           float64 `json:"hp"`
		Radius       float64 `json:"radius"`
		Height       float64 `json:"height"`
		Cubes        float64 `json:"cubes"`
	} `json:"crates"`
	Loot struct {
		PickupRadius      float64            `json:"pickup_radius"`
		EquipS            float64            `json:"equip_s"`
		FloorCount        float64            `json:"floor_count"`
		FloorRoofFraction float64            `json:"floor_roof_fraction"`
		CrateWeaponChance float64            `json:"crate_weapon_chance"`
		DeathDrop         float64            `json:"death_drop"`
		Weights           map[string]float64 `json:"weights"`
	} `json:"loot"`
	NPC struct {
		Count           float64 `json:"count"`
		HP              float64 `json:"hp"`
		Radius          float64 `json:"radius"`
		Height          float64 `json:"height"`
		Speed           float64 `json:"speed"`
		Leash           float64 `json:"leash"`
		ChaseLeash      float64 `json:"chase_leash"`
		AggroRange      float64 `json:"aggro_range"`
		DeaggroRange    float64 `json:"deaggro_range"`
		Damage          float64 `json:"damage"`
		Cooldown        float64 `json:"cooldown"`
		ProjectileSpeed float64 `json:"projectile_speed"`
		SpreadDeg       float64 `json:"spread_deg"`
		Cubes           float64 `json:"cubes"`
	} `json:"npc"`
	Drops struct {
		Times    []float64 `json:"times"`
		FallS    float64   `json:"fall_s"`
		Radius   float64   `json:"radius"`
		CaptureS float64   `json:"capture_s"`
		Cubes    float64   `json:"cubes"`
		Repair   float64   `json:"repair"`
		Spread   float64   `json:"spread"`
	} `json:"drops"`
	Zone struct {
		StartCenter []float64   `json:"start_center"`
		StartRadius float64     `json:"start_radius"`
		Phases      []ZonePhase `json:"phases"`
	} `json:"zone"`
	Upgrades []UpgradeDef `json:"upgrades"`
}

type Hello struct {
	Type     string  `json:"type"`
	Protocol string  `json:"protocol"`
	Slot     int     `json:"slot"`
	Seed     float64 `json:"seed"`
	Map      struct {
		Dir   string   `json:"dir"`
		Files []string `json:"files"`
	} `json:"map"`
	Config Config `json:"config"`
	Raw    []byte `json:"-"`
}

type WeaponObs struct {
	Kind     string      `json:"kind"`
	Cooldown float64     `json:"cooldown"`
	Ammo     float64     `json:"ammo"`
	Reload   float64     `json:"reload"`
	Charge   float64     `json:"charge"`
	Stats    WeaponStats `json:"stats"`
}

type SelfObs struct {
	ID        int          `json:"id"`
	Pos       Vec3         `json:"pos"`
	Vel       Vec3         `json:"vel"`
	Aim       Vec2         `json:"aim"`
	HP        float64      `json:"hp"`
	MaxHP     float64      `json:"max_hp"`
	Energy    float64      `json:"energy"`
	MaxEnergy float64      `json:"max_energy"`
	Grounded  bool         `json:"grounded"`
	Cubes     float64      `json:"cubes"`
	Upgrades  []string     `json:"upgrades"`
	Weapons   []*WeaponObs `json:"weapons"`
	InZone    bool         `json:"in_zone"`
	Stats     MechStats    `json:"stats"`
}

type ActorObs struct {
	ID       int      `json:"id"`
	Kind     string   `json:"kind"`
	Pos      Vec3     `json:"pos"`
	Vel      Vec3     `json:"vel"`
	Aim      Vec2     `json:"aim"`
	HP       float64  `json:"hp"`
	MaxHP    float64  `json:"max_hp"`
	Cubes    float64  `json:"cubes"`
	Weapons  []string `json:"weapons"`
	Charging bool     `json:"charging"`
}

type ProjObs struct {
	ID   int    `json:"id"`
	Kind string `json:"kind"`
	Pos  Vec3   `json:"pos"`
	Vel  Vec3   `json:"vel"`
}

type ItemObs struct {
	ID  int     `json:"id"`
	Pos Vec3    `json:"pos"`
	HP  float64 `json:"hp"`
}

type PickupObs struct {
	ID   int    `json:"id"`
	Kind string `json:"kind"`
	Pos  Vec3   `json:"pos"`
}

type SoundObs struct {
	Kind    string  `json:"kind"`
	Bearing float64 `json:"bearing"`
	Dist    string  `json:"dist"`
	Age     float64 `json:"age"`
}

type BeamObs struct {
	By   int     `json:"by"`
	Kind string  `json:"kind"`
	Path []Vec3  `json:"path"`
	Age  float64 `json:"age"`
}

type Event struct {
	Type    string          `json:"type"`
	Amount  float64         `json:"amount"`
	Weapon  string          `json:"weapon"`
	Bearing *float64        `json:"bearing"`
	By      *int            `json:"by"`
	Rear    bool            `json:"rear"`
	Target  int             `json:"target"`
	Effect  string          `json:"effect"`
	ID      json.RawMessage `json:"id"`
	Cubes   float64         `json:"cubes"`
	Kind    string          `json:"kind"`
	Slot    int             `json:"slot"`
	Reason  string          `json:"reason"`
}

type DropObs struct {
	ID       int     `json:"id"`
	Pos      Vec3    `json:"pos"`
	Radius   float64 `json:"radius"`
	LandsIn  float64 `json:"lands_in"`
	CaptureS float64 `json:"capture_s"`
	Progress float64 `json:"progress"`
	Holder   *int    `json:"holder"`
}

type ZoneObs struct {
	Center      Vec2    `json:"center"`
	Radius      float64 `json:"radius"`
	NextCenter  Vec2    `json:"next_center"`
	NextRadius  float64 `json:"next_radius"`
	ShrinkStart float64 `json:"shrink_start"`
	ShrinkEnd   float64 `json:"shrink_end"`
	DPS         float64 `json:"dps"`
}

type OfferChoice struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Desc string `json:"desc"`
}

type Offer struct {
	ID      int           `json:"id"`
	Expires float64       `json:"expires"`
	Choices []OfferChoice `json:"choices"`
}

type Obs struct {
	Type        string      `json:"type"`
	Tick        int         `json:"tick"`
	T           float64     `json:"t"`
	Alive       int         `json:"alive"`
	Self        SelfObs     `json:"self"`
	Mechs       []ActorObs  `json:"mechs"`
	Projectiles []ProjObs   `json:"projectiles"`
	Crates      []ItemObs   `json:"crates"`
	Cubes       []ItemObs   `json:"cubes"`
	Pickups     []PickupObs `json:"pickups"`
	Sounds      []SoundObs  `json:"sounds"`
	Beams       []BeamObs   `json:"beams"`
	Events      []Event     `json:"events"`
	Drops       []DropObs   `json:"drops"`
	Zone        ZoneObs     `json:"zone"`
	Offer       *Offer      `json:"offer"`
}

type Action struct {
	Tick   int     `json:"tick"`
	Move   Vec2    `json:"move"`
	Boost  bool    `json:"boost"`
	Jump   bool    `json:"jump"`
	Aim    Vec2    `json:"aim"`
	Fire   [2]bool `json:"fire"`
	Reload bool    `json:"reload"`
	Pick   *int    `json:"pick,omitempty"`
	Take   *int    `json:"take,omitempty"`
	Drop   *int    `json:"drop,omitempty"`
	Debug  *Debug  `json:"debug,omitempty"`
}

type Debug struct {
	Label string `json:"label,omitempty"`
}
