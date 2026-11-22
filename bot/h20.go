package bot

import (
	"fmt"
	"math"
	"os"
)

func (b *Bot) h20On() bool { return b.ffaSure && !b.practice }

type h20state struct {
	corpses []corpseHeap

	n    [27]int
	seen map[int]bool
}

type corpseHeap struct {
	pos  [2]float64
	t    float64
	id   int
	took bool
}

const (
	hLasSeen = iota
	hLasGate
	hGateQuiet
	hGateShy
	hGateLow
	hGateNear
	hGatePeer
	hGateBoss
	hGateKeep
	hGateOther
	hLasGo
	hLasUnsafe
	hLasWait
	hLasWaitD
	hGlGo
	hGlHurt
	hGlBusy
	hGlCubes
	hGlNo
	hCorpseNew
	hCorpseGo
	hCorpseUnsafe
	hCorpseAge
	hCorpseTook
	hDuelTrack
	hDuelZone
	hDecide
)

var H20Debug = os.Getenv("MECH_H20") != ""

func (b *Bot) lasRushOn() bool {
	return b.P.LasRush != 0 && b.h20On() && !b.holds("laser") && (b.P.LasRushT <= 0 || b.t < b.P.LasRushT)
}

func (b *Bot) lasActive() bool {
	return b.P.LasRush > 0 && b.h20On() && !b.holds("laser") && (b.P.LasRushT <= 0 || b.t < b.P.LasRushT)
}

func lasObj(ob *objective) bool {
	return (ob.kind == "weapon" || ob.kind == "roof") && ob.item != nil && ob.item.Kind == "laser"
}

func (b *Bot) lasWant(ob *objective, d float64) bool {
	return b.lasRushOn() && lasObj(ob) && d <= math.Abs(b.P.LasRush)
}

func (b *Bot) lasNear(x, y float64) float64 {
	m := math.Inf(1)
	for _, t := range b.trackL {
		if t.NPC || b.t-t.Seen > 10 {
			continue
		}
		m = math.Min(m, hyp(x, y, t.Pos[0], t.Pos[1]))
	}
	for i := range b.noise {
		n := &b.noise[i]
		if n.far || b.t-n.t > 12 {
			continue
		}
		m = math.Min(m, hyp(x, y, n.pos[0], n.pos[1]))
	}
	return m
}

func (b *Bot) lasSafe(ob *objective) bool {
	if !b.lasActive() {
		return false
	}
	if b.P.LasSafe <= 0 {
		return true
	}
	return b.lasNear(b.me.Pos[0], b.me.Pos[1]) >= b.P.LasSafe && b.lasNear(ob.pos[0], ob.pos[1]) >= b.P.LasSafe
}

func (b *Bot) lasSkip(lasWant, lasGo bool, gate int) bool {
	if !lasWant {
		return true
	}
	b.h20.n[gate]++
	b.h20.n[hLasGate]++
	return !lasGo
}

func (b *Bot) lasWaiting() bool {
	if !b.P.LasWait || !b.lasActive() {
		return false
	}
	blocked := false
	for _, it := range b.pickupL {
		if it.Kind != "laser" || b.t < it.Bad {
			continue
		}
		if b.roughDist(it.Pos[0], it.Pos[1]) > math.Abs(b.P.LasRush) {
			continue
		}
		ob := objective{kind: "weapon", pos: it.Pos, item: it}
		if b.lasSafe(&ob) {
			return false
		}
		blocked = true
	}
	return blocked
}

func (b *Bot) greenLight(t *Track, d, hpFrac float64) bool {
	if !b.h20On() || b.practice || t.NPC || t.Ghost || !t.LOS || b.t-t.Seen > 0.3 || b.boss != nil {
		return false
	}
	if b.P.GlHurt <= 0 && !b.P.GlBusy && b.P.GlCubes <= 0 {
		return false
	}
	las, _ := b.weapon("laser")
	if b.P.GlLaser && las == nil {
		return false
	}
	if b.P.GlHP > 0 && hpFrac < b.P.GlHP {
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

	if las != nil {
		if d < 25 || d > las.Stats.Range-15 {
			return false
		}
	} else if b.P.OpenBusyD > 0 && d > b.P.OpenBusyD {
		return false
	}
	ok := false
	if b.P.GlHurt > 0 && t.MaxHP > 0 && t.HP/t.MaxHP < b.P.GlHurt {
		ok = true
		b.h20.n[hGlHurt]++
	}
	if b.P.GlBusy && b.t-t.busyT < 3 {
		ok = true
		b.h20.n[hGlBusy]++
	}
	if b.P.GlCubes > 0 && b.me.Cubes-t.Cubes >= float64(b.P.GlCubes) {
		ok = true
		b.h20.n[hGlCubes]++
	}
	if ok {
		b.h20.n[hGlGo]++
	} else {
		b.h20.n[hGlNo]++
	}
	return ok
}

func (b *Bot) corpseOn() bool { return b.P.CorpseD > 0 && b.h20On() }

func (b *Bot) corpseDeath(id int, x, y, at float64) {
	if !b.corpseOn() {
		return
	}
	if hyp(x, y, b.me.Pos[0], b.me.Pos[1]) > b.P.CorpseD {
		return
	}
	keep := b.h20.corpses[:0]
	for _, c := range b.h20.corpses {
		if at-c.t < 120 && c.id != id {
			keep = append(keep, c)
		}
	}
	b.h20.corpses = append(keep, corpseHeap{pos: [2]float64{x, y}, t: at, id: id})
	b.h20.n[hCorpseNew]++
}

func (b *Bot) corpseFree(ob *objective) bool {
	if !b.corpseOn() {
		return false
	}
	switch ob.kind {
	case "cube", "site":
	case "roof":
		if ob.item == nil || b.cubes[ob.item.ID] != ob.item {
			return false
		}
	default:
		return false
	}

	if ob.kind == "site" && ob.site != nil && !ob.site.far {
		if b.t-ob.site.t < b.P.CorpseAge {
			b.h20.n[hCorpseAge]++
		} else if b.P.CorpseSafe > 0 && b.lasNear(ob.pos[0], ob.pos[1]) < b.P.CorpseSafe {
			b.h20.n[hCorpseUnsafe]++
		} else {
			b.h20.n[hCorpseGo]++
			return true
		}
	}
	for i := range b.h20.corpses {
		c := &b.h20.corpses[i]
		if hyp(ob.pos[0], ob.pos[1], c.pos[0], c.pos[1]) > 25 {
			continue
		}
		if b.t-c.t < b.P.CorpseAge {
			b.h20.n[hCorpseAge]++
			continue
		}
		if b.P.CorpseSafe > 0 && b.lasNear(ob.pos[0], ob.pos[1]) < b.P.CorpseSafe {
			b.h20.n[hCorpseUnsafe]++
			continue
		}
		b.h20.n[hCorpseGo]++
		if !c.took && hyp(b.me.Pos[0], b.me.Pos[1], c.pos[0], c.pos[1]) < 6 {
			c.took = true
			b.h20.n[hCorpseTook]++
		}
		return true
	}
	return false
}

func (b *Bot) corpseBonus(ob *objective) float64 {
	if b.corpseFree(ob) {
		return 150
	}
	return 0
}

func (b *Bot) duelAim() (float64, bool) {
	if !b.P.DuelHold || !b.h20On() || b.o.Alive != 2 || b.othersDied == 0 {
		return 0, false
	}
	if math.Hypot(b.me.Vel[0], b.me.Vel[1]) > 1.0 {
		return 0, false
	}
	px, py := b.me.Pos[0], b.me.Pos[1]
	mem := b.P.DuelMem
	if mem <= 0 {
		mem = 25
	}
	for _, t := range b.trackL {
		if t.NPC || b.t-t.Seen > mem {
			continue
		}
		b.h20.n[hDuelTrack]++
		return math.Atan2(t.Pos[1]-py, t.Pos[0]-px), true
	}
	z := &b.o.Zone
	cx, cy := z.NextCenter[0], z.NextCenter[1]
	if hyp(cx, cy, px, py) < 15 {
		cx, cy = z.Center[0], z.Center[1]
	}
	if hyp(cx, cy, px, py) < 15 {
		return 0, false
	}
	b.h20.n[hDuelZone]++
	return math.Atan2(cy-py, cx-px), true
}

func (b *Bot) h20Trace() {
	if !H20Debug || !b.h20On() {
		return
	}
	b.h20.n[hDecide]++
	if int(b.t*10)%50 != 0 {
		return
	}
	las := 0
	if b.holds("laser") {
		las = 1
	}
	fmt.Fprintf(os.Stderr, "H20 %s %d %d t=%.0f alive=%d cubes=%.0f las=%d corpses=%d n=%v\n",
		b.P.Name, b.seed, b.slot, b.t, b.o.Alive, b.me.Cubes, las, len(b.h20.corpses), b.h20.n)
}
