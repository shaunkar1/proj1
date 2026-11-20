package bot

// Variant returns a named parameter set (for A/B runs in the simulator).
func Variant(name string) Params {
	p := DefaultParams()
	p.Name = name
	switch name {
	case "", "default", "paw":
		p.Name = "paw"
	case "passive":
		p.HuntPassive = false
	case "nofarm":
		p.FarmNPC = false
	case "nodrops":
		p.UseDrops = false
	case "bold":
		p.EngageRatio = 1.4
		p.RetreatHP = 0.25
	case "timid":
		p.EngageRatio = 0.7
		p.RetreatHP = 0.45
	}
	return p
}
