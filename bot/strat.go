package bot

import (
	"fmt"
	"math"
	"os"

	"mechbot/proto"
)

func (b *Bot) cubeMul(c float64) float64 {
	return 1 + b.cfg.Cubes.DamagePerCube*math.Min(c, b.cfg.Cubes.StatCap)
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
func (b *Bot) myDps(t *Track, d float64) float64 {
	mul := b.dmgMul()
	dps := 0.0
	for _, w := range b.me.Weapons {
		if w == nil {
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
	for i, c := range o.Choices {
		r := 1 << 20
		for k, id := range b.P.UpgradePrio {
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

func (b *Bot) slot1Rank() float64 {
	if len(b.me.Weapons) > 1 && b.me.Weapons[1] != nil {
		return b.P.WeaponRank[b.me.Weapons[1].Kind]
	}
	return 0
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
	return b.P.WeaponRank[it.Kind] - b.slot1Rank()
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
	// do not swap a charging laser
	if len(me.Weapons) > 1 && me.Weapons[1] != nil {
		if me.Weapons[1].Kind == "laser" && me.Weapons[1].Charge > 0 {
			return
		}
		one := 1
		act.Drop = &one
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
		if age < 0.3 {
			mechs = append(mechs, t)
			if t.LOS && hostile && (facing < 0.6 || b.t-t.HitMe < 3) {
				incoming += b.enemyDps(t, d)
				nHostile++
			}
		}
	}

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
		if b.t-b.zonePathT > 1.5 || hyp(zoneGoal[0], zoneGoal[1], b.zonePathG[0], b.zonePathG[1]) > 25 {
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
		if md > 0 {
			myTTK = tgt.HP / md
		}
		inc := b.enemyDps(tgt, dEval)
		// others already shooting at us
		for _, t := range mechs {
			if t != tgt && t.LOS && b.t < t.Hostile {
				inc += b.enemyDps(t, hyp(px, py, t.Pos[0], t.Pos[1]))
			}
		}
		theirTTK = me.HP / math.Max(inc, 1)
		engage = myTTK < b.P.EngageRatio*theirTTK || adv > 25
		if nHostile >= 2 && myTTK > 2.5 {
			engage = engage && myTTK*1.5 < theirTTK
		}
	}
	_ = adv
	finishing := tgt != nil && myTTK < 1.6
	if !b.retreating {
		if hpFrac < b.P.RetreatHP && (incoming > 0 || b.t-b.lastDmgT < 3) && !finishing {
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
	case b.retreating || (tgt != nil && !engage && (underFire || incoming > 0)):
		mode = "retreat"
		sp := b.findSafe(threats)
		goal, hasGoal = sp, true
		ctx.hide = true
		ctx.boost = me.Energy > 12 && (underFire || incoming > 0)
		ctx.target = tgt
		ctx.wGoal = 1.5
	case tgt != nil && engage:
		mode = "fight"
		ctx.target = tgt
		ctx.strafe = true
		ctx.wantLOS = 1
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
		if zoneGo {
			// drift toward the zone while fighting
			goal, hasGoal = zoneGoal, true
			ctx.wGoal = 0.6
		}
	default:
		ob := (*objective)(nil)
		if !(hpFrac < 0.7 && b.t-b.lastDmgT < 40 && len(threats) > 0) {
			ob = b.pickObjective(zEnd, hpFrac, len(mechs) > 0, budget)
		}
		if zoneGo && ob == nil {
			mode = "zone"
			goal, hasGoal = zoneGoal, true
			ctx.wGoal = 2
			ctx.boost = me.Energy > 60 && needIn > 60
		} else if hpFrac < 0.7 && b.t-b.lastDmgT < 40 && len(threats) > 0 {
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
				if !b.w.Clear(b.muz[0], b.muz[1], b.muz[2], cp[0], cp[1], cp[2]) {
					ctx.arrive = 3 // no line on it from here: walk up to it
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
	if b.climbID >= 0 {
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
	if shootT != nil && !holdFire {
		shot = b.shootMech(act, shootT)
	} else if tgt != nil && !holdFire {
		// no line: pre-aim where it is
		y, p, _ := b.aimAt(b.predict(tgt, latency))
		act.Aim = [2]float64{y, p}
		if las, li := b.weapon("laser"); las != nil && las.Charge > 0 {
			act.Fire[li] = true // keep the charge
		} else if las != nil && las.Cooldown <= 0.11 && b.t-tgt.Seen < 0.3 && mode == "fight" && hyp(px, py, tgt.Pos[0], tgt.Pos[1]) < las.Stats.Range {
			act.Fire[li] = true
		}
		shot = true
	}
	healing := hpFrac < 0.85 && b.t-b.lastDmgT < 40
	if !shot && shootT == nil && healing {
		// keep still on the trigger: dealing damage restarts the repair timer
		for _, t := range b.trackL {
			if t.NPC && t.LOS && b.t-t.Seen < 0.3 && b.t-t.HitMe < 2 {
				shot = b.shootMech(act, t)
				break
			}
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
		if !shot && farm != nil {
			shot = b.shootCrate(act, farm)
		}
		if !shot && !b.retreating {
			// opportunistic: any crate in easy reach
			var bc *Item
			bd := 75.0
			for _, c := range b.crateL {
				if c.Roof && c.Pos[2] > me.Pos[2]+25 {
					continue
				}
				d := hyp(c.Pos[0], c.Pos[1], px, py)
				if d < bd && b.t-c.Seen < 0.5 {
					bd, bc = d, c
				}
			}
			if bc != nil && bd < 45 {
				shot = b.shootCrate(act, bc)
			}
		}
	}
	if las, li := b.weapon("laser"); las != nil && las.Charge > 0 && !act.Fire[li] && shootT == nil {
		act.Fire[li] = true // never let go without a target
	}
	if !shot {
		b.scan(act, hasGoal, goal)
	}
	// reload when quiet
	if gun, _ := b.weapon("gun"); gun != nil && gun.Reload <= 0 && gun.Ammo < gun.Stats.Mag {
		quiet := shootT == nil && (tgt == nil || !tgt.LOS)
		if (quiet && gun.Ammo <= gun.Stats.Mag*0.6) || (quiet && b.t-b.lastDmgT > 4 && gun.Ammo < gun.Stats.Mag) {
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
	if b.t < t.Hostile {
		return true
	}
	if b.P.Aggressive {
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
			if b.exposed(threats[i].eye, b.safeSpot[0], b.safeSpot[1], z) {
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
			if b.exposed(th.eye, x, y, z) {
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
	b.safeSpot = best
	b.safeT = b.t + 2.5
	return best
}

func (b *Bot) pickObjective(zEnd zoneC, hpFrac float64, enemiesNear bool, budget float64) *objective {
	me := b.me
	px, py := me.Pos[0], me.Pos[1]
	var best *objective
	bs := 0.0
	consider := func(ob objective) {
		d := b.roughDist(ob.pos[0], ob.pos[1])
		// reachable before the zone takes it?
		eta := b.t + d/me.Stats.WalkSpeed
		if b.zoneAt(eta+3).outside(ob.pos[0], ob.pos[1]) > -5 {
			return
		}
		// outside the next circle: only if the detour fits in the spare time
		if zEnd.outside(ob.pos[0], ob.pos[1]) > -8 && d/me.Stats.WalkSpeed*1.6+4 > budget {
			return
		}
		s := ob.value - d
		if s > bs {
			bs = s
			o := ob
			best = &o
		}
	}
	for _, it := range b.cubeL {
		if it.Roof && math.Abs(it.Pos[2]-me.Pos[2]) > 3 || b.t < it.Bad {
			continue
		}
		consider(objective{kind: "cube", pos: it.Pos, value: 85, item: it})
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
		if it.Roof && math.Abs(it.Pos[2]-me.Pos[2]) > 3 {
			// on a roof: climb for it when it is reachable and nobody is around
			h := it.Pos[2] - me.Pos[2]
			need := h/8.0*me.Stats.JetDrain + 12
			if !b.P.Roof || enemiesNear || hpFrac < 0.6 || h < 0 || need > me.MaxEnergy*0.9 {
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
			consider(objective{kind: "crate", pos: it.Pos, value: 75, item: it})
		}
	}
	if b.P.FarmNPC && hpFrac > 0.6 && !enemiesNear {
		for _, t := range b.trackL {
			if t.NPC && b.t-t.Seen < 3 {
				consider(objective{kind: "npc", pos: t.Pos, value: 95, trk: t})
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
	if b.P.HuntPassive && hpFrac > 0.8 && b.t > b.P.HuntAfter {
		for _, t := range b.trackL {
			if t.NPC || b.t-t.Seen > 20 || (b.t > t.Hostile && !b.P.Aggressive) {
				continue
			}
			consider(objective{kind: "hunt", pos: t.Pos, value: 120 - (b.t-t.Seen)*3, trk: t})
		}
	}
	// stay on an objective until done unless something clearly better shows up
	_ = px
	_ = py
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
	case b.t < 1.7:
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
	b.scanPhase += 0.1
	off := math.Sin(b.scanPhase*1.1) * 1.9
	act.Aim = [2]float64{angDiff(base+off, 0), 0}
}
