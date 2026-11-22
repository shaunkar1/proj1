package sim

type Recorder struct {
	Frames []Frame
}

type Frame struct {
	T     float64
	Zone  [3]float64
	Mechs [][5]float64
}

func (r *Recorder) frame(s *Sim) {
	f := Frame{T: s.T, Zone: [3]float64{s.zc[0], s.zc[1], s.zr}}
	for _, m := range s.Mechs {
		hp := m.HP
		if !m.Alive {
			hp = -1
		}
		f.Mechs = append(f.Mechs, [5]float64{m.Pos[0], m.Pos[1], m.Pos[2], hp, m.Aim[0]})
	}
	r.Frames = append(r.Frames, f)
}

func (s *Sim) ZoneNow() ([2]float64, float64) { return s.zc, s.zr }
