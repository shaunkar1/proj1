package main

import (
	"math"
	"sort"

	"mechbot/sim"
)

type samp struct {
	alive     bool
	x, y      float64
	hp, maxhp float64
	cubes     int
	shots     int
}

type mrec struct {
	ID      int     `json:"id"`
	Name    string  `json:"name"`
	Place   int     `json:"place"`
	Kills   int     `json:"kills"`
	DeathT  float64 `json:"death_t"`
	Alive   bool    `json:"alive"`
	Cubes   int     `json:"cubes"`
	C60     int     `json:"c60"`
	C120    int     `json:"c120"`
	C180    int     `json:"c180"`
	Dealt   int     `json:"dealt"`
	Taken   int     `json:"taken"`
	By      int     `json:"by"`
	With    string  `json:"with"`
	Loadout string  `json:"loadout"`
	FW      float64 `json:"fw"`
	FH      float64 `json:"fh"`
	FD      float64 `json:"fd"`
	Still   float64 `json:"st"`
	Las     float64 `json:"las"`
	KC      int     `json:"kc"`
	KHP     int     `json:"khp"`
	MHP     int     `json:"mhp"`
	AD      int     `json:"ad"`
}

type engRec struct {
	T      float64 `json:"t"`
	A      int     `json:"a"`
	B      int     `json:"b"`
	D      int     `json:"d"`
	ClA    int     `json:"cla"`
	ClB    int     `json:"clb"`
	Near   bool    `json:"near"`
	NC     int     `json:"nc"`
	Weak   bool    `json:"weak"`
	Few    bool    `json:"few"`
	MC     int     `json:"mc"`
	TC     int     `json:"tc"`
	HPR    float64 `json:"hpr"`
	NAlive int     `json:"alive"`
}

type hitRec struct {
	t    float64
	a, b int
	amt  float64
	w    string
}

func sample(s *sim.Sim) []samp {
	row := make([]samp, len(s.Mechs))
	for i, m := range s.Mechs {
		row[i] = samp{m.Alive, m.Pos[0], m.Pos[1], m.HP, m.MaxHP, m.Cubes, m.Shots}
	}
	return row
}

func matchStats(s *sim.Sim, names map[int]string, pos [][]samp) ([]mrec, []engRec, [][]interface{}) {
	n := len(s.Mechs)
	ms := make([]mrec, n)
	var hits []hitRec
	for _, m := range s.Mechs {
		for _, d := range m.DmgLog {
			if d.By >= 0 && d.By < 100 && d.By != m.ID {
				hits = append(hits, hitRec{d.T, d.By, m.ID, d.Amount, d.Weapon})
			}
		}
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].t < hits[j].t })
	deaths := make([]float64, 0, n)
	for _, m := range s.Mechs {
		if !m.Alive {
			deaths = append(deaths, m.DeathT)
		}
	}
	for i, m := range s.Mechs {
		r := mrec{ID: m.ID, Name: names[m.ID], Place: m.Place, Kills: m.Kills, DeathT: math.Round(m.DeathT*10) / 10, Alive: m.Alive, Cubes: m.Cubes, C60: m.Cubes60, C120: m.Cubes120, C180: -1,
			Dealt: int(m.DmgDealt), Taken: int(m.DmgTaken), By: -1, FW: m.FirstW, FH: -1, FD: -1, Still: -1, Las: -1, KC: -1, KHP: -1, MHP: int(m.MaxHP)}
		if !m.Alive && len(m.DmgLog) > 0 {
			l := m.DmgLog[len(m.DmgLog)-1]
			if m.DeathT-l.T < 5 {
				r.By, r.With = l.By, l.Weapon
			}
		}
		if !m.Alive && r.With == "" {
			r.With = "zone"
		}
		for _, w := range m.W {
			if w != nil {
				if r.Loadout != "" {
					r.Loadout += "+"
				}
				r.Loadout += w.Kind
			}
		}
		if len(pos) > 180 && pos[180][i].alive {
			r.C180 = pos[180][i].cubes
		}
		still, tot := 0, 0
		for sec := 31; sec < len(pos); sec++ {
			if pos[sec][i].alive && pos[sec-1][i].alive {
				tot++
				if math.Hypot(pos[sec][i].x-pos[sec-1][i].x, pos[sec][i].y-pos[sec-1][i].y) < 1.0 {
					still++
				}
			}
		}
		if tot > 60 {
			r.Still = math.Round(1000*float64(still)/float64(tot)) / 1000
		}
		if !m.Alive {
			for _, d := range deaths {
				if d >= m.DeathT {
					r.AD++
				}
			}
			r.AD += s.Alive
			if r.By >= 0 && r.By < n {
				sec := int(m.DeathT)
				if sec >= len(pos) {
					sec = len(pos) - 1
				}
				if sec >= 0 {
					r.KC, r.KHP = pos[sec][r.By].cubes, int(pos[sec][r.By].hp)
				}
			}
		}
		ms[i] = r
	}
	lh := map[[2]int]float64{}
	dmg := make([]float64, n)
	las := make([]float64, n)
	var engs []engRec
	for _, h := range hits {
		a, b := h.a, h.b
		dmg[a] += h.amt
		if h.w == "laser" {
			las[a] += h.amt
		}
		if ms[a].FH < 0 {
			ms[a].FH = math.Round(h.t*10) / 10
		}
		if ms[b].FD < 0 {
			ms[b].FD = math.Round(h.t*10) / 10
		}
		t1, ok1 := lh[[2]int{a, b}]
		t2, ok2 := lh[[2]int{b, a}]
		if (!ok1 || h.t-t1 > 15) && (!ok2 || h.t-t2 > 15) {
			sec := int(h.t)
			if sec < len(pos) && pos[sec][a].alive && pos[sec][b].alive {
				A, B := pos[sec][a], pos[sec][b]
				dd := math.Hypot(A.x-B.x, A.y-B.y)
				e := engRec{T: math.Round(h.t*10) / 10, A: a, B: b, D: int(dd), ClA: -999, ClB: -999, MC: A.cubes, TC: B.cubes, HPR: math.Round(100*B.hp/math.Max(A.hp, 1)) / 100}
				if sec >= 10 && pos[sec-10][a].alive && pos[sec-10][b].alive {
					pa, pb := pos[sec-10][a], pos[sec-10][b]
					d0 := math.Max(math.Hypot(pa.x-pb.x, pa.y-pb.y), 1e-6)
					ux, uy := (pb.x-pa.x)/d0, (pb.y-pa.y)/d0
					e.ClA = int(math.Round((A.x-pa.x)*ux + (A.y-pa.y)*uy))
					e.ClB = int(math.Round(-((B.x-pb.x)*ux + (B.y-pb.y)*uy)))
				}
				lim := math.Max(200, dd+1)
				nearD, nearID := 1e18, -1
				weakV, weakID := 1e18, -1
				fewV := 1 << 30
				for j := 0; j < n; j++ {
					if j == a || !pos[sec][j].alive {
						continue
					}
					o := pos[sec][j]
					dq := math.Hypot(A.x-o.x, A.y-o.y)
					if dq < nearD {
						nearD, nearID = dq, j
					}
					if dq <= lim {
						e.NC++
						if v := o.hp / math.Max(o.maxhp, 1); v < weakV {
							weakV, weakID = v, j
						}
						if o.cubes < fewV {
							fewV = o.cubes
						}
					}
				}
				e.Near, e.Weak, e.Few = nearID == b, weakID == b, fewV == B.cubes
				for j := 0; j < n; j++ {
					if pos[sec][j].alive {
						e.NAlive++
					}
				}
				engs = append(engs, e)
			}
		}
		lh[[2]int{a, b}] = h.t
	}
	for i := range ms {
		if dmg[i] > 0 {
			ms[i].Las = math.Round(100*las[i]/dmg[i]) / 100
		}
	}
	var raw [][]interface{}
	for _, h := range hits {
		raw = append(raw, []interface{}{math.Round(h.t*10) / 10, h.a, h.b, int(h.amt), h.w})
	}
	return ms, engs, raw
}

func gunfireStats(s *sim.Sim, pos [][]samp, farm map[int][]float64) []int {
	out := make([]int, 18)
	n := len(s.Mechs)
	type mh struct {
		t    float64
		a, b int
	}
	var hits []mh
	for _, m := range s.Mechs {
		for _, d := range m.DmgLog {
			if d.By >= 0 && d.By < 100 && d.By != m.ID && d.T < 80 {
				hits = append(hits, mh{d.T, d.By, m.ID})
			}
		}
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].t < hits[j].t })
	for i := 0; i < n; i++ {
		for t := 5; t <= 60; t += 5 {
			if t >= len(pos) || !pos[t][i].alive {
				break
			}
			ft := float64(t)
			busy := false
			att, found := false, false
			for _, h := range hits {
				if h.a != i && h.b != i {
					continue
				}
				if h.t > ft-10 && h.t <= ft {
					busy = true
					break
				}
				if h.t > ft && h.t <= ft+15 && !found {
					found, att = true, h.b == i
				}
			}
			if busy {
				continue
			}
			near := 1e9
			for j := 0; j < n; j++ {
				if j != i && pos[t][j].alive {
					if d := math.Hypot(pos[t][i].x-pos[t][j].x, pos[t][i].y-pos[t][j].y); d < near {
						near = d
					}
				}
			}
			band := 2
			if near < 100 {
				band = 0
			} else if near < 200 {
				band = 1
			}
			grp := 1
			t0 := t - 10
			if t0 < 0 {
				t0 = 0
			}
			if pos[t][i].shots > pos[t0][i].shots {
				grp = 2
			}
			for _, ft2 := range farm[i] {
				if ft2 > ft-10 && ft2 <= ft {
					grp = 0
				}
			}
			out[(band*3+grp)*2]++
			if att {
				out[(band*3+grp)*2+1]++
			}
		}
	}
	return out
}
