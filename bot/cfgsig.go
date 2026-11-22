package bot

import (
	"encoding/json"
	"hash/fnv"
	"sort"
)

var cfgSections = []string{"protocol", "mapfiles", "match", "bot", "mech", "vision", "sound", "weapons", "projectile", "cubes", "crates", "loot", "npc", "drops", "zone", "debug", "replay", "upgrades"}

func CfgHashes(raw []byte) map[string]uint32 {
	var h struct {
		Protocol json.RawMessage `json:"protocol"`
		Map      struct {
			Files json.RawMessage `json:"files"`
		} `json:"map"`
		Config map[string]json.RawMessage `json:"config"`
	}
	out := map[string]uint32{}
	if json.Unmarshal(raw, &h) != nil {
		return out
	}
	sum := func(r json.RawMessage) uint32 {
		var v interface{}
		if len(r) == 0 || json.Unmarshal(r, &v) != nil {
			return 0
		}
		c, _ := json.Marshal(v)
		f := fnv.New32a()
		f.Write(c)
		return f.Sum32()
	}
	out["protocol"] = sum(h.Protocol)
	out["mapfiles"] = sum(h.Map.Files)
	for k, v := range h.Config {
		out[k] = sum(v)
	}
	return out
}

func cfgDiff(raw []byte) (first, n int) {
	if len(raw) == 0 {
		return 0, 0
	}
	got := CfgHashes(raw)
	for i, k := range cfgSections {
		if got[k] != kitCfg[k] {
			n++
			if first == 0 {
				first = i + 1
			}
		}
	}
	var extra []string
	for k := range got {
		if _, ok := kitCfg[k]; !ok {
			extra = append(extra, k)
		}
	}
	sort.Strings(extra)
	if len(extra) > 0 {
		n += len(extra)
		if first == 0 {
			first = len(cfgSections) + 1
		}
	}
	return first, n
}

func CfgDiffX(raw []byte) (int, int) { return cfgDiff(raw) }

func (b *Bot) FFASureX() bool { return b.ffaSure }
