package game

import (
	"slices"

	"ssg/model"
)

const governmentTechType = "government"

func governmentTechs(resources *Resources) []model.Tech {
	if resources == nil || resources.TechTree == nil {
		return nil
	}
	governments := make([]model.Tech, 0)
	for _, tech := range resources.TechTree.AllTechs {
		if isGovernmentTech(tech) {
			governments = append(governments, tech)
		}
	}
	return governments
}

func baseGovernmentTechs(resources *Resources) []model.Tech {
	governments := governmentTechs(resources)
	base := make([]model.Tech, 0, len(governments))
	for _, tech := range governments {
		if isBaseGovernment(tech) {
			base = append(base, tech)
		}
	}
	return base
}

func defaultGovernmentName(resources *Resources) string {
	for _, tech := range baseGovernmentTechs(resources) {
		if tech.Default {
			return tech.Name
		}
	}
	return "Dictatorship"
}

func findGovernmentTech(resources *Resources, name string) (model.Tech, bool) {
	for _, tech := range governmentTechs(resources) {
		if tech.Name == name {
			return tech, true
		}
	}
	return model.Tech{}, false
}

func activeGovernmentTech(g *Game, player *model.Player) (model.Tech, bool) {
	if g == nil || player == nil {
		return model.Tech{}, false
	}
	name := player.Race.Government
	if name == "" {
		name = defaultGovernmentName(g.Resources)
	}
	active, ok := findGovernmentTech(g.Resources, name)
	if !ok {
		return model.Tech{}, false
	}

	seen := map[string]bool{active.Name: true}
	for {
		next, found := researchedGovernmentUpgrade(g.Resources, player, active)
		if !found {
			return active, true
		}
		if seen[next.Name] {
			return active, true
		}
		seen[next.Name] = true
		active = next
	}
}

func researchedGovernmentUpgrade(resources *Resources, player *model.Player, active model.Tech) (model.Tech, bool) {
	for _, techName := range player.ResearchedTechs {
		candidate, ok := findGovernmentTech(resources, techName)
		if !ok || isBaseGovernment(candidate) {
			continue
		}
		if isGovernmentUpgradeAllowed(player, candidate, active) {
			return candidate, true
		}
	}
	return model.Tech{}, false
}

func isGovernmentTech(tech model.Tech) bool {
	return tech.Type == governmentTechType
}

func isBaseGovernment(tech model.Tech) bool {
	return isGovernmentTech(tech) && len(tech.Upgrades) == 0
}

func isGovernmentUpgradeAllowed(_ *model.Player, candidate model.Tech, active model.Tech) bool {
	return isGovernmentTech(candidate) && slices.Contains(candidate.Upgrades, active.Name)
}

func governmentModifierValue(g *Game, player *model.Player, modifierName string, fallback float64) float64 {
	tech, ok := activeGovernmentTech(g, player)
	if !ok {
		return fallback
	}
	for _, modifier := range tech.Grants.Modifiers {
		if modifier.Name == modifierName {
			return modifier.Value
		}
	}
	return fallback
}

func governmentHasModifier(g *Game, player *model.Player, modifierName string) bool {
	tech, ok := activeGovernmentTech(g, player)
	if !ok {
		return false
	}
	for _, modifier := range tech.Grants.Modifiers {
		if modifier.Name == modifierName {
			return true
		}
	}
	return false
}

func researchedTechModifierValue(g *Game, player *model.Player, techName string, modifierName string, fallback float64) float64 {
	if g == nil || player == nil || !isTechResearched(player, techName) {
		return fallback
	}
	tech, ok := g.findTechDefinition(techName)
	if !ok {
		return fallback
	}
	for _, modifier := range tech.Grants.Modifiers {
		if modifier.Name == modifierName {
			return modifier.Value
		}
	}
	return fallback
}

func psionicsPolicyActive(g *Game, player *model.Player) bool {
	if g == nil || player == nil || !isTechResearched(player, "Psionics") {
		return false
	}
	tech, ok := activeGovernmentTech(g, player)
	if !ok {
		return false
	}
	switch tech.Name {
	case "Feudal", "Dictatorship", "Confederation", "Imperium":
		return true
	default:
		return false
	}
}

func psionicsPolicyModifierValue(g *Game, player *model.Player, modifierName string, fallback float64) float64 {
	if !psionicsPolicyActive(g, player) {
		return fallback
	}
	return researchedTechModifierValue(g, player, "Psionics", modifierName, fallback)
}

func DefensiveSpyBonus(g *Game, player *model.Player) float64 {
	bonus := governmentModifierValue(g, player, "defensive_spy_bonus", 0)
	bonus += psionicsPolicyModifierValue(g, player, "defensive_spy_bonus", 0)
	return bonus
}

func governmentMoraleNeutral(g *Game, player *model.Player) bool {
	return governmentHasModifier(g, player, "morale_neutral")
}

func governmentLaborMultiplier(g *Game, player *model.Player, modifierName string) float64 {
	return governmentModifierValue(g, player, modifierName, 1)
}

func governmentTaxMultiplier(g *Game, player *model.Player) float64 {
	return governmentModifierValue(g, player, "tax_multiplier", 1)
}

func governmentShipCostMultiplier(g *Game, player *model.Player) float64 {
	return governmentModifierValue(g, player, "ship_cost_multiplier", 1)
}

func governmentCommandPointsMultiplier(g *Game, player *model.Player) float64 {
	return governmentModifierValue(g, player, "command_points_multiplier", 1)
}

func hydratePlanetGovernment(g *Game, p *model.Planet, player *model.Player) {
	if p == nil {
		return
	}
	tech, _ := activeGovernmentTech(g, player)
	p.Government = model.PlanetGovernmentOutput{
		Name:                    tech.Name,
		FoodLaborMultiplier:     governmentLaborMultiplier(g, player, "food_labor_multiplier"),
		IndustryLaborMultiplier: governmentLaborMultiplier(g, player, "industry_labor_multiplier"),
		ScienceLaborMultiplier:  governmentLaborMultiplier(g, player, "science_labor_multiplier"),
		TaxMultiplier:           governmentTaxMultiplier(g, player),
		MoraleNeutral:           governmentMoraleNeutral(g, player),
	}
}

func buildingHasMoraleModifier(building model.Building) bool {
	for _, modifier := range building.Grants.Modifiers {
		if modifier.Name == "morale_bonus" {
			return true
		}
	}
	return false
}

func techUnlocksMoraleBuilding(resources *Resources, tech model.Tech) bool {
	if resources == nil {
		return false
	}
	for _, buildingName := range tech.Grants.Unlocks.Buildings {
		for _, building := range resources.Buildings {
			if building.Name == buildingName && buildingHasMoraleModifier(building) {
				return true
			}
		}
	}
	return false
}
