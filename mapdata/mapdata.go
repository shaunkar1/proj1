package mapdata

import _ "embed"

//go:embed arena/v2/world.json
var WorldJSON []byte

//go:embed arena/v2/terrain.i16
var Terrain []byte
