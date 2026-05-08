package game

import (
	. "ssg/model"
)

func hydratePlanetOutputs(game *Game, p *Planet) {
	foodPerFarmer := int64(2)
	foodFromBuildings := int64(0)
	industryPerWorker := IndustryPerMineralClass(p.MineralClass)
	industryFromBuildings := int64(0)
	sciencePerScientist := int64(3)
	scienceFromBuildings := int64(0)

	for _, buildingName := range p.Buildings {
		if building, err := getBuildingFromName(game, buildingName); err == nil {
			for _, effect := range building.Effects {
				switch effect.Name {
				case "foodBonus":
					foodPerFarmer += effect.Bonus
				case "flatFoodBonus":
					foodFromBuildings += effect.Bonus
				case "industryBonus":
					industryPerWorker += effect.Bonus
				case "flatIndustryBonus":
					industryFromBuildings += effect.Bonus
				case "scienceBonus":
					sciencePerScientist += effect.Bonus
				case "flatScienceBonus":
					scienceFromBuildings += effect.Bonus
				}
			}
		}
	}

	p.Food = PlanetFoodOutput{
		Farmers:             p.Food.Farmers,
		ProducedPerFarmer:   foodPerFarmer,
		ProducedPerBuilding: foodFromBuildings,
	}

	p.Industry = PlanetIndustryOutput{
		Workers:             p.Industry.Workers,
		ProducedPerWorker:   industryPerWorker,
		ProducedPerBuilding: industryFromBuildings,
	}

	p.Science = PlanetScienceOutput{
		Scientists:           p.Science.Scientists,
		ProducedPerScientist: sciencePerScientist,
		ProducedPerBuilding:  scienceFromBuildings,
	}

	hydratePlanetPollution(p)
}
