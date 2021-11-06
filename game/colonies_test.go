package game_test

import (
	"ssg/game"
	"ssg/model"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGame_CreateCapitalPlanet(t *testing.T) {
	playerName := "test_player"
	var numStars uint = 1
	g := game.New(playerName, numStars)

	colonies, _ := g.GetColonyList(1)
	assert.Equal(t, len(colonies), 1, "there should be only one colony")

	// default specs for a capitol planet
	capitol := colonies[0]
	assert.Equal(t, capitol.Environment, model.PlanetEnvironmentTerran)
	assert.Equal(t, capitol.Size, model.PlanetSizeMedium)
	assert.Equal(t, capitol.MaxPop, model.CalculatePlanetMaxPop(model.PlanetSizeMedium, model.PlanetEnvironmentTerran))
	assert.Equal(t, capitol.MineralClass, model.PlanetMineralAbundant)
	assert.Equal(t, capitol.Gravity, model.PlanetGravityNormal)
}
