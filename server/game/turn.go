package game

import (
	"fmt"
	"math"
	. "ssg/model"
)

func planetTurn(p *Planet, commands Commands, player *Player, game *Game) error {

	hydratePlanetOutputs(game, p)

	// population movement
	if commands.MovePopulation != nil {
		newTotal := commands.MovePopulation.Farmers + commands.MovePopulation.Workers + commands.MovePopulation.Scientists
		if newTotal > p.Population {
			return fmt.Errorf("bad pop movement: too many units")
		}
		p.Food.Farmers = commands.MovePopulation.Farmers
		p.Industry.Workers = commands.MovePopulation.Workers
		p.Science.Scientists = commands.MovePopulation.Scientists
	}

	// population growth calculations
	oldPopulation := p.Population
	growth := PopulationGrowth(p.Population, p.Food.ProducedPerBuilding, p.Food.ProducedPerFarmer, p.Food.Farmers, MaxPop(p))
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

	// industry output
	grossIndustry := p.Industry.ProducedPerBuilding + p.Industry.Workers*p.Industry.ProducedPerWorker
	industryProduced := grossIndustry - Pollution(p)

	var buildingName string
	for _, buildCommand := range commands.Build {
		if buildCommand.PlanetID == p.ID {
			buildingName = buildCommand.Building
		}
	}
	if buildingName == "" && !(p.IsBuilding && p.CurrentProject != "") {
		p.CurrentProject = "Trade Goods"
		p.IsBuilding = false
		p.BuildBucket = 0
	} else if buildingName == "Trade Goods" {
		p.CurrentProject = "Trade Goods"
		p.IsBuilding = false
		p.BuildBucket = 0
	} else if buildingName != "" {
		if !Build(p, buildingName, player.Available.Buildings, game) {
			p.CurrentProject = "Trade Goods"
			p.IsBuilding = false
			p.BuildBucket = 0
		}
	}

	hydratePlanetIncome(game, p, industryProduced)

	// Advance build project
	if p.IsBuilding {
		nextBucket := p.BuildBucket - industryProduced
		if nextBucket <= 0 {
			// project is complete
			p.Buildings = append(p.Buildings, p.CurrentProject)
			p.IsBuilding = false
			p.IndustryStack += -nextBucket
			p.BuildBucket = 0
			p.CurrentProject = ""
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

func Build(p *Planet, buildingName string, availableBuildings []string, game *Game) bool {
	// is the building already built on this planet
	found := false
	for _, buildingInPlanet := range p.Buildings {
		if buildingName == buildingInPlanet {
			found = true
			break
		}
	}

	// does the player have the tech to build this?
	known := false
	for _, availableBuilding := range availableBuildings {
		if buildingName == availableBuilding {
			known = true
			break
		}
	}

	if found || !known {
		return false
	}

	building, err := getBuildingFromName(game, buildingName)
	if err != nil {
		return false
	}

	if p.IsBuilding && p.CurrentProject == buildingName {
		return true
	}

	p.CurrentProject = buildingName
	p.IsBuilding = true
	p.BuildBucket = building.Cost

	return true
}
