//go:build ignore

package main

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"os"
	"strings"

	"mechbot/bot"
)

func main() {
	f, _ := os.Open(os.Args[1])
	g, _ := gzip.NewReader(f)
	sc := bufio.NewScanner(g)
	sc.Buffer(make([]byte, 1<<20), 1<<26)
	sc.Scan()
	raw := sc.Text()
	fmt.Println(bot.CfgDiffX([]byte(raw)))
	fmt.Println(bot.CfgDiffX([]byte(strings.Replace(raw, `"tick_budget_ms":50`, `"tick_budget_ms": 60`, 1))))
	fmt.Println(bot.CfgDiffX([]byte(strings.Replace(raw, `"frame_every":2`, `"frame_every":1`, 1))))
	fmt.Println(bot.CfgDiffX([]byte(strings.Replace(raw, `"../arena"`, `"/arena"`, 1))))
	fmt.Println(bot.CfgDiffX([]byte(strings.Replace(raw, `"upgrades":`, `"extra":{"a":1},"upgrades":`, 1))))
}
