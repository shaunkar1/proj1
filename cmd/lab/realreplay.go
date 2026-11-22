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
	"strings"

	"mechbot/bot"
	"mechbot/geo"
	"mechbot/proto"
)

type ev struct {
	T    int       `json:"t"`
	E    string    `json:"e"`
	ID   *int      `json:"id"`
	Src  *int      `json:"src"`
	Dst  *int      `json:"dst"`
	By   *int      `json:"by"`
	Amt  float64   `json:"amt"`
	W    string    `json:"w"`
	Op   string    `json:"op"`
	Kind string    `json:"kind"`
	At   []float64 `json:"at"`
	Up   string    `json:"up"`
}

type frame struct {
	T  int         `json:"t"`
	M  [][]float64 `json:"m"`
	N  [][]float64 `json:"n"`
	Z  []float64   `json:"z"`
	Ev []ev        `json:"ev"`
}

type rec struct {
	Seed string `json:"seed"`
	Hdr  struct {
		Mechs []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"mechs"`
		Crates []struct {
			ID  int        `json:"id"`
			Pos [3]float64 `json:"pos"`
		} `json:"crates"`
		Pickups []struct {
			ID   int        `json:"id"`
			Kind string     `json:"kind"`
			Pos  [3]float64 `json:"pos"`
		} `json:"pickups"`
		WeaponKinds []string `json:"weapon_kinds"`
		Zone        []struct {
			Center      [2]float64 `json:"center"`
			Radius      float64    `json:"radius"`
			ShrinkStart float64    `json:"shrink_start"`
			ShrinkEnd   float64    `json:"shrink_end"`
		} `json:"zone"`
	} `json:"hdr"`
	Frames []frame `json:"frames"`
}

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

func angDiff(a, b float64) float64 {
	d := math.Mod(a-b, 2*math.Pi)
	if d > math.Pi {
		d -= 2 * math.Pi
	}
	if d < -math.Pi {
		d += 2 * math.Pi
	}
	return d
}

func main() {
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	var all []rec
	if err := json.Unmarshal(raw, &all); err != nil {
		panic(err)
	}
	var r *rec
	for i := range all {
		if strings.HasSuffix(all[i].Seed, os.Args[2]) {
			r = &all[i]
		}
	}
	if r == nil {
		panic("no such match")
	}
	variant := "paw"
	if len(os.Args) > 3 {
		variant = os.Args[3]
	}
	t0, t1 := 0.0, 1e9
	if len(os.Args) > 5 {
		t0, _ = strconv.ParseFloat(os.Args[4], 64)
		t1, _ = strconv.ParseFloat(os.Args[5], 64)
	}
	me := -1
	for _, m := range r.Hdr.Mechs {
		if strings.Contains(m.Name, "TigerTown") {
			me = m.ID
		}
	}
	h := loadHello("/home/claude/kit/mechfight-kit/check/stream.jsonl.gz")
	h.Slot = me
	w, _ := geo.Load("/home/claude/kit/mechfight-kit/arena")
	w.SetMargin(30)
	w.BuildNav(3.0, 9.5, 1.2)
	b := bot.New(h, w, bot.Variant(variant))
	kinds := r.Hdr.WeaponKinds

	type crate struct {
		pos [3]float64
		hp  float64
	}
	crates := map[int]*crate{}
	for _, c := range r.Hdr.Crates {
		crates[c.ID] = &crate{c.Pos, 300}
	}
	cubes := map[int][3]float64{}
	type pk struct {
		kind string
		pos  [3]float64
	}
	pickups := map[int]pk{}
	for _, p := range r.Hdr.Pickups {
		pickups[p.ID] = pk{p.Kind, p.Pos}
	}
	lastSeen := map[int]float64{}
	prev := map[int][]float64{}
	var upgrades []string
	capac := 0

	ammo, reload, cool := 12.0, 0.0, 0.0
	lasCharge, lasCool := 0.0, 0.0
	var act proto.Action
	realShots, replayShots := 0, 0
	var pending []proto.Event
	myCubes := 0.0

	for fi, f := range r.Frames {
		ts := float64(f.T) / 20
		cur := map[int][]float64{}
		alive := 0
		for _, m := range f.M {
			cur[int(m[0])] = m
			if m[6] > 0 {
				alive++
			}
		}
		mm := cur[me]
		if mm == nil || mm[6] <= 0 {
			break
		}

		realFire := ""
		var sounds []proto.SoundObs
		hear := func(id int, kind string) {
			m := cur[id]
			if m == nil {
				return
			}
			rng := h.Config.Sound.Range[kind]
			d := math.Hypot(m[1]-mm[1], m[2]-mm[2])
			if rng <= 0 || d > rng {
				return
			}
			band := "far"
			if d < 60 {
				band = "near"
			} else if d < 160 {
				band = "mid"
			}
			sounds = append(sounds, proto.SoundObs{Kind: kind, Bearing: math.Atan2(m[2]-mm[2], m[1]-mm[1]) + 0.12*math.Sin(float64(f.T*7+id*13)), Dist: band, Age: 0.05})
		}
		for _, e := range f.Ev {
			switch e.E {
			case "hit":
				if e.Dst != nil && *e.Dst >= 1000 {
					if c := crates[*e.Dst]; c != nil {
						c.hp -= e.Amt
					}
				}
				if e.Dst != nil && *e.Dst == me && e.Src != nil {
					by := *e.Src
					var bearing *float64
					if s := cur[by]; s != nil {
						bb := math.Atan2(s[2]-mm[2], s[1]-mm[1])
						bearing = &bb
					}
					wk := e.W
					if by >= 100 {
						wk = "npc"
					}
					pending = append(pending, proto.Event{Type: "damaged", Amount: e.Amt, Weapon: wk, By: &by, Bearing: bearing})
				}
				if e.Src != nil && *e.Src == me && e.Dst != nil {
					pending = append(pending, proto.Event{Type: "hit", Target: *e.Dst, Amount: e.Amt, Weapon: e.W})
				}
			case "crate":
				if e.ID != nil {
					delete(crates, *e.ID)
				}
			case "cube":
				if e.ID != nil && len(e.At) == 3 {
					cubes[*e.ID] = [3]float64{e.At[0], e.At[1], e.At[2]}
				}
			case "pickup":
				if e.ID != nil {
					delete(cubes, *e.ID)
					if e.By != nil && *e.By == me {
						myCubes = mm[9]
						pending = append(pending, proto.Event{Type: "pickup", Cubes: mm[9]})
					}
				}
			case "weapon":
				if e.ID == nil {
					break
				}
				switch e.Op {
				case "take":
					delete(pickups, *e.ID)
				case "spawn", "drop":
					if len(e.At) == 3 {
						pickups[*e.ID] = pk{e.Kind, [3]float64{e.At[0], e.At[1], e.At[2]}}
					}
				}
			case "death":
				if e.By != nil && *e.By == me && e.ID != nil && *e.ID < 16 {
					pending = append(pending, proto.Event{Type: "kill", Target: *e.ID})
				}
			case "upgrade":
				if e.ID != nil && *e.ID == me {
					upgrades = append(upgrades, e.Up)
					if e.Up == "capacitor" {
						capac++
					}
				}
			case "fire":
				if e.ID != nil && *e.ID == me {
					realFire += " REAL-FIRE:" + e.W
					realShots++
				} else if e.ID != nil && *e.ID < 100 {
					hear(*e.ID, e.W)
				}
			case "laser":
				if e.ID != nil && *e.ID == me {
					realFire += " REAL-LASER"
				} else if e.ID != nil && *e.ID < 100 {
					hear(*e.ID, "laser")
				}
			case "zap":
				if e.ID != nil && *e.ID != me && *e.ID < 100 {
					hear(*e.ID, "lightning")
				}
			}
		}
		_ = myCubes

		o := proto.Obs{Type: "obs", Tick: f.T, T: ts, Alive: alive}
		st := h.Config.Mech
		sc := 1 + 0.05*mm[9]
		st.Radius *= sc
		st.Height *= sc
		st.EyeHeight *= sc
		st.MuzzleHeight *= sc
		var vel proto.Vec3
		if p := prev[me]; p != nil && fi > 0 {
			dt := float64(f.T-r.Frames[fi-1].T) / 20
			if dt > 0 {
				vel = proto.Vec3{(mm[1] - p[1]) / dt, (mm[2] - p[2]) / dt, (mm[3] - p[3]) / dt}
			}
		}
		o.Self = proto.SelfObs{ID: me, Pos: proto.Vec3{mm[1], mm[2], mm[3]}, Vel: vel, Aim: proto.Vec2{mm[4], mm[5]}, HP: mm[6], MaxHP: mm[10],
			Energy: mm[7], MaxEnergy: 100 + 30*float64(capac), Grounded: int(mm[8])&1 != 0, Cubes: mm[9], Upgrades: upgrades, InZone: true, Stats: st}
		for s := 0; s < 2; s++ {
			k := int(mm[11+s])
			if k < 0 || k >= len(kinds) {
				o.Self.Weapons = append(o.Self.Weapons, nil)
				continue
			}
			wo := &proto.WeaponObs{Kind: kinds[k], Stats: h.Config.Weapons[kinds[k]]}
			switch kinds[k] {
			case "gun":
				wo.Ammo, wo.Reload, wo.Cooldown = ammo, reload, math.Max(0, cool)
			case "laser":
				wo.Charge, wo.Cooldown = lasCharge, math.Max(0, lasCool)
			}
			o.Self.Weapons = append(o.Self.Weapons, wo)
		}
		ex, ey, ez := mm[1], mm[2], mm[3]+st.EyeHeight
		see := func(x, y, zc, zt, rng float64) bool {
			d := math.Hypot(x-ex, y-ey)
			if d > rng {
				return false
			}
			los := w.Clear(ex, ey, ez, x, y, zc) || w.Clear(ex, ey, ez, x, y, zt)
			if !los {
				return false
			}
			if d < 45 {
				return true
			}
			if math.Abs(angDiff(math.Atan2(y-ey, x-ex), mm[4])) > math.Pi/3 {
				return false
			}
			return w.Sees(ex, ey, ez, x, y, zc) || w.Sees(ex, ey, ez, x, y, zt)
		}
		for id, m := range cur {
			if id == me || m[6] <= 0 {
				continue
			}
			s2 := 1 + 0.05*m[9]
			if see(m[1], m[2], m[3]+4.5*s2, m[3]+9*s2, 200) {
				lastSeen[id] = ts
			}
			if ls, ok := lastSeen[id]; !ok || ts-ls > 5 {
				continue
			}
			a := proto.ActorObs{ID: id, Kind: "mech", Pos: proto.Vec3{m[1], m[2], m[3]}, Aim: proto.Vec2{m[4], m[5]}, HP: m[6], MaxHP: m[10], Cubes: m[9], Charging: int(m[8])&4 != 0}
			if p := prev[id]; p != nil {
				a.Vel = proto.Vec3{(m[1] - p[1]) / 0.1, (m[2] - p[2]) / 0.1, (m[3] - p[3]) / 0.1}
			}
			for s := 0; s < 2; s++ {
				if k := int(m[11+s]); k >= 0 && k < len(kinds) {
					a.Weapons = append(a.Weapons, kinds[k])
				}
			}
			o.Mechs = append(o.Mechs, a)
		}
		for _, n := range f.N {
			id := int(n[0])
			if n[5] <= 0 {
				continue
			}
			if see(n[1], n[2], n[3]+1.75, n[3]+3.5, 200) {
				lastSeen[id] = ts
			}
			if ls, ok := lastSeen[id]; !ok || ts-ls > 5 {
				continue
			}
			o.Mechs = append(o.Mechs, proto.ActorObs{ID: id, Kind: "npc", Pos: proto.Vec3{n[1], n[2], n[3]}, HP: n[5], MaxHP: 350})
		}
		for id, c := range crates {
			if see(c.pos[0], c.pos[1], c.pos[2]+1.2, c.pos[2]+2.4, 200) {
				o.Crates = append(o.Crates, proto.ItemObs{ID: id, Pos: proto.Vec3{c.pos[0], c.pos[1], c.pos[2]}, HP: c.hp})
			}
		}
		for id, c := range cubes {
			if see(c[0], c[1], c[2]+0.5, c[2]+1, 200) {
				o.Cubes = append(o.Cubes, proto.ItemObs{ID: id, Pos: proto.Vec3{c[0], c[1], c[2]}})
			}
		}
		for id, p := range pickups {
			if see(p.pos[0], p.pos[1], p.pos[2]+0.5, p.pos[2]+1, 200) {
				o.Pickups = append(o.Pickups, proto.PickupObs{ID: id, Kind: p.kind, Pos: proto.Vec3{p.pos[0], p.pos[1], p.pos[2]}})
			}
		}

		zs := r.Hdr.Zone
		o.Zone = proto.ZoneObs{Center: proto.Vec2{f.Z[0], f.Z[1]}, Radius: f.Z[2], NextCenter: proto.Vec2{f.Z[0], f.Z[1]}, NextRadius: f.Z[2]}
		for k := 1; k < len(zs); k++ {
			if ts < zs[k].ShrinkEnd {
				o.Zone.NextCenter = proto.Vec2{zs[k].Center[0], zs[k].Center[1]}
				o.Zone.NextRadius = zs[k].Radius
				o.Zone.ShrinkStart, o.Zone.ShrinkEnd = zs[k].ShrinkStart, zs[k].ShrinkEnd
				if k-1 < len(h.Config.Zone.Phases) {
					o.Zone.DPS = h.Config.Zone.Phases[k-1].DPS
				}
				break
			}
		}
		o.Self.InZone = math.Hypot(mm[1]-f.Z[0], mm[2]-f.Z[1]) <= f.Z[2]
		o.Events = pending
		pending = nil
		o.Sounds = sounds

		act = b.Act(&o)

		fired := ""
		gi, li := -1, -1
		for s := 0; s < 2; s++ {
			if k := int(mm[11+s]); k >= 0 && k < len(kinds) {
				if kinds[k] == "gun" {
					gi = s
				}
				if kinds[k] == "laser" {
					li = s
				}
			}
		}
		for k := 0; k < 2; k++ {
			if reload > 0 {
				reload -= 0.05
				if reload <= 1e-9 {
					reload, ammo = 0, 12
				}
			}
			cool -= 0.05
			lasCool -= 0.05
			if k == 0 && act.Reload && reload == 0 && ammo < 12 {
				reload = 2.4
				fired += " RELOAD"
			}
			if gi >= 0 && act.Fire[gi] && reload == 0 && cool <= 1e-9 && ammo > 0 {
				ammo--
				cool = 0.24
				fired += " fire"
				replayShots++
				if ammo == 0 {
					reload = 2.4
				}
			}
			if li >= 0 {
				if act.Fire[li] && lasCool <= 0 {
					lasCharge += 0.05
				} else if lasCharge > 0 {
					fired += " laser"
					lasCharge, lasCool = 0, 1.65
				}
			}
		}
		if ts >= t0 && ts <= t1 {
			nc, bd := -1, 1e9
			for id, c := range crates {
				if d := math.Hypot(c.pos[0]-mm[1], c.pos[1]-mm[2]); d < bd {
					bd, nc = d, id
				}
			}
			if a := os.Getenv("ATT"); a != "" {
				id, _ := strconv.Atoi(a)
				if am := cur[id]; am != nil {
					dx, dy := am[1]-mm[1], am[2]-mm[2]
					dd := math.Hypot(dx, dy)
					mv := math.Hypot(act.Move[0], act.Move[1])
					c := 0.0
					if mv > 0.2 && dd > 1 {
						c = (act.Move[0]*dx + act.Move[1]*dy) / (mv * dd)
					}
					realFire += fmt.Sprintf(" ATT cos %+.2f d %.0f", c, dd)
				}
			}
			fmt.Printf("%6.1f %-34s hp %4.0f ammo %2.0f rl %.1f mv(%+.1f,%+.1f) fire %v reload %v crate %d@%.0f vis(c%d q%d p%d m%d)%s%s\n",
				ts, b.Info(), mm[6], ammo, reload, act.Move[0], act.Move[1], act.Fire, act.Reload, nc, bd, len(o.Crates), len(o.Cubes), len(o.Pickups), len(o.Mechs), fired, realFire)
		}
		for id, m := range cur {
			prev[id] = m
		}
	}
	fmt.Println("real gun shots", realShots, "replayed bot gun shots", replayShots)
}
