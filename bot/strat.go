package bot

import (
	"fmt"
	"math"
	"os"
	"sort"

	"mechbot/geo"
	"mechbot/proto"
)

func (b *Bot) cubeMul(c float64) float64 {
	return 1 + b.cfg.Cubes.DamagePerCube*math.Min(c, b.cfg.Cubes.StatCap)
}

func (b *Bot) laserIn(t *Track) float64 {
	if t.NPC || !t.has[2] {
		return 1e9
	}
	las := b.cfg.Weapons["laser"]
	chg := las.Charge
	if t.Cubes >= 3 {
		chg *= 0.6
	}
	if t.Charging {
		return math.Max(0, t.lasChargeT+chg-b.t)
	}
	cd := math.Max(0, t.lasFireT+las.Cooldown-b.t)
	return cd + chg
}

func (b *Bot) enemyDpsNL(t *Track, d float64) float64 {
	if !t.has[2] {
		return b.enemyDps(t, d)
	}
	save := t.has[2]
	t.has[2] = false
	v := b.enemyDps(t, d)
	if !(t.has[0] || t.has[1] || t.has[3]) {
		v = 0
	}
	t.has[2] = save
	return v
}

func (b *Bot) enemyDps(t *Track, d float64) float64 {
	if t.NPC {
		if d < 60 {
			return 12
		}
		return 0
	}
	mul := b.cubeMul(t.Cubes) * (1 - b.me.Stats.Armor)

	for _, th := range b.cfg.Cubes.UpgradeThresholds {
		if t.Cubes >= th {
			mul *= 1.07
		}
	}
	dps := 0.0
	known := t.has[0] || t.has[1] || t.has[2] || t.has[3]
	if t.has[0] || !known {
		g := b.cfg.Weapons["gun"]
		if d < g.Range {

			ph := clamp(0.9-(d-40)*0.0125, 0.06, 0.9)
			dps += 105 * ph
		}
	}
	if t.has[2] && d < b.cfg.Weapons["laser"].Range {
		if b.holds("lightning") && d < 42 && b.me.Energy > 15 {
			dps += 20
		} else {
			dps += 125
		}
	}
	if t.has[3] && d < b.cfg.Weapons["lightning"].Range+4 {
		dps += 230
	}
	if t.has[1] && d < 110 {
		dps += clamp(70-d*0.5, 15, 70)
	}
	return dps * mul
}

func (b *Bot) myDps(t *Track, d float64) float64 { return b.myDpsX(t, d, false) }

func (b *Bot) myDpsX(t *Track, d float64, noLaser bool) float64 {
	mul := b.dmgMul()
	dps := 0.0
	for _, w := range b.me.Weapons {
		if w == nil || (noLaser && w.Kind == "laser") {
			continue
		}
		switch w.Kind {
		case "gun":
			if d < w.Stats.Range {
				cyc := w.Stats.Mag*w.Stats.Cooldown + w.Stats.Reload
				sus := w.Stats.Damage * w.Stats.Mag / cyc
				dps += sus * b.gunHitProb(t, d)
			}
		case "laser":
			if d < w.Stats.Range {
				v := w.Stats.Damage / (w.Stats.Charge + w.Stats.Cooldown)
				if t.has[1] {
					v *= 1 + w.Stats.Cookoff
				}
				if t.has[3] && d < 50 {
					v *= 0.3
				}
				dps += v
			}
		case "lightning":
			if d < w.Stats.Range {
				dps += w.Stats.Damage / w.Stats.Cooldown * clamp(b.me.Energy/40, 0.2, 1)
			}
		case "mortar":
			if d < 90 && d > 12 {
				dps += 55
			}
		}
	}
	return dps * mul
}

func (b *Bot) gunHitProb(t *Track, d float64) float64 {
	base := func(d float64) float64 { return clamp(0.97-(d-40)*0.0035, 0.3, 0.97) }
	pb := base(d)
	if t.NPC || t.shots < 6 {
		return pb
	}
	dm := hyp(t.Pos[0], t.Pos[1], b.me.Pos[0], b.me.Pos[1])
	k := clamp((t.hits/t.shots)/base(dm), 0.05, 1.1)

	k += (1 - k) * clamp((60-d)/30, 0, 1)
	return clamp(pb*k, 0.02, 0.97)
}

func (b *Bot) bestRange(t *Track) (float64, float64) {
	if b.P.FixRange > 0 && !t.NPC {
		d := b.P.FixRange
		return d, b.myDps(t, d) - 1.15*b.enemyDps(t, d)
	}
	if b.practice && b.P.PracRange > 0 && !t.NPC {
		d := b.P.PracRange
		return d, b.myDps(t, d) - 1.15*b.enemyDps(t, d)
	}
	bestD, bestA := 130.0, math.Inf(-1)
	for _, d := range [...]float64{28, 40, 55, 75, 100, 130, 160, 190} {
		a := b.myDps(t, d) - 1.15*b.enemyDps(t, d)
		if a > bestA+2 {
			bestA, bestD = a, d
		}
	}
	return bestD, bestA
}

func (b *Bot) pickUpgrade(o *proto.Offer) int {
	best, bi := 1<<30, 0
	prio := b.P.UpgradePrio
	if b.practice && len(b.P.PracUpgradePrio) > 0 {
		prio = b.P.PracUpgradePrio
	}
	for i, c := range o.Choices {
		r := 1 << 20
		for k, id := range prio {
			if id == c.ID {
				r = k
				break
			}
		}
		if r < best {
			best, bi = r, i
		}
	}
	return bi
}

func (b *Bot) heldRank(w *proto.WeaponObs) float64 {
	if w == nil {
		return 0
	}
	if w.Kind == "gun" {

		if (b.P.DropGunForLtn || (b.practice && b.P.PracLasLtn)) && b.holds("laser") {
			return 0.5
		}
		return 99
	}
	return b.rank(w.Kind)
}

func (b *Bot) rank(kind string) float64 {
	if b.practice && b.P.PracLtnRank > 0 && kind == "lightning" {
		return b.P.PracLtnRank
	}
	return b.P.WeaponRank[kind]
}

func (b *Bot) slot1Rank() float64 {
	r := 99.0
	for _, w := range b.me.Weapons {
		if v := b.heldRank(w); v < r {
			r = v
		}
	}
	if r == 99 {
		return 0
	}
	return r
}

func (b *Bot) holds(kind string) bool {
	w, _ := b.weapon(kind)
	return w != nil
}

func (b *Bot) wantPickup(it *Item) float64 {
	if it.Kind == "gun" || b.holds(it.Kind) {
		return 0
	}
	v := b.rank(it.Kind) - b.slot1Rank()
	if it.Kind == "lightning" && b.slot1Rank() == 0.5 {
		v = 1.2
	}
	return v
}

func (b *Bot) tryTake(act *proto.Action) {
	me := b.me
	if b.t < b.takeTryT {
		return
	}
	var best *Item
	bv := 0.0
	for _, it := range b.pickupL {
		if hyp(it.Pos[0], it.Pos[1], me.Pos[0], me.Pos[1]) > b.cfg.Loot.PickupRadius-0.4 || math.Abs(it.Pos[2]-me.Pos[2]) > 3.6 {
			continue
		}
		if v := b.wantPickup(it); v > bv {
			bv, best = v, it
		}
	}
	if best == nil {
		return
	}

	slot, low := -1, 1e9
	for i, w := range me.Weapons {
		if w == nil {
			slot, low = -1, -1
			break
		}
		if r := b.heldRank(w); r < low {
			low, slot = r, i
		}
	}
	if slot >= 0 {
		if w := me.Weapons[slot]; w.Kind == "laser" && w.Charge > 0 {
			return
		}
		sl := slot
		act.Drop = &sl
	}
	id := best.ID
	act.Take = &id
	b.takeTryT = b.t + 0.3
	delete(b.pickups, id)
}

type objective struct {
	kind  string
	pos   [3]float64
	value float64
	item  *Item
	trk   *Track
	site  *deathSite
}

func (b *Bot) decide(act *proto.Action) {
	o := b.o
	me := b.me
	px, py := me.Pos[0], me.Pos[1]
	if o.Offer != nil && len(o.Offer.Choices) > 0 {
		i := b.pickUpgrade(o.Offer)
		act.Pick = &i
	}
	b.tryTake(act)
	hpFrac := me.HP / math.Max(1, me.MaxHP)

	var ghost *Track
	var threats []threat
	var mechs []*Track
	incoming := 0.0
	nHostile := 0
	for _, t := range b.trackL {
		age := b.t - t.Seen
		if t.NPC {
			if age < 0.3 {
				d := hyp(px, py, t.Pos[0], t.Pos[1])
				if d < 70 {
					threats = append(threats, threat{t: t, eye: [3]float64{t.Pos[0], t.Pos[1], t.Pos[2] + 2.5}, weight: 0.5})
				}
			}
			continue
		}
		if age > 12 {
			continue
		}
		if age > 1 {
			t.closeT = 0
		}
		d := hyp(px, py, t.Pos[0], t.Pos[1])
		w := 0.3
		bearingToMe := math.Atan2(py-t.Pos[1], px-t.Pos[0])
		facing := math.Abs(angDiff(t.Aim[0], bearingToMe))
		hostile := b.t < t.Hostile
		if hostile {
			w = 1
		} else if d < 60 {
			w = 0.5
		}
		if age > 0.3 {
			w *= clamp(1-age/12, 0.1, 1)
		}
		eh := 8.0 * t.Scale
		threats = append(threats, threat{t: t, eye: [3]float64{t.Pos[0], t.Pos[1], t.Pos[2] + eh}, weight: w})
		if t.Ghost && age < 2.5 {
			ghost = t
		}
		if age < 0.3 && !t.Ghost {
			if b.P.FleeD > 0 && d > 1 {

				cv := (t.AvgVel[0]*(px-t.Pos[0]) + t.AvgVel[1]*(py-t.Pos[1])) / d
				if cv >= b.P.FleeV {
					if t.closeT == 0 {
						t.closeT = b.t
					}
				} else if cv < b.P.FleeV*0.5 {
					t.closeT = 0
				}
			}
			mechs = append(mechs, t)
			if t.LOS && hostile && (facing < 0.6 || b.t-t.HitMe < 3) {
				incoming += b.enemyDps(t, d)
				nHostile++
			}
		}
	}

	b.farmPhase = b.P.PracFarmCubes > 0 && !b.sawOtherDeath && int(me.Cubes) < b.P.PracFarmCubes && b.t < b.P.PracFarmT

	b.E = effP{b.P.BossRatio, b.P.HardRatio, b.P.BossObjD, b.P.BossAvoidD, b.P.BossSeenD, b.P.PileBonus, b.P.ExploreOld, b.P.Calm, b.P.ObjHide, !b.P.NoWatchdog}
	if !b.ffaSure && !b.P.NoLegacy {
		b.E = effP{0.85, 1.5, 160, 150, 1e9, 0, true, false, false, false}
	}
	if b.P.QuietT > 0 && (b.t < b.P.QuietT || (b.P.QuietAlive > 0 && b.ffaSure && b.o.Alive > b.P.QuietAlive && b.t < b.P.QuietCap)) && (b.ffaSure || (b.P.QuietAlways && (b.P.ShyT0 <= 0 || b.t < b.P.ShyT0))) && !b.practice {

		if b.P.QuietHide {
			b.E.BossRatio, b.E.HardRatio, b.E.BossObjD, b.E.BossAvoidD, b.E.BossSeenD = 0.85, 0, b.P.QuietObjD, b.P.QuietAvoidD, 1e9
		}
		b.quiet = true
	} else {
		b.quiet = false
	}
	if b.P.HideCover > 0 && b.hideOn() {
		b.hideSurvey()
	}
	b.endLow = b.P.EndLow > 0 && b.ffaSure && !b.practice && b.othersDied > 0 && o.Alive <= b.P.EndLow && o.Alive > 2
	if b.endLow && b.P.EndLowZoneR > 0 && o.Zone.Radius < b.P.EndLowZoneR {
		b.endLow = false
	}
	if b.endLow {

		b.quiet = true
	}
	b.hush = false
	if b.P.HushT > 0 && b.ffaSure && !b.practice && b.t < b.P.HushT && !(b.P.HushLaser && b.holds("laser")) {

		if b.P.HushD >= 1e8 {
			b.hush = true
		}
		seenMem, heardMem := 10.0, 12.0
		if b.P.HushMem > 0 {
			seenMem, heardMem = b.P.HushMem, b.P.HushMem
		}
		for _, t := range b.trackL {
			if !b.hush && !t.NPC && !t.Ghost && b.t-t.Seen < seenMem && hyp(px, py, t.Pos[0], t.Pos[1]) < b.P.HushD {
				b.hush = true
			}
		}
		for i := range b.noise {
			if !b.hush && b.t-b.noise[i].t < heardMem && hyp(px, py, b.noise[i].pos[0], b.noise[i].pos[1]) < b.P.HushD {
				b.hush = true
			}
		}
	}
	b.e8Update()
	b.memUpdate()
	b.boss = nil
	b.peer = nil
	bossD := 1e9
	peerD := 1e9
	if b.E.BossRatio > 0 && !b.practice && o.Alive > 2 {
		for _, t := range b.trackL {
			if t.NPC || t.Ghost || b.t-t.Seen > 15 {
				continue
			}
			d := hyp(px, py, t.Pos[0], t.Pos[1])
			if d < 300 {
				dr := b.dangerRatio(t)
				if dr <= b.E.BossRatio {
					continue
				}
				if b.P.BossHot > 0 && dr < b.P.BossHot && b.t-t.HitMe < 4 {
					continue
				}
				if b.E.HardRatio > 0 && dr <= b.E.HardRatio {

					if d < peerD {
						b.peer, peerD = t, d
					}
					continue
				}
				if d < bossD {
					b.boss, bossD = t, d
				}
			}
		}
		if b.boss != nil {
			for i := range threats {
				if threats[i].t == b.boss && threats[i].weight < 2 {
					threats[i].weight = 2
				}
			}
		}
	}
	if b.P.EndRefuse > 0 && b.ffaSure && !b.practice && o.Alive == 2 && b.e8T == nil {

		if t := b.e8Opp(); t != nil && b.t-t.Seen < 15 && t.Seen > b.e8SiteT+0.15 && b.e8Ratio(t) > b.P.EndRefuse {

			b.boss, bossD = t, hyp(px, py, t.Pos[0], t.Pos[1])
		}
	}
	boss := b.boss
	b.patient = b.P.Endgame > 0 && b.sawOtherDeath && o.Alive <= b.P.Endgame && o.Alive > 2
	if b.P.EndgameFFA > 0 && b.ffaSure && b.othersDied > 0 && o.Alive <= b.P.EndgameFFA && o.Alive > 2 {
		b.patient = true
	}

	var tgt *Track
	bestV := -1e18
	for _, t := range mechs {
		d := hyp(px, py, t.Pos[0], t.Pos[1])
		if !b.fair(t, d, hpFrac) {
			continue
		}
		v := -t.HP - d*2.5
		if t.LOS {
			v += 400
		}
		if b.t-t.HitMe < 4 {
			v += 350
		}
		if t.ID == b.targetID {
			v += 150
		}
		if b.practice && b.P.PracIso > 0 {

			for _, u := range b.trackL {
				if u != t && !u.NPC && !u.Ghost && b.t-u.Seen < 8 && hyp(u.Pos[0], u.Pos[1], t.Pos[0], t.Pos[1]) < 90 {
					v -= b.P.PracIso
				}
			}
		}
		if v > bestV {
			bestV, tgt = v, t
		}
	}
	if tgt == nil && b.e8T != nil && b.t-b.e8T.Seen < 12 {

		tgt = b.e8T
	}
	if tgt != nil {
		b.targetID = tgt.ID
	} else {
		b.targetID = -1
	}

	zEnd := b.zoneAt(o.Zone.ShrinkEnd + 0.01)
	if o.Zone.ShrinkEnd <= b.t {
		zEnd = zoneC{o.Zone.NextCenter, o.Zone.NextRadius}
	}
	dNext := hyp(px, py, zEnd.c[0], zEnd.c[1])
	marginR := math.Min(14, zEnd.r*0.35)
	needIn := dNext - (zEnd.r - marginR)
	zoneGo := false
	zoneLate := false
	var zoneGoal [2]float64
	budget := 1e9
	if needIn > 0 {
		f := (zEnd.r - marginR - 4) / math.Max(dNext, 1)
		if f < 0 {
			f = 0
		}
		zoneGoal = [2]float64{zEnd.c[0] + (px-zEnd.c[0])*f, zEnd.c[1] + (py-zEnd.c[1])*f}
		if c := b.w.Nav.Nearest(zoneGoal[0], zoneGoal[1], 80); c >= 0 {
			x, y := b.w.Nav.Pos(c)
			zoneGoal = [2]float64{x, y}
		}
		if b.P.ZoneGoalIn && b.ffaSure && !b.practice {

			if in := math.Min(3, zEnd.r*0.3); zEnd.outside(zoneGoal[0], zoneGoal[1]) > -in {
				zoneGoal = b.zoneGoalInside(zEnd, zoneGoal, marginR, in)
			}
		}
		if b.P.ZoneGoalNear && b.ffaSure && !b.practice {

			in := math.Min(3, zEnd.r*0.3)
			if b.zgnOn && (b.zgnC != zEnd || zEnd.outside(b.zgnG[0], b.zgnG[1]) > -in) {
				b.zgnOn = false
			}
			if !b.zgnOn && b.t >= b.zgnT && b.searchTick != o.Tick-2 {
				b.zgnT = b.t + 2
				rc := b.pathCostTo(zoneGoal[0], zoneGoal[1])
				if rc < 0 || rc > 1.5*needIn+60 {
					sT, sG, sC := b.zonePathT, b.zonePathG, b.zonePathC
					b.zoneAltT = 0
					alt := b.zoneGoalInside(zEnd, zoneGoal, marginR, in)
					if alt != zoneGoal && b.zonePathG == alt && b.zonePathT == b.t && (rc < 0 || b.zonePathC*1.3+30 < rc) {
						b.zgnG, b.zgnC, b.zgnOn = alt, zEnd, true
					} else {
						b.zonePathT, b.zonePathG, b.zonePathC = sT, sG, sC
						if rc >= 0 {
							b.zonePathT, b.zonePathG, b.zonePathC = b.t, zoneGoal, rc
						}
					}
				} else {
					b.zonePathT, b.zonePathG, b.zonePathC = b.t, zoneGoal, rc
				}
			}
			if b.zgnOn {
				zoneGoal = b.zgnG
			}
		}

		if (b.t-b.zonePathT > 1.5 || hyp(zoneGoal[0], zoneGoal[1], b.zonePathG[0], b.zonePathG[1]) > 25) && b.searchTick != o.Tick-2 {
			b.zonePathT = b.t
			b.zonePathG = zoneGoal
			b.zonePathC = b.pathCostTo(zoneGoal[0], zoneGoal[1])
			if b.zonePathC < 0 {
				b.zonePathC = needIn * 2.5
				if b.P.ZoneGoalIn && b.ffaSure && !b.practice && zoneGoal == b.zoneAltG && b.t < b.zoneAltT {

					b.zoneAltBad = append(b.zoneAltBad, [3]float64{zoneGoal[0], zoneGoal[1], b.t + 12})
					b.zoneAltT = 0
				}
			}
		}
		travel := b.zonePathC/me.Stats.WalkSpeed*1.1 + 3

		tEdge := o.Zone.ShrinkEnd
		for k := 0; k <= 20; k++ {
			tt := b.t + (o.Zone.ShrinkEnd-b.t)*float64(k)/20
			if tt < b.t {
				tt = b.t
			}
			if b.zoneAt(tt).outside(px, py) > -6 {
				tEdge = tt
				break
			}
		}
		leaveBy := math.Min(tEdge-5, o.Zone.ShrinkEnd-travel-6)
		budget = leaveBy - b.t
		if budget <= 0 || !me.InZone {
			zoneLate = true
		}
		zoneGo = true
	}
	zoneUrgent := zoneGo && (zoneLate || b.zoneAt(b.t+6).outside(px, py) > -3)
	if b.P.ZoneCommit > 0 && b.x10On() {

		if zoneUrgent {
			b.x10.urgT = b.t
		} else if zoneGo && b.x10.urgT > 0 && b.t-b.x10.urgT < b.P.ZoneCommit {
			zoneUrgent = true
			b.x10.n[xDither]++
		}
	}
	zoneQuiet := false
	if b.P.QuietWalk > 0 && b.P.QuietWalkZone && b.ffaSure && !b.practice && zoneGo && !zoneUrgent && b.t > b.P.QuietWalkT {

		v := math.Min(b.P.QuietWalk, me.Stats.WalkSpeed)
		extra := b.zonePathC/v*1.1 - b.zonePathC/me.Stats.WalkSpeed*1.1
		zoneQuiet = budget-extra <= 0
	}
	if zoneGo && b.ffaSure && !b.practice && b.P.ZoneLead > 0 && budget < b.P.ZoneLead {

		zoneUrgent = true
	}
	zoneBolt, zoneHold := false, false
	if b.P.ZoneStall > 0 && b.ffaSure && !b.practice {

		if zoneUrgent {
			if b.t-b.zsUrgT > 1.5 || b.zsBest == 0 || b.zsCirc != zEnd {

				b.zsBest, b.zsBestC, b.zsT, b.zsN, b.zsCirc = needIn, b.zonePathC, b.t, 0, zEnd
			}
			b.zsUrgT = b.t
			if needIn < b.zsBest-2 || b.zonePathC < b.zsBestC-3 {
				b.zsBest, b.zsBestC, b.zsT, b.zsN = math.Min(needIn, b.zsBest), math.Min(b.zonePathC, b.zsBestC), b.t, 0
			}
			if b.t >= b.boltT && b.t-b.zsT > b.P.ZoneStall && dNext > 12 {
				if me.Energy < b.P.ZoneStallE && b.t-b.zsT < b.P.ZoneStall+2.5 {
					zoneHold = true
				} else {

					dx, dy := zEnd.c[0]-px, zEnd.c[1]-py
					n := b.zsN % 6
					if n == 0 {
						if cx, cy, ok := b.navTo(zoneGoal[0], zoneGoal[1]); ok && hyp(cx, cy, px, py) > 2 {
							dx, dy = cx-px, cy-py
						}
					}
					a := math.Atan2(dy, dx)
					if n >= 2 {
						k := float64((n-2)/2 + 1)
						if n%2 == 1 {
							k = -k
						}
						a += k * math.Pi / 3
					}
					b.boltDir = [2]float64{math.Cos(a), math.Sin(a)}
					b.boltT = b.t + 2.2
					b.boltZ = me.Pos[2]
					b.zsT = b.t + 2.2 - b.P.ZoneStall + 1
					b.zsN++
				}
			}
			zoneBolt = b.t < b.boltT
		} else {
			b.boltT = 0
		}
	}

	engage := false
	myTTK, theirTTK := 1e9, 1e9
	prefD, adv := 0.0, 0.0
	if tgt != nil {
		d := hyp(px, py, tgt.Pos[0], tgt.Pos[1])
		prefD, adv = b.bestRange(tgt)

		dEval := d
		if math.Abs(d-prefD) < 60 || prefD > d {
			dEval = prefD
		}
		md := b.myDps(tgt, dEval)
		inc := b.enemyDps(tgt, dEval)
		if b.peekOn() && tgt.has[2] && b.coverNear {

			inc = b.enemyDpsNL(tgt, dEval)*0.7 + 35
			md *= 0.7
		}
		if md > 0 {
			myTTK = tgt.HP / md
		}

		for _, t := range mechs {
			if t != tgt && t.LOS && b.t < t.Hostile {
				inc += b.enemyDps(t, hyp(px, py, t.Pos[0], t.Pos[1]))
			}
		}
		theirTTK = me.HP / math.Max(inc, 1)
		er := b.P.EngageRatio
		if b.P.EarlyEngage > 0 && !b.practice && o.Alive > b.P.EarlyAlive {

			er *= b.P.EarlyEngage
		}
		advOK := adv > 25 && er >= b.P.EngageRatio
		if b.ffaSure && !b.practice {

			if b.quiet && b.P.QuietEngage > 0 {
				er *= b.P.QuietEngage
			}
			lim := b.P.FFAAdvD
			if b.quiet && b.P.QuietAdvD > 0 && (lim <= 0 || b.P.QuietAdvD < lim) {
				lim = b.P.QuietAdvD
			}
			if lim > 0 && d-prefD > lim {
				advOK = false
			}
		}
		engage = myTTK < er*theirTTK || advOK
		if b.ffaSure && !b.practice && b.P.FleeLaser > 0 && (b.P.FleeLaser >= 2 || b.quiet) && tgt.has[2] && !b.holds("laser") && myTTK > 2.5 {

			engage = false
		}
		if nHostile >= 2 && myTTK > 2.5 {
			engage = engage && myTTK*1.5 < theirTTK
		}
	}
	_ = adv
	if b.e8T != nil && tgt == b.e8T {

		engage = true
		b.retreating = false
	}
	if b.e8Las != nil && tgt == b.e8Las && b.t >= tgt.Hostile {
		engage = false
	}
	retreatHP := b.P.RetreatHP
	if b.practice {
		if b.P.PracRetreatHP >= 0 {
			retreatHP = b.P.PracRetreatHP
		}
		if b.P.PracBerserk && tgt != nil {
			engage = true
		}
	}
	finishing := tgt != nil && myTTK < 1.6
	if !b.retreating {
		if hpFrac < retreatHP && (incoming > 0 || b.t-b.lastDmgT < 3) && !finishing {
			b.retreating = true
		}
		if b.practice && b.P.PracTW > 0 && hpFrac < 0.8 && !finishing && b.t-b.lastDmgT < 2 {

			n := 0
			for _, t := range b.trackL {
				if !t.NPC && !t.Ghost && b.t-t.HitMe < 4 {
					n++
				}
			}
			if n >= b.P.PracTW {
				b.retreating = true
			}
		}
	} else if hpFrac > b.P.ReengageHP || (len(mechs) == 0 && b.t-b.lastDmgT > 6 && hpFrac > 0.55) {
		b.retreating = false
	}
	if b.e8T != nil && tgt == b.e8T {
		b.retreating = false
	}
	if tgt != nil && !engage && !finishing && (incoming > 0 || b.t-b.lastDmgT < 3) {

	}
	stand := false
	if b.practice && b.P.PracStand > 0 {

		fleeing := b.retreating || (tgt != nil && !engage && (b.t-b.lastDmgT < 2.5 || incoming > 0))
		if !fleeing {
			b.standT, b.standOn = 0, false
		} else {
			if b.standT == 0 {
				b.standT = b.t
			}
			if hpFrac < b.P.PracStand && b.t-b.standT > 1.0 && b.t-b.lastDmgT < 0.6 {
				b.standOn = true
			}
		}
		if b.standOn && b.t-b.lastDmgT < 2.5 {
			if st := b.standTarget(mechs); st != nil {
				tgt, stand, engage = st, true, true
				b.targetID = st.ID
				st.Hostile = math.Max(st.Hostile, b.t+4)
				prefD, adv = b.bestRange(st)
			}
		}
	}

	ctx := &moveCtx{threats: threats, wGoal: b.P.WGoal, arrive: 1.5}
	b.zrOn = 0
	mode := "roam"
	var goal [2]float64
	hasGoal := false
	var farm *Item
	var farmNPC *Track

	underFire := b.t-b.lastDmgT < 2.5

	var grab *Item
	if b.P.Grab > 0 && (b.ffaSure || b.P.GrabAlways) && hpFrac > 0.3 {
		gd := b.P.Grab
		for _, it := range b.pickupL {
			if b.t < it.Bad || math.Abs(it.Pos[2]-me.Pos[2]) > 3 {
				continue
			}
			lim := b.P.Grab
			if it.Kind != "laser" || b.holds("laser") {
				if b.slot1Rank() != 0 || b.wantPickup(it) <= 0 {
					continue
				}
				lim = b.P.Grab * 0.55
			}
			d := hyp(px, py, it.Pos[0], it.Pos[1])
			if d > lim || d >= gd {
				continue
			}

			near := false
			for _, t := range mechs {
				if hyp(it.Pos[0], it.Pos[1], t.Pos[0], t.Pos[1]) < 30 && b.t-t.Seen < 2 {
					near = true
				}
			}
			if near {
				continue
			}
			if rd := b.roughDist(it.Pos[0], it.Pos[1]); rd > lim*1.4 {
				continue
			}
			if b.capWeapon(it) {
				b.x10.n[xCapGrab]++
				continue
			}
			gd, grab = d, it
		}
		if grab == nil && b.P.GrabCube > 0 && !b.cubeFull() {

			gd = b.P.GrabCube
			for _, it := range b.cubeL {
				if b.t < it.Bad || math.Abs(it.Pos[2]-me.Pos[2]) > 3 {
					continue
				}
				d := hyp(px, py, it.Pos[0], it.Pos[1])
				if d >= gd {
					continue
				}
				near := false
				for _, t := range mechs {
					if hyp(it.Pos[0], it.Pos[1], t.Pos[0], t.Pos[1]) < 25 && b.t-t.Seen < 2 {
						near = true
					}
				}
				if near || b.roughDist(it.Pos[0], it.Pos[1]) > b.P.GrabCube*1.5 {
					continue
				}
				if b.cubeNo(it) {
					b.x10.n[xCapGrab]++
					continue
				}
				gd, grab = d, it
			}
		}
	}

	var dropGo *proto.DropObs
	ffaDrop := b.P.FFADrop > 0 && b.ffaSure && !b.practice && hpFrac < b.P.FFADrop && boss == nil

	ffaFree := b.P.FFADropFree > 0 && b.ffaSure && !b.practice && !ffaDrop && boss == nil && tgt == nil && nHostile == 0 && hpFrac > 0.5 && b.t-b.lastDmgT > 6

	e8Drop := b.P.EndDrop > 0 && b.P.FFADropFree <= 0 && b.ffaSure && !b.practice && o.Alive <= b.P.EndDrop && !ffaDrop && tgt == nil && nHostile == 0 && hpFrac > 0.5 && b.t-b.lastDmgT > 6
	if e8Drop {
		ffaFree = true
	}
	dropWise := b.practice && b.P.PracDropWise > 0
	if (b.P.PracDrop > 0 && b.practice && hpFrac < b.P.PracDrop && b.dropWanted(hpFrac)) || ffaDrop || ffaFree {
		bd := b.P.PracDropD
		if ffaFree {
			bd = b.P.FFADropFreeD
		}
		if e8Drop {
			bd = b.P.EndDropD
		}
		for i := range o.Drops {
			d := &o.Drops[i]
			dist := hyp(px, py, d.Pos[0], d.Pos[1])
			if dist >= bd || zEnd.outside(d.Pos[0], d.Pos[1]) > -4 {
				continue
			}
			if ffaFree && !e8Drop && ((d.Holder != nil && *d.Holder != me.ID) || !b.dropClear(d.Pos[0], d.Pos[1], dist) || b.roughDist(d.Pos[0], d.Pos[1]) >= bd) {
				continue
			}
			if e8Drop && ((d.Holder != nil && *d.Holder != me.ID) || !b.e8DropClear(d.Pos[0], d.Pos[1], dist) || b.roughDist(d.Pos[0], d.Pos[1]) >= bd) {
				continue
			}
			if ffaDrop {

				if b.dropSeen == nil {
					b.dropSeen = map[int]float64{}
				}
				if d.LandsIn > 0 {
					continue
				}
				if _, ok := b.dropSeen[d.ID]; !ok {
					b.dropSeen[d.ID] = b.t
				}
				if b.t-b.dropSeen[d.ID] < b.P.FFADropWait && hpFrac > 0.5 {
					continue
				}
				if d.Holder != nil && *d.Holder != me.ID {
					continue
				}
				taken := false
				for _, t := range mechs {
					if b.t-t.Seen < 4 && hyp(d.Pos[0], d.Pos[1], t.Pos[0], t.Pos[1]) < 40 && t != tgt {
						taken = true
					}
				}
				if taken {
					continue
				}
			}

			if dist/me.Stats.WalkSpeed+3 < d.LandsIn {
				continue
			}
			if dropWise {

				if dist/me.Stats.WalkSpeed+1.5 < d.LandsIn {
					continue
				}
				if (b.t-b.lastDmgT < 2.5 || nHostile > 0) && dist > 60 {
					continue
				}
			}
			if b.practice && b.P.PracDropCrowd > 0 && dist > 25 {
				nn := 0
				for _, t := range b.trackL {
					if t.NPC || t.Ghost || b.t-t.Seen > 6 {
						continue
					}
					if hyp(d.Pos[0], d.Pos[1], t.Pos[0], t.Pos[1]) < b.P.PracDropCrowd || hyp(px, py, t.Pos[0], t.Pos[1]) < 0.6*b.P.PracDropCrowd {
						nn++
					}
				}
				if nn >= 2 {
					continue
				}
			}
			bd, dropGo = dist, d
		}
	}
	e8Stalk := false
	var e8G [2]float64
	if b.P.EndStalk > 0 && b.ffaSure && !b.practice && o.Alive == 3 && (b.endLow || b.patient) && hpFrac >= 0.9 && b.t-b.lastDmgT > 8 && nHostile == 0 && !b.retreating && (tgt == nil || tgt == b.e8Las) {
		if g, _, ok := b.e8StalkGoal(zEnd); ok {
			e8Stalk, e8G = true, g
			for _, t := range b.trackL {
				if t.NPC || t.Ghost || b.t-t.Seen > 3 {
					continue
				}
				if b.t < t.Hostile || hyp(px, py, t.Pos[0], t.Pos[1]) < 80 {
					e8Stalk = false
				}
			}
		}
	}
	switch {
	case zoneUrgent:
		mode = "zone"
		goal, hasGoal = zoneGoal, true
		ctx.boost = me.Energy > 8
		if b.P.ZoneStall > 0 && b.P.ZoneStallE > 8 && b.ffaSure && !b.practice {
			ctx.boost = me.Energy > b.P.ZoneStallE
		}
		ctx.wGoal = 4
		ctx.mustMove = true
		if tgt != nil {
			ctx.target = tgt
			ctx.strafe = false
		}
		if b.P.ZoneBrave > 0 && b.x10On() && (!me.InZone || b.zoneAt(b.t+b.P.ZoneBrave).outside(px, py) > -3) {

			ctx.brave = true
			b.x10.n[xFace]++
		}
		if (b.P.ZoneRun > 0 || b.P.ZoneRunStall > 0) && b.ffaSure && !b.practice {

			if b.t-b.zrUrgT > 1.0 {
				b.zrAt, b.zrT, b.zrFree = [2]float64{px, py}, b.t, false
			} else if hyp(px, py, b.zrAt[0], b.zrAt[1]) > 8 {
				b.zrAt, b.zrT = [2]float64{px, py}, b.t
			}
			b.zrUrgT = b.t
			if b.P.ZoneRunStall > 0 && b.t-b.zrT > b.P.ZoneRunStall {
				b.zrFree = true
			}
			switch {
			case b.P.ZoneRun > 0 && (!me.InZone || b.zoneAt(b.t+b.P.ZoneRun).outside(px, py) > -2):
				ctx.threats = nil
				b.zrOn = 2
			case b.zrFree:
				var th2 []threat
				for i := range ctx.threats {
					if t := ctx.threats[i].t; t.LOS && b.t-t.Seen < 0.3 {
						th2 = append(th2, ctx.threats[i])
					}
				}
				ctx.threats = th2
				b.zrOn = 1
			}
		}
	case zoneQuiet && tgt == nil && nHostile == 0 && !underFire && !b.retreating:
		mode = "zone"
		goal, hasGoal = zoneGoal, true
		ctx.wGoal = 4
		ctx.mustMove = true
	case dropGo != nil:
		mode = "drop"
		goal, hasGoal = [2]float64{dropGo.Pos[0], dropGo.Pos[1]}, true
		ctx.arrive = 2
		ctx.wGoal = 4
		ctx.mustMove = hyp(px, py, dropGo.Pos[0], dropGo.Pos[1]) > 4
		ctx.boost = me.Energy > 8 && hyp(px, py, dropGo.Pos[0], dropGo.Pos[1]) > 12
		ctx.target = tgt
		if dropWise && tgt != nil && hyp(px, py, dropGo.Pos[0], dropGo.Pos[1]) < dropGo.Radius-1 {

			ctx.arrive = math.Max(2, dropGo.Radius-4)
			ctx.wGoal = 60
			ctx.mustMove = false
			ctx.strafe = true
		}
	case grab != nil:
		mode = "grab"
		goal, hasGoal = [2]float64{grab.Pos[0], grab.Pos[1]}, true
		ctx.arrive = 1.0
		ctx.boost = me.Energy > 8
		ctx.wGoal = 4
		ctx.mustMove = true
		ctx.target = tgt
		if b.grabID != grab.ID {
			b.grabID, b.grabT = grab.ID, b.t
		} else if b.t-b.grabT > 6 {
			grab.Bad = b.t + 60
		}
	case ghost != nil && b.t-ghost.HitMe < 2.5 && (tgt == nil || !tgt.LOS || (b.P.EvadeSniper && b.ffaSure && ghost.has[2] && !finishing && !b.practice)):

		mode = "evade"
		sp := b.findSafe(threats)
		goal, hasGoal = sp, true
		ctx.hide = true
		ctx.boost = me.Energy > 10
		ctx.wGoal = 1.5
		ctx.mustMove = true
	case e8Stalk:

		mode = "stalk"
		goal, hasGoal = e8G, true
		ctx.arrive = 4
		ctx.wGoal = 2
		if tgt != nil {
			ctx.target = tgt
		}
	case boss == nil && b.peer != nil && peerD < b.P.PeerAvoidD && b.peer.LOS && b.t-b.peer.Seen < 1.5 && !(tgt == b.peer && finishing) && tgt != b.peer:

		mode = "avoid"
		sp := b.findSafe(threats)
		goal, hasGoal = sp, true
		ctx.hide = true
		ctx.target = tgt
		ctx.wGoal = 1.5
		ctx.mustMove = true
	case boss != nil && (bossD < b.E.BossAvoidD || (boss.LOS && b.t-boss.Seen < 1.5 && bossD < b.E.BossSeenD && !(b.E.Calm && b.t-boss.HitMe > 6 && boss.aimedAt < 0.3))) && !(tgt == boss && finishing):

		mode = "avoid"
		sp := b.findSafe(threats)
		goal, hasGoal = sp, true
		ctx.hide = true
		ctx.boost = me.Energy > 12 && boss.LOS && b.t-boss.Seen < 1 && bossD < 170
		ctx.target = tgt
		ctx.wGoal = 1.5
		ctx.mustMove = boss.LOS && b.t-boss.Seen < 1
		if b.t < b.farCover {
			ctx.wGoal = 4
			ctx.mustMove = true
			ctx.boost = me.Energy > 6
		}
	case !stand && (b.retreating || (tgt != nil && !engage && (underFire || incoming > 0))):
		mode = "retreat"
		sp := b.findSafe(threats)
		goal, hasGoal = sp, true
		if !underFire && incoming == 0 && !b.P.NoRetreatCubes && !b.cubeFull() {

			bd := 28.0
			for _, it := range b.cubeL {
				if b.t < it.Bad || (it.Roof && math.Abs(it.Pos[2]-me.Pos[2]) > 3) {
					continue
				}
				if d := hyp(px, py, it.Pos[0], it.Pos[1]); d < bd && !b.cubeNo(it) {
					bd = d
					goal = [2]float64{it.Pos[0], it.Pos[1]}
					mode = "rcube"
				}
			}
		}
		ctx.hide = mode == "retreat"
		if mode == "rcube" {
			ctx.arrive = 1
		}
		ctx.boost = me.Energy > 12 && (underFire || incoming > 0)
		if b.t < b.farCover && mode == "retreat" {
			ctx.wGoal = 4
			ctx.mustMove = true
		}
		ctx.target = tgt
		ctx.wGoal = 1.5
	case tgt != nil && engage:
		mode = "fight"
		ctx.target = tgt
		ctx.strafe = true
		ctx.wantLOS = 1
		chase := b.P.ChaseHP > 0 && tgt.HP < b.P.ChaseHP*tgt.MaxHP && hpFrac > 0.5 && nHostile <= 1 && tgt.HP < me.HP
		if chase && prefD > b.P.ChaseD {

			prefD = b.P.ChaseD
		}
		if b.P.QuietNoClose && b.ffaSure && b.quiet && !b.practice && !finishing && tgt.HP > 0.3*tgt.MaxHP {

			if dNow := hyp(px, py, tgt.Pos[0], tgt.Pos[1]); dNow > prefD {
				prefD = math.Min(dNow+10, 190)
			}
		}
		ctx.pref = prefD
		for i := range threats {
			if threats[i].t == tgt {
				threats[i].isTgt = true
			}
		}
		if !tgt.LOS {
			goal, hasGoal = [2]float64{tgt.Pos[0], tgt.Pos[1]}, true
			ctx.arrive = math.Max(8, ctx.pref*0.6)
			ctx.wGoal = 0.8
			if b.P.QuietNoChase && b.ffaSure && b.quiet && !b.practice && !finishing && tgt.HP > 0.3*tgt.MaxHP {

				goal, hasGoal = b.findSafe(threats), true
				ctx.arrive = 1.5
				ctx.hide = true
				ctx.wantLOS = 0
				ctx.strafe = false
				ctx.wGoal = 1.5
			}
		}
		d := hyp(px, py, tgt.Pos[0], tgt.Pos[1])
		if b.P.LtnKite && tgt.has[3] && !b.holds("lightning") {

			if prefD < 150 {
				prefD = 150
				ctx.pref = prefD
			}
			if d < 80 && me.Energy > 6 {
				ctx.boost = true
			}
			b.ltnClose = d < 65
		}
		if prefD < 50 && d > prefD+12 && d < 120 && me.Energy > 45 {
			ctx.boost = true
		}
		if prefD > d+25 && me.Energy > 30 {
			ctx.boost = true
		}
		if chase && d > prefD+15 && me.Energy > 35 {
			ctx.boost = true
		}
		if b.e8T != nil && tgt == b.e8T && !tgt.LOS {
			goal, hasGoal = [2]float64{tgt.Pos[0], tgt.Pos[1]}, true
			ctx.arrive = 8
			ctx.wGoal = 3
			ctx.mustMove = true
			if d > 80 && me.Energy > 50 && !b.holds("lightning") {
				ctx.boost = true
			}

			var th2 []threat
			for i := range ctx.threats {
				if ctx.threats[i].t != tgt {
					th2 = append(th2, ctx.threats[i])
				}
			}
			ctx.threats = th2
		} else if zoneGo {

			goal, hasGoal = zoneGoal, true
			ctx.wGoal = 0.6
		}
	case (b.e8T != nil && tgt == nil) || (b.e8Go && tgt == nil):

		mode = "pounce"
		if b.e8T != nil {
			goal, hasGoal = [2]float64{b.e8T.Pos[0], b.e8T.Pos[1]}, true
		} else {
			goal, hasGoal = b.e8Site, true
		}
		ctx.arrive = 10
		ctx.wGoal = 3
		ctx.mustMove = hyp(px, py, goal[0], goal[1]) > 12
		ctx.boost = me.Energy > 50 && hyp(px, py, goal[0], goal[1]) > 80 && !b.holds("lightning")
	case b.fleeFrom(mechs, hpFrac) != nil || (b.t < b.fleeUntil && b.ffaSure && !b.practice && b.P.FleeD > 0):

		from := b.fleeFrom(mechs, hpFrac)
		if from == nil {
			from = b.tracks[b.fleeID]
		}
		if from != nil {
			from.fledT = b.t
			if b.t >= b.fleeUntil || from.ID != b.fleeID || hyp(px, py, b.fleeG[0], b.fleeG[1]) < 8 {
				b.fleeG = b.fleeGoal(from, threats)
				b.fleeUntil = b.t + b.P.FleeHold
				b.fleeID = from.ID
			}
		}
		mode = "flee"
		goal, hasGoal = b.fleeG, true
		ctx.arrive = 5
		ctx.wGoal = 3
		ctx.mustMove = hyp(px, py, b.fleeG[0], b.fleeG[1]) > 6
		ctx.boost = me.Energy > 10 && hyp(px, py, b.fleeG[0], b.fleeG[1]) > 15
	case b.E.Watchdog && !b.P.NoBreak && b.t < b.breakT && !b.practice:
		mode = "break"
		goal, hasGoal = b.breakG, true
		ctx.arrive = 8
	default:
		ob := (*objective)(nil)
		hurtHP := 0.7
		if b.P.HurtHP > 0 && b.ffaSure && !b.practice {
			hurtHP = b.P.HurtHP
		}
		hurt := hpFrac < hurtHP && b.t-b.lastDmgT < 40 && len(threats) > 0
		b.nearOnly = hurt && !b.P.NoRetreatCubes
		if !hurt || b.nearOnly {
			ob = b.pickObjective(zEnd, hpFrac, len(mechs) > 0, budget)
		}
		b.nearOnly = false
		if ob == nil && !hurt && b.quiet && b.P.ShyRoam && !b.P.QuietHide && (b.ffaSure || !b.P.NoShyRoam0) {

			mode = "shy"
			if !b.ffaSure && b.P.ShyMove0 {
				goal, hasGoal = b.shyMove(zEnd), true
				ctx.arrive = 6
			} else {
				goal, hasGoal = b.shySpot(), true
				ctx.arrive = 3
			}
		} else if zoneGo && ob == nil {
			mode = "zone"
			goal, hasGoal = zoneGoal, true
			ctx.wGoal = 2
			ctx.boost = me.Energy > 60 && needIn > 60
		} else if hurt && ob == nil {

			mode = "heal"
			goal, hasGoal = b.findSafe(threats), true
			ctx.hide = true
			ctx.arrive = 2
		} else if ob != nil {
			mode = ob.kind
			goal, hasGoal = [2]float64{ob.pos[0], ob.pos[1]}, true
			if b.E.Watchdog && ob.item != nil && (ob.kind == "crate" || ob.kind == "cube") {

				it := ob.item
				d := b.roughDist(ob.pos[0], ob.pos[1])
				switch {
				case it.wdT == 0 || d < it.wdD-5 || it.HP < it.wdHP-1 || b.t-it.wdT > 40:
					it.wdT, it.wdD, it.wdHP = b.t, d, it.HP
				case b.t-it.wdT > 12:
					it.Bad = b.t + 45
					it.wdT = 0
				}
			}
			if b.E.Watchdog && ob.item != nil && (ob.kind == "roof" || ob.kind == "weapon") && b.climbID != ob.item.ID {

				it := ob.item
				b.dbgObj = it.ID
				d := b.roughDist(ob.pos[0], ob.pos[1])
				switch {
				case it.wdT == 0 || d < it.wdD-5:
					it.wdT, it.wdD = b.t, d
				case b.t-it.wdT > 14:
					it.Bad = b.t + 60
					it.wdT = 0
				}
			}
			switch ob.kind {
			case "crate":
				farm = ob.item
				ctx.arrive = 14
				cp := [3]float64{ob.item.Pos[0], ob.item.Pos[1], ob.item.Pos[2] + 1.2}

				minD := (me.Stats.MuzzleHeight-1.2+me.Pos[2]-ob.item.Pos[2])/math.Tan(math.Max(0.2, -me.Stats.MinPitch*0.85)) + 1.5
				if minD < 8 {
					minD = 8
				}
				ctx.arrive = math.Max(14, minD+5)
				if !b.w.Clear(b.muz[0], b.muz[1], b.muz[2], cp[0], cp[1], cp[2]) {
					ctx.arrive = minD + 1
				}
				ctx.keepOff = minD + 1
				if dC := hyp(px, py, cp[0], cp[1]); dC < minD {

					ux, uy := b.lastMove[0], b.lastMove[1]
					if dC > 0.5 {
						ux, uy = (px-cp[0])/dC, (py-cp[1])/dC
					}
					if ux == 0 && uy == 0 {
						ux = 1
					}
					for _, rot := range [...]float64{0, 0.7, -0.7, 1.4, -1.4, 2.4, -2.4, math.Pi} {
						c, sn := math.Cos(rot), math.Sin(rot)
						vx, vy := ux*c-uy*sn, ux*sn+uy*c
						if b.w.Nav.Walkable(px+vx*3, py+vy*3) && b.w.Nav.Walkable(px+vx*6, py+vy*6) {
							b.backOff = [2]float64{vx, vy}
							b.backOffT = b.t + 0.15
							break
						}
					}
				}
				if b.objID != ob.item.ID {
					b.objID, b.objT = ob.item.ID, b.t
				} else if b.t-b.objT > 14 {
					ob.item.Bad = b.t + 90
				}
			case "npc":
				farmNPC = ob.trk
				ctx.arrive = 22
				ctx.target = ob.trk
				ctx.strafe = true
			case "roof":
				ctx.arrive = 1.2
				need := (ob.item.Pos[2]-me.Pos[2])/8.0*me.Stats.JetDrain + 12
				if hyp(goal[0], goal[1], px, py) < 2.6 && me.Grounded && me.Energy >= need && b.climbID < 0 {
					b.climbID = ob.item.ID
					b.climbCube = b.cubes[ob.item.ID] == ob.item
					b.climbT = b.t
					b.climbPos = ob.item.Pos
					b.climbFrom = [2]float64{px, py}
				}
				if hyp(goal[0], goal[1], px, py) < 30 {
					ctx.boost = false
				}
			case "cube":
				ctx.arrive = 1.0
				if ob.item != nil {

					if b.objID != ob.item.ID {
						b.objID, b.objT, b.objNearT = ob.item.ID, b.t, 0
					}
					if hyp(px, py, ob.item.Pos[0], ob.item.Pos[1]) < 2.8 {
						b.objNearT += 0.1
					}
					if b.objNearT > 1.2 || b.t-b.objT > 6+hyp(px, py, ob.item.Pos[0], ob.item.Pos[1])/8 {
						ob.item.Bad = b.t + 60
					}
				}
			case "npcgo":
				ctx.arrive = 30
			case "explore":
				ctx.arrive = 20
			case "site":
				ctx.arrive = 10
				if st := ob.site; st != nil {
					if st.goT == 0 {
						st.goT = b.t
					} else if b.t-st.goT > 45 {
						st.bad = b.t + 120
					}
				}
			case "hunt":
				ctx.arrive = 30
			case "ear":
				ctx.arrive = 25
			case "drop":
				ctx.arrive = 2
			default:
				ctx.arrive = 1.0
			}
			if me.Energy > 75 && hyp(goal[0], goal[1], px, py) > 40 && ob.kind != "roof" {
				ctx.boost = true
			}
		} else if b.campPhase() {
			mode = "camp"
			goal, hasGoal = [2]float64{px, py}, true
			ctx.arrive = 6
		} else {
			goal, hasGoal = b.roam(zEnd), true
			ctx.arrive = 6
		}
	}
	b.campHold = mode == "camp"
	b.mode = mode
	if b.E.Watchdog && !b.practice {

		if b.dithT == 0 || hyp(px, py, b.dithAt[0], b.dithAt[1]) > 12 {
			b.dithAt, b.dithT, b.dithPath = [2]float64{px, py}, b.t, 0
		} else {
			switch mode {
			case "crate", "cube", "weapon", "roof", "explore", "roam", "npcgo", "site", "hunt", "drop", "zone":
				b.dithPath += hyp(px, py, b.dithPrev[0], b.dithPrev[1])
			}
		}
		b.dithPrev = [2]float64{px, py}
		if !b.P.NoBreak && b.t-b.dithT > 14 && b.dithPath > 45 && b.t >= b.breakT {
			if hyp(px, py, b.breakC[0], b.breakC[1]) > 45 {
				b.breakTried = b.breakTried[:0]
			}
			b.breakC = [2]float64{px, py}
			best, bd := [2]float64{zEnd.c[0], zEnd.c[1]}, math.Inf(1)
			nv := b.w.Nav
			for pass := 0; pass < 2 && math.IsInf(bd, 1); pass++ {
				for k := 0; k < 12; k++ {
					a := float64(k) * math.Pi / 6
					x, y := px+90*math.Cos(a), py+90*math.Sin(a)
					if !nv.WalkableMain(x, y) || zEnd.outside(x, y) > -15 {
						continue
					}
					tried := false
					for _, ta := range b.breakTried {
						if math.Abs(angDiff(a, ta)) < 1.0 {
							tried = true
						}
					}
					if tried && pass == 0 {
						continue
					}
					c := hyp(x, y, zEnd.c[0], zEnd.c[1])
					if bo := b.boss; bo != nil && hyp(x, y, bo.Pos[0], bo.Pos[1]) < 120 {
						c += 300
					}
					if c < bd {
						bd, best = c, [2]float64{x, y}
					}
				}
				if pass == 0 && math.IsInf(bd, 1) {
					b.breakTried = b.breakTried[:0]
				}
			}
			b.breakG, b.breakT = best, b.t+9
			if d := hyp(best[0], best[1], px, py); d > 1 {

				b.breakTried = append(b.breakTried, math.Atan2(best[1]-py, best[0]-px))
				b.unstickDir = [2]float64{(best[0] - px) / d, (best[1] - py) / d}
				b.unstickT = b.t + 1.3
			}
			b.dithT = 0
			for _, it := range b.crateL {
				if hyp(it.Pos[0], it.Pos[1], px, py) < 45 {
					it.Bad = math.Max(it.Bad, b.t+40)
				}
			}
			for _, it := range b.pickupL {
				if hyp(it.Pos[0], it.Pos[1], px, py) < 45 {
					it.Bad = math.Max(it.Bad, b.t+40)
				}
			}
		}
	}
	b.DbgGoal, b.DbgHasGoal = goal, hasGoal
	if hideDebug {
		b.hideTrace()
	}
	if hasGoal {
		ctx.goal = &goal
	}

	nvw := b.w.Nav
	if myc := nvw.Cell(px, py); b.hopFly || (myc >= 0 && nvw.Walk[myc] && nvw.Comp[myc] != nvw.Main && me.Grounded) {
		if !b.hopFly {
			exits := nvw.Exits[nvw.Comp[myc]]
			var be *geo.Exit
			bc := math.Inf(1)
			for i := range exits {
				e := &exits[i]
				d := hyp(e.LX, e.LY, px, py)

				toward := 0.0
				if hasGoal {
					toward = hyp(e.TX, e.TY, goal[0], goal[1]) * 0.25
				}
				c := d*1.4 + e.Energy*1.2 + toward
				if b.hop != nil && e.LX == b.hop.LX && e.LY == b.hop.LY {
					c -= 25
				}
				if b.t < b.hopFail && b.hop != nil && e.LX == b.hop.LX && e.LY == b.hop.LY {
					c += 400
				}
				if c < bc {
					bc, be = c, e
				}
			}
			b.hop = be
		}
		if b.hop != nil && mode != "fight" {
			mode = "escape"
			goal, hasGoal = [2]float64{b.hop.LX, b.hop.LY}, true
			ctx.goal = &goal
			ctx.arrive = 0.8
			ctx.mustMove = false
			ctx.wGoal = 3
			ctx.boost = false
			if !b.hopFly && hyp(b.hop.LX, b.hop.LY, px, py) < 1.8 && me.Grounded && me.Energy >= math.Min(b.hop.Energy+10, me.MaxEnergy*0.97) {
				b.hopFly = true
				b.hopT = b.t
			}
		}
	} else if !b.hopFly {
		b.hop = nil
	}

	spd := math.Hypot(me.Vel[0], me.Vel[1])
	moved := hyp(px, py, b.lastPos[0], b.lastPos[1])
	if b.wantMove && spd < 1.0 && moved < 0.08 && me.Grounded {
		b.stuckT += 0.1
	} else if spd > 3 {
		b.stuckT = 0
	}
	var mx, my float64
	var boost bool
	climbing := false
	if b.hopFly && b.hop != nil {

		e := b.hop
		dx, dy := e.TX-px, e.TY-py
		l := math.Max(math.Hypot(dx, dy), 0.01)
		ahead := b.w.MaxTopAlong(px, py, e.TX, e.TY)
		landed := me.Grounded && b.t-b.hopT > 0.4
		switch {
		case landed && (nvw.WalkableMain(px, py) || l < 3):
			b.hopFly, b.hop = false, nil
		case landed || b.t-b.hopT > 14:

			b.hopFly = false
			b.hopFail = b.t + 25
		default:
			climbing = true
			mx, my = dx/l, dy/l
			if me.Grounded {
				act.Jump = true
			}
			boost = me.Pos[2] < ahead+1.5 && l > 2.0
			if l <= 2.0 {
				mx, my = 0, 0
			}
			mode = "hop"
		}
	}
	if !climbing && b.climbID >= 0 {
		it := b.pickups[b.climbID]
		if it == nil && b.climbCube {
			it = b.cubes[b.climbID]
		}
		onRoof := me.Grounded && math.Abs(me.Pos[2]-b.climbPos[2]) < 1.6
		switch {
		case it == nil, b.t-b.climbT > 11, underFire, mode == "fight", mode == "retreat", zoneUrgent,
			!onRoof && me.Energy < 2 && me.Vel[2] <= 0 && b.t-b.climbT > 1:
			if it != nil {
				it.Bad = b.t + 45
			}
			b.climbID = -1
		default:
			climbing = true
			dx, dy := b.climbPos[0]-px, b.climbPos[1]-py
			l := math.Max(math.Hypot(dx, dy), 0.01)
			mx, my = dx/l, dy/l
			if l < 1.0 {
				mx, my = mx*l, my*l
			}
			if !onRoof {
				if me.Grounded {
					act.Jump = true
				}
				boost = me.Pos[2] < b.climbPos[2]+1.0
			}
			mode = "climb"
		}
	}
	if b.P.Periscope > 0 && b.t < 3.4 && !climbing && !b.hopFly && !underFire && tgt == nil && mode != "zone" {

		if b.t < 0.15 {
			b.periZ = me.Pos[2]
		}
		if (me.Grounded && b.t < 1.2) || (!me.Grounded && b.t < 3.0) {
			mx, my = 0, 0
			climbing = true
			mode = "peri"
			if me.Grounded {
				act.Jump = true
			}
			boost = !me.Grounded && me.Pos[2] < b.periZ+b.P.Periscope && me.Energy > 45 && b.t < 2.4
		}
	}
	if b.t < b.backOffT && !climbing && mode == "crate" {
		mx, my = b.backOff[0], b.backOff[1]
		boost = false
		climbing = true
	}
	bolt := zoneBolt && !climbing && mode == "zone"
	if bolt {

		mx, my = b.boltDir[0], b.boltDir[1]
		boost = me.Energy > 2
		if b.P.ZoneStallE > 0 {
			boost = !me.Grounded && me.Pos[2] < b.boltZ+7 && me.Energy > 0
		}
		if me.Grounded {
			act.Jump = true
		}
		b.stuckT = 0
		b.pathT = -10
		mode = "zbolt"
	} else if climbing {
		b.stuckT = 0
	} else if b.t < b.unstickT {
		mx, my = b.unstickDir[0], b.unstickDir[1]
		boost = true
		if me.Grounded {
			act.Jump = true
		}
	} else if b.t < b.freeT {

		var on bool
		if mx, my, boost, on = b.freeFlight(act); on {
			b.stuckT = 0
			mode = "free"
		} else {
			mx, my, boost = b.steer(ctx)
		}
	} else if b.stuckT > 0.5 && b.P.Unwedge && b.ffaSure && !b.practice && b.unwedge(hasGoal, goal) {

		b.stuckT = 0
		b.pathT = -10
		mx, my, boost, _ = b.freeFlight(act)
		mode = "free"
	} else if b.stuckT > 0.5 {

		if c := b.w.Nav.Cell(px+b.lastMove[0]*3, py+b.lastMove[1]*3); c >= 0 {
			b.blockedC[c] = b.t + 20
		}
		if c := b.w.Nav.Cell(px, py); c >= 0 {
			b.blockedC[c] = b.t + 20
		}
		a := b.rng.Float64() * 2 * math.Pi
		if hasGoal && b.rng.Float64() < 0.6 {
			a = math.Atan2(goal[1]-py, goal[0]-px) + (b.rng.Float64()-0.5)*1.2
		}
		b.unstickDir = [2]float64{math.Cos(a), math.Sin(a)}
		b.unstickT = b.t + 0.9
		b.stuckT = 0
		b.pathT = -10
		mx, my = b.unstickDir[0], b.unstickDir[1]
		act.Jump = true
		boost = true
	} else {
		mx, my, boost = b.steer(ctx)
	}
	if b.P.HideStill && b.hideOn() && mode == "shy" && hasGoal && !climbing && !bolt && me.Grounded && b.t >= b.unstickT &&
		hyp(px, py, goal[0], goal[1]) <= 3 && len(b.o.Projectiles) == 0 && b.t-b.lastDmgT > 3 {
		mx, my, boost = 0, 0, false
	}
	if v := b.P.QuietWalk; v > 0 && b.ffaSure && !b.practice && b.t > b.P.QuietWalkT && (!b.P.QuietWalkQ || b.quiet) && me.Grounded && !climbing && !bolt &&
		b.t >= b.unstickT && b.stuckT <= 0.5 && !zoneUrgent && !underFire && b.t-b.lastDmgT > 5 && nHostile == 0 && tgt == nil && len(b.o.Projectiles) == 0 {
		switch mode {
		case "fight", "flee", "retreat", "escape", "evade", "avoid", "zbolt", "hop", "climb", "break", "pounce", "stalk":
		default:

			boost = false
			b.quietWalkN++
			if ws := me.Stats.WalkSpeed; ws > v {
				if l, f := math.Hypot(mx, my), v/ws; l > f {
					mx, my = mx*f/l, my*f/l
				}
			}
		}
	}
	if !me.Grounded && b.t >= b.unstickT && !climbing && !bolt {

		boost = false
	}
	if zoneHold && !bolt && !climbing {
		boost = false
	}
	act.Move = proto.Vec2{mx, my}
	act.Boost = boost
	b.wantMove = math.Hypot(mx, my) > 0.3
	if b.wantMove {
		b.lastMove = [2]float64{mx, my}
	}

	b.lasCrate = false
	b.lasFree = true
	for _, t := range b.trackL {
		if !t.NPC && b.t-t.Seen < 4 {
			b.lasFree = false
		}
	}
	shot := false
	var shootT *Track

	bs := -1e18
	for _, t := range mechs {
		if !t.LOS {
			continue
		}
		d := hyp(px, py, t.Pos[0], t.Pos[1])
		if !b.fair(t, d, hpFrac) {
			continue
		}
		v := -t.HP*0.6 - d*2
		if t == tgt {
			v += 200
		}
		if b.t-t.HitMe < 3 {
			v += 150
		}
		if v > bs {
			bs, shootT = v, t
		}
	}
	holdFire := mode == "retreat" && b.retreating && !underFire && incoming == 0

	var biter *Track
	if !b.P.NoBiter {
		for _, t := range b.trackL {
			if t.NPC && b.t-t.HitMe < 3 && b.t-t.Seen < 6 && (biter == nil || t.HitMe > biter.HitMe) {
				biter = t
			}
		}
	}
	npcShot := false
	if shootT != nil && !holdFire {
		shot = b.shootMech(act, shootT)
	} else if biter != nil {
		if biter.LOS && b.t-biter.Seen < 0.3 {
			shot = b.shootMech(act, biter)
		}
		if !shot {

			y, p, _ := b.aimAt([3]float64{biter.Pos[0], biter.Pos[1], biter.Pos[2] + 1.5})
			act.Aim = [2]float64{y, p}
			shot = true
		}
		npcShot = true
		if las, li := b.weapon("laser"); las != nil && las.Charge > 0 && !act.Fire[li] {
			act.Fire[li] = true
		}
	} else if tgt != nil && !holdFire {

		y, p, _ := b.aimAt(b.predict(tgt, latency))
		act.Aim = [2]float64{y, p}
		if las, li := b.weapon("laser"); las != nil && las.Charge > 0 {
			act.Fire[li] = true
		} else if las != nil && las.Cooldown <= 0.11 && b.t-tgt.Seen < 0.3 && mode == "fight" && hyp(px, py, tgt.Pos[0], tgt.Pos[1]) < las.Stats.Range &&
			(!b.peekOn() || !(tgt.has[2] && tgt.Charging)) {
			act.Fire[li] = true
		}
		shot = true
	}
	healing := hpFrac < 0.85 && b.t-b.lastDmgT < 40
	farmShot := false

	noFarm := b.capNoFarm()

	lowSilent := b.endLow && !b.P.EndLowFarm
	if lowSilent && b.farmShots() {
		lowSilent = false
		b.x10.n[xFarmShot]++
	}
	hushLook := b.P.HushLook && b.x10On() && b.t < 1.7
	silent := lowSilent || b.hush || noFarm || hushLook
	if !shot && shootT == nil && healing {

		for _, t := range b.trackL {
			if t.NPC && t.LOS && b.t-t.Seen < 0.3 && b.t-t.HitMe < 2 {
				shot = b.shootMech(act, t)
				break
			}
		}

		if !shot && !b.P.OldReload {
			if farm != nil && !silent {
				shot = b.shootCrate(act, farm)
			}
			if !shot && !b.retreating && !lowSilent && !b.hush && !noFarm {
				var bc *Item
				bd := 60.0
				for _, c := range b.crateL {
					if c.Roof && c.Pos[2] > me.Pos[2]+3 {
						continue
					}
					d := hyp(c.Pos[0], c.Pos[1], px, py)
					if d < bd && b.t-c.Seen < 0.5 && b.t >= c.Bad && !b.farmBlocked(c.Pos[0], c.Pos[1]) {
						bd, bc = d, c
					}
				}
				if bc != nil {
					shot = b.shootCrate(act, bc)
				}
			}
			farmShot = shot
		}
	} else if !shot && shootT == nil {

		if farmNPC != nil && farmNPC.LOS && !(silent && b.t-farmNPC.HitMe >= 3) {
			shot = b.shootMech(act, farmNPC)
		}
		if !shot {
			for _, t := range b.trackL {
				if t.NPC && t.LOS && b.t-t.Seen < 0.3 && (b.t-t.HitMe < 3 || (b.P.FarmNPC && !lowSilent && !b.hush && !noFarm && !hushLook && hpFrac > 0.5 && hyp(px, py, t.Pos[0], t.Pos[1]) < 60)) {
					shot = b.shootMech(act, t)
					break
				}
			}
		}
		look := b.P.ScanFirst && b.t < 1.7 && (b.ffaSure || b.P.ScanAlways)
		if !shot && farm != nil && !look && !silent {
			shot = b.shootCrate(act, farm)
		}
		if !shot && !b.retreating && !look && !lowSilent && !b.hush && !noFarm {

			var bc *Item
			bd := 110.0
			for _, c := range b.crateL {
				if c.Roof && c.Pos[2] > me.Pos[2]+3 && !b.roofLootOK(c, hpFrac) {
					continue
				}
				d := hyp(c.Pos[0], c.Pos[1], px, py)
				if d < bd && b.t-c.Seen < 0.5 && b.t >= c.Bad && !b.farmBlocked(c.Pos[0], c.Pos[1]) {
					bd, bc = d, c
				}
			}
			if bc != nil && (bd < 50 || b.lasFree) {
				shot = b.shootCrate(act, bc)
			}
		}
		farmShot = shot
	}
	if las, li := b.weapon("laser"); las != nil && las.Charge > 0 && !act.Fire[li] && shootT == nil && !b.lasCrate {
		act.Fire[li] = true
	}
	if b.ltnClose {
		if las, li := b.weapon("laser"); las != nil && las.Charge == 0 {
			act.Fire[li] = false
		}
		b.ltnClose = false
	}
	if !shot {
		b.scan(act, hasGoal, goal)
		if mode == "stalk" && b.t-b.e8FightT < 4 {

			act.Aim = [2]float64{angDiff(math.Atan2(b.e8Fight[1]-py, b.e8Fight[0]-px)+0.5*math.Sin(b.scanPhase*1.1), 0), 0}
		}
	}

	if gun, _ := b.weapon("gun"); gun != nil && gun.Reload <= 0 && gun.Ammo < gun.Stats.Mag {

		quiet := shootT == nil && (tgt == nil || !tgt.LOS) && !farmShot && !npcShot
		if quiet {
			b.quietT += 0.1
		} else {
			b.quietT = 0
		}

		if b.P.OldReload {
			if (quiet && gun.Ammo <= gun.Stats.Mag*0.5) || (quiet && tgt == nil && len(mechs) == 0 && gun.Ammo < gun.Stats.Mag) {
				act.Reload = true
			}
		} else if (quiet && gun.Ammo <= gun.Stats.Mag*0.34) ||
			(b.quietT > 1.5 && tgt == nil && len(mechs) == 0 && gun.Ammo <= gun.Stats.Mag*0.67 && mode != "crate" && mode != "npc") ||
			(b.quietT > 6 && len(mechs) == 0 && gun.Ammo < gun.Stats.Mag && mode != "crate" && mode != "npc") {
			act.Reload = true
		}
	}
	b.dbg = mode
	b.e8Trace()
	b.trace13(act, mode, goal, hasGoal, zoneGo, zoneUrgent, budget, needIn, tgt)
	b.x10Trace(zoneUrgent, act, farmShot)
	if Debug && b.P.NoFire {
		fmt.Fprintf(os.Stderr, "    -> move=(%.2f,%.2f) mode=%s hp=%.0f\n", act.Move[0], act.Move[1], mode, me.HP)
	}
}

func (b *Bot) fair(t *Track, d, hpFrac float64) bool {
	if b.e8T != nil && t == b.e8T {
		t.Hostile = math.Max(t.Hostile, b.t+8)
		return true
	}
	if b.patient {

		fin := 0.3
		if b.P.PatientFinish > 0 && b.ffaSure && !b.practice {
			fin = b.P.PatientFinish
		}
		if b.t-t.HitMe < 4 || (b.t < t.Hostile && d < 70) || (t.HP < fin*t.MaxHP && d < 130 && hpFrac > 0.5) {
			return true
		}
		if b.lateStrike(t, d, hpFrac) {
			t.Hostile = math.Max(t.Hostile, b.t+8)
			return true
		}
		if b.P.EndThird > 0 && b.e8Third(t, d, hpFrac) {
			return true
		}
		return false
	}
	if b.farmPhase {
		if b.practice {
			return b.t-t.HitMe < 4 && d < 100
		}

		return b.t < t.Hostile
	}
	if b.t < t.Hostile {
		return true
	}
	if b.P.Aggressive && (b.P.AggroD <= 0 || d < b.P.AggroD) && b.t >= b.P.AggroAfter && hpFrac >= b.P.AggroHP {
		return true
	}
	if b.isPrey(t, d, hpFrac) {
		return true
	}
	if b.isOpening(t, d, hpFrac) {
		t.Hostile = math.Max(t.Hostile, b.t+8)
		return true
	}
	if b.isBlind(t, d, hpFrac) {
		t.Hostile = math.Max(t.Hostile, b.t+6)
		return true
	}
	if b.practice && b.P.PracBerserk {
		return true
	}
	if b.practice && b.P.PracStand > 0 && t.LOS && b.t-t.Seen < 0.3 {

		if md := b.myDps(t, d); md > 0 && t.HP < 2*md {
			return true
		}
	}
	if b.practice && b.P.PracAggro && hpFrac > b.P.PracAggroHP && d < b.P.PracAggroD {
		return true
	}
	if t.HP < 0.4*t.MaxHP && hpFrac > 0.6 && d < 150 {
		return true
	}
	return false
}

func (b *Bot) prefRange(t *Track) float64 {
	haveL := b.holds("lightning")
	haveLas := b.holds("laser")
	switch {
	case haveL && b.me.Energy > 25 && !(t.has[3] && t.HP > b.me.HP*1.3):
		return b.P.PrefRangeLtn
	case t.has[3]:
		if haveLas {
			return b.P.PrefRangeLas
		}
		return math.Max(70, b.P.PrefRangeGun)
	case haveLas:
		if t.has[2] {
			return 60
		}
		return b.P.PrefRangeLas
	case t.has[2]:
		return 45
	}
	return b.P.PrefRangeGun
}

func (b *Bot) findSafe(threats []threat) [2]float64 {
	me := b.me
	px, py := me.Pos[0], me.Pos[1]
	if b.t < b.safeT {

		ok := true
		z := b.w.Ground(b.safeSpot[0], b.safeSpot[1]) + me.Stats.Height*0.5
		for i := range threats {
			if threats[i].t.NPC {
				continue
			}
			if b.bodyExposed(threats[i].eye, b.safeSpot[0], b.safeSpot[1], z-me.Stats.Height*0.5) {
				ok = false
				break
			}
		}
		if ok && b.P.SafeAway > 0 && b.ffaSure && !b.practice {
			for i := range threats {
				th := &threats[i]
				if th.t.NPC || th.weight < 1 {
					continue
				}
				dNow := hyp(px, py, th.t.Pos[0], th.t.Pos[1])
				dS := hyp(b.safeSpot[0], b.safeSpot[1], th.t.Pos[0], th.t.Pos[1])
				if dS < dNow-10 && hyp(b.safeSpot[0], b.safeSpot[1], px, py) > 12 {
					ok = false
					break
				}
			}
		}
		if ok && b.zoneAt(b.t+8).outside(b.safeSpot[0], b.safeSpot[1]) < -4 {
			return b.safeSpot
		}
	}
	nv := b.w.Nav
	zs := b.zoneAt(b.t + 12)
	best := [2]float64{px, py}
	bs := math.Inf(-1)
	for k := 0; k < 48; k++ {
		a := b.rng.Float64() * 2 * math.Pi
		r := 8 + b.rng.Float64()*70
		x, y := px+r*math.Cos(a), py+r*math.Sin(a)
		if !nv.WalkableMain(x, y) {
			continue
		}
		if zs.outside(x, y) > -6 {
			continue
		}
		z := b.w.Ground(x, y) + me.Stats.Height*0.5
		s := -b.roughDist(x, y) * 1.0
		for i := range threats {
			th := &threats[i]
			if th.t.NPC {
				continue
			}
			d := hyp(x, y, th.t.Pos[0], th.t.Pos[1])
			if b.bodyExposed(th.eye, x, y, z-me.Stats.Height*0.5) {
				s -= 150 * th.weight
			}
			s += math.Min(d, 120) * 0.5 * th.weight
			if b.P.SafeAway > 0 && b.ffaSure && !b.practice && th.weight >= 1 {

				dNow := hyp(px, py, th.t.Pos[0], th.t.Pos[1])
				if d < dNow-10 {
					s -= 120 + b.P.SafeAway*(dNow-d)
				} else if hyp(x, y, px, py) > dNow && (x-px)*(th.t.Pos[0]-px)+(y-py)*(th.t.Pos[1]-py) > 0 {
					s -= 120
				}
			}

			if d < 30 {
				s -= 100
			}
		}
		if s > bs {
			bs, best = s, [2]float64{x, y}
		}
	}
	keep := 2.5
	if !b.P.NoFarCover && b.t >= b.farTry {

		var hs [2]*threat
		for i := range threats {
			th := &threats[i]
			if th.t.NPC || th.weight < 1 || b.t-th.t.Seen > 3 {
				continue
			}
			d := hyp(px, py, th.t.Pos[0], th.t.Pos[1])
			if hs[0] == nil || d < hyp(px, py, hs[0].t.Pos[0], hs[0].t.Pos[1]) {
				hs[1], hs[0] = hs[0], th
			} else if hs[1] == nil || d < hyp(px, py, hs[1].t.Pos[0], hs[1].t.Pos[1]) {
				hs[1] = th
			}
		}
		exposedAt := func(x, y float64) bool {
			z := b.w.Ground(x, y)
			for _, th := range hs {
				if th != nil && b.bodyExposed(th.eye, x, y, z) {
					return true
				}
			}
			return false
		}
		if hs[0] != nil && exposedAt(best[0], best[1]) {
			b.farTry = b.t + 1.2
			found := false
			for ring := 0; ring < 4 && !found; ring++ {
				r := 55 + float64(ring)*30
				fd := math.Inf(1)
				for k := 0; k < 16; k++ {
					a := (float64(k) + b.rng.Float64()) * 2 * math.Pi / 16
					x, y := px+r*math.Cos(a), py+r*math.Sin(a)
					if !nv.WalkableMain(x, y) || zs.outside(x, y) > -6 {
						continue
					}

					d0 := hyp(x, y, hs[0].t.Pos[0], hs[0].t.Pos[1])
					if d0 < hyp(px, py, hs[0].t.Pos[0], hs[0].t.Pos[1])*0.8 || exposedAt(x, y) {
						continue
					}
					if c := b.roughDist(x, y) - d0*0.2; c < fd {
						fd, best = c, [2]float64{x, y}
						found = true
					}
				}
			}
			if found {
				keep = 6
				b.farCover = b.t + 6
			}
		}
	}
	b.safeSpot = best
	b.safeT = b.t + keep
	return best
}

func (b *Bot) pickObjective(zEnd zoneC, hpFrac float64, enemiesNear bool, budget float64) *objective {
	me := b.me
	px, py := me.Pos[0], me.Pos[1]
	var best *objective
	bs := 0.0

	var keep []*Track
	if b.P.KeepAwayD > 0 && b.ffaSure && !b.practice && !b.quiet && b.o.Alive > b.P.KeepAwayAlive {
		for _, t := range b.trackL {
			if !t.NPC && !t.Ghost && b.t-t.Seen < 8 && b.dangerRatio(t) >= b.P.KeepAwayRatio {
				keep = append(keep, t)
			}
		}
	}
	consider := func(ob objective) {
		d := b.roughDist(ob.pos[0], ob.pos[1])
		if b.P.CampD > 0 && d > b.P.CampD && b.campPhase() && ob.kind != "ear" {
			return
		}
		if b.x10Skip(&ob, d) {
			return
		}
		if b.practice && b.P.PracSqD > 0 && d > 12 {
			nn := 0
			for _, t := range b.trackL {
				if !t.NPC && !t.Ghost && b.t-t.Seen < 8 && hyp(ob.pos[0], ob.pos[1], t.Pos[0], t.Pos[1]) < b.P.PracSqD {
					nn++
				}
			}
			if nn >= 2 {
				return
			}
		}
		for _, t := range keep {
			db := hyp(ob.pos[0], ob.pos[1], t.Pos[0], t.Pos[1])
			if db < b.P.KeepAwayD && db < hyp(px, py, t.Pos[0], t.Pos[1])+5 && d > 12 {
				return
			}
		}
		if (b.P.HitMem > 0 || b.P.LtnObjD > 0) && b.ffaSure && !b.practice && d > 12 {
			for _, t := range b.trackL {
				if t.NPC {
					continue
				}
				db := hyp(ob.pos[0], ob.pos[1], t.Pos[0], t.Pos[1])
				if b.P.HitMem > 0 && t.HitMe > 0 && b.t-t.HitMe < b.P.HitMem && db < 220 && db < hyp(px, py, t.Pos[0], t.Pos[1])-5 {
					return
				}
				if b.P.LtnObjD > 0 && t.has[3] && !t.Ghost && b.t-t.Seen < 20 && db < b.P.LtnObjD && !b.holds("lightning") {
					return
				}
			}
		}
		if b.P.CrossAlive > 0 && b.ffaSure && !b.practice && b.o.Alive <= b.P.CrossAlive && d > 12 {

			lanes := 0
			gz := b.w.Ground(ob.pos[0], ob.pos[1]) + 2.0
			for _, t := range b.trackL {
				if t.NPC || t.Ghost || b.t-t.Seen > 10 || !t.has[2] {
					continue
				}
				if db := hyp(ob.pos[0], ob.pos[1], t.Pos[0], t.Pos[1]); db < b.P.CrossD && b.w.ClearCoarse(t.Pos[0], t.Pos[1], t.Pos[2]+2.0, ob.pos[0], ob.pos[1], gz) {
					lanes++
				}
			}
			if lanes >= 2 {
				return
			}
		}
		if b.hush && (ob.kind == "crate" || ob.kind == "npc" || ob.kind == "npcgo") {
			return
		}
		if b.P.FleeD > 0 && b.P.FleeMem > 0 && b.ffaSure && !b.practice && d > 12 {
			for _, t := range b.trackL {
				if t.NPC || t.fledT == 0 || b.t-t.fledT > b.P.FleeMem {
					continue
				}
				dt := hyp(px, py, t.Pos[0], t.Pos[1])
				db := hyp(ob.pos[0], ob.pos[1], t.Pos[0], t.Pos[1])
				if db < 220 && db < dt-5 {
					return
				}
				if dt > 1 && d > dt*0.7 && ((ob.pos[0]-px)*(t.Pos[0]-px)+(ob.pos[1]-py)*(t.Pos[1]-py))/(hyp(ob.pos[0], ob.pos[1], px, py)*dt+1e-9) > 0.7 {
					return
				}
			}
		}
		lowFree := false
		if b.endLow && b.P.EndLowD > 0 {

			if b.farmFree(&ob) {
				lowFree = true
				b.x10.n[xFarmObj]++
			} else if b.laserFree(&ob) {
				lowFree = true
				b.x10.n[xLasObj]++
			} else if b.silentFree(&ob, d) {
				lowFree = true
				b.x10.n[xSilObj]++
			}
		}
		if b.endLow && b.P.EndLowD > 0 && !lowFree {
			switch ob.kind {
			case "cube", "weapon", "roof", "drop":
				lim := b.P.EndLowD
				if (ob.kind == "weapon" || ob.kind == "roof") && ob.item != nil && ob.item.Kind == "laser" && !b.holds("laser") && b.P.ShyLaserD > lim {
					lim = b.P.ShyLaserD
				}
				if d > lim {
					return
				}
			default:

				return
			}
		}
		if b.nearOnly && (d > 35 || (ob.kind != "cube" && ob.kind != "crate")) {

			if !(b.P.RoofBold && !b.practice && (ob.kind == "weapon" || ob.kind == "roof") && d < 70 && !b.holds("laser")) {
				return
			}
		}
		if bo := b.boss; bo != nil {

			db := hyp(ob.pos[0], ob.pos[1], bo.Pos[0], bo.Pos[1])
			if db < b.E.BossObjD || (db < hyp(px, py, bo.Pos[0], bo.Pos[1])-15 && (!b.ffaSure || b.P.BossFarD <= 0 || db < b.P.BossFarD)) {
				return
			}
		}
		shyPen := 0.0
		if b.quiet && !b.P.QuietHide {
			soft := !b.ffaSure && b.P.ShySoft0 > 0

			for _, t := range b.trackL {
				if !b.ffaSure && !soft && (d <= b.P.ShyNear0 || (b.P.ShyWeapon0 && (ob.kind == "weapon" || ob.kind == "roof"))) {
					break
				}
				if mem := 8.0; t.NPC || t.Ghost || b.t-t.Seen > func() float64 {
					if b.ffaSure && b.P.ShyMem > 0 {
						mem = b.P.ShyMem
					}
					return mem
				}() {
					continue
				}
				db := hyp(ob.pos[0], ob.pos[1], t.Pos[0], t.Pos[1])
				if soft {
					if c := hyp(px, py, t.Pos[0], t.Pos[1]) - db; db < b.P.ShyFarD && c > shyPen {
						shyPen = c
					}
					continue
				}
				if db < b.P.QuietObjD && db < hyp(px, py, t.Pos[0], t.Pos[1])+5 && d > 12 {
					return
				}
			}
			if soft {
				for i := range b.noise {
					n := &b.noise[i]
					if b.t-n.t > 12 {
						continue
					}
					db := hyp(ob.pos[0], ob.pos[1], n.pos[0], n.pos[1])
					if c := hyp(px, py, n.pos[0], n.pos[1]) - db; db < b.P.ShyFarD && c > shyPen {
						shyPen = c
					}
				}
			}
			wantLas := (ob.kind == "weapon" || ob.kind == "roof") && ob.item != nil && ob.item.Kind == "laser" && !b.holds("laser") && d <= b.P.ShyLaserD
			if !wantLas && b.laserFree(&ob) {
				wantLas = true
			}
			if b.P.ShyWeapon0 && !b.ffaSure && (ob.kind == "weapon" || ob.kind == "roof") {
				wantLas = true
			}
			if near := b.P.ShyNear0; b.P.ShyD > 0 && d > 12 && (b.ffaSure || d > near) && !wantLas && !soft {

				for i := range b.noise {
					n := &b.noise[i]
					if b.t-n.t > 12 {
						continue
					}
					lim := b.P.ShyD
					if n.far {
						lim = b.P.ShyFarD
					}
					db := hyp(ob.pos[0], ob.pos[1], n.pos[0], n.pos[1])
					if db < lim && db < hyp(px, py, n.pos[0], n.pos[1])-8 {
						return
					}
				}
			}
		}
		if pe := b.peer; pe != nil && b.t-pe.Seen < 8 {
			db := hyp(ob.pos[0], ob.pos[1], pe.Pos[0], pe.Pos[1])
			if db < b.P.PeerObjD || db < d-10 {
				return
			}
		}

		eta := b.t + d/me.Stats.WalkSpeed
		if b.zoneAt(eta+3).outside(ob.pos[0], ob.pos[1]) > -5 {
			return
		}

		if zEnd.outside(ob.pos[0], ob.pos[1]) > -8 && d/me.Stats.WalkSpeed*1.6+4 > budget {
			return
		}
		k := 0.6
		if b.practice && b.P.PracNearFarm {
			k = 1.3
		}
		s := ob.value - d*k - shyPen*b.P.ShySoft0
		if ob.kind == b.lastObjKind && hyp(ob.pos[0], ob.pos[1], b.lastObjPos[0], b.lastObjPos[1]) < 3 {
			s += 15
		}
		if s > bs {
			if bo := b.boss; bo != nil && b.E.ObjHide && d > 12 && b.t-bo.Seen < 8 && hyp(ob.pos[0], ob.pos[1], bo.Pos[0], bo.Pos[1]) < 260 {

				eye := [3]float64{bo.Pos[0], bo.Pos[1], bo.Pos[2] + me.Stats.EyeHeight}
				if b.bodyExposed(eye, ob.pos[0], ob.pos[1], b.w.Ground(ob.pos[0], ob.pos[1])) {
					return
				}
			}
			bs = s
			o := ob
			best = &o
		}
	}
	full := b.cubeFull()
	for _, it := range b.cubeL {
		if full {
			break
		}
		if it.Roof && math.Abs(it.Pos[2]-me.Pos[2]) > 3 || b.t < it.Bad {
			if b.t >= it.Bad && b.roofLootOK(it, hpFrac) && !enemiesNear {

				if c := b.w.Nav.Nearest(it.Pos[0], it.Pos[1], 30); c >= 0 {
					if ax, ay := b.w.Nav.Pos(c); hyp(ax, ay, it.Pos[0], it.Pos[1]) <= 22 {
						consider(objective{kind: "roof", pos: [3]float64{ax, ay, me.Pos[2]}, value: 110 - (it.Pos[2]-me.Pos[2])*2.5 - 25, item: it})
					}
				}
			}
			continue
		}
		v := 110.0
		if b.E.PileBonus > 0 {

			n := 0
			for _, o := range b.cubeL {
				if o != it && b.t >= o.Bad && hyp(o.Pos[0], o.Pos[1], it.Pos[0], it.Pos[1]) < 14 {
					n++
				}
			}
			v += b.E.PileBonus * math.Min(float64(n), 6)
		}
		consider(objective{kind: "cube", pos: it.Pos, value: v, item: it})
	}
	if b.P.Sites && !b.practice {
		b.tidySites()
		for _, st := range b.sites {
			if b.t < st.bad {
				continue
			}
			consider(objective{kind: "site", pos: [3]float64{st.pos[0], st.pos[1], b.w.Ground(st.pos[0], st.pos[1])}, value: st.val, site: st})
		}
	}
	for _, it := range b.pickupL {
		if b.t < it.Bad {
			continue
		}
		v := b.wantPickup(it)
		if v <= 0 {
			continue
		}
		val := 70 + 45*v
		if b.slot1Rank() == 0 {
			val = 200 + 30*v
		}
		if it.Kind == "laser" && !b.P.NoLaserRush {
			val = math.Max(val, 270)
		}
		if it.Roof && math.Abs(it.Pos[2]-me.Pos[2]) > 3 {

			h := it.Pos[2] - me.Pos[2]
			need := h/8.0*me.Stats.JetDrain + 12
			blocked := enemiesNear || hpFrac < 0.6
			if b.P.RoofBold && !b.holds("laser") && !b.practice {

				blocked = hpFrac < 0.5 || b.t-b.lastDmgT < 3
				for _, t := range b.trackL {
					if t.NPC || t.Ghost || b.t-t.Seen > 3 {
						continue
					}
					if hyp(px, py, t.Pos[0], t.Pos[1]) < 110 && (b.t < t.Hostile || t.LOS) {
						blocked = true
						break
					}
				}
			}
			if !b.P.Roof || blocked || h < 0 || need > me.MaxEnergy*0.9 {
				continue
			}
			c := b.w.Nav.Nearest(it.Pos[0], it.Pos[1], 30)
			if c < 0 {
				continue
			}
			ax, ay := b.w.Nav.Pos(c)
			if hyp(ax, ay, it.Pos[0], it.Pos[1]) > 22 {
				continue
			}
			consider(objective{kind: "roof", pos: [3]float64{ax, ay, me.Pos[2]}, value: val - h*2.5 - 25, item: it})
			continue
		}
		consider(objective{kind: "weapon", pos: it.Pos, value: val, item: it})
	}
	if hpFrac > 0.45 && !(full && b.slot1Rank() != 0) {
		for _, it := range b.crateL {
			if (it.Roof && it.Pos[2] > me.Pos[2]+3) || b.t < it.Bad {
				continue
			}
			consider(objective{kind: "crate", pos: it.Pos, value: 105, item: it})
		}
	}
	if b.P.FarmNPC && hpFrac > 0.6 && !enemiesNear && !full {
		for _, t := range b.trackL {
			if t.NPC && b.t-t.Seen < 3 {
				consider(objective{kind: "npc", pos: t.Pos, value: b.P.NPCValue, trk: t})
			}
		}
		if b.P.NPCMemory {
			ids := make([]int, 0, len(b.npcMem))
			for id := range b.npcMem {
				ids = append(ids, id)
			}
			sort.Ints(ids)
			for _, id := range ids {
				it := b.npcMem[id]
				if tr := b.tracks[id]; tr != nil && b.t-tr.Seen < 3 {
					continue
				}
				consider(objective{kind: "npcgo", pos: it.Pos, value: b.P.NPCValue - 15})
			}
		}
	}
	if b.P.UseDrops && !(full && hpFrac > 0.8) {
		for i := range b.o.Drops {
			d := &b.o.Drops[i]
			dist := b.roughDist(d.Pos[0], d.Pos[1])
			eta := dist / me.Stats.WalkSpeed
			if eta > d.LandsIn+25 {
				continue
			}
			if b.practice && b.P.PracDropWise > 0 && (!b.dropWanted(hpFrac) || eta+2 < d.LandsIn) {
				continue
			}
			v := 170.0
			if d.Holder != nil && *d.Holder != me.ID {
				v = 60
			}
			if b.P.FFADropFree > 0 && b.ffaSure && !b.practice && (v < 100 || !b.dropClear(d.Pos[0], d.Pos[1], hyp(px, py, d.Pos[0], d.Pos[1]))) {
				continue
			}
			consider(objective{kind: "drop", pos: d.Pos, value: v})
		}
	}
	if !b.P.NoLaserRush && !b.P.NoExplore && !b.holds("laser") && !b.practice && (b.E.ExploreOld && !enemiesNear && b.t > 20 || !b.E.ExploreOld && !b.dangerClose()) {

		if g, ok := b.unseenGoal(zEnd); ok {
			v := 100.0
			if b.slot1Rank() == 0 {
				v = 125
			}
			consider(objective{kind: "explore", pos: [3]float64{g[0], g[1], me.Pos[2]}, value: v})
		}
	} else if b.P.QuietScout > 0 && b.ffaSure && !b.practice && b.quiet && b.holds("laser") && hpFrac > 0.8 && !enemiesNear && b.t-b.lastDmgT > 10 && b.nobodyWithin(px, py, b.P.QuietScoutClear, 12) {

		if g, ok := b.unseenGoal(zEnd); ok {
			consider(objective{kind: "explore", pos: [3]float64{g[0], g[1], me.Pos[2]}, value: b.P.QuietScout})
		}
	} else if b.P.FarmScout > 0 && b.farmOn() && hpFrac > 0.8 && b.t-b.lastDmgT > 10 {

		if g, ok := b.unseenGoal(zEnd); ok {
			consider(objective{kind: "explore", pos: [3]float64{g[0], g[1], me.Pos[2]}, value: b.P.FarmScout})
		}
	}
	if g, ok := b.earGoal(); ok && hpFrac > 0.6 {
		consider(objective{kind: "ear", pos: [3]float64{g[0], g[1], me.Pos[2]}, value: 260})
	}
	if b.P.HuntPassive && !(b.P.FFAPassive && b.ffaSure && !b.practice) && hpFrac > 0.8 && b.t > b.P.HuntAfter && !b.farmPhase && !b.patient {
		for _, t := range b.trackL {
			if t.NPC || b.t-t.Seen > 20 || (b.t > t.Hostile && !b.P.Aggressive && !(b.practice && b.P.PracAggro) && !b.isPrey(t, hyp(px, py, t.Pos[0], t.Pos[1]), hpFrac)) {
				continue
			}
			consider(objective{kind: "hunt", pos: t.Pos, value: 120 - (b.t-t.Seen)*3, trk: t})
		}
	}
	_ = px
	_ = py
	if best != nil {
		b.lastObjKind, b.lastObjPos = best.kind, best.pos
	}
	return best
}

func (b *Bot) zoneGoalInside(zEnd zoneC, snapped [2]float64, marginR, in float64) [2]float64 {
	me := b.me
	px, py := me.Pos[0], me.Pos[1]
	if b.t < b.zoneAltT && b.zoneAltC == zEnd && zEnd.outside(b.zoneAltG[0], b.zoneAltG[1]) <= -in {
		return b.zoneAltG
	}
	nv := b.w.Nav

	if c := nv.Nearest(zEnd.c[0], zEnd.c[1], 80); c >= 0 {
		if cx, cy := nv.Pos(c); zEnd.outside(cx, cy) <= -in {
			b.searchTick = b.o.Tick
			if path, cost := nv.Path(px, py, cx, cy, 30000, b.navExtra()); path != nil {
				depth := math.Min(marginR+4, zEnd.r*0.6)
				g, acc, tot := path[len(path)-1], 0.0, 0.0
				found := false
				for i := 1; i < len(path); i++ {
					tot += hyp(path[i][0], path[i][1], path[i-1][0], path[i-1][1])
					if !found && zEnd.outside(path[i][0], path[i][1]) <= -depth {
						g, acc, found = path[i], tot, true
					}
				}
				if !found {
					acc = tot
				}
				if tot > 1 {
					acc *= cost / tot
				}
				b.zonePathT, b.zonePathG, b.zonePathC = b.t, g, acc
				b.zoneAltG, b.zoneAltT, b.zoneAltC = g, b.t+3, zEnd
				return g
			}
		}
	}
	rT := math.Max(zEnd.r-marginR-4, 0)
	th0 := math.Atan2(py-zEnd.c[1], px-zEnd.c[0])
	type cand struct {
		p [2]float64
		d float64
	}
	var cs []cand
	for _, rf := range []float64{1, 0.8, 0.6, 0.4, 0.2, 0} {
		for _, da := range []float64{0, 0.2, -0.2, 0.4, -0.4, 0.6, -0.6, 0.9, -0.9, 1.2, -1.2, 1.6, -1.6, 2.2, -2.2, math.Pi} {
			x, y := zEnd.c[0]+rT*rf*math.Cos(th0+da), zEnd.c[1]+rT*rf*math.Sin(th0+da)
			if !nv.WalkableMain(x, y) || zEnd.outside(x, y) > -in {
				continue
			}
			bad := false
			for _, q := range b.zoneAltBad {
				if b.t < q[2] && hyp(x, y, q[0], q[1]) < 12 {
					bad = true
				}
			}
			if !bad {
				cs = append(cs, cand{[2]float64{x, y}, b.roughDist(x, y)})
			}
			if rf == 0 {
				break
			}
		}
	}
	if len(cs) == 0 {
		return snapped
	}
	sort.Slice(cs, func(i, j int) bool { return cs[i].d < cs[j].d })
	best, bc := cs[0].p, math.Inf(1)
	for i := 0; i < len(cs) && i < 3; i++ {
		if c := b.pathCostTo(cs[i].p[0], cs[i].p[1]); c >= 0 && c < bc {
			best, bc = cs[i].p, c
		}
	}
	if !math.IsInf(bc, 1) {
		b.zonePathT, b.zonePathG, b.zonePathC = b.t, best, bc
	}
	b.zoneAltG, b.zoneAltT, b.zoneAltC = best, b.t+3, zEnd
	return best
}

func (b *Bot) shySpot() [2]float64 {
	me := b.me
	px, py := me.Pos[0], me.Pos[1]
	if b.hideOn() {
		return b.hideSpot()
	}
	if b.t < b.shyT {
		return b.shyGoal
	}
	b.shyT = b.t + 2.5
	prev, hadPrev := b.shyGoal, b.shyGoal != [2]float64{}
	var oth [][2]float64
	for _, t := range b.trackL {
		if !t.NPC && !t.Ghost && b.t-t.Seen < 12 {
			oth = append(oth, [2]float64{t.Pos[0], t.Pos[1]})
		}
	}
	for i := range b.noise {
		if b.t-b.noise[i].t < 12 {
			oth = append(oth, b.noise[i].pos)
		}
	}
	b.shyGoal = [2]float64{px, py}
	if len(oth) == 0 && !b.x10Any() {
		return b.shyGoal
	}
	score := func(x, y float64) float64 {
		m := 260.0
		for _, o := range oth {
			if d := hyp(x, y, o[0], o[1]); d < m {
				m = d
			}
		}
		return b.x10Spot(x, y, m)
	}
	zs := b.zoneAt(b.t + 25)
	nv := b.w.Nav
	bs := score(px, py) + 12
	in := 0.0
	var zE zoneC
	if b.P.ShyInside > 0 && b.ffaSure {
		zE = b.zoneAt(b.o.Zone.ShrinkEnd + 0.01)
		if b.o.Zone.ShrinkEnd <= b.t {
			zE = zoneC{b.o.Zone.NextCenter, b.o.Zone.NextRadius}
		}
		in = math.Min(b.P.ShyInside, zE.r*0.5)
		if zE.outside(px, py) > -in {
			bs = math.Inf(-1)
		}
	}
	for ring := 1; ring <= 3; ring++ {
		r := 30 * float64(ring)
		for k := 0; k < 12; k++ {
			a := float64(k) * math.Pi / 6
			x, y := px+r*math.Cos(a), py+r*math.Sin(a)
			if !nv.WalkableMain(x, y) || zs.outside(x, y) > -12 {
				continue
			}
			if in > 0 && zE.outside(x, y) > -in && !math.IsInf(bs, -1) {
				continue
			}
			sc := score(x, y) - 0.12*r
			if in > 0 && zE.outside(x, y) > -in {

				sc = -1e6 - hyp(x, y, zE.c[0], zE.c[1])
			}
			if sc > bs {
				bs, b.shyGoal = sc, [2]float64{x, y}
			}
		}
	}
	if b.P.ShyHold > 0 && b.ffaSure && !b.practice && hadPrev && in <= 0 {

		if dp := hyp(px, py, prev[0], prev[1]); dp > 8 && dp < 140 && nv.WalkableMain(prev[0], prev[1]) && zs.outside(prev[0], prev[1]) <= -12 {
			if bs < score(prev[0], prev[1])+b.P.ShyHold {
				b.shyGoal = prev
			}
		}
	}
	return b.shyGoal
}

func (b *Bot) shyMove(zEnd zoneC) [2]float64 {
	me := b.me
	px, py := me.Pos[0], me.Pos[1]
	if b.t < b.shyMoveT && hyp(b.shyMoveG[0], b.shyMoveG[1], px, py) > 10 {
		return b.shyMoveG
	}
	var oth [][2]float64
	for _, t := range b.trackL {
		if !t.NPC && !t.Ghost && b.t-t.Seen < 12 {
			oth = append(oth, [2]float64{t.Pos[0], t.Pos[1]})
		}
	}
	for i := range b.noise {
		if b.t-b.noise[i].t < 12 {
			oth = append(oth, b.noise[i].pos)
		}
	}
	nv := b.w.Nav
	best := [2]float64{px, py}
	bs := math.Inf(-1)
	for k := 0; k < 24; k++ {
		a := float64(k) * math.Pi / 12
		r := 70.0 + 30*float64(k%3)
		x, y := px+r*math.Cos(a), py+r*math.Sin(a)
		if !nv.WalkableMain(x, y) || zEnd.outside(x, y) > -10 {
			continue
		}
		m := 260.0
		for _, o := range oth {
			if d := hyp(x, y, o[0], o[1]); d < m {
				m = d
			}
		}
		s := m - 0.15*r
		if len(b.seen) > 0 {
			cx, cy := int((x-b.w.BMinX)/seenCell), int((y-b.w.BMinY)/seenCell)
			if cx >= 0 && cy >= 0 && cx < b.seenW && cy < b.seenH && !b.seen[cy*b.seenW+cx] {
				s += 40
			}
		}
		if s > bs {
			bs, best = s, [2]float64{x, y}
		}
	}
	b.shyMoveG, b.shyMoveT = best, b.t+6
	return best
}

func (b *Bot) roam(zEnd zoneC) [2]float64 {
	me := b.me
	if b.t < b.roamT && hyp(b.roamGoal[0], b.roamGoal[1], me.Pos[0], me.Pos[1]) > 8 && zEnd.outside(b.roamGoal[0], b.roamGoal[1]) < -5 {
		return b.roamGoal
	}
	nv := b.w.Nav
	best := [2]float64{zEnd.c[0], zEnd.c[1]}
	bs := math.Inf(-1)
	if (!b.P.IdleXUnarmed || !b.holds("laser")) && !b.practice && b.t > 20 {
		if g, ok := b.unseenGoal(zEnd); ok {
			b.roamGoal = g
			b.roamT = b.t + 12
			return g
		}
	}
	for k := 0; k < 24; k++ {
		a := b.rng.Float64() * 2 * math.Pi
		r := zEnd.r * (0.25 + 0.6*b.rng.Float64())
		x, y := zEnd.c[0]+r*math.Cos(a), zEnd.c[1]+r*math.Sin(a)
		if !nv.WalkableMain(x, y) {
			continue
		}
		s := -hyp(x, y, me.Pos[0], me.Pos[1])*0.5 + b.rng.Float64()*40
		if s > bs {
			bs, best = s, [2]float64{x, y}
		}
	}
	b.roamGoal = best
	b.roamT = b.t + 20
	return best
}

func (b *Bot) scan(act *proto.Action, hasGoal bool, goal [2]float64) {
	me := b.me
	base := me.Aim[0]
	if math.Hypot(act.Move[0], act.Move[1]) > 0.3 {
		base = math.Atan2(act.Move[1], act.Move[0])
	}
	switch {
	case b.t < 1.7 || (b.P.Periscope > 0 && b.t < 3.3 && !me.Grounded):

		act.Aim = [2]float64{angDiff(b.spawnYaw+me.Stats.TurnRate*(b.t+0.3), 0), 0}
		if b.cfgN == 0 && b.wallOn {

			switch {
			case b.t >= 0.3 && b.t < 0.6:
				act.Aim[1] = gapCode(b.readyGap)
			case b.t >= 0.85 && b.t < 1.15:
				act.Aim[1] = wallCode(b.wallMed)
			case b.t >= 1.15 && b.t < 1.42:
				act.Aim[1] = wallCode(b.wallMin)
			case b.t >= 1.45 && b.t < 1.68 && b.ffaSure && !b.practice:

				act.Aim[1] = -0.15 - 0.03*Build
			}
		}
		if b.cfgN > 0 {

			switch {
			case b.t >= 0.3 && b.t < 0.9:
				act.Aim[1] = 0.2 + 0.04*float64(b.cfgFirst)
			case b.t >= 1.0 && b.t < 1.6:
				act.Aim[1] = 0.2 + 0.04*math.Min(float64(b.cfgN), 20)
			}
		}
		return
	case b.t-b.lastDmgT < 2.5:
		act.Aim = [2]float64{b.lastDmgDir, 0}
		return
	case b.t-b.hearT < 1.5:
		act.Aim = [2]float64{b.hearDir, 0}
		return
	case b.P.EarP > 0 && b.earT > 0 && b.t-b.earHeardT < 3:
		act.Aim = [2]float64{b.earDir, 0}
		return
	}
	if b.P.SpinScan {

		act.Aim = [2]float64{angDiff(me.Aim[0]+0.36, 0), 0}
		return
	}
	b.scanPhase += 0.1
	amp := 1.9
	if b.P.ScanAhead > 0 && b.ffaSure && !b.practice && b.othersDied > 0 && b.o.Alive <= b.P.EndScan && b.o.Alive > 2 && math.Hypot(act.Move[0], act.Move[1]) > 0.3 {

		amp = b.P.ScanAhead
	}
	off := math.Sin(b.scanPhase*1.1) * amp
	act.Aim = [2]float64{angDiff(base+off, 0), 0}
}

const seenCell = 20.0

func (b *Bot) markSeen() {
	w := b.w
	if b.seen == nil {
		b.seenW = int((w.BMaxX-w.BMinX)/seenCell) + 1
		b.seenH = int((w.BMaxY-w.BMinY)/seenCell) + 1
		b.seen = make([]bool, b.seenW*b.seenH)
	}
	me := b.me
	px, py := me.Pos[0], me.Pos[1]
	ez := me.Pos[2] + me.Stats.EyeHeight
	rng := b.cfg.Vision.Range - 15
	if rng < 60 {
		rng = 160
	}
	half := b.cfg.Vision.FovDeg * math.Pi / 360
	if half <= 0 {
		half = math.Pi / 3
	}

	for k := 0; k < 10; k++ {
		a := me.Aim[0] + (b.rng.Float64()*2-1)*half*0.95
		r := 30 + b.rng.Float64()*(rng-30)
		x, y := px+r*math.Cos(a), py+r*math.Sin(a)
		cx, cy := int((x-w.BMinX)/seenCell), int((y-w.BMinY)/seenCell)
		if cx < 0 || cy < 0 || cx >= b.seenW || cy >= b.seenH || b.seen[cy*b.seenW+cx] {
			continue
		}
		gx := w.BMinX + (float64(cx)+0.5)*seenCell
		gy := w.BMinY + (float64(cy)+0.5)*seenCell
		if hyp(gx, gy, px, py) > rng {
			continue
		}
		if w.Sees(px, py, ez, gx, gy, w.Ground(gx, gy)+1.5) {
			b.seen[cy*b.seenW+cx] = true
		}
	}
}

func (b *Bot) unseenGoal(zEnd zoneC) ([2]float64, bool) {
	var best [2]float64
	if b.P.NoExplore || len(b.seen) == 0 {
		return best, false
	}
	if b.t < b.unseenT {
		cx, cy := int((b.unseenG[0]-b.w.BMinX)/seenCell), int((b.unseenG[1]-b.w.BMinY)/seenCell)
		if cx >= 0 && cy >= 0 && cx < b.seenW && cy < b.seenH && !b.seen[cy*b.seenW+cx] && zEnd.outside(b.unseenG[0], b.unseenG[1]) < -10 {
			return b.unseenG, true
		}
	}
	me := b.me
	nv := b.w.Nav
	w := b.w
	bs := math.Inf(-1)
	for cy := 0; cy < b.seenH; cy++ {
		for cx := 0; cx < b.seenW; cx++ {
			if b.seen[cy*b.seenW+cx] {
				continue
			}
			x := w.BMinX + (float64(cx)+0.5)*seenCell
			y := w.BMinY + (float64(cy)+0.5)*seenCell
			if zEnd.outside(x, y) > -10 || !nv.WalkableMain(x, y) {
				continue
			}
			d := hyp(x, y, me.Pos[0], me.Pos[1])
			if d < 25 {
				b.seen[cy*b.seenW+cx] = true
				continue
			}

			un := 0
			for _, o := range [...][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := cx+o[0], cy+o[1]
				if nx >= 0 && ny >= 0 && nx < b.seenW && ny < b.seenH && !b.seen[ny*b.seenW+nx] {
					un++
				}
			}
			s := -d + float64(un)*12
			if s > bs {
				bs, best = s, [2]float64{x, y}
			}
		}
	}
	if bs == math.Inf(-1) {
		return best, false
	}
	b.unseenG, b.unseenT = best, b.t+6
	return best, true
}

func (b *Bot) dangerRatio(t *Track) float64 {
	thp := t.HP
	if thp <= 0 {
		thp = t.MaxHP
	}
	theirs := thp * b.enemyDps(t, 90)
	ours := b.me.HP * math.Max(b.myDps(t, 90), 25)
	return theirs / math.Max(ours, 1)
}

func (b *Bot) dangerClose() bool {
	if b.t-b.lastDmgT < 3 {
		return true
	}
	px, py := b.me.Pos[0], b.me.Pos[1]
	for _, t := range b.trackL {
		if t.NPC || t.Ghost || b.t-t.Seen > 3 {
			continue
		}
		if hyp(px, py, t.Pos[0], t.Pos[1]) < 110 && (b.t < t.Hostile || t.LOS) {
			return true
		}
	}
	return false
}

func (b *Bot) isBlind(t *Track, d, hpFrac float64) bool {
	if b.P.Ambush <= 0 && b.P.Snipe <= 0 && b.P.Preempt <= 0 {
		return false
	}
	if b.practice || t.NPC || t.Ghost || b.boss != nil || hpFrac < 0.6 || !t.LOS || b.t-t.Seen > 0.3 || !b.holds("laser") {
		return false
	}
	dr := b.dangerRatio(t)
	if b.P.Ambush > 0 && d > 50 && d < b.P.Ambush && dr <= b.P.AmbushRatio {
		brg := math.Atan2(b.me.Pos[1]-t.Pos[1], b.me.Pos[0]-t.Pos[0])
		if math.Abs(angDiff(t.Aim[0], brg)) > b.P.AmbushAng {
			return true
		}
	}
	if b.P.Snipe > 0 && !t.has[2] && d > 110 && d < b.P.Snipe && dr <= 1.5 {
		return true
	}
	if b.P.Preempt > 0 && d > 60 && d < b.P.Preempt && dr <= 1.1 {
		return true
	}
	return false
}

func (b *Bot) isOpening(t *Track, d, hpFrac float64) bool {
	if !b.P.Open || b.P.FFAPassive || !b.ffaSure || b.practice || t.NPC || t.Ghost || !t.LOS || b.t-t.Seen > 0.3 || hpFrac < b.P.OpenHP || b.boss != nil {
		return false
	}
	if b.t-b.lastDmgT < 3 {
		return false
	}
	for _, o := range b.trackL {
		if o != t && !o.NPC && b.t < o.Hostile && b.t-o.Seen < 3 {
			return false
		}
	}
	if b.dangerRatio(t) > b.P.OpenRatio {
		return false
	}
	if las, _ := b.weapon("laser"); las != nil && !b.P.NoOpenLas && d > 25 && d < las.Stats.Range-25 {
		return true
	}
	return b.t-t.busyT < 3 && d < b.P.OpenBusyD
}

func (b *Bot) lateStrike(t *Track, d, hpFrac float64) bool {
	if b.P.LateStrike <= 0 || !b.ffaSure || b.practice || b.o.Alive > b.P.LateStrike || t.NPC || t.Ghost || !t.LOS || b.t-t.Seen > 0.3 {
		return false
	}
	if hpFrac < b.P.LateStrikeHP || d > b.P.LateStrikeD || d < 20 || b.boss != nil {
		return false
	}
	if las, _ := b.weapon("laser"); las == nil || d > las.Stats.Range-15 {
		return false
	}
	hpLim, ratioLim := 1.0, b.P.LateStrikeRatio
	if b.P.LateApproach > 0 && d > 1 {

		ux, uy := (b.me.Pos[0]-t.Pos[0])/d, (b.me.Pos[1]-t.Pos[1])/d
		if t.AvgVel[0]*ux+t.AvgVel[1]*uy >= b.P.LateApproach {
			ar := 1.5
			if b.P.LateApproachR > 0 {
				ar = b.P.LateApproachR
			}
			hpLim, ratioLim = 1.35, math.Max(ratioLim, ar)
		}
	}
	if t.HP > hpLim*b.me.HP || b.dangerRatio(t) > ratioLim {
		return false
	}
	for _, o := range b.trackL {
		if o != t && !o.NPC && !o.Ghost && b.t-o.Seen < 4 {
			return false
		}
	}
	return true
}

func (b *Bot) isPrey(t *Track, d, hpFrac float64) bool {
	if b.P.PreyRatio <= 0 || b.practice || t.NPC || t.Ghost || b.boss != nil || hpFrac < 0.55 || d > 230 {
		return false
	}
	return b.dangerRatio(t) < b.P.PreyRatio
}

func (b *Bot) nobodyWithin(x, y, r, age float64) bool {
	for _, t := range b.trackL {
		if !t.NPC && b.t-t.Seen < age && hyp(x, y, t.Pos[0], t.Pos[1]) < r {
			return false
		}
	}
	for i := range b.noise {
		n := &b.noise[i]
		if b.t-n.t < age && !n.far && hyp(x, y, n.pos[0], n.pos[1]) < r {
			return false
		}
	}
	return true
}

func (b *Bot) dropClear(x, y, myD float64) bool {
	if !b.nobodyWithin(x, y, b.P.FFADropFree, 12) {
		return false
	}
	for _, t := range b.trackL {
		if !t.NPC && b.t-t.Seen < 12 && hyp(x, y, t.Pos[0], t.Pos[1]) < myD+20 {
			return false
		}
	}
	return true
}

func (b *Bot) roofLootOK(it *Item, hpFrac float64) bool {
	if !b.P.RoofLoot || !b.ffaSure || b.practice || !b.P.Roof || !it.Roof {
		return false
	}
	me := b.me
	h := it.Pos[2] - me.Pos[2]
	if h < 3 || h/8.0*me.Stats.JetDrain+12 > me.MaxEnergy*0.9 || hpFrac < 0.6 || b.t-b.lastDmgT < 5 {
		return false
	}
	return b.nobodyWithin(me.Pos[0], me.Pos[1], b.P.RoofLootClear, 8)
}

func (b *Bot) fleeFrom(mechs []*Track, hpFrac float64) *Track {
	if b.P.FleeD <= 0 || !b.ffaSure || b.practice || !b.quiet || b.o.Alive <= 2 {
		return nil
	}
	px, py := b.me.Pos[0], b.me.Pos[1]
	var best *Track
	bd := 1e9
	for _, t := range mechs {
		if t.closeT == 0 || b.t-t.closeT < b.P.FleeT {
			continue
		}
		lim := b.P.FleeD
		if t.has[2] && t.LOS && b.P.FleeLasD > lim {
			lim = b.P.FleeLasD
		}
		d := hyp(px, py, t.Pos[0], t.Pos[1])
		if d > lim || d >= bd {
			continue
		}
		if t.HP < 0.3*t.MaxHP || b.dangerRatio(t) < b.P.FleeRatio {
			continue
		}
		best, bd = t, d
	}
	return best
}

func (b *Bot) fleeGoal(from *Track, threats []threat) [2]float64 {
	me := b.me
	px, py := me.Pos[0], me.Pos[1]
	d0 := hyp(px, py, from.Pos[0], from.Pos[1])
	base := math.Atan2(py-from.Pos[1], px-from.Pos[0])
	zs := b.zoneAt(b.t + 15)
	nv := b.w.Nav
	eye := [3]float64{from.Pos[0], from.Pos[1], from.Pos[2] + 8.0*from.Scale}
	best, bs := [2]float64{}, math.Inf(-1)
	for k := -7; k <= 7; k++ {
		a := base + float64(k)*0.2
		for _, r := range [...]float64{70, 110, 150} {
			x, y := px+r*math.Cos(a), py+r*math.Sin(a)
			if !nv.WalkableMain(x, y) || zs.outside(x, y) > -8 {
				continue
			}
			s := hyp(x, y, from.Pos[0], from.Pos[1]) - d0
			if !b.bodyExposed(eye, x, y, b.w.Ground(x, y)) {
				s += 60
			}
			if rd := b.roughDist(x, y); rd > r*1.1 {
				s -= 40
			}
			for _, t := range b.trackL {
				if t == from || t.NPC || b.t-t.Seen > 8 {
					continue
				}
				if dd := hyp(x, y, t.Pos[0], t.Pos[1]); dd < 110 {
					s -= 110 - dd
				}
			}
			for i := range b.noise {
				if b.t-b.noise[i].t > 12 {
					continue
				}
				if dd := hyp(x, y, b.noise[i].pos[0], b.noise[i].pos[1]); dd < 110 {
					s -= (110 - dd) * 0.7
				}
			}
			if s > bs {
				bs, best = s, [2]float64{x, y}
			}
		}
	}
	if math.IsInf(bs, -1) {
		return b.findSafe(threats)
	}
	return best
}

func (b *Bot) cubeFull() bool {
	return b.practice && b.P.PracCubeMax > 0 && int(b.me.Cubes) >= b.P.PracCubeMax
}

func (b *Bot) dropWanted(hpFrac float64) bool {
	if !b.practice || b.P.PracDropWise <= 0 {
		return true
	}
	return hpFrac < b.P.PracDropWise || b.me.Cubes+b.cfg.Drops.Cubes <= b.cfg.Cubes.StatCap
}

func (b *Bot) standTarget(mechs []*Track) *Track {
	me := b.me
	var best *Track
	bt := math.Inf(1)
	for _, t := range mechs {
		if !t.LOS || b.t-t.Seen > 0.3 {
			continue
		}
		d := hyp(me.Pos[0], me.Pos[1], t.Pos[0], t.Pos[1])
		md := b.myDps(t, d)
		if md <= 0 {
			continue
		}
		ttk := t.HP / md
		if b.t-t.HitMe < 3 {
			ttk *= 0.8
		}
		if ttk < bt {
			bt, best = ttk, t
		}
	}
	return best
}
