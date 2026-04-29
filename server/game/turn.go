package game

import . "ssg/model"

func planetTurn(p *Planet) {
	if p.Owner != 0 {
		p.Population += PopulationGrowth(p)
		if p.Population < 0 {
			p.Population = 0
		}
		if maxPop := MaxPop(p); p.Population > maxPop {
			p.Population = maxPop
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
}
