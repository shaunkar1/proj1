// Package mapdata embeds a copy of the arena so the bot still has a map if
// the mounted one cannot be read.
package mapdata

import _ "embed"

//go:embed arena/v2/world.json
var WorldJSON []byte

//go:embed arena/v2/terrain.i16
var Terrain []byte
