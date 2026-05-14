package game

import (
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/assert"
)

func TestNewRandomPlanetSetsSpecialWorldSizes(t *testing.T) {
	SetRandomSeed(1)

	seenGasGiant := false
	seenAsteroidBelt := false

	for i := 0; i < 1000; i++ {
		planet := NewRandomPlanet()

		switch planet.Environment {
		case model.PlanetEnvironmentGasGiant:
			seenGasGiant = true
			assert.Equal(t, model.PlanetSizeHuge, planet.Size)
		case model.PlanetEnvironmentAsteroidBelt:
			seenAsteroidBelt = true
			assert.Equal(t, model.PlanetSizeLarge, planet.Size)
		}
	}

	assert.True(t, seenGasGiant)
	assert.True(t, seenAsteroidBelt)
}
