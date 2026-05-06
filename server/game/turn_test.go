package game_test

import (
	"ssg/game"
	"ssg/model"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGame_TurnBasic(t *testing.T) {
	g, _ := game.NewGame("test_player", 0, "")
	testPlanet := game.NewAverageTerranPlanet()
	testStar := game.NewStar(1, "star_name", model.StarColorYellow, []*model.Planet{testPlanet})
	g.Galaxy.Stars = append(g.Galaxy.Stars, testStar)

	assert.Equal(t, g.CurrentTurn, int64(3500))
	_ = g.Turn(1, model.Commands{})
	assert.Equal(t, g.CurrentTurn, int64(3501))
	_ = g.Turn(1, model.Commands{})
	_ = g.Turn(1, model.Commands{})
	_ = g.Turn(1, model.Commands{})
	assert.Equal(t, g.CurrentTurn, int64(3504))
}

func TestGame_Turn_AppliesPopulationGrowth(t *testing.T) {
	g, _ := game.NewGame("test_player", 1, "")

	planet := g.Galaxy.Stars[0].Planets[0]
	assert.Equal(t, int64(8000), planet.Population)

	err := g.Turn(1, model.Commands{})
	assert.NoError(t, err)
	assert.Equal(t, int64(7873), planet.Population)
}

func TestGame_TurnWithMovePopulationCommand(t *testing.T) {
	g, _ := game.NewGame("test_player", 1, "")
	planet := g.Galaxy.Stars[0].Planets[0]

	planet.Food.Farmers = 5
	planet.Industry.Workers = 3
	planet.Science.Scientists = 2

	commands := model.Commands{
		MovePopulation: []model.MovePopulationCommand{
			{PlanetID: planet.ID, From: model.PopCategoryFarmers, To: model.PopCategoryWorkers, Count: 1},
		},
	}

	err := g.Turn(1, commands)
	assert.NoError(t, err)
	assert.Equal(t, planet.Food.Farmers, int64(4))
	assert.Equal(t, planet.Industry.Workers, int64(4))
	assert.Equal(t, planet.Science.Scientists, int64(2))
}

func TestGame_TurnMovePopulation_NotEnoughPop(t *testing.T) {
	g, _ := game.NewGame("test_player", 1, "")
	planet := g.Galaxy.Stars[0].Planets[0]

	planet.Food.Farmers = 2

	commands := model.Commands{
		MovePopulation: []model.MovePopulationCommand{
			{PlanetID: planet.ID, From: model.PopCategoryFarmers, To: model.PopCategoryWorkers, Count: 5},
		},
	}

	err := g.Turn(1, commands)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not enough")
}

func TestGame_TurnPopUnitIncrease(t *testing.T) {
	testCases := []struct {
		description        string
		originalPopulation int64
		farmersOriginal    int64
		farmersExpected    int64
		workersOriginal    int64
		workersExpected    int64
		scientistsOriginal int64
		scientistsExpected int64
	}{
		{"1 farmer increased", 3999, 3, 4, 1, 1, 1, 1},
		{"worker unchanged after food penalty", 7999, 2, 2, 3, 3, 2, 2},
		{"scientist unchanged after food penalty", 7999, 2, 2, 2, 2, 3, 3},
		{"no increase", 3199, 3, 3, 1, 1, 1, 1},
		{"all being equal, no increase after food penalty", 9999, 3, 3, 3, 3, 3, 3},
		{"farmers and scientists unchanged after food penalty", 9999, 3, 3, 2, 2, 3, 3},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			g, _ := game.NewGame("test_player", 1, "")
			planet := g.Galaxy.Stars[0].Planets[0]

			planet.Food.Farmers = tc.farmersOriginal
			planet.Industry.Workers = tc.workersOriginal
			planet.Science.Scientists = tc.scientistsOriginal
			planet.Population = tc.originalPopulation

			err := g.Turn(1, model.Commands{})
			assert.Nil(t, err)
			assert.Equal(t, tc.farmersExpected, planet.Food.Farmers)
			assert.Equal(t, tc.workersExpected, planet.Industry.Workers)
			assert.Equal(t, tc.scientistsExpected, planet.Science.Scientists)
		})
	}

}
