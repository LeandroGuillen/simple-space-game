package game

import (
	"math"
	. "ssg/model"
)

func MaxPop(game *Game, p *Planet, player *Player) int64 {
	maxPop := baseMaxPop(p)
	if playerHasRaceTrait(player, "subterranean") {
		maxPop += baseMaxPop(p)
	}

	if game != nil && game.Resources != nil && game.Resources.BuildingCollection != nil {
		for _, buildingName := range p.Buildings {
			if building, err := getBuildingFromName(game, buildingName); err == nil {
				maxPop += maxPopulationModifiers(building.Grants.Modifiers)
			}
		}
	}

	if game != nil && game.Resources != nil && game.Resources.TechTree != nil && player != nil {
		for _, techName := range player.ResearchedTechs {
			if tech, ok := game.findTechDefinition(techName); ok {
				maxPop += maxPopulationModifiers(tech.Grants.Modifiers)
			}
		}
	}

	return maxPop
}

func hydratePlanetPopulationCapacity(game *Game, p *Planet, player *Player) {
	p.MaxPopulation = MaxPop(game, p, player)
}

func baseMaxPop(p *Planet) int64 {
	// TODO should take into account other factors
	// Planetary Env.:  Gaia Terran Arid Swamp Other
	// Tiny              5     4     3    2     1     <-- The table shows how
	// Small            10     8     6    4     3         the base max. pop. of
	// Medium           15    12     9    6     4         a planet is determined
	// Large            20    16    12    8     5     <-- by size and climate.
	// Huge             25    20    15   10     6
	return int64(p.Size*4) * 1000
}

func maxPopulationModifiers(modifiers []Modifier) int64 {
	var maxPop int64
	for _, modifier := range modifiers {
		if modifier.Name == "max_population" {
			maxPop += int64(modifier.Value)
		}
	}
	return maxPop
}

func PopulationGrowthBonus(game *Game, p *Planet) int64 {
	var growthBonus int64
	if game == nil || game.Resources == nil || game.Resources.BuildingCollection == nil {
		return growthBonus
	}

	for _, buildingName := range p.Buildings {
		if building, err := getBuildingFromName(game, buildingName); err == nil {
			growthBonus += populationGrowthModifiers(building.Grants.Modifiers)
		}
	}

	return growthBonus
}

func populationGrowthModifiers(modifiers []Modifier) int64 {
	var growthBonus int64
	for _, modifier := range modifiers {
		if modifier.Name == "population_growth" {
			growthBonus += int64(modifier.Value)
		}
	}
	return growthBonus
}

// PopulationGrowth returns the per-turn population delta for a planet.
func PopulationGrowth(population, foodPerBuilding, foodPerFarmer, numFarmers, maxPop, raceGrowthPercent, flatGrowthBonus int64) int64 {
	totalFoodProduced := foodPerBuilding + foodPerFarmer*numFarmers
	return PopulationGrowthFromFood(population, totalFoodProduced, maxPop, raceGrowthPercent, flatGrowthBonus)
}

func PopulationGrowthFromFood(population, totalFoodProduced, maxPop, raceGrowthPercent, flatGrowthBonus int64) int64 {
	if population <= 0 {
		return 0
	}

	planet_capacity := float64(maxPop / 1000)
	pop_units := math.Floor(float64(population / 1000))
	free_space := planet_capacity - pop_units
	basic_increment := math.Floor(math.Sqrt(2000 * pop_units * free_space / planet_capacity))

	race_bonus := float64(raceGrowthPercent)
	medicine_bonus := float64(0) // TODO: sum of technology bonus and leader bonus
	// housing_bonus = math.Floor( PP * 40 / float64(p.Population) )
	housing_bonus := float64(0) // TODO appears if 'housing' is picked in the build menu

	all_bonuses := race_bonus + medicine_bonus + housing_bonus
	population_growth := math.Floor(basic_increment * (100 + all_bonuses) / 100)

	// enough food? check for starvation
	food_lack := pop_units - float64(totalFoodProduced)
	var food_lack_penalty float64
	if food_lack > 0 {
		food_lack_penalty = 50 * food_lack
	}

	population_increment := population_growth + float64(flatGrowthBonus) - food_lack_penalty

	return int64(population_increment)
}
