package game

import (
	"math"
	"math/rand"
	"ssg/model"
)

func (g *Game) CreateCapitalPlanet(ownerID, planetID uint) error {
	if _, err := g.getPlayer(ownerID); err != nil {
		return err
	}

	planet, err := g.getPlanet(planetID)
	if err != nil {
		return err
	}

	// default capital terran world
	planet.Environment = model.PlanetEnvironmentTerran
	planet.Size = model.PlanetSizeMedium
	planet.MaxPop = model.CalculatePlanetMaxPop(model.PlanetSizeMedium, model.PlanetEnvironmentTerran)
	planet.MineralClass = model.PlanetMineralAbundant
	planet.Gravity = model.PlanetGravityNormal

	planet.Owner = ownerID
	planet.CurrentPop = 8
	planet.CurrentPopMillis = 8000
	planet.PopGrowth = 50
	planet.Workers = 3
	planet.Farmers = 2
	planet.Scientists = 3

	planet.Buildings = []string{"capitol"}

	return nil
}

func (g *Game) ColonizePlanet(ownerID, planetID uint) error {
	if _, err := g.getPlayer(ownerID); err != nil {
		return err
	}

	planet, err := g.getPlanet(planetID)
	if err != nil {
		return err
	}

	ColonizePlanet2(ownerID, planet)

	return nil
}

func ColonizePlanet2(ownerID uint, planet *model.Planet) {
	planet.Owner = ownerID
	planet.CurrentPop = 1
	planet.CurrentPopMillis = 1000
	planet.PopGrowth = 50
	planet.Workers = 1
}

// GetColonyList returns a list of planets that have a colony owned by given player
func (g *Game) GetColonyList(playerID uint) ([]*model.Planet, error) {
	if _, err := g.getPlayer(playerID); err != nil {
		return nil, err
	}

	colonyList := make([]*model.Planet, 0)
	for _, star := range g.Galaxy.Stars {
		for _, planet := range star.Planets {
			if planet.Owner == playerID {
				colonyList = append(colonyList, planet)
			}
		}
	}
	return colonyList, nil
}

func (g *Game) BuildOnPlanet(planetID, ownerID, buildingID uint) error {
	if _, err := g.getPlayer(ownerID); err != nil {
		return err
	}

	p, err := g.getPlanet(planetID)
	if err != nil {
		return err
	}

	p.CurrentProject = buildingID
	return nil
}

func IsColonized(p *model.Planet) bool {
	return p.CurrentPop > 0
}

func getNewPopulation(planet *model.Planet) {
	// should the colony population grow or shrink?
	planet.CurrentPopMillis += planet.PopGrowth
	expectedPopulation := int64(math.Floor(float64(planet.CurrentPopMillis) / 1000))
	if expectedPopulation < planet.CurrentPop {
		// somebody died
		removePopulation(planet)
	} else if expectedPopulation >= planet.MaxPop {
		// cannot fit more people on this planet
		expectedPopulation = planet.MaxPop
		planet.CurrentPopMillis = planet.MaxPop * 1000
	} else if expectedPopulation > planet.CurrentPop {
		// somebody was born
		addPopulation(planet, 1)
	}
	planet.CurrentPop = expectedPopulation
}

func addPopulation(p *model.Planet, pop int64) {
	// put one more where most people are
	if p.Farmers >= p.Workers && p.Farmers >= p.Scientists {
		p.Farmers += pop
	} else if p.Workers >= p.Farmers && p.Workers >= p.Scientists {
		p.Workers += pop
	} else {
		p.Scientists += pop
	}
}

func removePopulation(p *model.Planet) {
	if p.CurrentPop <= 0 {
		return
	}
	found := false
	for !found {
		popGroup := rand.Intn(3)
		switch popGroup {
		case 0:
			if p.Farmers > 0 {
				p.Farmers--
				found = true
			}
		case 1:
			if p.Workers > 0 {
				p.Workers--
				found = true
			}
		case 2:
			if p.Scientists > 0 {
				p.Scientists--
				found = true
			}
		}
	}
}
