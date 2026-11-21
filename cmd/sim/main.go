// Command sim plays local matches and reports placement statistics.
package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"mechbot/bot"
	"mechbot/geo"
	"mechbot/house"
	"mechbot/proto"
	"mechbot/sim"
)

type timed struct {
	inner sim.Controller
	max   time.Duration
	sum   time.Duration
	n     int
}

func (t *timed) Act(o *proto.Obs) proto.Action {
	t0 := time.Now()
	a := t.inner.Act(o)
	d := time.Since(t0)
	if d > t.max {
		t.max = d
	}
	t.sum += d
	t.n++
	return a
}

type result struct {
	place, kills       int
	deathT             float64
	maxMs, avgMs       float64
	cubes              int
	dealt, taken, zone float64
	name               string
	shots, hits        int
}

func loadHello(path string) *proto.Hello {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		panic(err)
	}
	sc := bufio.NewScanner(gz)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	sc.Scan()
	var h proto.Hello
	if err := json.Unmarshal(sc.Bytes(), &h); err != nil {
		panic(err)
	}
	return &h
}

func main() {
	n := flag.Int("n", 20, "matches")
	mode := flag.String("mode", "practice", "practice | ffa")
	seed := flag.Int64("seed", 1, "first seed")
	cand := flag.String("cand", "paw", "candidate variant(s), comma separated (ffa: the field)")
	workers := flag.Int("j", 2, "parallel matches")
	verbose := flag.Bool("v", false, "log deaths")
	stream := flag.String("hello", "/home/claude/kit/mechfight-kit/check/stream.jsonl.gz", "recorded stream with a hello")
	late := flag.Float64("late", 290, "squad all-hunt time")
	expert := flag.String("expert", "paw", "variant used for expert house bots")
	wa := flag.String("wa", "", "duel: slot-1 weapon for the candidate")
	wb := flag.String("wb", "", "duel: slot-1 weapon for the opponent")
	aggro := flag.Int("aggro", 1, "non-hunter aggression 0..2")
	opp := flag.String("opp", "house0", "duel opponent: houseN or a variant")
	sep := flag.Float64("sep", 130, "duel spawn separation")
	loot := flag.Bool("loot", false, "duel: keep loot, crates, NPCs")
	trace := flag.String("trace", "", "write a trace of the first match to this file")
	flag.Parse()

	base := loadHello(*stream)
	w0, err := geo.Load("mapdata/arena")
	if err != nil {
		panic(err)
	}
	w0.BuildNav(3.0, 9.5, 1.2)
	cands := strings.Split(*cand, ",")

	var mu sync.Mutex
	res := map[string][]result{}
	jobs := make(chan int64)
	var wg sync.WaitGroup
	for k := 0; k < *workers; k++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := w0.CloneScratch()
			for sd := range jobs {
				var tm []*timed
				names := map[int]string{}
				sq := house.NewSquad()
				sq.LateT = *late
				sq.Aggro = *aggro
				mySlot := int(sd*7) % 16
				hello := *base
				if *mode == "duel" {
					mySlot = int(sd) % 2
					hello.Config.Match.Mechs = 2
					hello.Config.Match.SpawnMinSeparation = *sep
					hello.Config.Match.SpawnZoneFraction = 0.5
					if !*loot {
						hello.Config.Loot.FloorCount = 0
						hello.Config.Crates.Count = 0
						hello.Config.NPC.Count = 0
						hello.Config.Drops.Times = nil
					}
				}
				bot.DebugSlot = mySlot
				mk := func(slot int, h *proto.Hello) (sim.Controller, string) {
					if *mode == "duel" {
						if slot == mySlot {
							v := cands[0]
							var inner sim.Controller = bot.New(h, w, bot.Variant(v))
							if v == "starter" {
								inner = &house.Starter{Reach: h.Config.Loot.PickupRadius}
							}
							t := &timed{inner: inner}
							tm = append(tm, t)
							names[slot] = v
							return t, v
						}
						v := *opp
						names[slot] = "opp:" + v
						if strings.HasPrefix(v, "house") {
							return house.New(h, w, nil, int(v[len(v)-1]-'0')), v
						}
						return bot.New(h, w, bot.Variant(v)), v
					}
					if *mode == "practice" {
						if slot == mySlot {
							v := cands[int(sd)%len(cands)]
							var inner sim.Controller = bot.New(h, w, bot.Variant(v))
							if v == "starter" {
								inner = &house.Starter{Reach: h.Config.Loot.PickupRadius}
							}
							t := &timed{inner: inner}
							tm = append(tm, t)
							names[slot] = v
							return t, v
						}
						k := slot
						if slot > mySlot {
							k--
						}
						switch {
						case k < 4:
							return house.New(h, w, sq, 0), "easy"
						case k < 8:
							return house.New(h, w, sq, 1), "medium"
						case k < 12:
							return house.New(h, w, sq, 2), "hard"
						default:
							return house.NewExpert(h, w, sq, bot.Variant(*expert)), "expert"
						}
					}
					v := cands[(slot+int(sd))%len(cands)]
					names[slot] = v
					if strings.HasPrefix(v, "house") {
						lvl := int(v[len(v)-1] - '0')
						return house.New(h, w, nil, lvl), v
					}
					t := &timed{inner: bot.New(h, w, bot.Variant(v))}
					tm = append(tm, t)
					return t, v
				}
				s := sim.New(w, &hello, sd, mk)
				if *mode == "duel" {
					if *wa != "" {
						s.Give(s.Mechs[mySlot], *wa)
					}
					if *wb != "" {
						s.Give(s.Mechs[1-mySlot], *wb)
					}
				}
				if *verbose {
					s.Log = func(f string, a ...interface{}) { fmt.Printf("[%d] "+f+"\n", append([]interface{}{sd}, a...)...) }
				}
				if *trace != "" && sd == *seed {
					s.Rec = &sim.Recorder{}
				}
				if *mode == "practice" && s.Rec == nil {
					for !s.Done && s.Mechs[mySlot].Alive {
						s.Step()
					}
				} else {
					s.Run()
				}
				if s.Rec != nil {
					f, _ := os.Create(*trace)
					json.NewEncoder(f).Encode(map[string]interface{}{"frames": s.Rec.Frames, "me": mySlot})
					f.Close()
				}
				mu.Lock()
				ti := 0
				for _, m := range s.Mechs {
					nm, ok := names[m.ID]
					if !ok {
						continue
					}
					r := result{place: m.Place, kills: m.Kills, deathT: m.DeathT, cubes: m.Cubes, dealt: m.DmgDealt, taken: m.DmgTaken, zone: m.ZoneDmg, name: nm, shots: m.Shots, hits: m.Hits}
					if ti < len(tm) {
						r.maxMs = float64(tm[ti].max.Microseconds()) / 1000
						r.avgMs = float64(tm[ti].sum.Microseconds()) / 1000 / float64(max(tm[ti].n, 1))
						ti++
					}
					res[nm] = append(res[nm], r)
					if *verbose && *mode != "ffa" && m.ID == mySlot {
						type agg struct {
							amt  float64
							n    int
							dist float64
						}
						by := map[string]*agg{}
						for _, d := range m.DmgLog {
							if d.T < m.DeathT-25 {
								continue
							}
							nm := "npc"
							if d.By < 100 && d.By >= 0 {
								nm = fmt.Sprintf("%d:%s", d.By, s.Mechs[d.By].Name)
							}
							k := nm + "/" + d.Weapon
							if by[k] == nil {
								by[k] = &agg{}
							}
							by[k].amt += d.Amount
							by[k].n++
							by[k].dist += d.Dist
						}
						var ks []string
						for k, a := range by {
							ks = append(ks, fmt.Sprintf("%s %.0f(n%d,d%.0f)", k, a.amt, a.n, a.dist/float64(a.n)))
						}
						sort.Strings(ks)
						wk := ""
						for _, w := range m.W {
							if w != nil {
								wk += w.Kind + " "
							}
						}
						fmt.Printf("[%d]   last25s: %s | my weapons: %s| ups %v\n", sd, strings.Join(ks, "; "), wk, m.Upgrades)
					}
					if *verbose && *mode != "ffa" {
						fmt.Printf("[%d] %s place %d kills %d deathT %.0f cubes %d dealt %.0f taken %.0f zone %.0f\n", sd, nm, m.Place, m.Kills, m.DeathT, m.Cubes, m.DmgDealt, m.DmgTaken, m.ZoneDmg)
					}
				}
				mu.Unlock()
			}
		}()
	}
	t0 := time.Now()
	for i := 0; i < *n; i++ {
		jobs <- *seed + int64(i)
	}
	close(jobs)
	wg.Wait()
	var keys []string
	for k := range res {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		rs := res[k]
		var pts, kills, dt, mx, av, cubes, dealt, taken, zone float64
		hist := make([]int, 17)
		shots, hits := 0, 0
		for _, r := range rs {
			pts += float64(16 - r.place)
			kills += float64(r.kills)
			dt += r.deathT
			cubes += float64(r.cubes)
			dealt += r.dealt
			taken += r.taken
			zone += r.zone
			if r.maxMs > mx {
				mx = r.maxMs
			}
			av += r.avgMs
			hist[r.place]++
			shots += r.shots
			hits += r.hits
		}
		c := float64(len(rs))
		fmt.Printf("%-10s n=%-4d score %.2f  kills %.2f  life %.0fs  cubes %.1f  dealt %.0f taken %.0f zone %.0f  ms avg %.2f max %.1f  acc %.2f  places %v\n",
			k, len(rs), pts/c, kills/c, dt/c, cubes/c, dealt/c, taken/c, zone/c, av/c, mx, float64(hits)/float64(max(shots, 1)), hist[1:])
	}
	fmt.Println("elapsed", time.Since(t0).Round(time.Millisecond))
}
