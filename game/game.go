package game

import (
	"errors"
	"fmt"
	"ssg/game/builder"
	"ssg/game/model"

	"github.com/google/uuid"
)

type Turnable interface {
	Turn()
}

type Game struct {
	Galaxy      model.Galaxy    `json:"galaxy"`
	Players     []*model.Player `json:"players"`
	ID          string          `json:"id"`
	CurrentTurn uint64          `json:"current_turn"`
}

func NewGame(playerName string, numStars uint) *Game {
	g := &Game{
		Galaxy: builder.NewGalaxy(numStars),
		Players: []*model.Player{
			{ID: 1, Name: playerName},
		},
		ID:          uuid.New().String(),
		CurrentTurn: 0,
	}

	_ = g.CreateCapitalPlanet(1, 1)

	return g
}

func (g *Game) Turn(playerID uint) error {
	player, err := g.getPlayer(playerID)
	if err != nil {
		return err
	}

	player.Treasury += 2

	colonyList, err := g.GetColonyList(playerID)
	if err != nil {
		return err
	}
	for _, colony := range colonyList {
		colony.Turn()
	}

	g.CurrentTurn++
	return nil
}

func (g *Game) getPlayer(ID uint) (*model.Player, error) {
	for _, player := range g.Players {
		if player.ID == ID {
			return player, nil
		}
	}
	return nil, errors.New(fmt.Sprintf("player with ID %d not found", ID))
}

func (g *Game) getPlanet(ID uint) (*model.Planet, error) {
	for _, star := range g.Galaxy.Stars {
		for _, planet := range star.Planets {
			if planet.ID == ID {
				return planet, nil
			}
		}
	}
	return nil, errors.New(fmt.Sprintf("planet with ID %d not found", ID))
}

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
	planet.MineralClass = model.PlanetMineralAbundant
	planet.Gravity = model.PlanetGravityNormal

	planet.Owner = ownerID
	planet.Population = 8000
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

	planet.Colonize(ownerID)

	return nil
}

// GetColonyList returns a list of planets that have a colony owned by given player.
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
