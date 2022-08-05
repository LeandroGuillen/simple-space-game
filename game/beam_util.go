package game

import (
	"math"
	. "ssg/game/model"
)

func GetDamage(b Beam, distance int64) int64 {
	// base damage
	totalDamage := b.MaxDamage
	// ---------------------------------
	// apply range dissipation for beams
	// ---------------------------------
	//	Ranges       Multiplier
	//	------       ----------
	//	0 to 2          1
	//	3 to 5          0.9
	//	6 to 8          0.8
	//	9 to 11         0.7
	//	12 to 14        0.6
	//	15 to 17        0.5
	//	18 to 20        0.4
	//	21 to 23        0.35
	dissipation := map[int64]float64{
		0: 1, 1: 1, 2: 1,
		3: 0.9, 4: 0.9, 5: 0.9,
		6: 0.8, 7: 0.8, 8: 0.8,
		9: 0.7, 10: 0.7, 11: 0.7,
		12: 0.6, 13: 0.6, 14: 0.6,
		15: 0.5, 16: 0.5, 17: 0.5,
		18: 0.4, 19: 0.4, 20: 0.4,
		21: 0.35, 22: 0.35, 23: 0.35,
	}
	actualDisipation, ok := dissipation[distance]
	if ok {
		rangedDamage := actualDisipation * float64(totalDamage)
		totalDamage = int64(math.Round(rangedDamage))
	} else if distance > 23 {
		// TODO this depends on weapon
		totalDamage = 0
	}

	return totalDamage
}

// NewBasicBeam creates a new beam with damage range 1-{damage}.
func NewBasicBeam(damage int64) Beam {
	return Beam{
		Name:      "Basic",
		MinDamage: 1,
		MaxDamage: damage,
	}
}

// NewMultipleBasicBeams creates `n` new beams with damage range 1-{damage}.
func NewMultipleBasicBeams(n int, damage int64) []Beam {
	var beamArray = make([]Beam, n)
	for i := 0; i < n; i++ {
		beamArray[i] = NewBasicBeam(damage)
	}
	return beamArray
}
