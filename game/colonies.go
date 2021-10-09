package game

import "ssg/model"

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

	planet.Owner = ownerID
	planet.CurrentPop = 1
	planet.Workers = 1

	return nil
}

// GetColonyList returns a list of planets that have a colony owned by given player
func (g *Game) GetColonyList(playerID uint) ([]*model.Planet, error) {
	if _, err := g.getPlayer(playerID); err != nil {
		return nil, err
	}

	colonyList := make([]*model.Planet, 0)
	for _, star := range g.Stars {
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
