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
