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
	for _, buildingName := range planet.Buildings {
		if building, err := getBuildingFromName(game, buildingName); err == nil {
			for _, modifier := range building.Grants.Modifiers {
				if modifier.Name == "income_flat" {
					incomeFromBuildings += int64(modifier.Value)
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

	return model.PlanetIncomeOutput{
		Taxes:       taxes,
		Buildings:   incomeFromBuildings,
		TradeGoods:  tradeGoods,
		Maintenance: maintenance,
		Total:       taxes + incomeFromBuildings + tradeGoods - maintenance,
	}
}

func maxInt64(left, right int64) int64 {
	if left > right {
		return left
	}
	return right
}
