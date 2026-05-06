package game

import "ssg/model"

func hydratePlanetIncome(p *model.Planet) {
	p.Income = PlanetIncome(p)
}

func PlanetIncome(p *model.Planet) model.PlanetIncomeOutput {
	taxes := TaxIncome(p)
	buildings := BuildingIncome(p)
	tradeGoods := TradeGoodsIncome(p)
	maintenance := MaintenanceCost(p)

	return model.PlanetIncomeOutput{
		Taxes:       taxes,
		Buildings:   buildings,
		TradeGoods:  tradeGoods,
		Maintenance: maintenance,
		Total:       taxes + buildings + tradeGoods - maintenance,
	}
}

func TaxIncome(p *model.Planet) int64 {
	return p.Population / 1000
}

func BuildingIncome(p *model.Planet) int64 {
	var income int64
	for _, building := range p.Buildings {
		for _, effect := range building.Effects {
			if effect.Name == "income" {
				income += effect.Bonus
			}
		}
	}
	return income
}

func TradeGoodsIncome(p *model.Planet) int64 {
	if p.IsBuilding {
		return 0
	}
	return TradeGoodsFromOutput(Industry(p), FoodSurplus(p))
}

func TradeGoodsFromOutput(industry, foodSurplus int64) int64 {
	if foodSurplus < 0 {
		foodSurplus = 0
	}
	return industry/2 + foodSurplus/2
}

func MaintenanceCost(p *model.Planet) int64 {
	var maintenance int64
	for _, building := range p.Buildings {
		maintenance += building.Maintenance
	}
	return maintenance
}

func Food(p *model.Planet) int64 {
	return p.Food.Farmers*p.Food.ProducedPerFarmer + p.Food.ProducedPerBuilding
}

func FoodSurplus(p *model.Planet) int64 {
	return Food(p) - p.Population/1000
}
