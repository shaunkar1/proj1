// Command bot speaks mechfight/1 on stdin/stdout.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"runtime/debug"
	"strconv"
	"time"

	"mechbot/bot"
	"mechbot/geo"
	"mechbot/mapdata"
	"mechbot/proto"
)

func logf(f string, a ...interface{}) { fmt.Fprintf(os.Stderr, f+"\n", a...) }

type job struct {
	obs *proto.Obs
	res chan proto.Action
}

func ff(v float64) string { return strconv.FormatFloat(v, 'f', 4, 64) }

func encode(a *proto.Action) []byte {
	b := make([]byte, 0, 160)
	b = append(b, `{"tick":`...)
	b = strconv.AppendInt(b, int64(a.Tick), 10)
	b = append(b, `,"move":[`...)
	b = append(b, ff(a.Move[0])...)
	b = append(b, ',')
	b = append(b, ff(a.Move[1])...)
	b = append(b, `],"aim":[`...)
	b = append(b, ff(a.Aim[0])...)
	b = append(b, ',')
	b = append(b, ff(a.Aim[1])...)
	b = append(b, `],"boost":`...)
	b = strconv.AppendBool(b, a.Boost)
	b = append(b, `,"jump":`...)
	b = strconv.AppendBool(b, a.Jump)
	b = append(b, `,"fire":[`...)
	b = strconv.AppendBool(b, a.Fire[0])
	b = append(b, ',')
	b = strconv.AppendBool(b, a.Fire[1])
	b = append(b, `],"reload":`...)
	b = strconv.AppendBool(b, a.Reload)
	if a.Pick != nil {
		b = append(b, `,"pick":`...)
		b = strconv.AppendInt(b, int64(*a.Pick), 10)
	}
	if a.Take != nil {
		b = append(b, `,"take":`...)
		b = strconv.AppendInt(b, int64(*a.Take), 10)
	}
	if a.Drop != nil {
		b = append(b, `,"drop":`...)
		b = strconv.AppendInt(b, int64(*a.Drop), 10)
	}
	b = append(b, "}\n"...)
	return b
}

func main() {
	debug.SetGCPercent(400)
	debug.SetMemoryLimit(380 << 20)
	out := bufio.NewWriterSize(os.Stdout, 1<<16)
	send := func(b []byte) {
		out.Write(b)
		out.Flush()
	}
	lines := make(chan []byte, 256)
	go func() {
		r := bufio.NewReaderSize(os.Stdin, 1<<20)
		for {
			ln, err := r.ReadBytes('\n')
			if len(ln) > 1 {
				lines <- ln
			}
			if err != nil {
				close(lines)
				return
			}
		}
	}()

	var pilot *bot.Bot
	jobs := make(chan job, 1)

	go func() {
		for j := range jobs {
			func() {
				defer func() {
					if r := recover(); r != nil {
						logf("panic at tick %d: %v\n%s", j.obs.Tick, r, debug.Stack())
						j.res <- proto.Action{Tick: -1}
					}
				}()
				j.res <- pilot.Act(j.obs)
			}()
		}
	}()

	var last proto.Action
	var budget = 34 * time.Millisecond
	busy := false
	var pending chan proto.Action
	for ln := range lines {

		newest := ln
	drain:
		for {
			select {
			case more, ok := <-lines:
				if !ok {
					break drain
				}

				if len(newest) > 12 && (string(newest[:14]) == `{"type":"hello` || string(newest[:12]) == `{"type":"end`) {
					handle(newest, &pilot, send)
				}
				newest = more
			default:
				break drain
			}
		}
		var head struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(newest, &head); err != nil {
			continue
		}
		switch head.Type {
		case "hello":
			handle(newest, &pilot, send)
		case "end":
			out.Flush()
			return
		case "obs":
			var o proto.Obs
			err := json.Unmarshal(newest, &o)
			if _, typeErr := err.(*json.UnmarshalTypeError); typeErr && o.Type == "obs" && len(o.Self.Weapons) > 0 {

				logf("odd field in obs: %v", err)
				err = nil
			}
			if err != nil {

				logf("bad obs: %v", err)
				var tk struct {
					Tick int `json:"tick"`
				}
				if json.Unmarshal(newest, &tk) == nil {
					fb := proto.Action{Tick: tk.Tick, Move: last.Move, Aim: last.Aim, Boost: last.Boost}
					send(encode(&fb))
				}
				continue
			}
			if busy {

				select {
				case <-pending:
					busy = false
				default:
				}
			}
			fallback := proto.Action{Tick: o.Tick, Move: last.Move, Aim: last.Aim, Boost: last.Boost}
			if pilot == nil || busy {
				send(encode(&fallback))
				continue
			}
			res := make(chan proto.Action, 1)
			jobs <- job{&o, res}
			select {
			case a := <-res:
				if a.Tick != o.Tick {
					a = fallback
				}
				last = a
				send(encode(&a))
			case <-time.After(budget):
				send(encode(&fallback))
				busy = true
				pending = res
				logf("slow decision at tick %d", o.Tick)
			}
		}
	}
	out.Flush()
}

func handle(ln []byte, pilot **bot.Bot, send func([]byte)) {
	var h proto.Hello
	if err := json.Unmarshal(ln, &h); err != nil || h.Type != "hello" {
		return
	}
	h.Raw = append([]byte(nil), ln...)
	t0 := time.Now()
	var w *geo.World
	var err error
	if h.Map.Dir != "" {
		w, err = geo.Load(h.Map.Dir)
	}
	if w == nil {
		if err != nil {
			logf("map load failed (%v), using embedded copy", err)
		}
		w, err = geo.FromBytes(mapdata.WorldJSON, mapdata.Terrain)
		if err != nil {
			logf("embedded map failed: %v", err)
		}
	}
	if w != nil {
		func() {
			defer func() {
				if r := recover(); r != nil {
					logf("init panic: %v", r)
				}
			}()
			m := h.Config.Match.ArenaMargin
			if m < 0 || m > 200 {
				m = 30
			}
			w.SetMargin(m)
			w.BuildNav(3.0, 9.5, 1.2)
			*pilot = bot.New(&h, w, bot.DefaultParams())
		}()
	}
	send([]byte(`{"type":"ready","name":"ProjectPaw"}` + "\n"))
	logf("ready in %v", time.Since(t0))
}
