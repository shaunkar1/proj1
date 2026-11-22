package bot

import "math"

func fieldBase(p *Params) {
	p.FFA0 = "always"
	p.QuietT = 0
	p.EndgameFFA = 0
	p.EndLow = 0
	p.KeepAwayD = 0
	p.BossRatio = 0
	p.FleeLaser = 0
	p.EvadeSniper = false
	p.QuietNoClose, p.QuietNoChase = false, false
	p.SafeAway = 0
	p.FFADrop = 0
	p.ObjHide = false
}

func (b *Bot) campPhase() bool {
	return b.P.Camp && (b.P.CampDuty <= 0 || math.Mod(b.t+7*float64(b.slot), 40) < 40*b.P.CampDuty)
}
