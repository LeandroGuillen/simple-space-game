package game

import (
	"math"

	. "ssg/model"
)

func hydratePlanetOutputs(game *Game, p *Planet) {
	foodPerFarmer := int64(2)
	foodFromBuildings := int64(0)
	industryPerWorker := IndustryPerMineralClass(p.MineralClass)
	industryFromBuildings := int64(0)
	industryPerColonist := int64(0)
	sciencePerScientist := int64(3)
	scienceFromBuildings := int64(0)

	for _, buildingName := range p.Buildings {
		if building, err := getBuildingFromName(game, buildingName); err == nil {
			for _, modifier := range building.Grants.Modifiers {
				switch modifier.Name {
				case "food_per_farmer":
					foodPerFarmer += int64(modifier.Value)
				case "food_flat":
					foodFromBuildings += int64(modifier.Value)
				case "industry_per_worker":
					industryPerWorker += int64(modifier.Value)
				case "industry_flat":
					industryFromBuildings += int64(modifier.Value)
				case "industry_per_colonist":
					industryPerColonist += int64(modifier.Value)
				case "science_per_scientist":
					sciencePerScientist += int64(modifier.Value)
				case "science_flat":
					scienceFromBuildings += int64(modifier.Value)
				}
			}
		}
	}
	if game != nil {
		if player, err := game.getPlayer(p.Owner); err == nil {
			for _, modifier := range player.Race.Grants.Modifiers {
				switch modifier.Name {
				case "food_per_farmer":
					foodPerFarmer += int64(modifier.Value)
				case "industry_per_worker":
					industryPerWorker += int64(modifier.Value)
				case "science_per_scientist":
					sciencePerScientist += int64(modifier.Value)
				}
			}
		}
	}
	sciencePerScientist += p.HomeworldScienceBonus
	p.GravityMultiplier = gravityOutputMultiplier(p, gamePlayerForPlanet(game, p))
	foodPerFarmer = maxInt64(0, foodPerFarmer)
	industryPerWorker = maxInt64(0, industryPerWorker)
	sciencePerScientist = maxInt64(0, sciencePerScientist)

	p.Food = PlanetFoodOutput{
		Farmers:             p.Food.Farmers,
		ProducedPerFarmer:   foodPerFarmer,
		ProducedPerBuilding: foodFromBuildings,
	}

	p.Industry = PlanetIndustryOutput{
		Workers:             p.Industry.Workers,
		ProducedPerWorker:   industryPerWorker,
		ProducedPerBuilding: industryFromBuildings,
		ProducedPerColonist: industryPerColonist,
	}

	p.Science = PlanetScienceOutput{
		Scientists:           p.Science.Scientists,
		ProducedPerScientist: sciencePerScientist,
		ProducedPerBuilding:  scienceFromBuildings,
	}

	hydratePlanetPollution(game, p)
}

func PlanetGrossIndustry(p *Planet) int64 {
	return moraleAdjustedLaborOutput(p.Industry.Workers, p.Industry.ProducedPerWorker, effectiveMoraleMultiplier(p.Morale)) +
		p.Industry.ProducedPerBuilding +
		colonistUnits(p)*p.Industry.ProducedPerColonist
}

func GamePlanetGrossIndustry(game *Game, p *Planet) int64 {
	player := gamePlayerForPlanet(game, p)
	return governmentAdjustedLaborOutput(
		p.Industry.Workers,
		p.Industry.ProducedPerWorker,
		governmentLaborMultiplier(game, player, "industry_labor_multiplier"),
		governmentMoraleMultiplier(game, player, p.Morale),
		gravityOutputMultiplier(p, player),
	) + p.Industry.ProducedPerBuilding + colonistUnits(p)*p.Industry.ProducedPerColonist
}

func PlanetIndustryProduced(game *Game, p *Planet) int64 {
	return PlanetGrossIndustry(p) - Pollution(game, p)
}

func GamePlanetIndustryProduced(game *Game, p *Planet) int64 {
	return GamePlanetGrossIndustry(game, p) - Pollution(game, p)
}

func PlanetFoodProduced(p *Planet) int64 {
	return moraleAdjustedLaborOutput(p.Food.Farmers, p.Food.ProducedPerFarmer, effectiveMoraleMultiplier(p.Morale)) +
		p.Food.ProducedPerBuilding
}

func GamePlanetFoodProduced(game *Game, p *Planet) int64 {
	player := gamePlayerForPlanet(game, p)
	return governmentAdjustedLaborOutput(
		p.Food.Farmers,
		p.Food.ProducedPerFarmer,
		governmentLaborMultiplier(game, player, "food_labor_multiplier"),
		governmentMoraleMultiplier(game, player, p.Morale),
		gravityOutputMultiplier(p, player),
	) + p.Food.ProducedPerBuilding
}

func PlanetScienceProduced(p *Planet) int64 {
	return moraleAdjustedLaborOutput(p.Science.Scientists, p.Science.ProducedPerScientist, effectiveMoraleMultiplier(p.Morale)) +
		p.Science.ProducedPerBuilding
}

func GamePlanetScienceProduced(game *Game, p *Planet) int64 {
	player := gamePlayerForPlanet(game, p)
	return governmentAdjustedLaborOutput(
		p.Science.Scientists,
		p.Science.ProducedPerScientist,
		governmentLaborMultiplier(game, player, "science_labor_multiplier"),
		governmentMoraleMultiplier(game, player, p.Morale),
		gravityOutputMultiplier(p, player),
	) + p.Science.ProducedPerBuilding
}

func gamePlayerForPlanet(game *Game, p *Planet) *Player {
	if game == nil || p == nil {
		return nil
	}
	player, err := game.getPlayer(p.Owner)
	if err != nil {
		return nil
	}
	return player
}

func governmentMoraleMultiplier(game *Game, player *Player, morale PlanetMoraleOutput) float64 {
	if governmentMoraleNeutral(game, player) {
		return 1
	}
	return effectiveMoraleMultiplier(morale)
}

func colonistUnits(p *Planet) int64 {
	if p.Population <= 0 {
		return 0
	}
	return p.Population / 1000
}

func moraleAdjustedLaborOutput(workers, outputPerWorker int64, multiplier float64) int64 {
	if multiplier == 0 {
		multiplier = 1
	}
	if multiplier < 0 {
		multiplier = 0
	}
	return int64(math.Floor(float64(workers*outputPerWorker) * multiplier))
}

func governmentAdjustedLaborOutput(workers, outputPerWorker int64, governmentMultiplier, moraleMultiplier, gravityMultiplier float64) int64 {
	if governmentMultiplier == 0 {
		governmentMultiplier = 1
	}
	if governmentMultiplier < 0 {
		governmentMultiplier = 0
	}
	if moraleMultiplier == 0 {
		moraleMultiplier = 1
	}
	if moraleMultiplier < 0 {
		moraleMultiplier = 0
	}
	if gravityMultiplier == 0 {
		gravityMultiplier = 1
	}
	if gravityMultiplier < 0 {
		gravityMultiplier = 0
	}
	return int64(math.Floor(float64(workers*outputPerWorker) * governmentMultiplier * moraleMultiplier * gravityMultiplier))
}
