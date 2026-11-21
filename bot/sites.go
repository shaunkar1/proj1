package bot

import "math"

// deathSite is a place where a mech died: its cubes plus two are lying there.
type deathSite struct {
	pos   [2]float64
	t     float64
	val   float64
	exact bool
	far   bool
	org   [2]float64
	brg   float64
	ext   int
	bad   float64
	goT   float64
}

// heardDeath turns a death sound (bearing with a few degrees of error and a
// coarse distance band) into a place worth visiting.
func (b *Bot) heardDeath(brg float64, band string, at float64) {
	me := b.me
	px, py := me.Pos[0], me.Pos[1]
	var who *Track
	bestErr := 0.25
	for _, t := range b.tracks {
		if t.NPC || t.Ghost || at-t.Seen > 6 {
			continue
		}
		d := hyp(px, py, t.Pos[0], t.Pos[1])
		in := (band == "near" && d < 75) || (band == "mid" && d > 45 && d < 180) || (band == "far" && d > 140)
		if !in {
			continue
		}
		e := math.Abs(angDiff(math.Atan2(t.Pos[1]-py, t.Pos[0]-px), brg))
		if e < bestErr {
			who, bestErr = t, e
		}
	}
	s := &deathSite{t: at, brg: brg, org: [2]float64{px, py}}
	if who != nil {
		s.pos = [2]float64{who.Pos[0], who.Pos[1]}
		s.exact = true
		s.val = 120 + 40*math.Min(who.Cubes+1, 4)
	} else {
		d := b.P.SiteFar
		s.val = b.P.SiteFarVal
		s.far = true
		switch band {
		case "near":
			d, s.val, s.far = 35, 175, false
		case "mid":
			d, s.val, s.far = 105, 170, false
		}
		s.pos = [2]float64{px + math.Cos(brg)*d, py + math.Sin(brg)*d}
	}
	if !b.snapSite(s) {
		return
	}
	b.sites = append(b.sites, s)
	if len(b.sites) > 12 {
		b.sites = b.sites[len(b.sites)-12:]
	}
}

// snapSite moves a site onto walkable ground; false if there is none near it.
func (b *Bot) snapSite(s *deathSite) bool {
	nv := b.w.Nav
	if nv.WalkableMain(s.pos[0], s.pos[1]) {
		return true
	}
	c := nv.Nearest(s.pos[0], s.pos[1], 40)
	if c < 0 {
		return false
	}
	x, y := nv.Pos(c)
	s.pos = [2]float64{x, y}
	return true
}

// tidySites drops sites that are stale, already looked at, or whose cubes we can see
// (visible cubes are ordinary objectives).
func (b *Bot) tidySites() {
	if len(b.sites) == 0 {
		return
	}
	me := b.me
	px, py := me.Pos[0], me.Pos[1]
	keep := b.sites[:0]
	for _, s := range b.sites {
		if b.t-s.t > b.P.SiteTTL {
			continue
		}
		d := hyp(px, py, s.pos[0], s.pos[1])
		seenCube := false
		for _, c := range b.cubeL {
			if b.t-c.Seen < 1.5 && hyp(c.Pos[0], c.Pos[1], s.pos[0], s.pos[1]) < 70 {
				seenCube = true
				break
			}
		}
		if seenCube {
			continue
		}
		gz := b.w.Ground(s.pos[0], s.pos[1])
		if d < 14 || (d < 45 && b.w.Clear(b.eye[0], b.eye[1], b.eye[2], s.pos[0], s.pos[1], gz+1.0)) {
			// looked at it: nothing here
			if s.far && s.ext < 1 {
				// the band only says "beyond 160 m": try further along the same bearing once
				s.ext++
				dd := hyp(s.org[0], s.org[1], s.pos[0], s.pos[1]) + 130
				s.pos = [2]float64{s.org[0] + math.Cos(s.brg)*dd, s.org[1] + math.Sin(s.brg)*dd}
				s.val -= 20
				if b.snapSite(s) {
					keep = append(keep, s)
				}
			}
			continue
		}
		keep = append(keep, s)
	}
	for i := len(keep); i < len(b.sites); i++ {
		b.sites[i] = nil
	}
	b.sites = keep
}
