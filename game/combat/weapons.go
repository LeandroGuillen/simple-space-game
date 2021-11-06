package combat

type Beam struct {
	Name      string
	MinDamage int64
	MaxDamage int64
}

var Beams = []Beam{
	{"Laser", 1, 4},
}
