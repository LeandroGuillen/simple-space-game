package game

import (
	"ssg/model"
)

func hydratePlanetOutputs(p *model.Planet) {
	p.Food = model.PlanetFoodOutput{
		Farmers:             p.Food.Farmers,
		ProducedPerFarmer:   2,
		ProducedPerBuilding: 0,
	}

	p.Industry = model.PlanetIndustryOutput{
		Workers:             p.Industry.Workers,
		ProducedPerWorker:   model.IndustryPerMineralClass(p.MineralClass),
		ProducedPerBuilding: 0,
	}

	p.Science = model.PlanetScienceOutput{
		Scientists:           p.Science.Scientists,
		ProducedPerScientist: 3,
		ProducedPerBuilding:  0,
	}
}
