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

func TestPopulationGrowth_BestCase_NormalGravityTerran(t *testing.T) {
	p := makeOwnedPlanet(model.PlanetGravityNormal, model.PlanetEnvironmentTerran, 8000)
	assert.Equal(t, int64(40), game.PopulationGrowth(p))
}

func TestPopulationGrowth_BestCase_NormalGravityGaia(t *testing.T) {
	p := makeOwnedPlanet(model.PlanetGravityNormal, model.PlanetEnvironmentGaia, 8000)
	assert.Equal(t, int64(40), game.PopulationGrowth(p))
}

func TestPopulationGrowth_WorstInhabitedCase_HighGravityToxic(t *testing.T) {
	p := makeOwnedPlanet(model.PlanetGravityHigh, model.PlanetEnvironmentToxic, 1000)
	// 40 * 0.5 * 0.15 = 3
	assert.Equal(t, int64(3), game.PopulationGrowth(p))
}

func TestPopulationGrowth_MidRange_LowGravityOcean(t *testing.T) {
	p := makeOwnedPlanet(model.PlanetGravityLow, model.PlanetEnvironmentOcean, 1000)
	// round(40 * 0.75 * 0.85) = round(25.5) = 26
	assert.Equal(t, int64(26), game.PopulationGrowth(p))
}

func TestPopulationGrowth_UnownedPlanet_ReturnsZero(t *testing.T) {
	p := game.NewPlanet(model.PlanetMineralAbundant, model.PlanetSizeMedium,
		model.PlanetGravityNormal, model.PlanetEnvironmentTerran)
	p.Owner = 0
	p.Population = 8000
	assert.Equal(t, int64(0), game.PopulationGrowth(p))
}

func TestPopulationGrowth_EmptyColony_ReturnsZero(t *testing.T) {
	p := makeOwnedPlanet(model.PlanetGravityNormal, model.PlanetEnvironmentTerran, 0)
	assert.Equal(t, int64(0), game.PopulationGrowth(p))
}

func TestPopulationGrowth_UninhabitableEnvironment_ReturnsZero(t *testing.T) {
	p := makeOwnedPlanet(model.PlanetGravityNormal, model.PlanetEnvironmentGasGiant, 1000)
	assert.Equal(t, int64(0), game.PopulationGrowth(p))
}

// Tests below exercise planetTurn indirectly via Game.Turn to validate the
// clamping of Population to [0, MaxPop] inside the turn hook.

func TestPlanetTurn_PopulationClampedToMaxPop(t *testing.T) {
	g, _ := game.NewGame("test_player", 0, "#FFFFFF")

	// Medium-size Terran planet -> MaxPop = 3 * 4 * 1000 = 12000.
	planet := makeOwnedPlanet(model.PlanetGravityNormal, model.PlanetEnvironmentTerran, 11990)
	planet.ID = 42
	star := game.NewStar(2, "test_star", model.StarColorYellow, []*model.Planet{planet})
	g.Galaxy.Stars = append(g.Galaxy.Stars, star)

	err := g.Turn(1, model.Commands{})
	assert.NoError(t, err)

	// Without a cap the next value would be 11990 + 40 = 12030; with the
	// MaxPop clamp it should top out at 12000 instead.
	assert.Equal(t, int64(12000), planet.Population)

	// A second turn at the cap leaves Population unchanged.
	err = g.Turn(1, model.Commands{})
	assert.NoError(t, err)
	assert.Equal(t, int64(12000), planet.Population)
}

func TestPlanetTurn_PopulationNeverGoesNegative(t *testing.T) {
	g, _ := game.NewGame("test_player", 0, "#FFFFFF")

	// A toxic planet with positive growth should still increase, but this
	// test guards the floor: drive the turn loop and ensure Population
	// never dips below 0 even with a low starting value.
	planet := makeOwnedPlanet(model.PlanetGravityHigh, model.PlanetEnvironmentToxic, 1)
	planet.ID = 43
	star := game.NewStar(3, "test_star_2", model.StarColorYellow, []*model.Planet{planet})
	g.Galaxy.Stars = append(g.Galaxy.Stars, star)

	for i := 0; i < 10; i++ {
		err := g.Turn(1, model.Commands{})
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, planet.Population, int64(0))
	}
}
