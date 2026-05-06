package game_test

import (
	"ssg/game"
	"ssg/model"
	"testing"

	"github.com/stretchr/testify/assert"
)

func makeOwnedPlanet(gravity model.PlanetGravity, env model.PlanetEnvironment, pop int64) *model.Planet {
	p := game.NewPlanet(model.PlanetMineralAbundant, model.PlanetSizeMedium, gravity, env)
	p.Owner = 1
	p.Population = pop
	return p
}

func TestPopulationGrowth_NoFoodOutput(t *testing.T) {
	p := makeOwnedPlanet(model.PlanetGravityNormal, model.PlanetEnvironmentTerran, 8000)
	assert.Equal(t, int64(-327), game.PopulationGrowth(p))
}

func TestPopulationGrowth_EmptyColony_ReturnsZero(t *testing.T) {
	p := makeOwnedPlanet(model.PlanetGravityNormal, model.PlanetEnvironmentTerran, 0)
	assert.Equal(t, int64(0), game.PopulationGrowth(p))
}

func TestPlanetTurn_PopulationClampedToMaxPop(t *testing.T) {
	g, _ := game.NewGame("test_player", 0, "#FFFFFF")

	// Medium-size Terran planet -> MaxPop = 3 * 4 * 1000 = 12000.
	planet := makeOwnedPlanet(model.PlanetGravityNormal, model.PlanetEnvironmentTerran, 11990)
	planet.ID = 42
	planet.Food.Farmers = 6
	star := game.NewStar(2, "test_star", model.StarColorYellow, []*model.Planet{planet})
	g.Galaxy.Stars = append(g.Galaxy.Stars, star)

	err := g.Turn(1, model.Commands{})
	assert.NoError(t, err)

	assert.Equal(t, int64(12000), planet.Population)

	// A second turn at the cap leaves Population unchanged.
	err = g.Turn(1, model.Commands{})
	assert.NoError(t, err)
	assert.Equal(t, int64(12000), planet.Population)
}

func TestPlanetTurn_PopulationNeverGoesNegative(t *testing.T) {
	g, _ := game.NewGame("test_player", 0, "#FFFFFF")

	planet := makeOwnedPlanet(model.PlanetGravityHigh, model.PlanetEnvironmentToxic, 1)
	planet.ID = 43
	star := game.NewStar(3, "test_star_2", model.StarColorYellow, []*model.Planet{planet})
	g.Galaxy.Stars = append(g.Galaxy.Stars, star)

	for range 10 {
		err := g.Turn(1, model.Commands{})
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, planet.Population, int64(0))
	}
}
