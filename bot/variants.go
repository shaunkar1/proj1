package bot

import (
	"strconv"
	"strings"
)

// Variant returns a named parameter set (for A/B runs in the simulator).
func Variant(name string) Params {
	p := DefaultParams()
	p.Name = name
	if strings.HasPrefix(name, "prof:") {
		p = baseParams()
		p.Name = name
		ApplyProfile(&p, name[5:])
		return p
	}
	if strings.HasPrefix(name, "dancer") && len(name) > 6 && name != "dancernd" {
		if v, err := strconv.Atoi(name[6:]); err == nil {
			p.NoFire = true
			p.Aggressive = true
			p.EngageRatio = 1e9
			p.RetreatHP = 0
			p.PrefRangeGun = float64(v)
			return p
		}
	}
	switch name {
	case "", "default", "paw":
		p.Name = "paw"
	case "turretna":
		p.Still = true
		p.Aggressive = true
		p.EngageRatio = 1e9
		p.RetreatHP = 0
		p.NoAdapt = true
	case "noadapt":
		p.NoAdapt = true
	case "turret":
		p.Still = true
		p.Aggressive = true
		p.EngageRatio = 1e9
		p.RetreatHP = 0
	case "dancer", "dancer130", "dancer60", "dancer160":
		p.NoFire = true
		p.Aggressive = true
		p.EngageRatio = 1e9
		p.RetreatHP = 0
		p.PrefRangeGun = 95
		if name == "dancer130" {
			p.PrefRangeGun = 130
		}
		if name == "dancer160" {
			p.PrefRangeGun = 160
		}
		if name == "dancer60" {
			p.PrefRangeGun = 60
		}
	case "dancernd":
		p.NoFire = true
		p.Aggressive = true
		p.EngageRatio = 1e9
		p.RetreatHP = 0
		p.NoDodge = true
	case "patient":
		p.Endgame = 3
	case "nomortar":
		p.WeaponRank = map[string]float64{"lightning": 2.6, "laser": 3, "mortar": 0, "gun": 0.5}
	case "lasltnnm":
		p.DropGunForLtn = true
		p.WeaponRank = map[string]float64{"lightning": 2.6, "laser": 3, "mortar": 0, "gun": 0.5}
	case "atlike":
		// stand-in for the laser+lightning chasers seen in showdowns
		p.Aggressive = true
		p.DropGunForLtn = true
		p.HuntAfter = 0
		p.FixRange = 40
		p.RetreatHP = 0.15
		p.EngageRatio = 3
		p.NoPeek = true
	case "atgun":
		p.Aggressive = true
		p.HuntAfter = 0
		p.FixRange = 60
		p.RetreatHP = 0.15
		p.EngageRatio = 3
		p.NoPeek = true
	case "chase":
		p.ChaseHP = 0.5
		p.ChaseD = 55
	case "chase7":
		p.ChaseHP = 0.7
		p.ChaseD = 45
	case "nolascheck":
		p.NoLasCheck = true
	case "nolaserrush":
		p.NoLaserRush = true
	case "llkite":
		p.DropGunForLtn = true
		p.FixRange = 120
	case "llchase":
		p.DropGunForLtn = true
		p.FixRange = 120
		p.ChaseHP = 0.6
		p.ChaseD = 35
	case "noboss":
		p.BossRatio = 0
	case "boss25":
		p.BossRatio = 2.5
	case "boss14":
		p.BossRatio = 1.4
	case "peri":
		p.Periscope = 20
	case "peri30":
		p.Periscope = 30
	case "spin":
		p.SpinScan = true
	case "spinperi":
		p.SpinScan = true
		p.Periscope = 20
	case "nofarcover":
		p.NoFarCover = true
	case "boss15":
		p.BossRatio = 1.5
	case "ambush":
		p.Ambush, p.AmbushAng, p.AmbushRatio = 200, 1.4, 1.3
	case "ambush250":
		p.Ambush, p.AmbushAng, p.AmbushRatio = 250, 1.6, 1.6
	case "ambushe":
		p.Ambush, p.AmbushAng, p.AmbushRatio = 200, 1.4, 1.3
		p.EarlyEngage = 0.6
		p.EarlyAlive = 8
	case "nobiter":
		p.NoBiter = true
	case "sitesb15":
		p.Sites = true
		p.BossRatio = 1.5
	case "sitesb13":
		p.Sites = true
		p.BossRatio = 1.3
	case "boss13":
		p.BossRatio = 1.3
	case "boss115":
		p.BossRatio = 1.15
	case "boss10":
		p.BossRatio = 1.0
	case "snb13":
		p.Sites = true
		p.SiteFarVal = 0
		p.BossRatio = 1.3
	case "snb115":
		p.Sites = true
		p.SiteFarVal = 0
		p.BossRatio = 1.15
	case "boss085":
		p.BossRatio = 0.85
	case "boss07":
		p.BossRatio = 0.7
	case "snb10":
		p.Sites = true
		p.SiteFarVal = 0
		p.BossRatio = 1.0
	case "snb085":
		p.Sites = true
		p.SiteFarVal = 0
		p.BossRatio = 0.85
	case "b7":
		// previous defaults
		p.HardRatio = 0
		p.BossRatio = 2.0
		p.BossObjD, p.BossAvoidD, p.BossSeenD = 160, 150, 1e9
		p.Sites = false
		p.NoExplore = true
		p.RoofBold = false
	case "b10":
		p.NoExplore = true
	case "nosites":
		p.Sites = false
	case "idlex":
		p.NoExplore = false
		p.NoLaserRush = true
	case "idlexl":
		p.NoExplore = false
		p.NoLaserRush = true
		p.IdleXUnarmed = true
	case "hot10":
		p.BossHot = 1.0
	case "hot12":
		p.BossHot = 1.2
	case "hot15":
		p.BossHot = 1.5
	case "noroofbold":
		p.RoofBold = false
	case "tier15":
		p.HardRatio = 1.5
	case "tier13":
		p.HardRatio = 1.3
	case "tier18":
		p.HardRatio = 1.8
	case "tier15w":
		p.HardRatio = 1.5
		p.PeerAvoidD = 130
		p.PeerObjD = 110
	case "tier15n":
		p.HardRatio = 1.5
		p.PeerAvoidD = 60
		p.PeerObjD = 45
	case "b12":
		p.BossRatio, p.HardRatio = 0.85, 0
		p.BossObjD, p.BossAvoidD, p.BossSeenD = 160, 150, 1e9
	case "er12":
		p.EngageRatio = 1.2
	case "er14":
		p.EngageRatio = 1.4
	case "er08":
		p.EngageRatio = 0.8
	case "hard13":
		p.HardRatio = 1.3
	case "hard20":
		p.HardRatio = 2.0
	case "nopeer":
		p.BossRatio = 1.5
		p.HardRatio = 0
	case "evsn":
		p.EvadeSniper = true
	case "evsner12":
		p.EvadeSniper = true
		p.EngageRatio = 1.2
	case "exold":
		p.ExploreOld = true
	case "b13":
		p.BossRatio, p.HardRatio = 0.85, 1.5
		p.BossObjD, p.BossAvoidD, p.BossSeenD = 160, 150, 1e9
	case "b15wide":
		p.BossObjD, p.BossAvoidD, p.BossSeenD = 160, 150, 1e9
	case "b15r20":
		p.BossRatio = 2.0
	case "b15r13":
		p.BossRatio = 1.3
	case "nopile":
		p.PileBonus = 0
	case "pile50":
		p.PileBonus = 50
	case "nolegacy":
		p.NoLegacy = true
	case "ffa0":
		p.FFA0 = "always"
	case "ffa0nograb":
		p.FFA0 = "always"
		p.Grab = 0
		p.ScanFirst = false
	case "ffa0noscan":
		p.FFA0 = "always"
		p.ScanFirst = false
	case "ffa0grab70":
		p.FFA0 = "always"
		p.Grab = 70
	case "noopen":
		p.Open = false
	case "openlas":
		p.OpenBusyD = 0
	case "openbusy":
		p.NoOpenLas = true
	case "open25":
		p.OpenRatio = 2.5
	case "ffa0open":
		p.FFA0 = "always"
	case "ffa0noopen":
		p.FFA0 = "always"
		p.Open = false
	case "nograb":
		p.Grab = 0
		p.ScanFirst = false
	case "sites":
		p.Sites = true
	case "sites2":
		p.Sites = true
		p.SiteFarVal = 190
		p.SiteFar = 280
	case "sitesnear":
		p.Sites = true
		p.SiteFarVal = 0
	case "snipe":
		p.Snipe = 265
	case "preempt":
		p.Preempt = 200
	case "opener":
		p.Ambush, p.AmbushAng, p.AmbushRatio = 220, 1.4, 1.3
		p.Snipe = 265
	case "opener3":
		p.Ambush, p.AmbushAng, p.AmbushRatio = 220, 1.4, 1.3
		p.Snipe = 265
		p.Preempt = 200
	case "early06":
		p.EarlyEngage = 0.6
		p.EarlyAlive = 8
	case "early06b15":
		p.EarlyEngage = 0.6
		p.EarlyAlive = 8
		p.BossRatio = 1.5
	case "early05a6":
		p.EarlyEngage = 0.5
		p.EarlyAlive = 6
	case "prey05":
		p.PreyRatio = 0.5
	case "prey03":
		p.PreyRatio = 0.3
	case "prey05e":
		p.PreyRatio = 0.5
		p.EarlyEngage = 0.6
		p.EarlyAlive = 8
	case "oldfarm":
		p.OldArrive = true
		p.OldReload = true
	case "npc0":
		p.NPCMemory = false
		p.NPCValue = 140
	case "npc200":
		p.NPCValue = 200
	case "open1":
		p.OpenCost = 1
	case "open2":
		p.OpenCost = 2.5
	case "noretcubes":
		p.NoRetreatCubes = true
	case "explore":
		p.NoExplore = false
	case "noexplore":
		p.NoExplore = true
	case "survivor":
		p.Endgame = 16
		p.HuntPassive = false
	case "patient6":
		p.Endgame = 6
	case "patient4":
		p.Endgame = 4
	case "nododge":
		p.NoDodge = true
	case "lasltn":
		p.DropGunForLtn = true
	case "pracaggro":
		p.PracAggro = true
	case "pracpeek":
		p.PracNoPeek = false
		p.PracNearFarm = false
	case "nopeek":
		p.NoPeek = true
	case "aggronp":
		p.Aggressive = true
		p.NoPeek = true
	case "noroof":
		p.Roof = false
	case "r95":
		p.PrefRangeGun = 95
	case "ltn":
		p.WeaponRank = map[string]float64{"lightning": 3, "laser": 2, "mortar": 1, "gun": 0.5}
	case "r60":
		p.PrefRangeGun = 60
	case "aggro":
		p.Aggressive = true
	case "passive":
		p.HuntPassive = false
	case "nofarm":
		p.FarmNPC = false
	case "nodrops":
		p.UseDrops = false
	case "bold":
		p.EngageRatio = 1.4
		p.RetreatHP = 0.25
	case "timid":
		p.EngageRatio = 0.7
		p.RetreatHP = 0.45
	}
	return p
}
