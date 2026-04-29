package game

import (
	"math"
	. "ssg/model"
)

func planetTurn(p *Planet, commands Commands) error {
	if err := applyMovePopulationCommands(p, commands.MovePopulation); err != nil {
		return err
	}

	// population growth calculations
	oldPopulation := p.Population
	p.Population += PopulationGrowth(p)
	if p.Population < 0 {
		p.Population = 0
	}
	if maxPop := MaxPop(p); p.Population > maxPop {
		p.Population = maxPop
	}

	// should we increase pop units?
	if math.Floor(float64(p.Population/1000)) > math.Floor(float64(oldPopulation/1000)) {
		// whoever has more units, add one more. This reinforces colonies that are
		// focused on one type of production
		if p.Industry.Workers >= p.Food.Farmers && p.Industry.Workers >= p.Science.Scientists {
			p.Industry.Workers++
		} else if p.Food.Farmers >= p.Industry.Workers && p.Food.Farmers >= p.Science.Scientists {
			p.Food.Farmers++
		} else {
			p.Science.Scientists++
		}
	}

	industryProduced := p.Industry.Workers * IndustryPerMineralClass(p.MineralClass)

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

	return nil
}

func applyMovePopulationCommands(planet *Planet, commands []MovePopulationCommand) error {
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
	return nil
}
