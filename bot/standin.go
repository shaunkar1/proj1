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

	p.HushT = 0
	p.FleeD = 0
	pinB3(p)
}

func pinB3(p *Params) {
	p.FFAPassive = false
	p.HideStick, p.HideStill, p.HideCover = 0, false, 0
	p.EndPounce, p.EndRushD, p.EndPounceT, p.EndRefuse = 0, 0, 0, 0
}

func (b *Bot) campPhase() bool {
	return b.P.Camp && (b.P.CampDuty <= 0 || math.Mod(b.t+7*float64(b.slot), 40) < 40*b.P.CampDuty)
}

func (b *Bot) earHear(brg float64, band string, at float64) {
	if at < b.P.EarAfter {
		return
	}
	px, py := b.me.Pos[0], b.me.Pos[1]

	for _, t := range b.tracks {
		if t.NPC || t.Ghost || at-t.Seen > 1 {
			continue
		}
		if math.Abs(angDiff(math.Atan2(t.Pos[1]-py, t.Pos[0]-px), brg)) < 0.3 {
			return
		}
	}
	near, mid := b.cfg.Sound.Near, b.cfg.Sound.Mid
	if near <= 0 || mid <= near {
		near, mid = 60, 160
	}
	d := near * 0.6
	if band != "near" {
		d = (near + mid) / 2
	}
	x, y := px+math.Cos(brg)*d, py+math.Sin(brg)*d
	if b.earT > 0 && at-b.earHeardT < 8 && (hyp(x, y, b.earPos[0], b.earPos[1]) < 60 || math.Abs(angDiff(brg, b.earDir)) < 0.35) {
		b.earHeardT, b.earDir, b.earPos = at, brg, [2]float64{x, y}
		return
	}
	keep := b.earRolls[:0]
	known := false
	for _, n := range b.earRolls {
		if at-n.t > 10 {
			continue
		}
		if hyp(n.pos[0], n.pos[1], x, y) < 60 || math.Abs(angDiff(math.Atan2(n.pos[1]-py, n.pos[0]-px), brg)) < 0.35 {
			n.t, known = at, true
		}
		keep = append(keep, n)
	}
	b.earRolls = keep
	if known {
		return
	}
	b.earRolls = append(b.earRolls, noiseSrc{pos: [2]float64{x, y}, t: at})
	if b.earT > 0 && at-b.earHeardT < 8 {
		return
	}
	if b.rng.Float64() < b.P.EarP {
		b.earT, b.earHeardT, b.earDir, b.earPos = at, at, brg, [2]float64{x, y}
	}
}

func (b *Bot) earGoal() ([2]float64, bool) {
	if b.P.EarP <= 0 || b.P.EarWalk <= 0 || b.earT <= 0 || b.t-b.earHeardT > b.P.EarWalk {
		return [2]float64{}, false
	}
	return b.earPos, true
}

func unpinB4(p *Params) {
	d := DefaultParams()
	p.FFAPassive = d.FFAPassive
	p.HideStick, p.HideStill, p.HideCover = d.HideStick, d.HideStill, d.HideCover
	p.EndPounce, p.EndRushD, p.EndPounceT, p.EndRefuse = d.EndPounce, d.EndRushD, d.EndPounceT, d.EndRefuse
}
