package bot

// Profile selects the squad-match (practice) tuning compiled into the bot.
const Profile = "f75a300"

// ApplyProfile adjusts only the parameters used in squad matches, so the
// free-for-all behaviour is the same under every profile.
func ApplyProfile(p *Params, name string) {
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
	case "aggro220":
		p.PracAggro = true
		p.PracAggroD = 220
		p.PracAggroHP = 0.45
	}
}

var dmgFirst = []string{"laser_damage", "gun_damage", "gun_rate", "laser_charge", "plating", "armor", "nanites", "lightning_efficiency",
	"servos", "lightning_chain", "capacitor", "reactor", "gyros", "gun_mag", "lightning_range", "mortar_reload", "mortar_splash",
	"gun_velocity", "mortar_velocity", "sensors", "jets", "laser_pierce"}
