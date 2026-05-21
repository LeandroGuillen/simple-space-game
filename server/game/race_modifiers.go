package game

import "ssg/model"

func raceModifierValue(player *model.Player, modifierName string, fallback float64) float64 {
	if player == nil {
		return fallback
	}
	for _, modifier := range player.Race.Grants.Modifiers {
		if modifier.Name == modifierName {
			return modifier.Value
		}
	}
	return fallback
}

// GroundCombatMultiplier returns the empire ground-combat strength multiplier from
// race modifiers (1.0 baseline). Apply to marine strength and invasion rolls when
// ground combat is implemented.
func GroundCombatMultiplier(player *model.Player) float64 {
	return 1 + raceModifierValue(player, "ground_combat_bonus", 0)
}

// OffensiveSpyBonus returns the offensive espionage effectiveness bonus from
// government and race modifiers (0 baseline). Combine with RaceSpyBonus for flat
// spy points when espionage is implemented.
func OffensiveSpyBonus(g *Game, player *model.Player) float64 {
	bonus := raceModifierValue(player, "offensive_spy_bonus", 0)
	if g != nil {
		bonus += governmentModifierValue(g, player, "offensive_spy_bonus", 0)
	}
	return bonus
}

// RaceSpyBonus returns flat spy points per turn granted by race design stats.
func RaceSpyBonus(player *model.Player) int64 {
	if player == nil {
		return 0
	}
	return player.Race.Stats.SpyBonus
}
