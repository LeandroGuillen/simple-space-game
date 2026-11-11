package game

import (
	"math"
	"ssg/model"
)

func hydratePlanetIncome(game *Game, p *model.Planet, industryProduced int64) {
	p.Income = PlanetIncome(game, p, industryProduced)
}

// industryAfterReplicators uses industry after pollution, before empire tax.
func industryAfterReplicators(game *Game, p *model.Planet) int64 {
	player := gamePlayerForPlanet(game, p)
	base := GamePlanetIndustryProduced(game, p)
	food := CalculateFoodWithReplicators(game, p, int64(math.Ceil(PopulationFoodDemand(player, p.Population))), base)
	return maxInt64(0, base-food.IndustryConsumed)
}

// splitIndustryTax follows the confirmed MoO2 1.50 assumptions: 1:1 BC,
// floor per colony, after pollution and food replicators. No income bonuses,
// morale effects or changes to pollution or population taxes.
func splitIndustryTax(net int64, player *model.Player) (buildIndustry, taxBC int64) {
	net = maxInt64(0, net)
	if player != nil {
		taxBC = int64(math.Floor(float64(net) * player.TaxRate))
	}
	return net - taxBC, taxBC
}

func PlanetIncome(game *Game, planet *model.Planet, industryProduced int64) model.PlanetIncomeOutput {
	var player *model.Player
	if game != nil {
		found, err := game.getPlayer(planet.Owner)
		if err == nil {
			player = found
		}
	}
	industryProduced, industryTax := splitIndustryTax(industryProduced, player)
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
	if !planet.IsBuilding && (planet.CurrentProject == "" || planet.CurrentProject == tradeGoodsProject) {
		food := GamePlanetFoodProduced(game, planet)
		foodSurplus := food - int64(math.Ceil(PopulationFoodDemand(player, planet.Population)))
		tradeGoods = int64(float64(maxInt64(0, industryProduced)/2+maxInt64(0, foodSurplus)/2) * tradeGoodsMultiplier(player))
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
	totalIncome := int64(float64(grossIncome)*incomeMultiplier) - maintenance + industryTax

	return model.PlanetIncomeOutput{
		Taxes:       taxes,
		IndustryTax: industryTax,
		Buildings:   incomeFromBuildings,
		TradeGoods:  tradeGoods,
		Maintenance: maintenance,
		Total:       totalIncome,
		Bonus:       incomeBonus,
	}
}

func tradeGoodsMultiplier(player *model.Player) float64 {
	if playerHasRaceTrait(player, "fantastic_traders") {
		return 1.5
	}
	return 1
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
