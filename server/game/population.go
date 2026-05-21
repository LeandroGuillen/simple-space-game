package game

import (
	"math"
	. "ssg/model"
)

func MaxPop(game *Game, p *Planet, player *Player) int64 {
	maxPop := baseMaxPop(p, player)
	if playerHasRaceTrait(player, "subterranean") {
		maxPop += baseMaxPop(p, player)
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

	maxPop = applyMaxPopPercentBonus(maxPop, maxPopEnvironmentBonusPercent(game, p, player))

	return maxPop
}

func hydratePlanetPopulationCapacity(game *Game, p *Planet, player *Player) {
	p.MaxPopulation = MaxPop(game, p, player)
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

// MedicineGrowthPercent returns the additive percentage population-growth bonus
// from researched empire-effect techs (e.g. Microbiotics, Universal Antidote).
// A tech that is upgraded by another researched tech is skipped so upgrade
// chains do not stack with their lower-tier predecessor.
func MedicineGrowthPercent(game *Game, player *Player) int64 {
	if game == nil || player == nil || game.Resources == nil || game.Resources.TechTree == nil {
		return 0
	}
	upgraded := researchedTechUpgradeTargets(game, player)
	var bonus int64
	for _, techName := range player.ResearchedTechs {
		if upgraded[techName] {
			continue
		}
		tech, ok := game.findTechDefinition(techName)
		if !ok {
			continue
		}
		for _, modifier := range tech.Grants.Modifiers {
			if modifier.Name == "population_growth_percent" {
				bonus += int64(modifier.Value)
			}
		}
	}
	return bonus
}

func researchedTechUpgradeTargets(game *Game, player *Player) map[string]bool {
	upgraded := map[string]bool{}
	if game == nil || player == nil {
		return upgraded
	}
	for _, techName := range player.ResearchedTechs {
		tech, ok := game.findTechDefinition(techName)
		if !ok {
			continue
		}
		for _, upgradedName := range tech.Upgrades {
			upgraded[upgradedName] = true
		}
	}
	return upgraded
}

// PopulationFoodDemand returns food units required per turn for the given population.
// Lithovore races need no food; cybernetic races need half the normal amount.
func PopulationFoodDemand(player *Player, population int64) float64 {
	popUnits := float64(population / 1000)
	if playerHasRaceTrait(player, "lithovore") {
		return 0
	}
	if playerHasRaceTrait(player, "cybernetic") {
		return popUnits * 0.5
	}
	return popUnits
}

// PopulationGrowth returns the per-turn population delta for a planet.
func PopulationGrowth(player *Player, population, foodPerBuilding, foodPerFarmer, numFarmers, maxPop, raceGrowthPercent, flatGrowthBonus int64) int64 {
	totalFoodProduced := foodPerBuilding + foodPerFarmer*numFarmers
	return PopulationGrowthFromFood(player, population, totalFoodProduced, maxPop, raceGrowthPercent, flatGrowthBonus)
}

func PopulationGrowthFromFood(player *Player, population, totalFoodProduced, maxPop, raceGrowthPercent, flatGrowthBonus int64) int64 {
	return PopulationGrowthFromFoodWithHousing(player, population, totalFoodProduced, maxPop, raceGrowthPercent, flatGrowthBonus, 0)
}

func PopulationGrowthFromFoodWithHousing(player *Player, population, totalFoodProduced, maxPop, raceGrowthPercent, flatGrowthBonus, housingGrowthBonusPercent int64) int64 {
	if population <= 0 {
		return 0
	}

	planet_capacity := float64(maxPop / 1000)
	pop_units := math.Floor(float64(population / 1000))
	free_space := planet_capacity - pop_units
	basic_increment := math.Floor(math.Sqrt(2000 * pop_units * free_space / planet_capacity))

	race_bonus := float64(raceGrowthPercent)
	medicine_bonus := float64(0) // TODO: sum of technology bonus and leader bonus
	housing_bonus := float64(housingGrowthBonusPercent)

	all_bonuses := race_bonus + medicine_bonus + housing_bonus
	population_growth := math.Floor(basic_increment * (100 + all_bonuses) / 100)

	// enough food? check for starvation
	food_lack := PopulationFoodDemand(player, population) - float64(totalFoodProduced)
	var food_lack_penalty float64
	if food_lack > 0 {
		food_lack_penalty = 50 * food_lack
	}

	population_increment := population_growth + float64(flatGrowthBonus) - food_lack_penalty

	return int64(population_increment)
}
