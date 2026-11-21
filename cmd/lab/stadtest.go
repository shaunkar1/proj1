//go:build ignore

package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"

	"mechbot/bot"
	"mechbot/geo"
	"mechbot/proto"
	"mechbot/sim"
)

func main() {
	f, _ := os.Open("/home/claude/kit/mechfight-kit/check/stream.jsonl.gz")
	gz, _ := gzip.NewReader(f)
	sc := bufio.NewScanner(gz)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	sc.Scan()
	var base proto.Hello
	json.Unmarshal(sc.Bytes(), &base)
	w, _ := geo.Load("mapdata/arena")
	w.BuildNav(3.0, 9.5, 1.2)
	hello := base
	hello.Config.Match.Mechs = 2
	hello.Config.NPC.Count = 0
	spots := [][2]float64{{-290.85, 138}, {-276, 128}, {-250, 200}, {-330, 100}, {-262, 60}, {-300, 30}, {-240, 230}}
	for i, sp := range spots {
		s := sim.New(w, &hello, int64(i+1), func(slot int, h *proto.Hello) (sim.Controller, string) {
			p := bot.DefaultParams()
			return bot.New(h, w, p), "paw"
		})
		m := s.Mechs[0]
		m.Pos = [3]float64{sp[0], sp[1], w.Ground(sp[0], sp[1])}
		s.Mechs[1].Pos = [3]float64{480, -300, w.Ground(480, -300)}
		out := -1.0
		for s.T < 200 && m.Alive {
			s.Step()
			if out < 0 && w.Nav.WalkableMain(m.Pos[0], m.Pos[1]) && m.Grounded {
				out = s.T
			}
			if os.Getenv("V") != "" && s.Tick%40 == 0 && i == 0 {
				fmt.Printf("  t=%.0f pos=(%.0f,%.0f,%.1f) en=%.0f hp=%.0f\n", s.T, m.Pos[0], m.Pos[1], m.Pos[2], m.Energy, m.HP)
			}
		}
		fmt.Printf("spawn (%.0f,%.0f): out at t=%.1f alive=%v hp=%.0f zoneDmg=%.0f\n", sp[0], sp[1], out, m.Alive, m.HP, m.ZoneDmg)
	}
}
