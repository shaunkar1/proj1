//go:build ignore

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"mechbot/bot"
	"mechbot/geo"
	"mechbot/proto"
)

func main() {
	f, _ := os.Open(os.Args[1])
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	sc.Scan()
	var h proto.Hello
	json.Unmarshal(sc.Bytes(), &h)
	w, _ := geo.Load("mapdata/arena")
	w.SetMargin(30)
	w.BuildNav(3.0, 9.5, 1.2)
	b := bot.New(&h, w, bot.DefaultParams())
	type rec struct {
		d    time.Duration
		tick int
		info string
	}
	var rs []rec
	for sc.Scan() {
		var o proto.Obs
		if json.Unmarshal(sc.Bytes(), &o) != nil || o.Type != "obs" {
			continue
		}
		t0 := time.Now()
		b.Act(&o)
		rs = append(rs, rec{time.Since(t0), o.Tick, b.Info()})
	}
	sort.Slice(rs, func(i, j int) bool { return rs[i].d > rs[j].d })
	for i := 0; i < 8 && i < len(rs); i++ {
		fmt.Printf("%.2f ms tick %d %s\n", float64(rs[i].d.Microseconds())/1000, rs[i].tick, rs[i].info)
	}
}
