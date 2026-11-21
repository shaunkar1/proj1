package bot

import (
	"fmt"
	"os"

	"mechbot/proto"
)

// Debug, when set, prints a status line per decision to stderr.
var Debug = os.Getenv("MECH_DEBUG") != ""

// DebugSlot limits tracing to one slot (-1 = all).
var DebugSlot = -1

func (b *Bot) trace(act interface{}) {
	if !Debug || (DebugSlot >= 0 && b.slot != DebugSlot) {
		return
	}
	me := b.me
	td, tl, thp := -1.0, false, 0.0
	lz := ""
	if t := b.tracks[b.targetID]; t != nil {
		td = hyp(me.Pos[0], me.Pos[1], t.Pos[0], t.Pos[1])
		tl = t.LOS
		thp = t.HP
		lz = fmt.Sprintf(" echg=%v elas=%.2f", t.Charging, b.laserIn(t))
	}
	if las, _ := b.weapon("laser"); las != nil {
		lz += fmt.Sprintf(" mychg=%.2f mycd=%.2f", las.Charge, las.Cooldown)
	}
	if os.Getenv("MECH_DEBUG") == "short" {
		a := act.(proto.Action)
		fmt.Fprintf(os.Stderr, "t=%.1f %s hp=%.0f pos=(%.1f,%.1f) d=%.0f los=%v thp=%.0f%s mv=(%.1f,%.1f) fire=%v\n", b.t, b.mode, me.HP, me.Pos[0], me.Pos[1], td, tl, thp, lz, a.Move[0], a.Move[1], a.Fire)
		return
	}
	fmt.Fprintf(os.Stderr, "t=%.1f %s hp=%.0f en=%.0f pos=(%.0f,%.0f,%.1f) v=%.1f tgt=%d d=%.0f los=%v thp=%.0f retreat=%v act=%+v\n",
		b.t, b.mode, me.HP, me.Energy, me.Pos[0], me.Pos[1], me.Pos[2], hyp(me.Vel[0], me.Vel[1], 0, 0), b.targetID, td, tl, thp, b.retreating, act)
}

// Info describes the last decision (for profiling tools).
func (b *Bot) Info() string {
	return fmt.Sprintf("mode=%s search=%v pathlen=%d tracks=%d proj=%d", b.mode, b.searchTick == b.o.Tick, len(b.path), len(b.tracks), len(b.o.Projectiles))
}
