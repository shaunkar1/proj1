package house

import (
	"mechbot/bot"
	"mechbot/geo"
	"mechbot/proto"
)

// Expert is a squad member driven by the real pilot: it never targets
// squad-mates and, when hunting, is fed the squad's shared sightings.
type Expert struct {
	inner *bot.Bot
	sq    *Squad
	id    int
}

func NewExpert(h *proto.Hello, w *geo.World, sq *Squad, p bot.Params) *Expert {
	sq.Members[h.Slot] = 3
	sq.Alive[h.Slot] = 0
	return &Expert{inner: bot.New(h, w, p), sq: sq, id: h.Slot}
}

func (e *Expert) Act(o *proto.Obs) proto.Action {
	e.sq.Report(o)
	hunter := e.sq.IsHunter(e.id, o.Alive)
	c := *o
	c.Mechs = nil
	seen := map[int]bool{}
	for _, m := range o.Mechs {
		if m.Kind == "mech" {
			if _, friend := e.sq.Members[m.ID]; friend {
				continue
			}
			if !hunter && (m.Pos[0]-o.Self.Pos[0])*(m.Pos[0]-o.Self.Pos[0])+(m.Pos[1]-o.Self.Pos[1])*(m.Pos[1]-o.Self.Pos[1]) > 90*90 {
				continue
			}
			seen[m.ID] = true
		}
		c.Mechs = append(c.Mechs, m)
	}
	if hunter {
		for id, s := range e.sq.Sight {
			if !seen[id] && o.T-s.T < 0.3 {
				c.Mechs = append(c.Mechs, proto.ActorObs{ID: id, Kind: "mech", Pos: s.Pos, Vel: s.Vel, HP: s.HP, MaxHP: 1400, Weapons: []string{"gun"}})
			}
		}
	}
	return e.inner.Act(&c)
}
