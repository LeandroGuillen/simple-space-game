package game

import (
	"math"
	"ssg/model"
)

func hydratePlanetIncome(game *Game, p *model.Planet, industryProduced int64) {
	p.Income = PlanetIncome(game, p, industryProduced)
}

func PlanetIncome(game *Game, planet *model.Planet, industryProduced int64) model.PlanetIncomeOutput {
	var player *model.Player
	if game != nil {
		found, err := game.getPlayer(planet.Owner)
		if err == nil {
			player = found
		}
	}
	taxes := int64(math.Floor(float64(planet.Population/1000) *
		governmentMoraleMultiplier(game, player, planet.Morale) *
		governmentTaxMultiplier(game, player)))

	var incomeFromBuildings int64
	var incomeBonus float64
	for _, buildingName := range planet.Buildings {
		if building, err := getBuildingFromName(game, buildingName); err == nil {
			buildingIncome, buildingBonus := incomeModifiers(building.Grants.Modifiers)
			incomeFromBuildings += buildingIncome
			incomeBonus += buildingBonus
		}
	}

	if game != nil && game.Resources != nil && game.Resources.TechTree != nil {
		if player != nil {
			_, raceBonus := incomeModifiers(player.Race.Grants.Modifiers)
			incomeBonus += raceBonus
			for _, techName := range player.ResearchedTechs {
				if tech, ok := game.findTechDefinition(techName); ok {
					_, techBonus := incomeModifiers(tech.Grants.Modifiers)
					incomeBonus += techBonus
				}
			}
		}
	}

	var tradeGoods int64 = 0
	if !planet.IsBuilding {
		food := GamePlanetFoodProduced(game, planet)
		foodSurplus := food - planet.Population/1000
		tradeGoods = maxInt64(0, industryProduced)/2 + maxInt64(0, foodSurplus)/2
	}

	var maintenance int64
	for _, buildingName := range planet.Buildings {
		if building, err := getBuildingFromName(game, buildingName); err == nil {
			maintenance += building.Maintenance
		}
	}

	grossIncome := taxes + incomeFromBuildings + tradeGoods
	incomeMultiplier := 1 + incomeBonus
	if incomeMultiplier < 0 {
		incomeMultiplier = 0
	}
	totalIncome := int64(float64(grossIncome)*incomeMultiplier) - maintenance

	return model.PlanetIncomeOutput{
		Taxes:       taxes,
		Buildings:   incomeFromBuildings,
		TradeGoods:  tradeGoods,
		Maintenance: maintenance,
		Total:       totalIncome,
		Bonus:       incomeBonus,
	}
}

func maxInt64(left, right int64) int64 {
	if left > right {
		return left
	}
	return right
}

func incomeModifiers(modifiers []model.Modifier) (int64, float64) {
	var incomeFlat int64
	var incomeBonus float64
	for _, modifier := range modifiers {
		switch modifier.Name {
		case "income_flat":
			incomeFlat += int64(modifier.Value)
		case "income_bonus":
			incomeBonus += modifier.Value
		}
	}
	return incomeFlat, incomeBonus
}
