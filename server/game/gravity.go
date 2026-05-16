package game

import "ssg/model"

const gravityGeneratorBuildingName = "Gravity Generator"

// preferredGravity returns the gravity class a player's race feels at home on.
// Races without a gravity adaptation trait prefer Normal-G worlds.
func preferredGravity(player *model.Player) model.PlanetGravity {
	if playerHasRaceTrait(player, "low_g_world") {
		return model.PlanetGravityLow
	}
	if playerHasRaceTrait(player, "high_g_world") {
		return model.PlanetGravityHigh
	}
	return model.PlanetGravityNormal
}

func planetHasGravityGenerator(planet *model.Planet) bool {
	if planet == nil {
		return false
	}
	for _, name := range planet.Buildings {
		if name == gravityGeneratorBuildingName {
			return true
		}
	}
	return false
}

// gravityGeneratorApplicable reports whether a Gravity Generator would have any
// effect on the planet for the given player. The building is only relevant when
// the planet's gravity differs from the race's preferred gravity.
func gravityGeneratorApplicable(planet *model.Planet, player *model.Player) bool {
	if planet == nil {
		return false
	}
	return planet.Gravity != preferredGravity(player)
}

// gravityOutputMultiplier returns the multiplier applied to per-unit labor
// output (food per farmer, industry per worker, science per scientist) for a
// given planet and player based on planetary gravity and any race adaptation
// trait. Income (taxes/buildings) is not affected. A Gravity Generator on the
// planet neutralizes any gravity penalty.
func gravityOutputMultiplier(planet *model.Planet, player *model.Player) float64 {
	if planet == nil {
		return 1
	}
	if planetHasGravityGenerator(planet) {
		return 1
	}
	lowG := playerHasRaceTrait(player, "low_g_world")
	highG := playerHasRaceTrait(player, "high_g_world")
	switch planet.Gravity {
	case model.PlanetGravityLow:
		if lowG {
			return 1
		}
		return 0.75
	case model.PlanetGravityHigh:
		if highG {
			return 1
		}
		return 0.5
	default:
		// Normal or unset
		if lowG {
			return 0.75
		}
		return 1
	}
}
