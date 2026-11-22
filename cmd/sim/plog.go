package main

import (
	"math"

	"mechbot/house"
	"mechbot/sim"
)

type pracLog struct {
	Seed   int64        `json:"seed"`
	Name   string       `json:"name"`
	Kills  int          `json:"kills"`
	DeathT float64      `json:"death_t"`
	Zone   float64      `json:"zone"`
	S      []pracSample `json:"s"`
	D      []pracDmg    `json:"d"`
	Ups    []string     `json:"ups"`
	W      []string     `json:"w"`
	Lvl    map[int]int  `json:"lvl"`
	KT     []float64    `json:"kt"`
	KL     []int        `json:"kl"`
	KX     [][5]float64 `json:"kx"`
	lastK  int
	alive  map[int]bool
	sq     *house.Squad
}

type pracSample struct {
	T     float64  `json:"t"`
	HP    float64  `json:"hp"`
	Max   float64  `json:"max"`
	En    float64  `json:"en"`
	Mode  string   `json:"mode"`
	Tgt   int      `json:"tgt"`
	Pos   [2]int   `json:"pos"`
	Cubes int      `json:"cubes"`
	Kills int      `json:"kills"`
	InZ   float64  `json:"inz"`
	H     [][7]int `json:"h"`
	Drops [][3]int `json:"drops"`
}

type pracDmg struct {
	T  float64 `json:"t"`
	By int     `json:"by"`
	W  string  `json:"w"`
	A  float64 `json:"a"`
	D  float64 `json:"d"`
	HP float64 `json:"hp"`
	R  int     `json:"r"`
}

func (p *pracLog) sample(s *sim.Sim, sq *house.Squad, me int, tm []*timed) {
	m := s.Mechs[me]
	p.sq = sq
	if p.alive == nil {
		p.alive = map[int]bool{}
		for _, o := range s.Mechs {
			p.alive[o.ID] = true
		}
	}
	for _, o := range s.Mechs {
		if p.alive[o.ID] && !o.Alive {
			p.alive[o.ID] = false
			if o.ID != me {
				p.KT = append(p.KT, math.Round(s.T*10)/10)
				p.KL = append(p.KL, sq.Members[o.ID])
				p.KX = append(p.KX, deathRec(s, o, me))
			}
		}
	}
	mode, tgt := "", -1
	if len(tm) > 0 {
		if d, ok := tm[0].inner.(interface{ DbgMode() string }); ok {
			mode = d.DbgMode()
		}
		if d, ok := tm[0].inner.(interface{ DbgTarget() int }); ok {
			tgt = d.DbgTarget()
		}
	}
	zc, zr := s.ZoneNow()
	sm := pracSample{T: math.Round(s.T*10) / 10, HP: math.Round(m.HP), Max: math.Round(m.MaxHP), En: math.Round(m.Energy), Mode: mode, Tgt: tgt,
		Pos: [2]int{int(m.Pos[0]), int(m.Pos[1])}, Cubes: m.Cubes, Kills: m.Kills, InZ: math.Round(zr - math.Hypot(m.Pos[0]-zc[0], m.Pos[1]-zc[1]))}
	alive := 0
	for _, o := range s.Mechs {
		if o.Alive {
			alive++
		}
	}
	for _, o := range s.Mechs {
		if o.ID == me || !o.Alive {
			continue
		}
		d := math.Hypot(o.Pos[0]-m.Pos[0], o.Pos[1]-m.Pos[1])
		if d > 320 {
			continue
		}
		h, c, wb := 0, 0, 0
		if sq.IsHunter(o.ID, alive) {
			h = 1
		}
		if s.W.Clear(o.Pos[0], o.Pos[1], o.Pos[2]+7, m.Pos[0], m.Pos[1], m.Pos[2]+4.5) {
			c = 1
		}
		for _, w := range o.W {
			if w == nil {
				continue
			}
			switch w.Kind {
			case "mortar":
				wb |= 1
			case "laser":
				wb |= 2
			case "lightning":
				wb |= 4
			}
		}
		sm.H = append(sm.H, [7]int{o.ID, int(d), int(o.HP), h, c, wb, o.Cubes})
	}
	for _, d := range s.Drops {
		li := d.LandT - s.T
		l := 0
		if d.Landed {
			l, li = 1, 0
		}
		sm.Drops = append(sm.Drops, [3]int{int(math.Hypot(d.Pos[0]-m.Pos[0], d.Pos[1]-m.Pos[1])), l, int(li)})
	}
	p.S = append(p.S, sm)
}

func (p *pracLog) finish(s *sim.Sim, me int, name string) {
	m := s.Mechs[me]
	p.Name, p.Kills, p.DeathT, p.Zone = name, m.Kills, math.Round(m.DeathT*10)/10, math.Round(m.ZoneDmg)
	for _, o := range s.Mechs {
		if p.alive != nil && p.alive[o.ID] && !o.Alive && o.ID != me {
			p.KT = append(p.KT, math.Round(s.T*10)/10)
			p.KX = append(p.KX, deathRec(s, o, me))
			if p.sq != nil {
				p.KL = append(p.KL, p.sq.Members[o.ID])
			}
		}
	}
	for _, d := range m.DmgLog {
		r := 0
		if d.Rear {
			r = 1
		}
		p.D = append(p.D, pracDmg{math.Round(d.T*10) / 10, d.By, d.Weapon, math.Round(d.Amount), math.Round(d.ADist), math.Round(d.HP), r})
	}
	p.Ups = m.Upgrades
	for _, w := range m.W {
		if w != nil {
			p.W = append(p.W, w.Kind)
		}
	}
	p.Lvl = map[int]int{}
	if p.sq != nil {
		for id, l := range p.sq.Members {
			p.Lvl[id] = l
		}
	}
}

func deathRec(s *sim.Sim, o *sim.Mech, me int) [5]float64 {
	r := [5]float64{math.Round(s.T*10) / 10, -1, 4, -1, 0}
	tot, mine := 0.0, 0.0
	for _, d := range o.DmgLog {
		tot += d.Amount
		if d.By == me {
			mine += d.Amount
			r[3] = math.Round((s.T-d.T)*10) / 10
		}
	}
	if n := len(o.DmgLog); n > 0 {
		d := o.DmgLog[n-1]
		r[1] = float64(d.By)
		switch d.Weapon {
		case "gun":
			r[2] = 0
		case "mortar":
			r[2] = 1
		case "laser":
			r[2] = 2
		case "lightning":
			r[2] = 3
		}
	}
	if tot > 0 {
		r[4] = math.Round(mine/tot*100) / 100
	}
	return r
}
