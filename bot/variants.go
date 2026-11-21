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
	case "explore":
		p.NoExplore = false
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
