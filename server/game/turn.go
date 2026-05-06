package game

import (
	"math"
	. "ssg/model"
)

func planetTurn(p *Planet, commands Commands) {
	applyMovePopulationCommands(p, commands.MovePopulation)
	hydratePlanetOutputs(p)

	// population growth calculations
	oldPopulation := p.Population
	growth := PopulationGrowth(p)
	p.Population += growth
	p.LastTurnPopGrowth = growth

	// minimum pop units is 1
	if p.Population < 1000 {
		p.Population = 1000
	}

	// capping population to planet max
	if maxPop := MaxPop(p); p.Population > maxPop {
		p.Population = maxPop
	}

	// should we increase/decrease pop units?
	popIncrease := int64(math.Floor(float64(p.Population/1000)) - math.Floor(float64(oldPopulation/1000)))
	if popIncrease != 0 {
		// whoever has more units, add one more. This reinforces colonies that are
		// focused on one type of production
		if p.Industry.Workers >= p.Food.Farmers && p.Industry.Workers >= p.Science.Scientists {
			p.Industry.Workers += popIncrease
		} else if p.Food.Farmers >= p.Industry.Workers && p.Food.Farmers >= p.Science.Scientists {
			p.Food.Farmers += popIncrease
		} else {
			p.Science.Scientists += popIncrease
		}
	}

	industryProduced := Industry(p)

	// Is this a new project or is it ongoing?
	if p.IsBuilding {
		nextBucket := p.BuildBucket - industryProduced
		if nextBucket <= 0 {
			// project is complete
			p.Buildings = append(p.Buildings, p.CurrentProject)
			p.IsBuilding = false
			p.IndustryStack += p.BuildBucket
			p.BuildBucket = 0
			p.CurrentProject = Building{}
		} else {
			// project is not complete
			p.BuildBucket = nextBucket
		}
	} else {
		// if nothing to work on, save production
		p.IndustryStack += industryProduced
	}
}

func applyMovePopulationCommands(planet *Planet, commands []MovePopulationCommand) {
	for _, cmd := range commands {
		switch cmd.From {
		case PopCategoryFarmers:
			planet.Food.Farmers -= cmd.Count
		case PopCategoryWorkers:
			planet.Industry.Workers -= cmd.Count
		case PopCategoryScientists:
			planet.Science.Scientists -= cmd.Count
		}

		switch cmd.To {
		case PopCategoryFarmers:
			planet.Food.Farmers += cmd.Count
		case PopCategoryWorkers:
			planet.Industry.Workers += cmd.Count
		case PopCategoryScientists:
			planet.Science.Scientists += cmd.Count
		}
	}
}
