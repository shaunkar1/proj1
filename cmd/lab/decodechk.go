//go:build ignore

package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"

	"mechbot/proto"
)

func main() {
	f, _ := os.Open("/home/claude/kit/mechfight-kit/check/stream.jsonl.gz")
	z, _ := gzip.NewReader(f)
	sc := bufio.NewScanner(z)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	n, bad := 0, 0
	evs := map[string]int{}
	for sc.Scan() {
		n++
		var o proto.Obs
		if err := json.Unmarshal(sc.Bytes(), &o); err != nil {
			bad++
			if bad < 6 {
				fmt.Println("line", n, "err:", err)
			}
		}
		var raw struct {
			Events []map[string]interface{} `json:"events"`
		}
		json.Unmarshal(sc.Bytes(), &raw)
		for _, e := range raw.Events {
			t, _ := e["type"].(string)
			evs[t]++
			if evs[t] <= 2 {
				b, _ := json.Marshal(e)
				fmt.Println("sample", string(b))
			}
		}
	}
	fmt.Println("lines", n, "bad", bad, evs)
}
