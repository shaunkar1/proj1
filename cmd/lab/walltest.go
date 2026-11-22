//go:build ignore

package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"mechbot/bot"
	"mechbot/geo"
	"mechbot/proto"
)

func main() {
	f, _ := os.Open(os.Args[1])
	g, _ := gzip.NewReader(f)
	sc := bufio.NewScanner(g)
	sc.Buffer(make([]byte, 1<<20), 1<<26)
	sc.Scan()
	var h proto.Hello
	json.Unmarshal(sc.Bytes(), &h)
	h.Raw = append([]byte(nil), sc.Bytes()...)
	w, _ := geo.Load("mapdata/arena")
	w.SetMargin(h.Config.Match.ArenaMargin)
	w.BuildNav(3.0, 9.5, 1.2)
	b := bot.New(&h, w, bot.Variant(os.Args[2]))
	ms, _ := time.ParseDuration(os.Args[3])
	for sc.Scan() {
		var o proto.Obs
		if json.Unmarshal(sc.Bytes(), &o) != nil || o.Type != "obs" {
			continue
		}
		time.Sleep(ms)
		if o.T == 0 {
			gap, _ := time.ParseDuration(os.Args[4])
			time.Sleep(gap)
		}
		a := b.Act(&o)
		if o.T > 1.9 {
			break
		}
		fmt.Printf("%.1f:%.2f ", o.T, a.Aim[1])
		if o.T > 1.75 && o.T < 1.85 {
			fmt.Printf("[ffaSure %v] ", b.FFASureX())
		}
	}
	fmt.Println()
}
