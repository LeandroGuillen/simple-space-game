package game

import (
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/assert"
)

func TestBuildOnPlanetAddsMissingBuilding(t *testing.T) {
	planet := &model.Planet{}

	buildOnPlanet(planet, "Capitol")
	buildOnPlanet(planet, "Star Base")
	buildOnPlanet(planet, "Capitol")

	assert.Equal(t, []string{"Capitol", "Star Base"}, planet.Buildings)
}

func TestNewPlayerStartsWithDefaultBuildingsAvailable(t *testing.T) {
	player := NewPlayer(1, "Ada", model.DefaultPlayerColor())

	assert.Equal(t, []string{"Capitol", "Colony Base", "Star Base", "Marine Barracks"}, player.Available.Buildings)
}
