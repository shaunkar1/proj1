package bot

import (
	"math"
	"sync/atomic"
)

var openCrowdFire, openCrowdMove int64

func OpenCrowdStats() (int64, int64) {
	return atomic.LoadInt64(&openCrowdFire), atomic.LoadInt64(&openCrowdMove)
}

func (b *Bot) openCrowdPars() (n int, d, tMax, age float64, ok bool) {
	if !b.practice || b.P.PracOpenCrowdN <= 0 {
		return 0, 0, 0, 0, false
	}
	d, tMax, age = b.P.PracOpenCrowdD, b.P.PracOpenCrowdT, b.P.PracOpenCrowdA
	if d <= 0 {
		d = 200
	}
	if tMax <= 0 {
		tMax = 90
	}
	if age <= 0 {
		age = 10
	}
	return b.P.PracOpenCrowdN, d, tMax, age, true
}

func (b *Bot) openCrowd() bool {
	n, d, tMax, age, ok := b.openCrowdPars()
	if !ok || b.t >= tMax {
		return false
	}
	cnt := 0
	for _, t := range b.trackL {
		if t.NPC || t.Ghost || b.t-t.Seen > age {
			continue
		}
		if hyp(b.me.Pos[0], b.me.Pos[1], t.Pos[0], t.Pos[1]) < d {
			cnt++
		}
	}
	if cnt < n {
		return false
	}
	atomic.AddInt64(&openCrowdFire, 1)
	return true
}

func (b *Bot) openSpreadGoal(zEnd zoneC) [2]float64 {
	me := b.me
	px, py := me.Pos[0], me.Pos[1]
	if b.t < b.ocHoldT && hyp(b.ocGoal[0], b.ocGoal[1], px, py) > 10 {
		return b.ocGoal
	}
	_, _, _, age, _ := b.openCrowdPars()
	var oth [][2]float64
	cx, cy := 0.0, 0.0
	for _, t := range b.trackL {
		if t.NPC || t.Ghost || b.t-t.Seen > age {
			continue
		}
		oth = append(oth, [2]float64{t.Pos[0], t.Pos[1]})
		cx += t.Pos[0]
		cy += t.Pos[1]
	}
	if len(oth) == 0 {
		return [2]float64{px, py}
	}
	cx /= float64(len(oth))
	cy /= float64(len(oth))
	d0 := hyp(px, py, cx, cy)
	nv := b.w.Nav
	best, bs := [2]float64{px, py}, math.Inf(-1)
	for k := 0; k < 24; k++ {
		a := float64(k) * math.Pi / 12
		for _, r := range [...]float64{60, 100, 140} {
			x, y := px+r*math.Cos(a), py+r*math.Sin(a)
			if !nv.WalkableMain(x, y) || zEnd.outside(x, y) > -10 {
				continue
			}

			s := hyp(x, y, cx, cy) - d0
			near := 300.0
			for _, o := range oth {
				if dd := hyp(x, y, o[0], o[1]); dd < near {
					near = dd
				}
			}
			s += 0.6 * math.Min(near, 220)
			if rd := b.roughDist(x, y); rd > r*1.25 {
				s -= 40
			}
			s -= 0.1 * r
			if s > bs {
				bs, best = s, [2]float64{x, y}
			}
		}
	}
	b.ocGoal, b.ocHoldT = best, b.t+4
	atomic.AddInt64(&openCrowdMove, 1)
	return best
}
