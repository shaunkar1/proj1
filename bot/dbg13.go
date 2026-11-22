package bot

import (
	"fmt"
	"math"
	"os"
	"strings"

	"mechbot/proto"
)

var b13Every, b13Names = func() (int, map[string]bool) {
	v := os.Getenv("B13_DEBUG")
	if v == "" {
		return 0, nil
	}
	parts := strings.Split(v, ",")
	n := 0
	fmt.Sscanf(parts[0], "%d", &n)
	if n <= 0 {
		n = 10
	}
	m := map[string]bool{}
	for _, p := range parts[1:] {
		m[p] = true
	}
	return n, m
}()

func (b *Bot) trace13(act *proto.Action, mode string, goal [2]float64, hasGoal, zoneGo, zoneUrgent bool, budget, needIn float64, tgt *Track) {
	if b13Every == 0 || b.o.Tick%b13Every != 0 {
		return
	}
	if len(b13Names) > 0 {
		if !b13Names[b.P.Name] {
			return
		}
	} else if b.P.Name != "ffa0" && b.P.Name != "b3" && !strings.HasPrefix(b.P.Name, "n4") {
		return
	}
	me := b.me
	px, py := me.Pos[0], me.Pos[1]
	z := &b.o.Zone
	outNow := hyp(px, py, z.Center[0], z.Center[1]) - z.Radius
	outNext := hyp(px, py, z.NextCenter[0], z.NextCenter[1]) - z.NextRadius
	gx, gy, gd := 0.0, 0.0, -1.0
	if hasGoal {
		gx, gy, gd = goal[0], goal[1], hyp(px, py, goal[0], goal[1])
	}
	near, nearAge, nearLOS := 999.0, -1.0, false
	for _, t := range b.trackL {
		if t.NPC || t.Ghost || b.t-t.Seen > 15 {
			continue
		}
		if d := hyp(px, py, t.Pos[0], t.Pos[1]); d < near {
			near, nearAge, nearLOS = d, b.t-t.Seen, t.LOS && b.t-t.Seen < 0.3
		}
	}
	td := -1.0
	tid := -1
	if tgt != nil {
		td, tid = hyp(px, py, tgt.Pos[0], tgt.Pos[1]), tgt.ID
	}
	bd := -1.0
	bid := -1
	if b.boss != nil {
		bd, bid = hyp(px, py, b.boss.Pos[0], b.boss.Pos[1]), b.boss.ID
	}
	fl := ""
	for _, f := range []struct {
		on bool
		c  string
	}{{b.quiet, "q"}, {b.endLow, "l"}, {b.hush, "h"}, {b.patient, "p"}, {b.retreating, "r"}, {zoneGo, "z"}, {zoneUrgent, "U"}, {me.InZone, "i"}, {me.Grounded, "g"}, {b.wantMove, "w"}, {act.Boost, "B"}, {act.Jump, "J"}, {act.Fire[0] || act.Fire[1], "F"}, {b.t < b.unstickT, "u"}, {nearLOS, "L"}, {b.zrOn == 1, "x"}, {b.zrOn == 2, "X"}, {b.zgnOn, "N"}} {
		if f.on {
			fl += f.c
		}
	}
	if fl == "" {
		fl = "-"
	}
	wp := ""
	for _, w := range me.Weapons {
		if w != nil && len(w.Kind) >= 2 {
			wp += w.Kind[:2]
		}
	}
	if budget > 999 {
		budget = 999
	}

	r, h, mz := me.Stats.Radius, me.Stats.Height, me.Pos[2]
	blocked := func(x, y float64) bool {
		if b.w.BodyBlocked(x, y, mz+1.2, mz+h) {
			return true
		}
		for k := 0; k < 8; k++ {
			a := float64(k) * math.Pi / 4
			if b.w.BodyBlocked(x+r*math.Cos(a), y+r*math.Sin(a), mz+1.2, mz+h) {
				return true
			}
		}
		return false
	}
	emb := 0
	if blocked(px, py) {
		emb = 1
	}
	free := 0
	for k := 0; k < 8; k++ {
		a := float64(k) * math.Pi / 4
		x, y := px+0.6*math.Cos(a), py+0.6*math.Sin(a)
		if !blocked(x, y) && b.w.Ground(x, y) <= mz+1.2 {
			free++
		}
	}
	ceil := 99.0
	for k := -1; k < 8; k++ {
		x, y := px, py
		if k >= 0 {
			a := float64(k) * math.Pi / 4
			x, y = px+r*math.Cos(a), py+r*math.Sin(a)
		}
		for _, iv := range b.w.Intervals(x, y) {
			if float64(iv.Base) > mz+1.2 && float64(iv.Base)-mz < ceil {
				ceil = float64(iv.Base) - mz
			}
		}
	}
	fmt.Fprintf(os.Stderr, "T13 %s %d %d %.2f al=%d %s pos=%.1f,%.1f,%.1f hp=%.0f/%.0f en=%.0f c=%.0f mv=%.2f,%.2f v=%.1f fl=%s on=%.0f ox=%.0f zr=%.0f>%.0f se=%.0f bud=%.0f need=%.0f goal=%.0f,%.0f gd=%.0f stk=%.1f dmg=%.1f pr=%d near=%.0f/%.1f tgt=%d/%.0f boss=%d/%.0f w=%s pl=%d emb=%d fr=%d ceil=%.1f h=%.1f zpc=%.0f te=%.0f tv=%.0f zpg=%.0f,%.0f zpt=%.1f\n",
		b.P.Name, b.seed, b.slot, b.t, b.o.Alive, mode, px, py, me.Pos[2], me.HP, me.MaxHP, me.Energy, me.Cubes, act.Move[0], act.Move[1], hyp(me.Vel[0], me.Vel[1], 0, 0),
		fl, outNow, outNext, z.Radius, z.NextRadius, z.ShrinkEnd-b.t, budget, needIn, gx, gy, gd, b.stuckT, b.t-b.lastDmgT, len(b.o.Projectiles), near, nearAge, tid, td, bid, bd, wp, len(b.path), emb, free, ceil, h, b.dbgZ13[0], b.dbgZ13[1], b.dbgZ13[2], b.zonePathG[0], b.zonePathG[1], b.t-b.zonePathT)
}
