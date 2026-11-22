package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"strings"

	"mechbot/bot"
	"mechbot/geo"
	"mechbot/house"
	"mechbot/proto"
	"mechbot/sim"
)

func loadHello(path string) *proto.Hello {
	f, _ := os.Open(path)
	defer f.Close()
	gz, _ := gzip.NewReader(f)
	sc := bufio.NewScanner(gz)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	sc.Scan()
	var h proto.Hello
	json.Unmarshal(sc.Bytes(), &h)
	return &h
}

func mk(v string, h *proto.Hello, w *geo.World) sim.Controller {
	if len(v) == 6 && v[:5] == "house" {
		return house.New(h, w, nil, int(v[5]-'0'))
	}
	return bot.New(h, w, bot.Variant(v))
}

func main() {
	a := flag.String("a", "turret", "shooter")
	b := flag.String("b", "dancer", "target")
	dist := flag.Float64("d", 100, "starting distance")
	secs := flag.Float64("t", 60, "seconds")
	n := flag.Int("n", 10, "runs")
	verbose := flag.Bool("v", false, "log hits")
	wall := flag.Bool("wall", false, "put a building at the origin; A starts behind it")
	wa := flag.String("wa", "", "slot-1 weapon for A")
	wb := flag.String("wb", "", "slot-1 weapon for B")
	fight := flag.Bool("fight", false, "real fight: no healing, report winner")
	flag.Parse()
	bot.DebugSlot = 0
	base := loadHello("/home/claude/kit/mechfight-kit/check/stream.jsonl.gz")
	wj := []byte(`{"bounds":{"minX":-500,"maxX":500,"minY":-330,"maxY":330},"rules":{"canopyDensity":0.35,"canopyBlockT":0.25},"terrain":{"minX":-513,"minY":-334,"step":1,"nx":1070,"ny":724},"prisms":[PRISMS],"trees":{"fields":["x","y","z","height","crownR","crownBase","trunkR","osm","crownX","crownY"],"data":[]}}`)
	pr := ""
	if *wall {
		pr = `{"kind":"building","base":-1,"top":20,"ring":[-30,-15,0,-15,0,15,-30,15],"holes":[]}`
	}
	wj = []byte(strings.Replace(string(wj), "PRISMS", pr, 1))
	w, err := geo.FromBytes(wj, make([]byte, 1070*724*2))
	if err != nil {
		panic(err)
	}
	w.BuildNav(3.0, 9.5, 1.2)
	ox, oy := -*dist/2, 0.0
	hello := *base
	hello.Config.Match.Mechs = 2
	hello.Config.Loot.FloorCount = 0
	hello.Config.Crates.Count = 0
	hello.Config.NPC.Count = 0
	hello.Config.Drops.Times = nil
	hello.Config.Zone.StartCenter = []float64{0, 0}
	hello.Config.Zone.Phases = []proto.ZonePhase{{HoldS: 1000, ShrinkS: 10, Radius: 500, DPS: 0}}
	var shA, hiA, shB, hiB, lsA, lhA int
	var winA, winB int
	var hpA, hpB, tEnd float64
	var dsum float64
	var dn int
	for r := 0; r < *n; r++ {
		s := sim.New(w, &hello, int64(r+1), func(slot int, h *proto.Hello) (sim.Controller, string) {
			if slot == 0 {
				return mk(*a, h, w), *a
			}
			return mk(*b, h, w), *b
		})
		A, B := s.Mechs[0], s.Mechs[1]
		A.Pos = [3]float64{ox, oy, w.Ground(ox, oy)}
		B.Pos = [3]float64{ox + *dist, oy, w.Ground(ox+*dist, oy)}
		if *wall {
			A.Pos = [3]float64{-36, 6, 0}
			B.Pos = [3]float64{*dist, 0, 0}
		}
		if *wa != "" {
			s.Give(A, *wa)
		}
		if *wb != "" {
			s.Give(B, *wb)
		}
		A.Aim[0], B.Aim[0] = 0, math.Pi
		A.CmdAim, B.CmdAim = A.Aim, B.Aim
		for s.T < *secs && !s.Done {
			s.Step()
			if s.Tick == 200 {
				A.Shots, A.Hits, B.Shots, B.Hits = 0, 0, 0, 0
			}
			if *fight {
				continue
			}
			if *verbose && B.HP < B.MaxHP {
				fmt.Fprintf(os.Stderr, "#### HIT at t=%.2f B pos=(%.1f,%.1f)\n", s.T, B.Pos[0], B.Pos[1])
			}
			A.HP, B.HP = A.MaxHP, B.MaxHP
			if s.Tick%10 == 0 {
				dsum += math.Hypot(A.Pos[0]-B.Pos[0], A.Pos[1]-B.Pos[1])
				dn++
			}
		}
		if *fight {
			if A.Alive && !B.Alive {
				winA++
			} else if B.Alive && !A.Alive {
				winB++
			}
			hpA += math.Max(A.HP, 0)
			hpB += math.Max(B.HP, 0)
			tEnd += s.T
		}
		lsA += A.LasShots
		lhA += A.LasHits
		shA += A.Shots
		hiA += A.Hits
		shB += B.Shots
		hiB += B.Hits
	}
	if *fight {
		fmt.Printf("FIGHT %s(%s) vs %s(%s) d=%.0f: wins %d-%d of %d, mean hp left %.0f vs %.0f, mean t %.1f\n", *a, *wa, *b, *wb, *dist, winA, winB, *n, hpA/float64(*n), hpB/float64(*n), tEnd/float64(*n))
		return
	}
	fmt.Printf("laser %d/%d ", lhA, lsA)
	fmt.Printf("%s -> %s : shots %d acc %.2f | %s -> %s : shots %d acc %.2f | mean dist %.0f\n", *a, *b, shA, float64(hiA)/float64(max(shA, 1)), *b, *a, shB, float64(hiB)/float64(max(shB, 1)), dsum/float64(max(dn, 1)))
}
