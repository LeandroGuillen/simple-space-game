package game

import (
	"corion/game/builder"
	. "corion/model"
	"errors"
	"fmt"
	"math"
	"math/rand"
)

type Player struct {
	ID uint `json:"id"`
}

type Game struct {
	Galaxy  `json:"galaxy"`
	Players []Player `json:"players"`
}

func (g *Game) NewGalaxy(numStars uint) Galaxy {
	galaxy := builder.NewGalaxy(numStars)
	return galaxy
}

func (g *Game) NewRandomGalaxy() Galaxy {
	// new galaxy parameters
	var stdDev float64 = 2
	var mean float64 = 6
	n := uint(math.Abs(math.Floor(rand.NormFloat64()*stdDev + mean)))
	galaxy := builder.NewGalaxy(n)
	return galaxy
}

func (g *Game) getPlayer(ID uint) (*Player, error) {
	for _, player := range g.Players {
		if player.ID == ID {
			return &player, nil
		}
	}
	return nil, errors.New(fmt.Sprintf("player with ID %d not found", ID))
}

func (g *Game) getPlanet(ID uint) (*Planet, error) {
	for _, star := range g.Stars {
		for _, planet := range star.Planets {
			if planet.ID == ID {
				return planet, nil
			}
		}
	}
	return nil, errors.New(fmt.Sprintf("planet with ID %d not found", ID))
}
