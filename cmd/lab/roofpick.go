//go:build ignore

package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"

	"mechbot/bot"
	"mechbot/geo"
	"mechbot/proto"
	"mechbot/sim"
)

func main() {
	f, _ := os.Open("sim/hello.jsonl.gz")
	gz, _ := gzip.NewReader(f)
	sc := bufio.NewScanner(gz)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	sc.Scan()
	var h proto.Hello
	json.Unmarshal(sc.Bytes(), &h)
	w, _ := geo.Load("mapdata/arena")
	w.SetMargin(h.Config.Match.ArenaMargin)
	w.BuildNav(3.0, 9.5, 1.2)
	fl := func(i int) float64 { v, _ := strconv.ParseFloat(os.Args[i], 64); return v }
	sx, sy := fl(1), fl(2)
	px, py, pz := fl(3), fl(4), fl(5)
	variant := "paw"
	if len(os.Args) > 6 {
		variant = os.Args[6]
	}
	h.Config.Match.Mechs = 2
	h.Config.Crates.Count = 0
	h.Config.NPC.Count = 0
	h.Config.Loot.FloorCount = 0
	h.Config.Drops.Times = nil
	var bb *bot.Bot
	s := sim.New(w, &h, 1, func(slot int, hh *proto.Hello) (sim.Controller, string) {
		b := bot.New(hh, w.CloneScratch(), bot.Variant(variant))
		if slot == 0 {
			bb = b
		}
		return b, variant
	})
	m := s.Mechs[0]
	m.Pos = [3]float64{sx, sy, w.Ground(sx, sy)}
	o := s.Mechs[1]
	o.Pos = [3]float64{sx + 400, sy + 300, w.Ground(sx+400, sy+300)}
	s.Picks = append(s.Picks, &sim.Pickup{ID: 8900, Kind: "laser", Pos: [3]float64{px, py, pz}})
	for i := 0; i < 600 && !s.Done; i++ {
		s.Step()
		if i%10 == 0 {
			has := ""
			for _, wp := range m.W {
				if wp != nil {
					has += wp.Kind + " "
				}
			}
			fmt.Printf("t=%.1f pos (%.1f,%.1f,%.1f) energy %.0f mode %s d %.1f has %s\n", s.T, m.Pos[0], m.Pos[1], m.Pos[2], m.Energy, bb.DbgMode(), math.Hypot(m.Pos[0]-px, m.Pos[1]-py), has)
		}
	}
}
