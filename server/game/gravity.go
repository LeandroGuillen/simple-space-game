package game

import "ssg/model"

// gravityOutputMultiplier returns the multiplier applied to per-unit labor
// output (food per farmer, industry per worker, science per scientist) for a
// given planet and player based on planetary gravity and any race adaptation
// trait. Income (taxes/buildings) is not affected.
func gravityOutputMultiplier(planet *model.Planet, player *model.Player) float64 {
	if planet == nil {
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
