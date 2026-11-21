package bot

// Profile selects the squad-match (practice) tuning compiled into the bot.
const Profile = "kite200"

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
	case "aggro220":
		p.PracAggro = true
		p.PracAggroD = 220
		p.PracAggroHP = 0.45
	}
}

var dmgFirst = []string{"laser_damage", "gun_damage", "gun_rate", "laser_charge", "plating", "armor", "nanites", "lightning_efficiency",
	"servos", "lightning_chain", "capacitor", "reactor", "gyros", "gun_mag", "lightning_range", "mortar_reload", "mortar_splash",
	"gun_velocity", "mortar_velocity", "sensors", "jets", "laser_pierce"}
