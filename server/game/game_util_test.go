package game

import (
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/assert"
)

func TestBuildOnPlanetAddsMissingBuilding(t *testing.T) {
	g := &Game{Resources: &Resources{BuildingCollection: &BuildingCollection{Buildings: []model.Building{{Name: "Capitol"}, {Name: "Star Base"}}}}}
	planet := &model.Planet{}

	buildOnPlanet(g, planet, "Capitol")
	buildOnPlanet(g, planet, "Star Base")
	buildOnPlanet(g, planet, "Capitol")

	assert.Equal(t, []string{"Capitol", "Star Base"}, planet.Buildings)
}

func TestNewGameStartsWithDefaultBuildingsAvailableFromStartingTech(t *testing.T) {
	resources, err := NewResourceManager()
	assert.NoError(t, err)

	game, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: DefaultRace("Human")})
	assert.NoError(t, err)

	player := game.Players[0]
	assert.Contains(t, player.ResearchedTechs, startingColonyInfrastructureTech)
	assert.Equal(t, []string{"Capitol", "Colony Base", "Star Base", "Marine Barracks", "Transport"}, player.Available.Buildings)
	assert.Contains(t, player.Available.Drives, "Nuclear")
	assert.Contains(t, player.Available.Armors, "Titanium")
}

func TestSetPlanetAutobuildPersistsOnOwnedPlanet(t *testing.T) {
	resources, err := NewResourceManager()
	assert.NoError(t, err)

	game, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: DefaultRace("Human")})
	assert.NoError(t, err)

	assert.NoError(t, game.SetPlanetAutobuild(1, 1, true))

	planet, err := game.getPlanet(1)
	assert.NoError(t, err)
	assert.True(t, planet.Autobuild)
}
