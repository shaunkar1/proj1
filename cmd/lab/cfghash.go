//go:build ignore

package main

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"os"
	"sort"

	"mechbot/bot"
)

func main() {
	f, _ := os.Open(os.Args[1])
	g, _ := gzip.NewReader(f)
	sc := bufio.NewScanner(g)
	sc.Buffer(make([]byte, 1<<20), 1<<26)
	sc.Scan()
	h := bot.CfgHashes(sc.Bytes())
	ks := make([]string, 0, len(h))
	for k := range h {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	fmt.Print("package bot\n\nvar kitCfg = map[string]uint32{")
	for _, k := range ks {
		fmt.Printf("%q: %d, ", k, h[k])
	}
	fmt.Println("}")
}
