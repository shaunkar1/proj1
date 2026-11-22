package bot

import (
	"fmt"
	"math"
	"os"

	"mechbot/proto"
)

func (b *Bot) x10On() bool { return b.ffaSure && !b.practice }

type hotSpot struct {
	pos [2]float64
	t   float64
	n   int
	far bool
}

type memMech struct {
	pos   [2]float64
	t     float64
	ratio float64
}

type deathAt struct {
	org   [2]float64
	brg   float64
	band  string
	t     float64
	exact bool
	pos   [2]float64
}

type x10state struct {
	hot    []hotSpot
	mem    map[int]*memMech
	deaths []deathAt
	farmT  float64
	urgT   float64

	n        [40]int
	dbgT     float64
	posRing  [32][2]float64
	posN     int
	modeRing [32]string
	modeDs   map[string]int
	secT     [12]float64
	maxCube  float64
}

const (
	xCapTick = iota
	xCapObj
	xCapGrab
	xHotNew
	xHotObj
	xHotSpot
	xHotMove
	xMemNew
	xMemObj
	xMemSpot
	xStall
	xZoneOut
	xPark
	xParkTurn
	xParkGap
	xHit
	xHitRear
	xHitRearK
	xFace
	xDither
	xFirst
	xFirstRear
	xFirstRearK
	xUrgent
	xPileObj
	xHushT
	xHushOn
	xFarmObj
	xFarmShot
	xLasObj
	xLowT
	xFarmBurst
	xLowFarmWin
	xLowFirst
	xLowFirstFarm
	xSilObj
)

var X10Debug = os.Getenv("MECH_X10") != ""

var x10TraceSlot = func() int {
	var n int
	if _, err := fmt.Sscanf(os.Getenv("MECH_X10"), "trace:%d", &n); err == nil {
		return n
	}
	return -1
}()

func (b *Bot) cubeLimit() float64 {
	if b.P.CubeCap <= 0 || !b.x10On() {
		return 0
	}
	lim := float64(b.P.CubeCap)
	if b.o.Alive <= 2 && lim < b.cfg.Cubes.StatCap {
		lim = b.cfg.Cubes.StatCap
	}
	return lim
}

func (b *Bot) cubeCapped() bool {
	lim := b.cubeLimit()
	return lim > 0 && b.me.Cubes >= lim
}

func (b *Bot) capOver(x, y float64) (over, near bool) {
	lim := b.cubeLimit()
	if lim <= 0 {
		return false, false
	}
	n := 0.0
	for _, it := range b.cubeL {
		if hyp(it.Pos[0], it.Pos[1], x, y) < 6 {
			n++
		}
	}
	return b.me.Cubes >= lim || b.me.Cubes+n > lim*1.5, n > 0
}

func (b *Bot) capCube(it *Item) bool {
	over, _ := b.capOver(it.Pos[0], it.Pos[1])
	return over
}

func (b *Bot) cubeNo(it *Item) bool {
	return b.capCube(it) || (b.P.PileAge > 0 && b.x10On() && b.youngAt(it.Pos[0], it.Pos[1]))
}

func (b *Bot) capWeapon(it *Item) bool {
	if it == nil || (it.Kind == "laser" && !b.holds("laser")) {
		return false
	}
	over, near := b.capOver(it.Pos[0], it.Pos[1])
	return over && near
}

func (b *Bot) capSkip(ob *objective) bool {
	if b.cubeLimit() <= 0 {
		return false
	}
	switch ob.kind {
	case "cube":
		return ob.item != nil && b.capCube(ob.item)
	case "site", "npc", "npcgo", "drop":
		return b.cubeCapped()
	case "roof":
		if ob.item != nil && b.cubes[ob.item.ID] == ob.item {
			return b.capCube(ob.item)
		}
		return b.capWeapon(ob.item)
	case "weapon":
		return b.capWeapon(ob.item)
	case "crate":
		return b.cubeCapped() && b.holds("laser")
	}
	return false
}

func (b *Bot) capNoFarm() bool { return b.cubeCapped() && b.holds("laser") }

func (b *Bot) hotOn() bool   { return (b.P.HotR > 0 || b.P.HotT > 0) && b.x10On() }
func (b *Bot) hotRule() bool { return b.P.HotR > 0 && b.x10On() }

func (b *Bot) hotT() float64 {
	if b.P.HotT > 0 {
		return b.P.HotT
	}
	return 25
}

func (b *Bot) hotHear(kind string, brg float64, band string, at float64) {
	px, py := b.me.Pos[0], b.me.Pos[1]
	rng := b.cfg.Sound.Range[kind]
	if rng <= 0 {
		rng = 320
	}
	near, mid := b.cfg.Sound.Near, b.cfg.Sound.Mid
	if near <= 0 || mid <= near {
		near, mid = 60, 160
	}
	d, far := 0.0, false
	switch band {
	case "near":
		d = math.Min(near, rng) * 0.6
	case "mid":
		d = (near + math.Min(mid, math.Max(rng, near))) / 2
	default:
		d, far = (mid+math.Max(rng, mid))/2, true
	}
	x, y := px+math.Cos(brg)*d, py+math.Sin(brg)*d
	for _, t := range b.tracks {
		if t.NPC || t.Ghost || at-t.Seen > 2 {
			continue
		}
		td := hyp(px, py, t.Pos[0], t.Pos[1])
		if math.Abs(angDiff(math.Atan2(t.Pos[1]-py, t.Pos[0]-px), brg)) < 0.3 && td > d*0.5-20 && td < d*1.6+30 {
			x, y, far = t.Pos[0], t.Pos[1], false
			break
		}
	}
	keep := b.x10.hot[:0]
	var hit *hotSpot
	for i := range b.x10.hot {
		h := b.x10.hot[i]
		if at-h.t > b.hotT() {
			continue
		}
		keep = append(keep, h)
	}
	b.x10.hot = keep
	for i := range b.x10.hot {
		h := &b.x10.hot[i]
		same := hyp(h.pos[0], h.pos[1], x, y) < 60
		if h.far && far && math.Abs(angDiff(math.Atan2(h.pos[1]-py, h.pos[0]-px), brg)) < 0.25 {
			same = true
		}
		if same && (hit == nil || hyp(h.pos[0], h.pos[1], x, y) < hyp(hit.pos[0], hit.pos[1], x, y)) {
			hit = h
		}
	}
	if hit != nil {
		fresh := at-hit.t > 6
		hit.pos = [2]float64{(hit.pos[0] + x) / 2, (hit.pos[1] + y) / 2}
		hit.far = hit.far && far
		hit.t = math.Max(hit.t, at)
		hit.n++
		if fresh {
			b.hotMove(hit)
		}
		return
	}
	b.x10.hot = append(b.x10.hot, hotSpot{pos: [2]float64{x, y}, t: at, n: 1, far: far})
	b.x10.n[xHotNew]++
	b.hotMove(&b.x10.hot[len(b.x10.hot)-1])
}

func (b *Bot) hotMove(h *hotSpot) {
	if hyp(h.pos[0], h.pos[1], b.me.Pos[0], b.me.Pos[1]) >= b.P.HotR || b.mode != "shy" {
		return
	}
	for _, t := range b.tracks {
		if !t.NPC && !t.Ghost && b.t-t.Seen < 0.3 {
			return
		}
	}
	if b.t < b.shyT {
		b.shyT = 0
		b.x10.n[xHotMove]++
	}
}

func hotAge(age, life float64) float64 {
	if age >= life || life <= 0 {
		return 0
	}
	return 1 - 0.5*math.Max(age, 0)/life
}

func (b *Bot) hotNear(x, y float64) bool {
	if !b.hotRule() {
		return false
	}
	for i := range b.x10.hot {
		h := &b.x10.hot[i]
		if w := hotAge(b.t-h.t, b.hotT()); w > 0 && hyp(x, y, h.pos[0], h.pos[1]) < b.P.HotR*w {
			return true
		}
	}
	return false
}

func (b *Bot) memOn() bool { return b.P.ThreatMem > 0 && b.x10On() }

func (b *Bot) memUpdate() {
	if !b.memOn() {
		return
	}
	if b.x10.mem == nil {
		b.x10.mem = map[int]*memMech{}
	}
	lim := b.P.ThreatMemRatio
	if lim <= 0 {
		lim = 1
	}
	for _, t := range b.trackL {
		if t.NPC || t.Ghost || b.t-t.Seen > 0.3 {
			continue
		}
		r := b.dangerRatio(t)
		if r < lim {
			delete(b.x10.mem, t.ID)
			continue
		}
		m := b.x10.mem[t.ID]
		if m == nil {
			m = &memMech{}
			b.x10.mem[t.ID] = m
			b.x10.n[xMemNew]++
		}
		m.pos, m.t, m.ratio = [2]float64{t.Pos[0], t.Pos[1]}, b.t, r
	}
	for id, m := range b.x10.mem {
		if b.t-m.t > b.P.ThreatMem {
			delete(b.x10.mem, id)
		}
	}
}

func (b *Bot) x10Any() bool { return b.hotRule() || b.memOn() }

func (b *Bot) x10Spot(x, y, m float64) float64 {
	if !b.x10Any() {
		return m
	}
	if b.memOn() {
		for _, mm := range b.x10.mem {
			age := b.t - mm.t
			if age <= 12 {
				continue
			}
			if w := hotAge(age, b.P.ThreatMem); w > 0 {
				if d := hyp(x, y, mm.pos[0], mm.pos[1]) / w; d < m {
					m = d
					b.x10.n[xMemSpot]++
				}
			}
		}
	}
	if b.hotRule() {
		for i := range b.x10.hot {
			h := &b.x10.hot[i]
			if w := hotAge(b.t-h.t, b.hotT()); w > 0 {
				if d := hyp(x, y, h.pos[0], h.pos[1]) / w; d < m {
					m = d
				}
			}
		}
		if b.hotNear(x, y) {
			m -= 1000
			b.x10.n[xHotSpot]++
		}
	}
	return m
}

func (b *Bot) x10Skip(ob *objective, d float64) bool {
	if b.capSkip(ob) {
		b.x10.n[xCapObj]++
		return true
	}
	if b.pileYoung(ob) {
		b.x10.n[xPileObj]++
		return true
	}
	if !b.x10Any() || d <= 12 {
		return false
	}
	px, py := b.me.Pos[0], b.me.Pos[1]
	if b.hotRule() && b.hotNear(ob.pos[0], ob.pos[1]) {

		wantLas := (ob.kind == "weapon" || ob.kind == "roof") && ob.item != nil && ob.item.Kind == "laser" && !b.holds("laser") && d <= b.P.ShyLaserD
		if !wantLas {
			b.x10.n[xHotObj]++
			return true
		}
	}
	if b.memOn() {

		for _, mm := range b.x10.mem {
			age := b.t - mm.t
			w := hotAge(age, b.P.ThreatMem)
			if w <= 0 {
				continue
			}
			db := hyp(ob.pos[0], ob.pos[1], mm.pos[0], mm.pos[1])
			if db < b.P.QuietObjD*w && db < hyp(px, py, mm.pos[0], mm.pos[1])+5 {
				if age > 8 {
					b.x10.n[xMemObj]++
				}
				return true
			}
		}
	}
	return false
}

func (b *Bot) clearAt(x, y, r float64) bool {
	for _, t := range b.trackL {
		if !t.NPC && b.t-t.Seen < 10 && hyp(x, y, t.Pos[0], t.Pos[1]) < r {
			return false
		}
	}
	for i := range b.noise {
		if n := &b.noise[i]; b.t-n.t < 12 && hyp(x, y, n.pos[0], n.pos[1]) < r {
			return false
		}
	}
	if b.hotOn() {
		for i := range b.x10.hot {
			h := &b.x10.hot[i]
			if w := hotAge(b.t-h.t, b.hotT()); w > 0 && hyp(x, y, h.pos[0], h.pos[1]) < r*w {
				return false
			}
		}
	}
	if b.memOn() {
		for _, mm := range b.x10.mem {
			if w := hotAge(b.t-mm.t, b.P.ThreatMem); w > 0 && hyp(x, y, mm.pos[0], mm.pos[1]) < r*w {
				return false
			}
		}
	}
	return true
}

func (b *Bot) farmOn() bool {
	if b.P.FarmClear <= 0 || !b.x10On() || !b.endLow {
		return false
	}
	t0 := b.P.FarmClearT
	if t0 <= 0 {
		t0 = b.P.HushT
	}
	lim := b.cubeLimit()
	if lim <= 0 {
		lim = b.cfg.Cubes.StatCap
	}
	return b.t >= t0 && b.me.Cubes < lim
}

func (b *Bot) farmFree(ob *objective) bool {
	if !b.farmOn() {
		return false
	}
	switch ob.kind {
	case "crate", "npc", "npcgo", "cube", "site":
	case "explore":
		if b.P.FarmScout <= 0 {
			return false
		}
	case "roof":
		if ob.item == nil || b.cubes[ob.item.ID] != ob.item {
			return false
		}
	default:
		return false
	}
	r := b.P.FarmClear
	return b.clearAt(b.me.Pos[0], b.me.Pos[1], r) && b.clearAt(ob.pos[0], ob.pos[1], r)
}

func (b *Bot) farmShots() bool {
	return b.farmOn() && b.clearAt(b.me.Pos[0], b.me.Pos[1], b.P.FarmClear)
}

func (b *Bot) farmBlocked(x, y float64) bool {
	return b.P.FarmClear > 0 && b.x10On() && b.endLow && !b.clearAt(x, y, b.P.FarmClear)
}

func (b *Bot) silentFree(ob *objective, d float64) bool {
	if b.P.SilentCollect <= 0 || !b.x10On() || !b.endLow || d > b.P.SilentCollect {
		return false
	}
	lim := b.cubeLimit()
	if lim <= 0 {
		lim = b.cfg.Cubes.StatCap
	}
	short := b.me.Cubes < lim
	switch ob.kind {
	case "cube", "site":
		if !short {
			return false
		}
	case "roof":
		if ob.item != nil && b.cubes[ob.item.ID] == ob.item && !short {
			return false
		}
	case "weapon":
	case "drop":
		ok := false
		for i := range b.o.Drops {
			dr := &b.o.Drops[i]
			if hyp(dr.Pos[0], dr.Pos[1], ob.pos[0], ob.pos[1]) < 1 && dr.LandsIn <= 0 && (dr.Holder == nil || *dr.Holder == b.me.ID) {
				ok = true
			}
		}
		if !ok || !short {
			return false
		}
	default:
		return false
	}
	r := b.P.SilentClear
	if r <= 0 {
		r = 150
	}
	return b.clearAt(b.me.Pos[0], b.me.Pos[1], r) && b.clearAt(ob.pos[0], ob.pos[1], r)
}

func (b *Bot) laserFree(ob *objective) bool {
	if b.P.LaserErrand <= 0 || !b.x10On() || ob.item == nil || ob.item.Kind != "laser" || (ob.kind != "weapon" && ob.kind != "roof") || b.holds("laser") {
		return false
	}
	r := b.P.LaserErrand
	return b.clearAt(b.me.Pos[0], b.me.Pos[1], r) && b.clearAt(ob.pos[0], ob.pos[1], r)
}

func (b *Bot) x10Death(s *deathSite, band string) {
	if b.P.PileAge <= 0 || !b.x10On() {
		return
	}
	keep := b.x10.deaths[:0]
	for _, d := range b.x10.deaths {
		if s.t-d.t < b.P.PileAge {
			keep = append(keep, d)
		}
	}
	b.x10.deaths = append(keep, deathAt{org: s.org, brg: s.brg, band: band, t: s.t, exact: s.exact, pos: s.pos})
}

func (b *Bot) youngAt(x, y float64) bool {
	for i := range b.x10.deaths {
		d := &b.x10.deaths[i]
		if b.t-d.t >= b.P.PileAge {
			continue
		}
		if d.exact {
			if hyp(x, y, d.pos[0], d.pos[1]) < 25 {
				return true
			}
			continue
		}
		dd := hyp(x, y, d.org[0], d.org[1])
		in := (d.band == "near" && dd < 75) || (d.band == "mid" && dd > 45 && dd < 180) || (d.band == "far" && dd > 140)
		if in && math.Abs(angDiff(math.Atan2(y-d.org[1], x-d.org[0]), d.brg)) < 0.25 {
			return true
		}
	}
	return false
}

func (b *Bot) pileYoung(ob *objective) bool {
	if b.P.PileAge <= 0 || !b.x10On() {
		return false
	}
	switch ob.kind {
	case "site":
		return ob.site != nil && b.t-ob.site.t < b.P.PileAge
	case "cube":
		return b.youngAt(ob.pos[0], ob.pos[1])
	case "roof":
		return ob.item != nil && b.cubes[ob.item.ID] == ob.item && b.youngAt(ob.item.Pos[0], ob.item.Pos[1])
	}
	return false
}

func (b *Bot) x10Trace(zoneUrgent bool, a *proto.Action, farmShot bool) {
	if !X10Debug || !b.x10On() {
		return
	}
	me := b.me
	px, py := me.Pos[0], me.Pos[1]
	x := &b.x10
	act := a.Aim
	if farmShot {
		if x.farmT == 0 || b.t-x.farmT > 2 {
			x.n[xFarmBurst]++
		}
		x.farmT = b.t
	}
	if b.endLow {
		if x.farmT > 0 && b.t-x.farmT < 15 {
			x.n[xLowFarmWin]++
		}
	}
	if b.slot == x10TraceSlot {
		z := &b.o.Zone
		fmt.Fprintf(os.Stderr, "XT t=%.1f %s al=%d pos=(%.1f,%.1f,%.1f) v=%.1f en=%.0f hp=%.0f cubes=%.0f in=%v urg=%v goal=(%.0f,%.0f) mv=(%.2f,%.2f) boost=%v jump=%v aim=%.2f->%.2f stuck=%.1f unstick=%.1f path=%d ok=%v zc=%.0f blocked=%d zone=(%.0f,%.0f,%.0f)->(%.0f,%.0f,%.0f) %.0f-%.0f hot=%d mem=%d\n",
			b.t, b.mode, b.o.Alive, px, py, me.Pos[2], math.Hypot(me.Vel[0], me.Vel[1]), me.Energy, me.HP, me.Cubes, me.InZone, zoneUrgent, b.DbgGoal[0], b.DbgGoal[1],
			a.Move[0], a.Move[1], a.Boost, a.Jump, me.Aim[0], a.Aim[0], b.stuckT, math.Max(b.unstickT-b.t, 0), len(b.path), b.pathOK, b.zonePathC, len(b.blockedC),
			z.Center[0], z.Center[1], z.Radius, z.NextCenter[0], z.NextCenter[1], z.NextRadius, z.ShrinkStart, z.ShrinkEnd, len(x.hot), len(x.mem))
		if zoneUrgent && int(b.t*10)%5 == 0 && len(b.path) > 0 {
			ps := ""
			for i, p := range b.path {
				if i >= 14 {
					break
				}
				ps += fmt.Sprintf(" (%.1f,%.1f,%.1f)", p[0], p[1], b.w.Ground(p[0], p[1]))
			}
			fmt.Fprintf(os.Stderr, "XP t=%.1f pathGoal=(%.0f,%.0f) pathT=%.1f cost=%.0f walkable=%v path:%s\n", b.t, b.pathGoal[0], b.pathGoal[1], b.pathT, b.pathCost, b.w.Nav.Walkable(px, py), ps)
		}
		if b.P.FarmClear > 0 && int(b.t*10)%20 == 0 {
			nc, ncl, bd := 0, 0, 1e9
			for _, c := range b.crateL {
				if b.t < c.Bad {
					continue
				}
				nc++
				if b.clearAt(c.Pos[0], c.Pos[1], b.P.FarmClear) {
					ncl++
					bd = math.Min(bd, hyp(px, py, c.Pos[0], c.Pos[1]))
				}
			}
			fmt.Fprintf(os.Stderr, "XF t=%.1f %s low=%v farmOn=%v clearMe=%v crates=%d clearCrates=%d nearest=%.0f cubesKnown=%d drones=%d hush=%v quiet=%v tracks=%d noise=%d lastObj=%s\n",
				b.t, b.mode, b.endLow, b.farmOn(), b.clearAt(px, py, b.P.FarmClear), nc, ncl, bd, len(b.cubeL), len(b.npcMem), b.hush, b.quiet, len(b.trackL), len(b.noise), b.lastObjKind)
		}
	}
	if b.cubeCapped() {
		x.n[xCapTick]++
	}
	if me.Cubes > x.maxCube {
		x.maxCube = me.Cubes
	}
	x.posRing[x.posN%len(x.posRing)] = [2]float64{px, py}
	x.modeRing[x.posN%len(x.modeRing)] = b.mode
	x.posN++
	if b.endLow {
		if x.modeDs == nil {
			x.modeDs = map[string]int{}
		}
		k := b.mode
		if k == "shy" && math.Hypot(me.Vel[0], me.Vel[1]) >= 0.5 {
			k = "shy-moving"
		}
		x.modeDs[k]++
	}
	if !me.InZone {
		x.n[xZoneOut]++
	}
	if zoneUrgent {
		x.n[xUrgent]++
		if x.posN > 30 {
			p := x.posRing[(x.posN-31)%len(x.posRing)]
			if hyp(px, py, p[0], p[1]) < 3 {
				x.n[xStall]++
				if x.n[xStall]%10 == 1 {
					fmt.Fprintf(os.Stderr, "X10STALL %s %d %d t=%.1f pos=(%.0f,%.0f,%.1f) mode=%s en=%.0f inzone=%v hp=%.0f\n", b.P.Name, b.seed, b.slot, b.t, px, py, me.Pos[2], b.mode, me.Energy, me.InZone, me.HP)
				}
			}
		}
	}
	for k := range x.secT {
		if math.Abs(angDiff(me.Aim[0], float64(k)*math.Pi/6)) < b.cfg.Vision.FovDeg*math.Pi/360 {
			x.secT[k] = b.t
		}
	}
	if b.mode == "shy" && math.Hypot(me.Vel[0], me.Vel[1]) < 0.5 {
		x.n[xPark]++
		if math.Abs(angDiff(act[0], me.Aim[0])) >= 0.1 {
			x.n[xParkTurn]++
		}
		g := 0.0
		for k := range x.secT {
			g = math.Max(g, b.t-x.secT[k])
		}
		x.n[xParkGap] += int(math.Min(g, 30) * 10)
	}
	for i := range b.o.Events {
		e := &b.o.Events[i]
		if e.Type != "damaged" || e.By == nil || *e.By >= 100 {
			continue
		}
		known := false
		if t := b.tracks[*e.By]; t != nil && !t.Ghost && b.t-t.Seen < 5 {
			known = true
		}
		first := x.dbgT == 0 || b.t-x.dbgT > 10
		x.dbgT = b.t
		x.n[xHit]++
		if first {
			x.n[xFirst]++
			fa := -1.0
			if x.farmT > 0 {
				fa = b.t - x.farmT
			}
			if b.endLow {
				x.n[xLowFirst]++
				if fa >= 0 && fa < 15 {
					x.n[xLowFirstFarm]++
				}
			}
			fmt.Fprintf(os.Stderr, "X10HIT %s %d %d t=%.1f alive=%d low=%v farmAge=%.1f hp=%.0f mode=%s prev=%s\n", b.P.Name, b.seed, b.slot, b.t, b.o.Alive, b.endLow, fa, me.HP, b.mode, x.modeRing[(x.posN+len(x.modeRing)-15)%len(x.modeRing)])
		}
		if e.Rear {
			x.n[xHitRear]++
			if known {
				x.n[xHitRearK]++
			}
			if first {
				x.n[xFirstRear]++
				if known {
					x.n[xFirstRearK]++
				}
			}
		}
	}
	if b.P.HushT > 0 && b.t < b.P.HushT {
		x.n[xHushT]++
		if b.hush {
			x.n[xHushOn]++
		}
	}
	if b.endLow {
		x.n[xLowT]++
	}
	if int(b.t*10)%50 == 0 && len(x.modeDs) > 0 {
		ms := ""
		for _, k := range []string{"shy", "shy-moving", "explore", "crate", "npc", "npcgo", "cube", "site", "weapon", "roof", "drop", "grab", "zone", "avoid", "fight", "retreat", "evade", "heal", "flee", "break", "roam"} {
			if v := x.modeDs[k]; v > 0 {
				ms += fmt.Sprintf(" %s=%d", k, v)
			}
		}
		fmt.Fprintf(os.Stderr, "X10MODE %s %d %d%s\n", b.P.Name, b.seed, b.slot, ms)
	}
	if int(b.t*10)%50 == 0 {
		las, w2 := 0, 0
		if b.holds("laser") {
			las = 1
		}
		if b.slot1Rank() > 0 {
			w2 = 1
		}
		fmt.Fprintf(os.Stderr, "X10 %s %d %d t=%.0f alive=%d cubes=%.0f max=%.0f las=%d w2=%d n=%v\n", b.P.Name, b.seed, b.slot, b.t, b.o.Alive, me.Cubes, x.maxCube, las, w2, x.n)
	}
}
