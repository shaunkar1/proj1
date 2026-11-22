//go:build ignore

package main

import (
	"bufio"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"os"

	"mechbot/bot"
	"mechbot/geo"
	"mechbot/proto"
)

func main() {
	f, _ := os.Open(os.Args[1])
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<26)
	sc.Scan()
	var h proto.Hello
	json.Unmarshal(sc.Bytes(), &h)
	w, _ := geo.Load("mapdata/arena")
	w.SetMargin(h.Config.Match.ArenaMargin)
	w.BuildNav(3.0, 9.5, 1.2)
	b := bot.New(&h, w, bot.Variant(os.Args[2]))
	hs := sha1.New()
	n := 0
	first := -1
	var prev string
	for sc.Scan() {
		var o proto.Obs
		if json.Unmarshal(sc.Bytes(), &o) != nil {
			continue
		}
		a := b.Act(&o)
		js, _ := json.Marshal(a)
		hs.Write(js)
		n++
		if len(os.Args) > 3 {
			cur := fmt.Sprintf("%x", sha1.Sum(js))
			_ = prev
			fmt.Println(n, o.T, cur[:8], b.DbgMode())
		}
		_ = first
	}
	fmt.Printf("obs %d hash %x\n", n, hs.Sum(nil))
}
