package game

import "corion/model"

func (g *Game) ColonizePlanet(ownerID, planetID uint) error {
	if _, err := g.getPlayer(ownerID); err != nil {
		return err
	}

	planet, err := g.getPlanet(planetID)
	if err != nil {
		return err
	}

	planet.Owner = ownerID

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
