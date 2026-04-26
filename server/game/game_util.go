package game

import (
	"fmt"
	"ssg/game/model"
)

func (g *Game) getPlayer(ID uint) (*model.Player, error) {
	for _, player := range g.Players {
		if player.ID == ID {
			return player, nil
		}
	}
	return nil, fmt.Errorf("player with ID %d not found", ID)
}

func (g *Game) getPlanet(ID uint) (*model.Planet, error) {
	for _, star := range g.Galaxy.Stars {
		for _, planet := range star.Planets {
			if planet.ID == ID {
				return planet, nil
			}
		}
	}
	return nil, fmt.Errorf("planet with ID %d not found", ID)
}
