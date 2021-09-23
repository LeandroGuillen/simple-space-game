package game

import (
	"corion/game/builder"
	. "corion/model"
	"errors"
	"fmt"
	"math"
	"math/rand"

	"github.com/google/uuid"
)

type Player struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`

	Race string `json:"race"`

	Treasury       int64   `json:"treasury"`
	TaxRate        float64 `json:"tax_rate"`
	ResearchPoints uint    `json:"research_points"`

	Fleets         []uint `json:"fleets"`
	Freighters     uint   `json:"freighters"`
	FreightersUsed uint   `json:"freighters_used"`
}

type Game struct {
	Galaxy  `json:"galaxy"`
	Players []*Player `json:"players"`
	ID      string    `json:"id"`
}

func New(playerName string, numStars uint) *Game {
	g := &Game{
		Players: []*Player{
			{ID: 1, Name: playerName},
		},
		Galaxy: builder.NewGalaxy(numStars),
		ID:     uuid.New().String(),
	}

	_ = g.ColonizePlanet(1, 1)

	return g
}

func (g *Game) NewRandomGalaxy() Galaxy {
	// new galaxy parameters
	var stdDev float64 = 2
	var mean float64 = 6
	n := uint(math.Abs(math.Floor(rand.NormFloat64()*stdDev + mean)))
	galaxy := builder.NewGalaxy(n)
	return galaxy
}

func (g *Game) Turn(playerID uint) error {
	player, err := g.getPlayer(playerID)
	if err != nil {
		return err
	}

	player.Treasury += 2

	return nil
}

func (g *Game) getPlayer(ID uint) (*Player, error) {
	for _, player := range g.Players {
		if player.ID == ID {
			return player, nil
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
