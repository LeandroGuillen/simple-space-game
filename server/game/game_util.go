package game

import (
	"fmt"
	"slices"
	. "ssg/model"
	"strings"
)

func (g *Game) getPlayer(ID uint) (*Player, error) {
	for _, player := range g.Players {
		if player.ID == ID {
			return player, nil
		}
	}
	return nil, fmt.Errorf("player with ID %d not found", ID)
}

func (g *Game) getPlanet(ID uint) (*Planet, error) {
	for _, star := range g.Galaxy.Stars {
		for _, planet := range star.Planets {
			if planet.ID == ID {
				return planet, nil
			}
		}
	}
	return nil, fmt.Errorf("planet with ID %d not found", ID)
}

func (g *Game) getStar(ID uint) (*Star, error) {
	for _, star := range g.Galaxy.Stars {
		if star.ID == ID {
			return star, nil
		}
	}
	return nil, fmt.Errorf("star with ID %d not found", ID)
}

func buildOnPlanet(game *Game, planet *Planet, buildingName string) {
	if slices.Contains(planet.Buildings, buildingName) {
		return
	}
	building, err := getBuildingFromName(game, buildingName)
	if err != nil {
		return
	}
	planet.Buildings = slices.DeleteFunc(planet.Buildings, func(existing string) bool {
		return slices.Contains(building.Upgrades, existing)
	})
	planet.Buildings = append(planet.Buildings, buildingName)
}

func hasBuilding(p *Planet, buildingName string) bool {
	for _, building := range p.Buildings {
		if strings.EqualFold(building, buildingName) {
			return true
		}
	}
	return false
}

func getBuildingFromName(g *Game, name string) (Building, error) {
	for _, building := range g.Resources.Buildings {
		if building.Name == name {
			return building, nil
		}
	}
	return Building{}, fmt.Errorf("building %s not found", name)
}
