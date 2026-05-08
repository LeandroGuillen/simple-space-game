package game

import (
	"ssg/model"
)

func hydratePlanetIncome(game *Game, p *model.Planet, industryProduced int64) {
	p.Income = PlanetIncome(game, p, industryProduced)
}

func PlanetIncome(game *Game, planet *model.Planet, industryProduced int64) model.PlanetIncomeOutput {
	taxes := planet.Population / 1000

	var incomeFromBuildings int64
	var incomeBonus float64
	for _, buildingName := range planet.Buildings {
		if building, err := getBuildingFromName(game, buildingName); err == nil {
			buildingIncome, buildingBonus := incomeModifiers(building.Grants.Modifiers)
			incomeFromBuildings += buildingIncome
			incomeBonus += buildingBonus
		}
	}

	if game.Resources != nil && game.Resources.TechTree != nil {
		player, err := game.getPlayer(planet.Owner)
		if err != nil {
			player = nil
		}
		if player != nil {
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
		food := planet.Food.Farmers*planet.Food.ProducedPerFarmer + planet.Food.ProducedPerBuilding
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
	totalIncome := int64(float64(grossIncome)*(1+incomeBonus)) - maintenance

	return model.PlanetIncomeOutput{
		Taxes:       taxes,
		Buildings:   incomeFromBuildings,
		TradeGoods:  tradeGoods,
		Maintenance: maintenance,
		Total:       totalIncome,
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
