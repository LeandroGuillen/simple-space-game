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
				case "science_per_scientist":
					sciencePerScientist += int64(modifier.Value)
				case "science_flat":
					scienceFromBuildings += int64(modifier.Value)
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
