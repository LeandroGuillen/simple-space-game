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

	hydratePlanetPollution(p)
}

func PlanetGrossIndustry(p *Planet) int64 {
	return moraleAdjustedLaborOutput(p.Industry.Workers, p.Industry.ProducedPerWorker, effectiveMoraleMultiplier(p.Morale)) +
		p.Industry.ProducedPerBuilding +
		colonistUnits(p)*p.Industry.ProducedPerColonist
}

func PlanetIndustryProduced(p *Planet) int64 {
	return PlanetGrossIndustry(p) - Pollution(p)
}

func PlanetFoodProduced(p *Planet) int64 {
	return moraleAdjustedLaborOutput(p.Food.Farmers, p.Food.ProducedPerFarmer, effectiveMoraleMultiplier(p.Morale)) +
		p.Food.ProducedPerBuilding
}

func PlanetScienceProduced(p *Planet) int64 {
	return moraleAdjustedLaborOutput(p.Science.Scientists, p.Science.ProducedPerScientist, effectiveMoraleMultiplier(p.Morale)) +
		p.Science.ProducedPerBuilding
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
