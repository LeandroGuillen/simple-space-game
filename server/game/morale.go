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

	return sources
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
