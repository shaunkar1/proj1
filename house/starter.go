package house

import (
	"math"

	"mechbot/proto"
)

type Starter struct {
	Reach float64
}

func (s *Starter) Act(o *proto.Obs) proto.Action {
	me := &o.Self
	x, y, z := me.Pos[0], me.Pos[1], me.Pos[2]
	goal := o.Zone.NextCenter
	act := proto.Action{Tick: o.Tick}
	if o.Offer != nil {
		zero := 0
		act.Pick = &zero
	}
	if me.Weapons[1] == nil {
		for _, p := range o.Pickups {
			if math.Hypot(p.Pos[0]-x, p.Pos[1]-y) < s.Reach {
				if p.Kind != "gun" {
					id := p.ID
					act.Take = &id
				}
				break
			}
		}
	}
	yaw := math.Atan2(goal[1]-y, goal[0]-x)
	act.Move = proto.Vec2{math.Cos(yaw), math.Sin(yaw)}
	act.Aim = proto.Vec2{yaw, 0}
	act.Boost = !me.InZone
	act.Jump = o.Tick%40 == 0
	var best *proto.ActorObs
	bd := math.Inf(1)
	for i := range o.Mechs {
		m := &o.Mechs[i]
		if m.Kind != "mech" {
			continue
		}
		d := math.Sqrt((m.Pos[0]-x)*(m.Pos[0]-x) + (m.Pos[1]-y)*(m.Pos[1]-y) + (m.Pos[2]-z)*(m.Pos[2]-z))
		if d < bd {
			bd, best = d, m
		}
	}
	if best != nil {
		t := best.Pos
		act.Aim = proto.Vec2{math.Atan2(t[1]-y, t[0]-x), math.Atan2(t[2]-z-2.5, math.Hypot(t[0]-x, t[1]-y))}
		act.Fire = [2]bool{true, true}
	}
	return act
}
