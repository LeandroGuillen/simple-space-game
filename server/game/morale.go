package game

import (
	"ssg/model"
)

func hydratePlanetMorale(game *Game, p *model.Planet, player *model.Player) {
	sources := PlanetMoraleSources(game, p, player)
	bonus := moraleSourcesBonus(sources)
	p.Morale = model.PlanetMoraleOutput{
		Bonus:      bonus,
		Multiplier: moraleMultiplier(bonus),
		Sources:    sources,
	}
}

func PlanetMoraleBonus(game *Game, p *model.Planet, player *model.Player) float64 {
	return moraleSourcesBonus(PlanetMoraleSources(game, p, player))
}

func PlanetMoraleSources(game *Game, p *model.Planet, player *model.Player) []model.PlanetMoraleSource {
	sources := []model.PlanetMoraleSource{}
	if governmentMoraleNeutral(game, player) {
		return sources
	}
	var bonus float64
	for _, buildingName := range p.Buildings {
		if building, err := getBuildingFromName(game, buildingName); err == nil {
			bonus = moraleModifiers(building.Grants.Modifiers)
			if bonus != 0 {
				sources = append(sources, model.PlanetMoraleSource{
					Name:  building.Name,
					Type:  "building",
					Bonus: bonus,
				})
			}
		}
	}

	if game.Resources != nil && game.Resources.TechTree != nil && player != nil {
		for _, techName := range player.ResearchedTechs {
			if tech, ok := game.findTechDefinition(techName); ok {
				bonus = moraleModifiers(tech.Grants.Modifiers)
				if bonus != 0 {
					sources = append(sources, model.PlanetMoraleSource{
						Name:  tech.Name,
						Type:  "tech",
						Bonus: bonus,
					})
				}
			}
		}
	}
	if player != nil {
		if bonus = psionicsPolicyModifierValue(game, player, "psionics_morale_bonus", 0); bonus != 0 {
			sources = append(sources, model.PlanetMoraleSource{
				Name:  "Psionics",
				Type:  "policy",
				Bonus: bonus,
			})
		}
		if bonus = governmentModifierValue(game, player, "missing_barracks_morale", 0); bonus != 0 && !planetHasBarracks(p) {
			sources = append(sources, model.PlanetMoraleSource{
				Name:  activeGovernmentName(game, player) + ": no Barracks",
				Type:  "government",
				Bonus: bonus,
			})
		}
		if bonus = governmentModifierValue(game, player, "barracks_morale_bonus", 0); bonus != 0 && planetHasBarracks(p) {
			sources = append(sources, model.PlanetMoraleSource{
				Name:  activeGovernmentName(game, player) + ": Barracks",
				Type:  "government",
				Bonus: bonus,
			})
		}
		if bonus = governmentModifierValue(game, player, "missing_capitol_morale", 0); bonus != 0 && !playerHasCapitol(game, player.ID) {
			sources = append(sources, model.PlanetMoraleSource{
				Name:  activeGovernmentName(game, player) + ": no Capitol",
				Type:  "government",
				Bonus: bonus,
			})
		}
	}

	return sources
}

func activeGovernmentName(game *Game, player *model.Player) string {
	if tech, ok := activeGovernmentTech(game, player); ok {
		return tech.Name
	}
	return "Government"
}

func planetHasBarracks(p *model.Planet) bool {
	return hasBuilding(p, "Marine Barracks") || hasBuilding(p, "Armor Barracks")
}

func playerHasCapitol(game *Game, playerID uint) bool {
	if game == nil {
		return false
	}
	for _, star := range game.Galaxy.Stars {
		for _, planet := range star.Planets {
			if planet.Owner == playerID && hasBuilding(planet, "Capitol") {
				return true
			}
		}
	}
	return false
}

func moraleSourcesBonus(sources []model.PlanetMoraleSource) float64 {
	var bonus float64
	for _, source := range sources {
		bonus += source.Bonus
	}
	return bonus
}

func moraleMultiplier(bonus float64) float64 {
	multiplier := 1 + bonus
	if multiplier < 0 {
		return 0
	}
	return multiplier
}

func effectiveMoraleMultiplier(output model.PlanetMoraleOutput) float64 {
	if output.Multiplier == 0 && output.Bonus == 0 {
		return 1
	}
	if output.Multiplier < 0 {
		return 0
	}
	return output.Multiplier
}

func moraleModifiers(modifiers []model.Modifier) float64 {
	var bonus float64
	for _, modifier := range modifiers {
		if modifier.Name == "morale_bonus" {
			bonus += modifier.Value
		}
	}
	return bonus
}
