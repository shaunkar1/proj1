//go:build ignore

package main

import (
	"fmt"
	"time"

	"mechbot/geo"
	"mechbot/mapdata"
)

func main() {
	t0 := time.Now()
	w, err := geo.FromBytes(mapdata.WorldJSON, mapdata.Terrain)
	t1 := time.Now()
	w.SetMargin(30)
	w.BuildNav(3.0, 9.5, 1.2)
	t2 := time.Now()
	fmt.Println("load", t1.Sub(t0), "nav", t2.Sub(t1), err)
}
