package game

import (
	"math"
	. "ssg/model"
)

// GetChanceToHit calculates the chance to hit.
// See section 2 in https://gamefaqs.gamespot.com/pc/197873-master-of-orion-ii-battle-at-antares/faqs/16743.
// `attackBonus` represents additional targeting helpers, e.g. Battle Scanner. For every unit of distance
// there is an attack penalty of one point.
func GetChanceToHit(c Computer, defenderDefense int64, attackBonus, distance int64) float64 {
	// Beam Attack
	BA := c.BeamAttackBonus + attackBonus
	// Beam Defense
	BD := defenderDefense + distance

	// Percent chance to hit = 100/(1 + 2^E)
	// where E = -(BA - BD)/16
	E := -float64(BA-BD) / 16
	chanceToHit := 1 / (1 + math.Pow(2, E))
	//fmt.Printf("--> BA:%d BD:%d (%.2f)\n", BA, BD, 100*chanceToHit)
	return chanceToHit
}
