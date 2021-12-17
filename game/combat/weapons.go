package combat

type Beam struct {
	Name      string
	MinDamage int64
	MaxDamage int64
}

var Beams = []Beam{
	{"Laser", 1, 4},
}

// NewBasicBeam creates a new beam with damage range 1-{damage}
func NewBasicBeam(damage int64) Beam {
	return Beam{
		"Basic",
		1,
		damage,
	}
}

func NewMultipleBasicBeams(n int, damage int64) []Beam {
	var beamArray = make([]Beam, n)
	for i := 0; i < n; i++ {
		beamArray[i] = NewBasicBeam(damage)
	}
	return beamArray
}
