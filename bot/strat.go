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

// laserIn: seconds until this mech's laser could next fire (large if it has none).
func (b *Bot) laserIn(t *Track) float64 {
	if t.NPC || !t.has[2] {
		return 1e9
	}
	las := b.cfg.Weapons["laser"]
	chg := las.Charge
	if t.Cubes >= 3 {
		chg *= 0.6 // may have bought faster capacitors
	}
	if t.Charging {
		return math.Max(0, t.lasChargeT+chg-b.t)
	}
	cd := math.Max(0, t.lasFireT+las.Cooldown-b.t)
	return cd + chg
}

// enemyDpsNL is enemyDps without the laser (which is handled by timing).
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

// enemyDps estimates what a mech can do to us per second at range d while it has a line on us.
func (b *Bot) enemyDps(t *Track, d float64) float64 {
	if t.NPC {
		if d < 60 {
			return 12
		}
		return 0
	}
	mul := b.cubeMul(t.Cubes) * (1 - b.me.Stats.Armor)
	// every upgrade threshold it has passed probably went partly into firepower
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
			// we dodge: their rounds only land when the flight time is short
			ph := clamp(0.9-(d-40)*0.0125, 0.06, 0.9)
			dps += 105 * ph
		}
	}
	if t.has[2] && d < b.cfg.Weapons["laser"].Range {
		if b.holds("lightning") && d < 42 && b.me.Energy > 15 {
			dps += 20 // our arcs keep its laser offline
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

// myDps estimates our damage per second against a mech at range d with a clear line.
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

// gunHitProb: chance one of our rounds lands at range d, using what we have
// measured against this particular target (does it dodge?).
func (b *Bot) gunHitProb(t *Track, d float64) float64 {
	base := func(d float64) float64 { return clamp(0.97-(d-40)*0.0035, 0.3, 0.97) }
	pb := base(d)
	if t.NPC || t.shots < 6 {
		return pb
	}
	dm := hyp(t.Pos[0], t.Pos[1], b.me.Pos[0], b.me.Pos[1])
	k := clamp((t.hits/t.shots)/base(dm), 0.05, 1.1)
	// up close nobody dodges
	k += (1 - k) * clamp((60-d)/30, 0, 1)
	return clamp(pb*k, 0.02, 0.97)
}

// bestRange picks the fighting distance where our damage most outweighs theirs.
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

// heldRank is how much we value keeping the weapon in a slot.
func (b *Bot) heldRank(w *proto.WeaponObs) float64 {
	if w == nil {
		return 0
	}
	if w.Kind == "gun" {
		// the gun is only ever given up for the laser+lightning pairing
		if (b.P.DropGunForLtn || (b.practice && b.P.PracLasLtn)) && b.holds("laser") {
			return 0.5
		}
		return 99
	}
	return b.rank(w.Kind)
}

// rank is how much we value a weapon kind (squad matches may value lightning differently).
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

// wantPickup: positive value if this pickup would improve the loadout.
func (b *Bot) wantPickup(it *Item) float64 {
	if it.Kind == "gun" || b.holds(it.Kind) {
		return 0
	}
	v := b.rank(it.Kind) - b.slot1Rank()
	if it.Kind == "lightning" && b.slot1Rank() == 0.5 {
		v = 1.2 // lightning in place of the gun, next to a laser
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
	// which slot gives way: an empty one, else the least valued
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
			return // cannot swap a charging laser
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

	// ---- threats
	var ghost *Track // someone hitting us from where we cannot see
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
			mechs = append(mechs, t)
			if t.LOS && hostile && (facing < 0.6 || b.t-t.HitMe < 3) {
				incoming += b.enemyDps(t, d)
				nHostile++
			}
		}
	}

	b.farmPhase = b.P.PracFarmCubes > 0 && !b.sawOtherDeath && int(me.Cubes) < b.P.PracFarmCubes && b.t < b.P.PracFarmT
	// ---- someone far stronger than us nearby: keep out of its sight
	// squad matches (and any match until another mech's death shows it is a
	// free-for-all) run on the settings the scored practice build used, so that
	// later free-for-all changes cannot move the practice score
	b.E = effP{b.P.BossRatio, b.P.HardRatio, b.P.BossObjD, b.P.BossAvoidD, b.P.BossSeenD, b.P.PileBonus, b.P.ExploreOld, b.P.Calm, b.P.ObjHide}
	if !b.ffaSure && !b.P.NoLegacy {
		b.E = effP{0.85, 1.5, 160, 150, 1e9, 0, true, false, false}
	}
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
					continue // it is already on us and not much stronger: an answered fight beats a walk to cover
				}
				if b.E.HardRatio > 0 && dr <= b.E.HardRatio {
					// about our strength: give it room, but it does not empty the neighbourhood
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
	boss := b.boss
	b.patient = b.P.Endgame > 0 && b.sawOtherDeath && o.Alive <= b.P.Endgame && o.Alive > 2
	// ---- target choice
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
		if v > bestV {
			bestV, tgt = v, t
		}
	}
	if tgt != nil {
		b.targetID = tgt.ID
	} else {
		b.targetID = -1
	}

	// ---- zone
	zEnd := b.zoneAt(o.Zone.ShrinkEnd + 0.01)
	if o.Zone.ShrinkEnd <= b.t {
		zEnd = zoneC{o.Zone.NextCenter, o.Zone.NextRadius}
	}
	dNext := hyp(px, py, zEnd.c[0], zEnd.c[1])
	marginR := math.Min(14, zEnd.r*0.35)
	needIn := dNext - (zEnd.r - marginR) // metres to travel to be safely inside the next circle
	zoneGo := false
	zoneLate := false
	var zoneGoal [2]float64
	budget := 1e9 // spare seconds before we must start walking in
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
		// real walking distance, refreshed now and then
		if (b.t-b.zonePathT > 1.5 || hyp(zoneGoal[0], zoneGoal[1], b.zonePathG[0], b.zonePathG[1]) > 25) && b.searchTick != o.Tick-2 {
			b.zonePathT = b.t
			b.zonePathG = zoneGoal
			b.zonePathC = b.pathCostTo(zoneGoal[0], zoneGoal[1])
			if b.zonePathC < 0 {
				b.zonePathC = needIn * 2.5
			}
		}
		travel := b.zonePathC/me.Stats.WalkSpeed*1.1 + 3
		// when does the edge reach us?
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

	// ---- fight or flight
	engage := false
	myTTK, theirTTK := 1e9, 1e9
	prefD, adv := 0.0, 0.0
	if tgt != nil {
		d := hyp(px, py, tgt.Pos[0], tgt.Pos[1])
		prefD, adv = b.bestRange(tgt)
		// judge the fight at the range we would actually hold
		dEval := d
		if math.Abs(d-prefD) < 60 || prefD > d {
			dEval = prefD
		}
		md := b.myDps(tgt, dEval)
		inc := b.enemyDps(tgt, dEval)
		if b.peekOn() && tgt.has[2] && b.coverNear {
			// with a corner to work from we only show ourselves while its laser is down
			inc = b.enemyDpsNL(tgt, dEval)*0.7 + 35
			md *= 0.7
		}
		if md > 0 {
			myTTK = tgt.HP / md
		}
		// others already shooting at us
		for _, t := range mechs {
			if t != tgt && t.LOS && b.t < t.Hostile {
				inc += b.enemyDps(t, hyp(px, py, t.Pos[0], t.Pos[1]))
			}
		}
		theirTTK = me.HP / math.Max(inc, 1)
		er := b.P.EngageRatio
		if b.P.EarlyEngage > 0 && !b.practice && o.Alive > b.P.EarlyAlive {
			// crowded early game: an even duel mostly feeds whoever shows up third
			er *= b.P.EarlyEngage
		}
		engage = myTTK < er*theirTTK || (adv > 25 && er >= b.P.EngageRatio)
		if nHostile >= 2 && myTTK > 2.5 {
			engage = engage && myTTK*1.5 < theirTTK
		}
	}
	_ = adv
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
	} else if hpFrac > b.P.ReengageHP || (len(mechs) == 0 && b.t-b.lastDmgT > 6 && hpFrac > 0.55) {
		b.retreating = false
	}
	if tgt != nil && !engage && !finishing && (incoming > 0 || b.t-b.lastDmgT < 3) {
		// outgunned right now: break contact (without latching the retreat flag)
	}

	ctx := &moveCtx{threats: threats, wGoal: b.P.WGoal, arrive: 1.5}
	mode := "roam"
	var goal [2]float64
	hasGoal := false
	var farm *Item
	var farmNPC *Track

	underFire := b.t-b.lastDmgT < 2.5
	// a weapon lying a few steps away comes before the fight it would win
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
			// not into the arms of whoever is shooting
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
			gd, grab = d, it
		}
		if grab == nil && b.P.GrabCube > 0 {
			// a cube a few steps away is health and damage for good: two seconds well spent
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
				gd, grab = d, it
			}
		}
	}
	switch {
	case zoneUrgent:
		mode = "zone"
		goal, hasGoal = zoneGoal, true
		ctx.boost = me.Energy > 8
		ctx.wGoal = 4
		ctx.mustMove = true
		if tgt != nil {
			ctx.target = tgt
			ctx.strafe = false
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
		// being hit by someone unseen: never stand there, get something solid in between
		mode = "evade"
		sp := b.findSafe(threats)
		goal, hasGoal = sp, true
		ctx.hide = true
		ctx.boost = me.Energy > 10
		ctx.wGoal = 1.5
		ctx.mustMove = true
	case boss == nil && b.peer != nil && peerD < b.P.PeerAvoidD && b.peer.LOS && b.t-b.peer.Seen < 1.5 && !(tgt == b.peer && finishing) && tgt != b.peer:
		// an equal looking our way from close: step out of its line, keep what we were doing nearby
		mode = "avoid"
		sp := b.findSafe(threats)
		goal, hasGoal = sp, true
		ctx.hide = true
		ctx.target = tgt
		ctx.wGoal = 1.5
		ctx.mustMove = true
	case boss != nil && (bossD < b.E.BossAvoidD || (boss.LOS && b.t-boss.Seen < 1.5 && bossD < b.E.BossSeenD && !(b.E.Calm && b.t-boss.HitMe > 6 && boss.aimedAt < 0.3))) && !(tgt == boss && finishing):
		// outclassed: break its line and open the distance; shoot only if it costs nothing
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
	case b.retreating || (tgt != nil && !engage && (underFire || incoming > 0)):
		mode = "retreat"
		sp := b.findSafe(threats)
		goal, hasGoal = sp, true
		if !underFire && incoming == 0 && !b.P.NoRetreatCubes {
			// a cube within a few steps is health now and for good: take it on the way
			bd := 28.0
			for _, it := range b.cubeL {
				if b.t < it.Bad || (it.Roof && math.Abs(it.Pos[2]-me.Pos[2]) > 3) {
					continue
				}
				if d := hyp(px, py, it.Pos[0], it.Pos[1]); d < bd {
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
			// wounded and alone: stay on it so it cannot break off and repair
			prefD = b.P.ChaseD
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
		}
		d := hyp(px, py, tgt.Pos[0], tgt.Pos[1])
		if prefD < 50 && d > prefD+12 && d < 120 && me.Energy > 45 {
			ctx.boost = true // close the gap for the short-range weapon
		}
		if prefD > d+25 && me.Energy > 30 {
			ctx.boost = true // open the range back up
		}
		if chase && d > prefD+15 && me.Energy > 35 {
			ctx.boost = true
		}
		if zoneGo {
			// drift toward the zone while fighting
			goal, hasGoal = zoneGoal, true
			ctx.wGoal = 0.6
		}
	default:
		ob := (*objective)(nil)
		hurt := hpFrac < 0.7 && b.t-b.lastDmgT < 40 && len(threats) > 0
		b.nearOnly = hurt && !b.P.NoRetreatCubes
		if !hurt || b.nearOnly {
			ob = b.pickObjective(zEnd, hpFrac, len(mechs) > 0, budget)
		}
		b.nearOnly = false
		if zoneGo && ob == nil {
			mode = "zone"
			goal, hasGoal = zoneGoal, true
			ctx.wGoal = 2
			ctx.boost = me.Energy > 60 && needIn > 60
		} else if hurt && ob == nil {
			// hurt and unobserved: stay out of sight until repaired
			mode = "heal"
			goal, hasGoal = b.findSafe(threats), true
			ctx.hide = true
			ctx.arrive = 2
		} else if ob != nil {
			mode = ob.kind
			goal, hasGoal = [2]float64{ob.pos[0], ob.pos[1]}, true
			switch ob.kind {
			case "crate":
				farm = ob.item
				ctx.arrive = 14
				cp := [3]float64{ob.item.Pos[0], ob.item.Pos[1], ob.item.Pos[2] + 1.2}
				// closer than this the gun cannot be depressed onto it
				minD := (me.Stats.MuzzleHeight-1.2+me.Pos[2]-ob.item.Pos[2])/math.Tan(math.Max(0.2, -me.Stats.MinPitch*0.85)) + 1.5
				if minD < 8 {
					minD = 8
				}
				ctx.arrive = math.Max(14, minD+5)
				if !b.w.Clear(b.muz[0], b.muz[1], b.muz[2], cp[0], cp[1], cp[2]) {
					ctx.arrive = minD + 1 // no line on it from here: walk up to it
				}
				ctx.keepOff = minD + 1
				if dC := hyp(px, py, cp[0], cp[1]); dC < minD {
					// too close to bring the gun down onto it: step straight back
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
					ob.item.Bad = b.t + 90 // taking too long: forget it for a while
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
					b.climbT = b.t
					b.climbPos = ob.item.Pos
					b.climbFrom = [2]float64{px, py}
				}
				if hyp(goal[0], goal[1], px, py) < 30 {
					ctx.boost = false // arrive with a full tank
				}
			case "cube":
				ctx.arrive = 1.0
				if ob.item != nil {
					// a cube we cannot reach (ledge, under a roof, already gone) must not hold us
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
						st.bad = b.t + 120 // cannot get there: give it up
					}
				}
			case "hunt":
				ctx.arrive = 30
			case "drop":
				ctx.arrive = 2
			default:
				ctx.arrive = 1.0
			}
			if me.Energy > 75 && hyp(goal[0], goal[1], px, py) > 40 && ob.kind != "roof" {
				ctx.boost = true
			}
		} else {
			goal, hasGoal = b.roam(zEnd), true
			ctx.arrive = 6
		}
	}
	b.mode = mode
	if hasGoal {
		ctx.goal = &goal
	}

	// ---- enclosed pocket (stadium bowl, courtyard): walk to the cheapest jet hop out
	nvw := b.w.Nav
	if myc := nvw.Cell(px, py); b.hopFly || (myc >= 0 && nvw.Walk[myc] && nvw.Comp[myc] != nvw.Main && me.Grounded) {
		if !b.hopFly {
			exits := nvw.Exits[nvw.Comp[myc]]
			var be *geo.Exit
			bc := math.Inf(1)
			for i := range exits {
				e := &exits[i]
				d := hyp(e.LX, e.LY, px, py)
				// prefer exits that lead toward where we want to be
				toward := 0.0
				if hasGoal {
					toward = hyp(e.TX, e.TY, goal[0], goal[1]) * 0.25
				}
				c := d*1.4 + e.Energy*1.2 + toward
				if b.hop != nil && e.LX == b.hop.LX && e.LY == b.hop.LY {
					c -= 25 // stick with the current choice
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

	// ---- unstick
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
		// flight controller: stay above everything still between us and the landing point
		e := b.hop
		dx, dy := e.TX-px, e.TY-py
		l := math.Max(math.Hypot(dx, dy), 0.01)
		ahead := b.w.MaxTopAlong(px, py, e.TX, e.TY)
		landed := me.Grounded && b.t-b.hopT > 0.4
		switch {
		case landed && (nvw.WalkableMain(px, py) || l < 3):
			b.hopFly, b.hop = false, nil
		case landed || b.t-b.hopT > 14:
			// came down short: try again (elsewhere if it keeps failing)
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
		// opening look: hop up on the jets and turn once, to see weapons and crates over the rooftops
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
		climbing = true // skip steering this decision
	}
	if climbing {
		b.stuckT = 0
	} else if b.t < b.unstickT {
		mx, my = b.unstickDir[0], b.unstickDir[1]
		boost = true
		if me.Grounded {
			act.Jump = true
		}
	} else if b.stuckT > 0.5 {
		// mark what is ahead as blocked and try a jump-jet over or around it
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
	if !me.Grounded && b.t >= b.unstickT && !climbing {
		// in the air without wanting to be: no jets
		boost = false
	}
	act.Move = proto.Vec2{mx, my}
	act.Boost = boost
	b.wantMove = math.Hypot(mx, my) > 0.3
	if b.wantMove {
		b.lastMove = [2]float64{mx, my}
	}

	// ---- aim and fire
	b.lasCrate = false
	b.lasFree = true
	for _, t := range b.trackL {
		if !t.NPC && b.t-t.Seen < 4 {
			b.lasFree = false
		}
	}
	shot := false
	var shootT *Track
	// prefer a visible mech we can hit; finishing blows first
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
	holdFire := mode == "retreat" && b.retreating && !underFire && incoming == 0 // let regen start
	// a drone that is hitting us: it blocks repairs, follows at walking pace and is
	// eight gun rounds from dead (lost a recorded match standing next to one at 163 HP)
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
			// not in view: turn to it
			y, p, _ := b.aimAt([3]float64{biter.Pos[0], biter.Pos[1], biter.Pos[2] + 1.5})
			act.Aim = [2]float64{y, p}
			shot = true
		}
		npcShot = true
		if las, li := b.weapon("laser"); las != nil && las.Charge > 0 && !act.Fire[li] {
			act.Fire[li] = true // keep any charge we were holding
		}
	} else if tgt != nil && !holdFire {
		// no line: pre-aim where it is
		y, p, _ := b.aimAt(b.predict(tgt, latency))
		act.Aim = [2]float64{y, p}
		if las, li := b.weapon("laser"); las != nil && las.Charge > 0 {
			act.Fire[li] = true // keep the charge
		} else if las != nil && las.Cooldown <= 0.11 && b.t-tgt.Seen < 0.3 && mode == "fight" && hyp(px, py, tgt.Pos[0], tgt.Pos[1]) < las.Stats.Range &&
			(!b.peekOn() || !(tgt.has[2] && tgt.Charging)) {
			act.Fire[li] = true // charge behind cover; step out when it is ready
		}
		shot = true
	}
	healing := hpFrac < 0.85 && b.t-b.lastDmgT < 40
	farmShot := false
	if !shot && shootT == nil && healing {
		// keep still on the trigger: dealing damage restarts the repair timer
		for _, t := range b.trackL {
			if t.NPC && t.LOS && b.t-t.Seen < 0.3 && b.t-t.HitMe < 2 {
				shot = b.shootMech(act, t)
				break
			}
		}
		// (crates do not count: in recorded matches repair kept running through crate hits)
		if !shot && !b.P.OldReload {
			if farm != nil {
				shot = b.shootCrate(act, farm)
			}
			if !shot && !b.retreating {
				var bc *Item
				bd := 60.0
				for _, c := range b.crateL {
					if c.Roof && c.Pos[2] > me.Pos[2]+3 {
						continue
					}
					d := hyp(c.Pos[0], c.Pos[1], px, py)
					if d < bd && b.t-c.Seen < 0.5 && b.t >= c.Bad {
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
		// farm: nearest NPC shooting us or chosen, then crates
		if farmNPC != nil && farmNPC.LOS {
			shot = b.shootMech(act, farmNPC)
		}
		if !shot {
			for _, t := range b.trackL {
				if t.NPC && t.LOS && b.t-t.Seen < 0.3 && (b.t-t.HitMe < 3 || (b.P.FarmNPC && hpFrac > 0.5 && hyp(px, py, t.Pos[0], t.Pos[1]) < 60)) {
					shot = b.shootMech(act, t)
					break
				}
			}
		}
		look := b.P.ScanFirst && b.t < 1.7 && (b.ffaSure || b.P.ScanAlways)
		if !shot && farm != nil && !look {
			shot = b.shootCrate(act, farm)
		}
		if !shot && !b.retreating && !look {
			// opportunistic: any crate in easy reach
			var bc *Item
			bd := 110.0
			for _, c := range b.crateL {
				if c.Roof && c.Pos[2] > me.Pos[2]+3 {
					continue // what falls out would stay on the roof
				}
				d := hyp(c.Pos[0], c.Pos[1], px, py)
				if d < bd && b.t-c.Seen < 0.5 && b.t >= c.Bad {
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
		act.Fire[li] = true // never let go without a target
	}
	if !shot {
		b.scan(act, hasGoal, goal)
	}
	// reload when quiet
	if gun, _ := b.weapon("gun"); gun != nil && gun.Reload <= 0 && gun.Ammo < gun.Stats.Mag {
		// (a crate or drone under fire is not quiet: reloading after every round made farming crawl)
		quiet := shootT == nil && (tgt == nil || !tgt.LOS) && !farmShot && !npcShot
		if quiet {
			b.quietT += 0.1
		} else {
			b.quietT = 0
		}
		// topping up a nearly full magazine locked the gun for 2.4 s every time a
		// single round had been fired: only reload when it is worth the pause
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
	if Debug && b.P.NoFire {
		fmt.Fprintf(os.Stderr, "    -> move=(%.2f,%.2f) mode=%s hp=%.0f\n", act.Move[0], act.Move[1], mode, me.HP)
	}
}

// fair reports whether a mech is a legitimate target right now: it is hostile
// to us, or weak enough to be worth starting a fight over.
func (b *Bot) fair(t *Track, d, hpFrac float64) bool {
	if b.patient {
		// three left: let the other two settle it unless one is on us
		return b.t-t.HitMe < 4 || (b.t < t.Hostile && d < 70) || (t.HP < 0.3*t.MaxHP && d < 130 && hpFrac > 0.5)
	}
	if b.farmPhase {
		if b.practice {
			return b.t-t.HitMe < 4 && d < 100
		}
		// not known to be a squad match (yet): answer whoever has hit us, at any range
		return b.t < t.Hostile
	}
	if b.t < t.Hostile {
		return true
	}
	if b.P.Aggressive {
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
	if b.practice && b.P.PracAggro && hpFrac > b.P.PracAggroHP && d < b.P.PracAggroD {
		return true
	}
	if t.HP < 0.4*t.MaxHP && hpFrac > 0.6 && d < 150 {
		return true
	}
	return false
}

// prefRange picks the fighting distance against a target from both loadouts.
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
		return 45 // close on laser users: their beam does not care about range, our gun does
	}
	return b.P.PrefRangeGun
}

// findSafe looks for a nearby spot out of every known enemy's sight.
func (b *Bot) findSafe(threats []threat) [2]float64 {
	me := b.me
	px, py := me.Pos[0], me.Pos[1]
	if b.t < b.safeT {
		// keep the spot while it is still hidden
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
			// do not run through the enemy
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
		// nothing close breaks every hostile line (open ground): commit to the nearest
		// cover further out instead of dithering where we stand
		var hs [2]*threat // the two nearest hostile mechs: enough to pick a side
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
					// not toward the enemy
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
	consider := func(ob objective) {
		d := b.roughDist(ob.pos[0], ob.pos[1])
		if b.nearOnly && (d > 35 || (ob.kind != "cube" && ob.kind != "crate")) {
			// repairing: only what is a few steps away and does not interrupt it -
			// except a weapon while we have none worth the name
			if !(b.P.RoofBold && !b.practice && (ob.kind == "weapon" || ob.kind == "roof") && d < 70 && !b.holds("laser")) {
				return
			}
		}
		if bo := b.boss; bo != nil {
			// nothing is worth walking toward the strongest mech around
			db := hyp(ob.pos[0], ob.pos[1], bo.Pos[0], bo.Pos[1])
			if db < b.E.BossObjD || db < hyp(px, py, bo.Pos[0], bo.Pos[1])-15 {
				return
			}
		}
		if pe := b.peer; pe != nil && b.t-pe.Seen < 8 {
			db := hyp(ob.pos[0], ob.pos[1], pe.Pos[0], pe.Pos[1])
			if db < b.P.PeerObjD || db < d-10 {
				return
			}
		}
		// reachable before the zone takes it?
		eta := b.t + d/me.Stats.WalkSpeed
		if b.zoneAt(eta+3).outside(ob.pos[0], ob.pos[1]) > -5 {
			return
		}
		// outside the next circle: only if the detour fits in the spare time
		if zEnd.outside(ob.pos[0], ob.pos[1]) > -8 && d/me.Stats.WalkSpeed*1.6+4 > budget {
			return
		}
		k := 0.6
		if b.practice && b.P.PracNearFarm {
			k = 1.3 // hunters are coming to us: do not wander off for loot
		}
		s := ob.value - d*k
		if ob.kind == b.lastObjKind && hyp(ob.pos[0], ob.pos[1], b.lastObjPos[0], b.lastObjPos[1]) < 3 {
			s += 15 // finish what we started
		}
		if s > bs {
			if bo := b.boss; bo != nil && b.E.ObjHide && d > 12 && b.t-bo.Seen < 8 && hyp(ob.pos[0], ob.pos[1], bo.Pos[0], bo.Pos[1]) < 260 {
				// work it cannot watch: standing in a stronger mech's view is how first strikes land
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
	for _, it := range b.cubeL {
		if it.Roof && math.Abs(it.Pos[2]-me.Pos[2]) > 3 || b.t < it.Bad {
			continue
		}
		v := 110.0
		if b.E.PileBonus > 0 {
			// a dead mech leaves a heap: the heap is the prize, not the nearest cube of it
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
			val = 200 + 30*v // a second weapon decides fights: get one first
		}
		if it.Kind == "laser" && !b.P.NoLaserRush {
			val = math.Max(val, 270) // every match we won, we held a laser early
		}
		if it.Roof && math.Abs(it.Pos[2]-me.Pos[2]) > 3 {
			// on a roof: climb for it when it is reachable and nobody is around
			h := it.Pos[2] - me.Pos[2]
			need := h/8.0*me.Stats.JetDrain + 12
			blocked := enemiesNear || hpFrac < 0.6
			if b.P.RoofBold && !b.holds("laser") && !b.practice {
				// a laser decides matches and many lie on low roofs: only an enemy that is
				// actually close and looking, or fire we are taking, keeps us on the ground
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
	if hpFrac > 0.45 {
		for _, it := range b.crateL {
			if (it.Roof && it.Pos[2] > me.Pos[2]+3) || b.t < it.Bad {
				continue
			}
			consider(objective{kind: "crate", pos: it.Pos, value: 105, item: it})
		}
	}
	if b.P.FarmNPC && hpFrac > 0.6 && !enemiesNear {
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
	if b.P.UseDrops {
		for i := range b.o.Drops {
			d := &b.o.Drops[i]
			dist := b.roughDist(d.Pos[0], d.Pos[1])
			eta := dist / me.Stats.WalkSpeed
			if eta > d.LandsIn+25 {
				continue
			}
			v := 170.0
			if d.Holder != nil && *d.Holder != me.ID {
				v = 60
			}
			consider(objective{kind: "drop", pos: d.Pos, value: v})
		}
	}
	if !b.P.NoLaserRush && !b.P.NoExplore && !b.holds("laser") && !b.practice && (b.E.ExploreOld && !enemiesNear && b.t > 20 || !b.E.ExploreOld && !b.dangerClose()) {
		// no laser yet: looking over unseen ground competes with farming
		if g, ok := b.unseenGoal(zEnd); ok {
			v := 100.0
			if b.slot1Rank() == 0 {
				v = 125
			}
			consider(objective{kind: "explore", pos: [3]float64{g[0], g[1], me.Pos[2]}, value: v})
		}
	}
	if b.P.HuntPassive && hpFrac > 0.8 && b.t > b.P.HuntAfter && !b.farmPhase && !b.patient {
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

// roam wanders inside the next circle, preferring spots away from the centre.
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

// scan sweeps the torso to look around when there is nothing to shoot.
func (b *Bot) scan(act *proto.Action, hasGoal bool, goal [2]float64) {
	me := b.me
	base := me.Aim[0]
	if math.Hypot(act.Move[0], act.Move[1]) > 0.3 {
		base = math.Atan2(act.Move[1], act.Move[0])
	}
	switch {
	case b.t < 1.7 || (b.P.Periscope > 0 && b.t < 3.3 && !me.Grounded):
		// first look: one full turn to find weapons and crates nearby
		act.Aim = [2]float64{angDiff(b.spawnYaw+me.Stats.TurnRate*(b.t+0.3), 0), 0}
		return
	case b.t-b.lastDmgT < 2.5:
		act.Aim = [2]float64{b.lastDmgDir, 0}
		return
	case b.t-b.hearT < 1.5:
		act.Aim = [2]float64{b.hearDir, 0}
		return
	}
	if b.P.SpinScan {
		// keep turning: every direction gets a look about every two seconds
		act.Aim = [2]float64{angDiff(me.Aim[0]+0.36, 0), 0}
		return
	}
	b.scanPhase += 0.1
	off := math.Sin(b.scanPhase*1.1) * 1.9
	act.Aim = [2]float64{angDiff(base+off, 0), 0}
}

const seenCell = 20.0

// markSeen records which patches of ground are in view right now.
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
	// a handful of rays per decision keeps this cheap; the torso sweep does the rest
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

// unseenGoal is the best patch of ground we have not looked at yet inside the
// next circle: weapons and crates are only known once seen.
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
				b.seen[cy*b.seenW+cx] = true // standing on it
				continue
			}
			// prefer near, and patches whose neighbours are unseen too
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

// dangerRatio compares what a mech could do to us with what we could do to it
// in a straight fight at medium range (health times damage rate, theirs over ours).
func (b *Bot) dangerRatio(t *Track) float64 {
	thp := t.HP
	if thp <= 0 {
		thp = t.MaxHP
	}
	theirs := thp * b.enemyDps(t, 90)
	ours := b.me.HP * math.Max(b.myDps(t, 90), 25)
	return theirs / math.Max(ours, 1)
}

// dangerClose: an enemy that matters right now - close, and either hostile or with a line on us.
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

// isBlind: openers. Recorded showdowns: whoever lands the first hit wins 70% of
// early duels, and the leading bots open on mechs that are not looking at them.
//
//	Ambush  - target inside this range is looking away (first beam is free, and on its rear)
//	Snipe   - target has no laser of its own and is far enough that its gun is weak
//	Preempt - any near-equal target inside this range
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

// isOpening: in recorded matches whoever landed the first laser hit took a quarter
// of the damage it dealt, and a mech already fighting someone else died four times
// in five when a third joined in. So strike first when nobody is on us.
func (b *Bot) isOpening(t *Track, d, hpFrac float64) bool {
	if !b.P.Open || !b.ffaSure || b.practice || t.NPC || t.Ghost || !t.LOS || b.t-t.Seen > 0.3 || hpFrac < b.P.OpenHP || b.boss != nil {
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

// isPrey: a mech we clearly outclass (kills are the main source of cubes).
func (b *Bot) isPrey(t *Track, d, hpFrac float64) bool {
	if b.P.PreyRatio <= 0 || b.practice || t.NPC || t.Ghost || b.boss != nil || hpFrac < 0.55 || d > 230 {
		return false
	}
	return b.dangerRatio(t) < b.P.PreyRatio
}
