package game

import (
	"math"
	. "ssg/model"
)

func MaxPop(p *Planet) int64 {
	// TODO should take into account other factors
	// Planetary Env.:  Gaia Terran Arid Swamp Other
	// Tiny              5     4     3    2     1     <-- The table shows how
	// Small            10     8     6    4     3         the base max. pop. of
	// Medium           15    12     9    6     4         a planet is determined
	// Large            20    16    12    8     5     <-- by size and climate.
	// Huge             25    20    15   10     6
	return int64(p.Size*4) * 1000
}

// PopulationGrowth returns the per-turn population delta for a planet.
func PopulationGrowth(p *Planet) int64 {
	if p.Population <= 0 {
		return 0
	}

	planet_capacity := float64(MaxPop(p) / 1000)
	pop_units := math.Floor(float64(p.Population / 1000))
	free_space := planet_capacity - pop_units
	basic_increment := math.Floor(math.Sqrt(2000 * pop_units * free_space / planet_capacity))

	race_bonus := float64(0)     // TODO: can be -50%, 0%, +50%, +100%
	medicine_bonus := float64(0) // TODO: sum of technology bonus and leader bonus
	// housing_bonus = math.Floor( PP * 40 / float64(p.Population) )
	housing_bonus := float64(0) // TODO appears if 'housing' is picked in the build menu

	all_bonuses := race_bonus + medicine_bonus + housing_bonus
	population_growth := math.Floor(basic_increment * (100 + all_bonuses) / 100)

	// building bonuses
	cloning_bonus := float64(0) // TODO if present, +100

	// enough food? check for starvation
	total_food_produced := float64(p.Food.ProducedPerBuilding + (p.Food.ProducedPerFarmer * p.Food.Farmers))
	food_lack := pop_units - total_food_produced
	var food_lack_penalty float64
	if food_lack > 0 {
		food_lack_penalty = 50 * food_lack
	}

	population_increment := population_growth + cloning_bonus - food_lack_penalty

	return int64(population_increment)
}
