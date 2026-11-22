//go:build ignore

package main

import (
	"fmt"

	"mechbot/bot"
)

func main() {
	mix := "mix:2:01|AAA+x3|BBB+x6"
	c := map[string]int{}
	for s := int64(1000000); s < 1000400; s++ {
		c[bot.MixPickX(mix, s*7919+13, int(s%16))]++
	}
	fmt.Println(c)
	c = map[string]int{}
	mix = "mix:8:01230123|P0|P1|P2|P3"
	for s := int64(5); s < 4005; s++ {
		c[bot.MixPickX(mix, s*104729, int(s%16))]++
	}
	fmt.Println(c)
	p := bot.Variant("prof:mix:2:01|f75a220+g+s+d100+x3+h35|f75a220+g+s+d100+x6")
	fmt.Println(p.Mix != "", p.RngSalt, p.PracAggroHP)
}
