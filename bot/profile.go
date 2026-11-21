package bot

import (
	"strconv"
	"strings"
)

// Profile selects the squad-match (practice) tuning compiled into the bot.
const Profile = "f75a220+g+s+d100+x9"

// ApplyProfile adjusts only the parameters used in squad matches, so the
// free-for-all behaviour is the same under every profile.
func ApplyProfile(p *Params, name string) {
	if len(name) > 6 && name[:6] == "probe:" {
		p.Probe = name[6:]
		name = "f75a220+g+s"
	}
	var mods []string
	if i := strings.IndexByte(name, '+'); i >= 0 {
		mods = strings.Split(name[i+1:], "+")
		name = name[:i]
	}
	defer func() {
		for _, m := range mods {
			if m == "" {
				continue
			}
			v, _ := strconv.ParseFloat(m[1:], 64)
			switch {
			case m == "g":
				p.GrabAlways = true
			case m == "s":
				p.ScanAlways = true
			case m == "o":
				p.QuietAlways = true
			case m == "o4":
				p.QuietAlways, p.ShyWeapon0 = true, true
			case m == "o5":
				p.QuietAlways, p.ShyWeapon0, p.NoShyRoam0 = true, true, true
			case m == "o6":
				p.QuietAlways, p.NoShyRoam0, p.ShyNear0 = true, true, 1e9
			case m == "o7":
				p.QuietAlways, p.ShyNear0 = true, 1e9
			case m == "o8":
				p.QuietAlways, p.NoShyRoam0, p.ShySoft0 = true, true, 0.8
			case m == "o9":
				p.QuietAlways, p.NoShyRoam0, p.ShySoft0 = true, true, 1.6
			case m == "o10":
				p.QuietAlways, p.ShyT0 = true, 8
			case m == "o11":
				p.QuietAlways, p.ShyT0 = true, 12
			case m == "o12":
				p.QuietAlways, p.ShyT0, p.ShyWeapon0 = true, 12, true
			case m == "o13":
				p.QuietAlways, p.ShySoft0, p.ShyMove0 = true, 0.8, true
			case m == "o14":
				p.QuietAlways, p.ShyMove0 = true, true
			case m == "o15":
				p.QuietAlways, p.ShyMove0, p.ShyWeapon0 = true, true, true
			case len(m) > 2 && m[:2] == "zt":
				if t, err := strconv.ParseInt(m[2:], 10, 64); err == nil {
					p.FFAAfter = t
				}
			case len(m) > 2 && m[:2] == "zg":
				p.FFA0 = "gap" + m[2:]
			case len(m) > 2 && m[:2] == "zw":
				p.FFA0 = "wall" + m[2:]
			case m == "o1":
				p.QuietAlways, p.NoShyRoam0 = true, true
			case m == "o2":
				p.QuietAlways, p.NoShyRoam0, p.ShyNear0 = true, true, 70
			case m == "o3":
				p.QuietAlways, p.ShyNear0 = true, 70
			case m == "ul":
				p.PracUpgradePrio = ltnFirst
			case m == "ud":
				p.PracUpgradePrio = dmgFirst
			case m == "ltn":
				p.PracLtnRank = 3.4
			case m[0] == 'p':
				p.PracRange = v
			case m[0] == 'q':
				p.PracDropD = v
			case m[0] == 'd':
				p.PracDrop = v / 100
			case m[0] == 'x':
				p.RngSalt = int(v)
			case m[0] == 't':
				p.PracFarmT = v
			case m[0] == 'c':
				p.PracFarmCubes = int(v)
			case m[0] == 'a':
				p.PracAggroD = v
			case m[0] == 'h':
				p.PracAggroHP = v / 100
			case m[0] == 'r':
				p.PracRetreatHP = v / 100
			}
		}
	}()
	switch name {
	case "base":
	case "aggro":
		p.PracAggro = true
	case "berserk":
		p.PracAggro = true
		p.PracBerserk = true
		p.PracRetreatHP = 0
	case "noret":
		p.PracAggro = true
		p.PracRetreatHP = 0
	case "lowret":
		p.PracAggro = true
		p.PracRetreatHP = 0.2
	case "farm8":
		p.PracAggro = true
		p.PracFarmCubes = 8
		p.PracFarmT = 110
	case "farm5":
		p.PracAggro = true
		p.PracFarmCubes = 5
		p.PracFarmT = 80
	case "aggropeek":
		p.PracAggro = true
		p.PracNoPeek = false
	case "aggrofar":
		p.PracAggro = true
		p.PracNearFarm = false
	case "dmg":
		p.PracAggro = true
		p.PracUpgradePrio = dmgFirst
	case "berserkdmg":
		p.PracAggro = true
		p.PracBerserk = true
		p.PracRetreatHP = 0
		p.PracUpgradePrio = dmgFirst
	case "lasltn":
		p.PracAggro = true
		p.PracLasLtn = true
	case "berserkltn":
		p.PracAggro = true
		p.PracBerserk = true
		p.PracRetreatHP = 0
		p.PracLasLtn = true
	case "rush45":
		p.PracAggro = true
		p.PracBerserk = true
		p.PracRetreatHP = 0
		p.PracRange = 45
	case "rush70":
		p.PracAggro = true
		p.PracBerserk = true
		p.PracRetreatHP = 0
		p.PracRange = 70
	case "rushltn":
		p.PracAggro = true
		p.PracBerserk = true
		p.PracRetreatHP = 0
		p.PracRange = 38
		p.PracLasLtn = true
	case "probe1", "probe2", "probe3", "probe4", "probe5", "probe6":
		p.PracAggro = true
		p.PracStopKills = int(name[5] - '0')
	case "kite200":
		p.PracAggro = true
		p.PracBerserk = true
		p.PracRetreatHP = 0
		p.PracRange = 200
	case "kite150":
		p.PracAggro = true
		p.PracBerserk = true
		p.PracRetreatHP = 0
		p.PracRange = 150
	case "farm4":
		p.PracAggro = true
		p.PracFarmCubes = 4
		p.PracFarmT = 50
	case "farm6":
		p.PracAggro = true
		p.PracFarmCubes = 6
		p.PracFarmT = 75
	case "farm8b":
		p.PracAggro = true
		p.PracFarmCubes = 8
		p.PracFarmT = 100
	case "farm10":
		p.PracAggro = true
		p.PracFarmCubes = 10
		p.PracFarmT = 130
	case "farm4t70":
		p.PracAggro = true
		p.PracFarmCubes = 4
		p.PracFarmT = 70
	case "farm4base":
		p.PracFarmCubes = 4
		p.PracFarmT = 50
	case "farm4dmg":
		p.PracAggro = true
		p.PracFarmCubes = 4
		p.PracFarmT = 50
		p.PracUpgradePrio = dmgFirst
	case "farm4kite":
		p.PracAggro = true
		p.PracFarmCubes = 4
		p.PracFarmT = 50
		p.PracRange = 150
	case "farm15":
		p.PracAggro = true
		p.PracFarmCubes = 15
		p.PracFarmT = 170
	case "farm21":
		p.PracAggro = true
		p.PracFarmCubes = 21
		p.PracFarmT = 220
	case "farm6base":
		p.PracFarmCubes = 6
		p.PracFarmT = 75
	case "farm6t110":
		p.PracAggro = true
		p.PracFarmCubes = 6
		p.PracFarmT = 110
	case "farm10t75":
		p.PracAggro = true
		p.PracFarmCubes = 10
		p.PracFarmT = 75
	case "farmT60":
		p.PracAggro = true
		p.PracFarmCubes = 99
		p.PracFarmT = 60
	case "farmT75":
		p.PracAggro = true
		p.PracFarmCubes = 99
		p.PracFarmT = 75
	case "farmT90":
		p.PracAggro = true
		p.PracFarmCubes = 99
		p.PracFarmT = 90
	case "farmT105":
		p.PracAggro = true
		p.PracFarmCubes = 99
		p.PracFarmT = 105
	case "farm10t75base":
		p.PracFarmCubes = 10
		p.PracFarmT = 75
	case "farm10t75dmg":
		p.PracAggro = true
		p.PracFarmCubes = 10
		p.PracFarmT = 75
		p.PracUpgradePrio = dmgFirst
	case "f75kite":
		p.PracAggro = true
		p.PracFarmCubes = 10
		p.PracFarmT = 75
		p.PracRange = 170
	case "f75kite130":
		p.PracAggro = true
		p.PracFarmCubes = 10
		p.PracFarmT = 75
		p.PracRange = 130
	case "f75ret50":
		p.PracAggro = true
		p.PracFarmCubes = 10
		p.PracFarmT = 75
		p.PracRetreatHP = 0.5
	case "f75ret20":
		p.PracAggro = true
		p.PracFarmCubes = 10
		p.PracFarmT = 75
		p.PracRetreatHP = 0.2
	case "f75a220":
		p.PracAggro = true
		p.PracFarmCubes = 10
		p.PracFarmT = 75
		p.PracAggroD = 220
		p.PracAggroHP = 0.45
	case "f75peek":
		p.PracAggro = true
		p.PracFarmCubes = 10
		p.PracFarmT = 75
		p.PracNoPeek = false
	case "f75far":
		p.PracAggro = true
		p.PracFarmCubes = 10
		p.PracFarmT = 75
		p.PracNearFarm = false
	case "f60base":
		p.PracFarmCubes = 10
		p.PracFarmT = 60
	case "f90base":
		p.PracFarmCubes = 10
		p.PracFarmT = 90
	case "f75basefar":
		p.PracFarmCubes = 10
		p.PracFarmT = 75
		p.PracNearFarm = false
	case "f75bers":
		p.PracAggro = true
		p.PracBerserk = true
		p.PracFarmCubes = 10
		p.PracFarmT = 75
	case "f75a300":
		p.PracAggro = true
		p.PracFarmCubes = 10
		p.PracFarmT = 75
		p.PracAggroD = 300
		p.PracAggroHP = 0.5
	case "f75a110":
		p.PracAggro = true
		p.PracFarmCubes = 10
		p.PracFarmT = 75
		p.PracAggroD = 110
	case "f75hp80":
		p.PracAggro = true
		p.PracFarmCubes = 10
		p.PracFarmT = 75
		p.PracAggroHP = 0.8
	case "f60a220":
		p.PracAggro = true
		p.PracFarmCubes = 10
		p.PracFarmT = 60
		p.PracAggroD = 220
		p.PracAggroHP = 0.45
	case "f90a220":
		p.PracAggro = true
		p.PracFarmCubes = 10
		p.PracFarmT = 90
		p.PracAggroD = 220
		p.PracAggroHP = 0.45
	case "f75a220r20":
		p.PracAggro = true
		p.PracFarmCubes = 10
		p.PracFarmT = 75
		p.PracAggroD = 220
		p.PracAggroHP = 0.45
		p.PracRetreatHP = 0.2
	case "f75a220h30":
		p.PracAggro = true
		p.PracFarmCubes = 10
		p.PracFarmT = 75
		p.PracAggroD = 220
		p.PracAggroHP = 0.3
	case "f75a220h60":
		p.PracAggro = true
		p.PracFarmCubes = 10
		p.PracFarmT = 75
		p.PracAggroD = 220
		p.PracAggroHP = 0.6
	case "f75a260":
		p.PracAggro = true
		p.PracFarmCubes = 10
		p.PracFarmT = 75
		p.PracAggroD = 260
		p.PracAggroHP = 0.45
	case "f75a190":
		p.PracAggro = true
		p.PracFarmCubes = 10
		p.PracFarmT = 75
		p.PracAggroD = 190
		p.PracAggroHP = 0.45
	case "f75a220far":
		p.PracAggro = true
		p.PracFarmCubes = 10
		p.PracFarmT = 75
		p.PracAggroD = 220
		p.PracAggroHP = 0.45
		p.PracNearFarm = false
	case "f75a220dmg":
		p.PracAggro = true
		p.PracFarmCubes = 10
		p.PracFarmT = 75
		p.PracAggroD = 220
		p.PracAggroHP = 0.45
		p.PracUpgradePrio = dmgFirst
	case "f75a220ltn":
		p.PracAggro = true
		p.PracFarmCubes = 10
		p.PracFarmT = 75
		p.PracAggroD = 220
		p.PracAggroHP = 0.45
		p.PracLtnRank = 3.4
	case "f75a220ll":
		p.PracAggro = true
		p.PracFarmCubes = 10
		p.PracFarmT = 75
		p.PracAggroD = 220
		p.PracAggroHP = 0.45
		p.PracLasLtn = true
	case "f75a220ltnb":
		p.PracAggro = true
		p.PracFarmCubes = 10
		p.PracFarmT = 75
		p.PracAggroD = 220
		p.PracAggroHP = 0.45
		p.PracLtnRank = 3.4
		p.PracBerserk = true
	case "aggro220":
		p.PracAggro = true
		p.PracAggroD = 220
		p.PracAggroHP = 0.45
	}
}

var ltnFirst = []string{"lightning_efficiency", "lightning_chain", "plating", "armor", "capacitor", "reactor", "nanites", "gun_damage", "gun_rate",
	"laser_damage", "laser_charge", "lightning_range", "servos", "gyros", "gun_mag", "mortar_reload", "mortar_splash",
	"gun_velocity", "mortar_velocity", "sensors", "jets", "laser_pierce"}

var dmgFirst = []string{"laser_damage", "gun_damage", "gun_rate", "laser_charge", "plating", "armor", "nanites", "lightning_efficiency",
	"servos", "lightning_chain", "capacitor", "reactor", "gyros", "gun_mag", "lightning_range", "mortar_reload", "mortar_splash",
	"gun_velocity", "mortar_velocity", "sensors", "jets", "laser_pierce"}
