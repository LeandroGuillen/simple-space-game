package game

import (
	"ssg/model"
)

func hydratePlanetOutputs(p *model.Planet) {
	foodPerFarmer := int64(2)
	foodFromBuildings := int64(0)
	industryPerWorker := model.IndustryPerMineralClass(p.MineralClass)
	industryFromBuildings := int64(0)
	sciencePerScientist := int64(3)
	scienceFromBuildings := int64(0)

	for _, building := range p.Buildings {
		for _, effect := range building.Effects {
			switch effect.Name {
			case "flatAgricultureBonus":
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

	p.Food = model.PlanetFoodOutput{
		Farmers:             p.Food.Farmers,
		ProducedPerFarmer:   foodPerFarmer,
		ProducedPerBuilding: foodFromBuildings,
	}

	p.Industry = model.PlanetIndustryOutput{
		Workers:             p.Industry.Workers,
		ProducedPerWorker:   industryPerWorker,
		ProducedPerBuilding: industryFromBuildings,
	}

	p.Science = model.PlanetScienceOutput{
		Scientists:           p.Science.Scientists,
		ProducedPerScientist: sciencePerScientist,
		ProducedPerBuilding:  scienceFromBuildings,
	}
}
