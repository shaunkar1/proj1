package bot

import (
	"fmt"
	"math"
	"os"
)

var E8Debug = os.Getenv("MECH_E8") != ""

type e8snd struct {
	t, brg float64
	band   string
	las    bool
}

func (b *Bot) e8Any() bool {
	return b.P.EndPounce > 0 || b.P.EndRefuse > 0 || b.P.EndStalk > 0 || b.P.EndThird > 0 || b.P.EndDrop > 0
}

func e8BandD(band string) float64 {
	switch band {
	case "near":
		return 35
	case "mid":
		return 110
	}
	return 230
}

func (b *Bot) e8Hear(kind string, brg float64, band string, at float64) {
	px, py := b.me.Pos[0], b.me.Pos[1]
	switch kind {
	case "gun", "laser", "lightning", "mortar":
		keep := b.e8Snd[:0]
		for _, s := range b.e8Snd {
			if at-s.t < 6 {
				keep = append(keep, s)
			}
		}
		b.e8Snd = append(keep, e8snd{at, brg, band, kind == "laser"})
	case "death":
		d := e8BandD(band)
		pos := [2]float64{px + math.Cos(brg)*d, py + math.Sin(brg)*d}

		if b.t-b.e8FightT < 8 && math.Abs(angDiff(math.Atan2(b.e8Fight[1]-py, b.e8Fight[0]-px), brg)) < 0.4 {
			pos = b.e8Fight
		}
		bestErr := 0.25
		for _, t := range b.tracks {
			if t.NPC || t.Ghost || at-t.Seen > 6 {
				continue
			}
			td := hyp(px, py, t.Pos[0], t.Pos[1])
			in := (band == "near" && td < 75) || (band == "mid" && td > 45 && td < 180) || (band == "far" && td > 140)
			if e := math.Abs(angDiff(math.Atan2(t.Pos[1]-py, t.Pos[0]-px), brg)); in && e < bestErr {
				bestErr = e
				pos = [2]float64{t.Pos[0], t.Pos[1]}
			}
		}
		b.e8Site, b.e8SiteT = pos, at
	}
}

func (b *Bot) e8Opp() *Track {
	var best *Track
	for _, t := range b.trackL {
		if t.NPC || t.Ghost {
			continue
		}
		if best == nil || t.Seen > best.Seen {
			best = t
		}
	}
	return best
}

func (b *Bot) e8Ratio(t *Track) float64 {
	dr := b.dangerRatio(t)
	age := b.t - t.Seen
	delay, rate := b.cfg.Mech.RegenDelay, b.cfg.Mech.RegenPerS
	if rate <= 0 {
		delay, rate = 5, 45
	}

	if t.HP > 0 && t.HP < t.MaxHP && age > delay {
		dr *= math.Min(t.MaxHP, t.HP+2*rate*(age-delay)) / t.HP
	}
	return dr
}

func (b *Bot) e8Update() {
	b.e8Las = nil
	wasT := b.e8T
	b.e8T, b.e8Go = nil, false
	if !b.ffaSure || b.practice || !b.e8Any() {
		return
	}
	o := b.o
	px, py := b.me.Pos[0], b.me.Pos[1]
	if o.Alive == 2 && b.P.EndPounce > 0 {
		lim := b.P.EndPounce
		t := b.e8Opp()

		if t != nil && b.t-t.Seen < 25 && t.Seen > b.e8SiteT+0.15 {
			if t == wasT {
				lim *= 1.25
			}
			if !b.holds("laser") {

				lim *= 0.5
			}
			if b.e8Ratio(t) <= lim {
				b.e8T = t
			}
		} else {
			ttl := b.P.EndPounceT
			if ttl <= 0 {
				ttl = 20
			}
			if d := hyp(px, py, b.e8Site[0], b.e8Site[1]); b.e8SiteT > 0 && b.t-b.e8SiteT < ttl && d > 25 && d < b.P.EndRushD {

				b.e8Go = true
			}
		}
	}
	if o.Alive == 3 && (b.P.EndStalk > 0 || b.P.EndThird > 0) {

		var busy *Track
		for _, t := range b.trackL {
			if t.NPC || t.Ghost || b.t-t.Seen > 1 || b.t-t.busyT > 3 {
				continue
			}
			if busy == nil || t.busyT > busy.busyT {
				busy = t
			}
		}
		if busy != nil {
			b.e8Fight, b.e8FightT, b.e8FightSeen, b.e8FightFar = [2]float64{busy.Pos[0], busy.Pos[1]}, b.t, true, false
		} else {
			n, las := 0, false
			var last *e8snd
			for i := range b.e8Snd {
				if s := &b.e8Snd[i]; b.t-s.t < 4 {
					n++
					las = las || s.las
					if last == nil || s.t > last.t {
						last = s
					}
				}
			}

			if n >= 3 && las && !(b.e8FightSeen && b.t-b.e8FightT < 2) {
				d := e8BandD(last.band)
				b.e8Fight, b.e8FightT, b.e8FightSeen, b.e8FightFar = [2]float64{px + math.Cos(last.brg)*d, py + math.Sin(last.brg)*d}, b.t, false, last.band == "far"
			}
		}
	}
}

func (b *Bot) e8Third(t *Track, d, hpFrac float64) bool {
	if b.P.EndThird <= 0 || !b.ffaSure || b.practice || b.o.Alive != 3 || t.NPC || t.Ghost || !t.LOS || b.t-t.Seen > 0.3 {
		return false
	}
	if b.t-t.busyT > 2.5 || t.aimedAt >= 0.3 || b.t-t.HitMe < 6 || hpFrac < 0.85 || b.t-b.lastDmgT < 5 || d < b.P.EndThird {
		return false
	}
	las, _ := b.weapon("laser")
	if las == nil || d > las.Stats.Range-15 {
		return false
	}
	mine := t.HP * b.cubeMul(t.Cubes)
	for _, o := range b.trackL {
		if o == t || o.NPC || o.Ghost || b.t-o.Seen > 3 {
			continue
		}
		if b.t < o.Hostile {
			return false
		}
		if o.LOS && b.t-o.Seen < 0.3 && b.t-o.busyT < 2.5 && o.HP*b.cubeMul(o.Cubes) > mine && hyp(b.me.Pos[0], b.me.Pos[1], o.Pos[0], o.Pos[1]) >= b.P.EndThird {
			return false
		}
	}
	b.e8Las = t
	return true
}

func (b *Bot) e8StalkGoal(zEnd zoneC) (goal [2]float64, hold, ok bool) {
	if b.P.EndStalk <= 0 || b.o.Alive != 3 || b.t-b.e8FightT > 4 {
		return goal, false, false
	}
	px, py := b.me.Pos[0], b.me.Pos[1]
	fx, fy := b.e8Fight[0], b.e8Fight[1]
	dF := hyp(px, py, fx, fy)
	want := b.P.EndStalk
	if dF <= want+15 {
		return [2]float64{px, py}, true, true
	}
	if b.t < b.e8HoldT && hyp(b.e8Hold[0], b.e8Hold[1], fx, fy) < want+40 && hyp(b.e8Hold[0], b.e8Hold[1], fx, fy) > want-40 {
		return b.e8Hold, false, true
	}
	nv := b.w.Nav
	best, bs := [2]float64{}, math.Inf(1)
	base := math.Atan2(py-fy, px-fx)
	for _, da := range [...]float64{0, 0.25, -0.25, 0.5, -0.5, 0.8, -0.8} {
		x, y := fx+math.Cos(base+da)*want, fy+math.Sin(base+da)*want
		if !nv.WalkableMain(x, y) || zEnd.outside(x, y) > -12 {
			continue
		}
		if c := hyp(px, py, x, y) + 40*math.Abs(da); c < bs {
			bs, best = c, [2]float64{x, y}
		}
	}
	if math.IsInf(bs, 1) {
		return goal, false, false
	}
	b.e8Hold, b.e8HoldT = best, b.t+3
	return best, false, true
}

func (b *Bot) e8DropClear(x, y, myD float64) bool {
	r := b.P.EndDropR
	if r <= 0 {
		r = 100
	}
	if !b.nobodyWithin(x, y, r, 12) {
		return false
	}
	for _, t := range b.trackL {
		if !t.NPC && b.t-t.Seen < 12 && hyp(x, y, t.Pos[0], t.Pos[1]) < myD+20 {
			return false
		}
	}
	return true
}

func (b *Bot) e8Trace() {
	if !E8Debug || !b.e8Any() || !b.ffaSure || b.practice || b.o.Alive > 3 {
		return
	}
	me := b.me
	if os.Getenv("MECH_E8") == "full" && b.o.Alive == 2 {
		w := ""
		for _, x := range me.Weapons {
			w += fmt.Sprintf(" %s(ammo %.0f rl %.1f cd %.2f chg %.2f)", x.Kind, x.Ammo, x.Reload, x.Cooldown, x.Charge)
		}
		td, tl, thp := -1.0, false, 0.0
		if t := b.tracks[b.targetID]; t != nil {
			td, tl, thp = hyp(me.Pos[0], me.Pos[1], t.Pos[0], t.Pos[1]), t.LOS, t.HP
		}
		fmt.Fprintf(os.Stderr, "E8F %s#%d t=%.1f %s hp=%.0f en=%.0f pos=(%.0f,%.0f,%.1f) tgt=%d d=%.0f los=%v thp=%.0f%s\n", b.P.Name, b.slot, b.t, b.mode, me.HP, me.Energy, me.Pos[0], me.Pos[1], me.Pos[2], b.targetID, td, tl, thp, w)
	}
	if b.o.Tick%20 != 0 {
		return
	}
	s := fmt.Sprintf("E8 %s#%d t=%.0f alive=%d hp=%.0f/%.0f c=%.0f pos=(%.0f,%.0f) zr=%.0f %s", b.P.Name, b.slot, b.t, b.o.Alive, me.HP, me.MaxHP, me.Cubes, me.Pos[0], me.Pos[1], b.o.Zone.Radius, b.InfoX())
	if b.e8T != nil {
		s += fmt.Sprintf(" POUNCE %d r=%.2f", b.e8T.ID, b.e8Ratio(b.e8T))
	}
	if b.e8Go {
		s += fmt.Sprintf(" RUSH (%.0f,%.0f)", b.e8Site[0], b.e8Site[1])
	}
	if b.e8Las != nil {
		s += fmt.Sprintf(" THIRD %d", b.e8Las.ID)
	}
	if b.t-b.e8FightT < 4 {
		s += fmt.Sprintf(" fight@(%.0f,%.0f;%.0fm seen=%v)", b.e8Fight[0], b.e8Fight[1], hyp(me.Pos[0], me.Pos[1], b.e8Fight[0], b.e8Fight[1]), b.e8FightSeen)
	}
	for _, t := range b.trackL {
		if !t.NPC && b.t-t.Seen < 25 {
			s += fmt.Sprintf(" [%d hp %.0f/%.0f c%.0f]", t.ID, t.HP, t.MaxHP, t.Cubes)
		}
	}
	fmt.Fprintln(os.Stderr, s)
}
