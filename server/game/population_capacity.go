package game

import (
	"math"

	. "ssg/model"
)

// MoO2 base max population (in population units) by planet size and environment column.
// Columns: Gaia, Terran, Arid, Swamp, Other.
var maxPopUnitsBySizeAndEnv = [5][5]int64{
	{5, 4, 3, 2, 1},     // Tiny
	{10, 8, 6, 4, 3},    // Small
	{15, 12, 9, 6, 4},   // Medium
	{20, 16, 12, 8, 5},  // Large
	{25, 20, 15, 10, 6}, // Huge
}

func isColonizableEnvironment(env PlanetEnvironment) bool {
	return env != PlanetEnvironmentInvalid &&
		env != PlanetEnvironmentGasGiant &&
		env != PlanetEnvironmentAsteroidBelt
}

func isHostileEnvironment(env PlanetEnvironment) bool {
	switch env {
	case PlanetEnvironmentBarren, PlanetEnvironmentToxic, PlanetEnvironmentRadiated:
		return true
	default:
		return false
	}
}

func planetEnvironmentPopColumn(env PlanetEnvironment, tolerant bool) int {
	if env == PlanetEnvironmentInvalid {
		// Legacy tests and uninitialized planets used size×4 (Terran column).
		return 1
	}
	if !isColonizableEnvironment(env) {
		return -1
	}
	switch env {
	case PlanetEnvironmentGaia:
		return 0
	case PlanetEnvironmentTerran:
		return 1
	case PlanetEnvironmentDesert, PlanetEnvironmentArid, PlanetEnvironmentTundra:
		return 2
	case PlanetEnvironmentSwamp, PlanetEnvironmentOcean:
		return 3
	case PlanetEnvironmentBarren, PlanetEnvironmentToxic, PlanetEnvironmentRadiated:
		if tolerant {
			return 1
		}
		return 4
	default:
		return 4
	}
}

// racePopulationCapacityMultiplier applies the Aquatic trait (2× on Ocean/Swamp, ½ on other habitable worlds).
func racePopulationCapacityMultiplier(player *Player, p *Planet) float64 {
	if p == nil || !isColonizableEnvironment(p.Environment) {
		return 1
	}
	if player == nil || !playerHasRaceTrait(player, "aquatic") {
		return 1
	}
	switch p.Environment {
	case PlanetEnvironmentOcean, PlanetEnvironmentSwamp:
		return 2
	default:
		return 0.5
	}
}

func baseMaxPop(p *Planet, player *Player) int64 {
	col := planetEnvironmentPopColumn(p.Environment, playerHasRaceTrait(player, "tolerant"))
	if col < 0 {
		return 0
	}
	sizeIdx := int(p.Size) - int(PlanetSizeTiny)
	if sizeIdx < 0 || sizeIdx >= len(maxPopUnitsBySizeAndEnv) {
		return 0
	}
	base := maxPopUnitsBySizeAndEnv[sizeIdx][col] * 1000
	return int64(math.Round(float64(base) * racePopulationCapacityMultiplier(player, p)))
}

func maxPopEnvironmentBonusPercent(game *Game, p *Planet, player *Player) float64 {
	if game == nil || p == nil || !isHostileEnvironment(p.Environment) {
		return 0
	}
	if game.Resources == nil || game.Resources.BuildingCollection == nil {
		return 0
	}
	tolerant := playerHasRaceTrait(player, "tolerant")
	var bonus float64
	for _, buildingName := range p.Buildings {
		building, err := getBuildingFromName(game, buildingName)
		if err != nil {
			continue
		}
		for _, modifier := range building.Grants.Modifiers {
			if modifier.Name != "environment_bonus" {
				continue
			}
			if tolerant && modifier.Value < 0 {
				continue
			}
			bonus += modifier.Value
		}
	}
	return bonus
}

func applyMaxPopPercentBonus(maxPop int64, percentBonus float64) int64 {
	if percentBonus == 0 {
		return maxPop
	}
	adjusted := float64(maxPop) * (1 + percentBonus)
	if adjusted < 0 {
		return 0
	}
	return int64(math.Round(adjusted))
}
