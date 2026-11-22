package bot

import (
	"strconv"
	"strings"
)

// Variant returns a named parameter set (for A/B runs in the simulator).
func Variant(name string) Params {
	p := DefaultParams()
	p.Name = name
	if strings.HasPrefix(name, "tk") {
		// short names for practice-profile tickets, for job files
		m := map[string]string{"tk0": "", "tko": "+o", "tko1": "+o1", "tko2": "+o2", "tko4": "+o4", "tko5": "+o5", "tko6": "+o6", "tko7": "+o7", "tko8": "+o8", "tko9": "+o9", "tko10": "+o10", "tko11": "+o11", "tko12": "+o12", "tko13": "+o13", "tko14": "+o14", "tko15": "+o15"}
		if suf, ok := m[name]; ok {
			p = baseParams()
			p.Name = name
			ApplyProfile(&p, "f75a220+g+s+d100+x3+h35"+suf)
			return p
		}
	}
	if strings.HasPrefix(name, "P_") || strings.HasPrefix(name, "Q_") {
		// practice-profile sweeps: P_ = the d100 ticket base, Q_ = without the supply-drop rush; _ separates modifiers
		base := "f75a220+g+s+d100"
		if name[0] == 'Q' {
			base = "f75a220+g+s"
		}
		p = baseParams()
		p.Name = name
		if rest := strings.Trim(name[2:], "_"); rest != "" {
			base += "+" + strings.ReplaceAll(rest, "_", "+")
		}
		ApplyProfile(&p, base)
		return p
	}
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
	case "qe1":
		p.FFA0 = "always"
		p.QuietAdvD = 50
	case "qe2":
		p.FFA0 = "always"
		p.QuietAdvD = 50
		p.QuietEngage = 0.7
	case "qe3":
		p.FFA0 = "always"
		p.FFAAdvD = 50
	case "qe4":
		p.FFA0 = "always"
		p.FFAAdvD = 50
		p.QuietEngage = 0.7
	case "qe5":
		p.FFA0 = "always"
		p.QuietAdvD = 25
		p.FFAAdvD = 80
	case "si30":
		p.FFA0 = "always"
		p.ShyInside = 30
	case "si60":
		p.FFA0 = "always"
		p.ShyInside = 60
	case "zl20":
		p.FFA0 = "always"
		p.ZoneLead = 20
	case "zl40":
		p.FFA0 = "always"
		p.ZoneLead = 40
	case "si30zl20":
		p.FFA0 = "always"
		p.ShyInside = 30
		p.ZoneLead = 20
	case "noadv":
		p.FFA0 = "always"
		p.QuietAdvD = 0
	case "xf7":
		p.FFA0 = "always"
		p.CrossAlive, p.CrossD = 7, 220
	case "xf16":
		p.FFA0 = "always"
		p.CrossAlive, p.CrossD = 16, 220
	case "xf7d150":
		p.FFA0 = "always"
		p.CrossAlive, p.CrossD = 7, 150
	case "xf1":
		p.FFA0 = "always"
		p.CrossAlive, p.CrossD = 7, 220
		p.QuietCap = 360
	case "qc360":
		p.FFA0 = "always"
		p.QuietCap = 360
	case "sm16":
		p.FFA0 = "always"
		p.ShyMem = 16
	case "sm24":
		p.FFA0 = "always"
		p.ShyMem = 24
	case "fl1":
		p.FFA0 = "always"
		p.FleeLaser = 1
	case "fl2":
		p.FFA0 = "always"
		p.FleeLaser = 2
	case "ffa0old":
		p.FFA0 = "always"
		p.EndgameFFA, p.QuietCap = 0, 240
	case "ffa0e":
		p.FFA0 = "always"
		p.QuietCap = 240
	case "ffa0c":
		p.FFA0 = "always"
		p.EndgameFFA = 0
	case "ffa0e6":
		p.FFA0 = "always"
		p.EndgameFFA = 6
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
	case "nokeep":
		p.KeepAwayD = 0
	case "keep130":
		p.KeepAwayD = 130
	case "keep60":
		p.KeepAwayD = 60
	case "keepall":
		p.KeepAwayRatio = 0.5
	case "noquiet":
		p.QuietT = 0
	case "quiet75":
		p.QuietT = 75
	case "quiet30":
		p.QuietT = 30
	case "noshy":
		p.ShyD, p.ShyRoam = 0, false
	case "shynoroam":
		p.ShyRoam = false
	case "shy0":
		p.QuietAlways = true
	case "shy0soft":
		p.QuietAlways, p.NoShyRoam0, p.ShySoft0 = true, true, 0.8
	case "shy0soft2":
		p.QuietAlways, p.NoShyRoam0, p.ShySoft0 = true, true, 1.6
	case "shy0o5":
		p.QuietAlways, p.ShyWeapon0, p.NoShyRoam0 = true, true, true
	case "shy0o4":
		p.QuietAlways, p.ShyWeapon0 = true, true
	case "shy0o1":
		p.QuietAlways, p.NoShyRoam0 = true, true
	case "shy0t8":
		p.QuietAlways, p.ShyT0 = true, 8
	case "shy0t12":
		p.QuietAlways, p.ShyT0 = true, 12
	case "shy0t12w":
		p.QuietAlways, p.ShyT0, p.ShyWeapon0 = true, 12, true
	case "shy0o13":
		p.QuietAlways, p.ShySoft0, p.ShyMove0 = true, 0.8, true
	case "shy0o14":
		p.QuietAlways, p.ShyMove0 = true, true
	case "shy0o15":
		p.QuietAlways, p.ShyMove0, p.ShyWeapon0 = true, true, true
	case "ffa0na":
		p.FFA0, p.QuietNoAim = "always", true
	case "ffa0q60":
		p.FFA0, p.QuietT = "always", 60
	case "ffa0na60":
		p.FFA0, p.QuietNoAim, p.QuietT = "always", true, 60
	case "noaim":
		p.QuietNoAim = true
	case "nobreak":
		p.NoBreak = true
	case "noaim60":
		p.QuietNoAim, p.QuietT = true, 60
	case "noaim90":
		p.QuietNoAim, p.QuietT = true, 90
	case "ffa0na90":
		p.FFA0, p.QuietNoAim, p.QuietT = "always", true, 90
	case "ffa0na120":
		p.FFA0, p.QuietNoAim, p.QuietT = "always", true, 120
	case "ffa0q90":
		p.FFA0, p.QuietT = "always", 90
	case "q45":
		p.QuietT, p.QuietNoAim = 45, false
	case "q120":
		p.QuietT = 120
	case "q150":
		p.QuietT = 150
	case "q200":
		p.QuietT = 200
	case "qa8":
		p.QuietAlive = 8
	case "qa6":
		p.QuietAlive = 6
	case "qa10":
		p.QuietAlive = 10
	case "q90aim":
		p.QuietNoAim = false
	case "qa0":
		p.QuietAlive = 0
	case "qa5":
		p.QuietAlive = 5
	case "qa4":
		p.QuietAlive = 4
	case "qa3":
		p.QuietAlive = 3
	case "qc300":
		p.QuietCap = 300
	case "qc180":
		p.QuietCap = 180
	case "qa4c300":
		p.QuietAlive, p.QuietCap = 4, 300
	case "lk":
		p.LtnKite = true
	case "nobossfar":
		p.BossFarD = 0
	case "bossfar110":
		p.BossFarD = 110
	case "shy0n12":
		p.QuietAlways = true
		p.ShyNear0 = 12
	case "shy0n70":
		p.QuietAlways = true
		p.ShyNear0 = 70
	case "shy0q60":
		p.QuietAlways = true
		p.QuietT = 60
	case "quiet60":
		p.QuietT = 60
	case "shy200":
		p.ShyD, p.ShyFarD = 200, 230
	case "quiet0":
		p.QuietAlways = true
	case "quiet0t75":
		p.QuietAlways = true
		p.QuietT = 75
	case "quiethide":
		p.QuietHide = true
		p.QuietObjD = 160
	case "quiet160":
		p.QuietObjD = 160
	case "quiet80":
		p.QuietObjD = 80
	case "noffadrop":
		p.FFADrop = 0
	case "ffadropall":
		p.FFADrop = 1.01
	case "ffadrop70":
		p.FFADrop = 0.7
	case "ffadropw0":
		p.FFADropWait = 0
	case "ffadropw20":
		p.FFADropWait = 20
	case "nowatchdog":
		p.NoWatchdog = true
	case "calm":
		p.Calm = true
	case "noobjhide":
		p.ObjHide = false
	case "calmnohide":
		p.Calm = true
		p.ObjHide = false
	case "noevsn":
		p.EvadeSniper = false
	case "noopen":
		p.Open = false
	case "openlas":
		p.NoOpenLas = false
	case "openbusy":
		p.NoOpenLas = true
	case "open25":
		p.OpenRatio = 2.5
	case "ffa0open":
		p.FFA0 = "always"
	case "ffa0noopen":
		p.FFA0 = "always"
		p.Open = false
	case "nograbcube":
		p.GrabCube = 0
	case "grabcube28":
		p.GrabCube = 28
	case "nograb":
		p.GrabCube = 0
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
	case "survq":
		p.Endgame = 16
		p.HuntPassive = false
		p.QuietCap = 300
	case "surv":
		p.Endgame = 16
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
