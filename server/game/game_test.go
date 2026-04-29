package game_test

import (
	"encoding/json"
	"ssg/game"
	"ssg/model"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewGame(t *testing.T) {
	playerName := "test_player"
	var numStars uint = 1
	g, _ := game.NewGame(playerName, numStars, "#FFFFFF")

	assert.Equal(t, len(g.Players), 1, "there should be only 1 player")
	assert.Equal(t, g.Players[0].Name, playerName, "player name is set up")
}

func TestGame_TurnPassing(t *testing.T) {
	testPlanet := game.NewAverageTerranPlanet()
	testStar := game.NewStar(1, "star_name", model.StarColorYellow, []*model.Planet{testPlanet})
	g, _ := game.NewGame("test_player", 0, "#FFFFFF")
	g.Galaxy.Stars = append(g.Galaxy.Stars, testStar)

	assert.Equal(t, g.CurrentTurn, int64(3500))
	_ = g.Turn(1, model.Commands{})
	assert.Equal(t, g.CurrentTurn, int64(3501))
	_ = g.Turn(1, model.Commands{})
	_ = g.Turn(1, model.Commands{})
	_ = g.Turn(1, model.Commands{})
	assert.Equal(t, g.CurrentTurn, int64(3504))
}

func TestGame_TurnPassing_GrowsCapitalPopulationByBaseline(t *testing.T) {
	g, _ := game.NewGame("test_player", 1, "#FFFFFF")

	capital := g.Galaxy.Stars[0].Planets[0]
	assert.Equal(t, int64(8000), capital.Population)

	err := g.Turn(1, model.Commands{})
	assert.NoError(t, err)
	assert.Equal(t, int64(8040), capital.Population)
}

func TestGame_VisibleSystems_AreScopedPerPlayerExploration(t *testing.T) {
	g, _ := game.NewGame("test_player", 2, "#FFFFFF")

	visible, err := g.GetVisibleGameForPlayer(1)
	assert.NoError(t, err)
	assert.Len(t, visible.Galaxy.Stars, 2)
	assert.NotEmpty(t, visible.Galaxy.Stars[0].Name)
	assert.NotEmpty(t, visible.Galaxy.Stars[0].Planets)
	assert.Empty(t, visible.Galaxy.Stars[1].Name)
	assert.Nil(t, visible.Galaxy.Stars[1].Planets)
}

func TestGame_ExploreStar_RevealsNameAndPlanets(t *testing.T) {
	g, _ := game.NewGame("test_player", 2, "#FFFFFF")

	err := g.ExploreStar(1, 2)
	assert.NoError(t, err)

	visible, err := g.GetVisibleGameForPlayer(1)
	assert.NoError(t, err)
	assert.NotEmpty(t, visible.Galaxy.Stars[1].Name)
	assert.NotEmpty(t, visible.Galaxy.Stars[1].Planets)
}

func TestGame_VisiblePlanet_IncludesOutputBreakdowns(t *testing.T) {
	g, _ := game.NewGame("test_player", 1, "#FFFFFF")
	planet := g.Galaxy.Stars[0].Planets[0]

	planet.Industry.Workers = 5
	planet.Food.Farmers = 3
	planet.Science.Scientists = 2
	planet.MineralClass = model.PlanetMineralAbundant // base 3
	planet.Buildings = []model.Building{
		{
			ID:   10,
			Name: "Automated Factory",
			Effects: []model.Effect{
				{Name: "industryBonus", Bonus: 1},
				{Name: "flatIndustryBonus", Bonus: 5},
			},
		},
		{
			ID:   20,
			Name: "Hydroponic Farm",
			Effects: []model.Effect{
				{Name: "flatAgricultureBonus", Bonus: 2},
			},
		},
		{
			ID:   30,
			Name: "Science Lab",
			Effects: []model.Effect{
				{Name: "scienceBonus", Bonus: 3},
				{Name: "flatScienceBonus", Bonus: 4},
			},
		},
	}
	g.Players[0].Stats.Farming = 1

	visible, err := g.GetVisibleGameForPlayer(1)
	assert.NoError(t, err)

	vPlanet := visible.Galaxy.Stars[0].Planets[0]

	assert.Equal(t, int64(5), vPlanet.Industry.Workers)
	assert.Equal(t, int64(3), vPlanet.Industry.BasePerWorker)
	assert.Equal(t, int64(1), vPlanet.Industry.PerWorkerBonusTotal)
	assert.Equal(t, int64(5), vPlanet.Industry.FlatBonusTotal)
	assert.Equal(t, int64(25), vPlanet.Industry.Total) // 5*(3+1)+5
	assert.NotEmpty(t, vPlanet.Industry.Contributors)

	assert.Equal(t, int64(3), vPlanet.Food.Farmers)
	assert.Equal(t, int64(1), vPlanet.Food.BasePerFarmer)
	assert.Equal(t, int64(1), vPlanet.Food.PerFarmerBonusTotal) // race bonus
	assert.Equal(t, int64(2), vPlanet.Food.FlatBonusTotal)
	assert.Equal(t, int64(8), vPlanet.Food.Total) // 3*(1+1)+2
	assert.NotEmpty(t, vPlanet.Food.Contributors)

	assert.Equal(t, int64(2), vPlanet.Science.Scientists)
	assert.Equal(t, int64(1), vPlanet.Science.BasePerScientist)
	assert.Equal(t, int64(3), vPlanet.Science.PerScientistBonusTotal)
	assert.Equal(t, int64(4), vPlanet.Science.FlatBonusTotal)
	assert.Equal(t, int64(12), vPlanet.Science.Total) // 2*(1+3)+4
	assert.NotEmpty(t, vPlanet.Science.Contributors)
}

func TestGame_VisiblePlanet_JsonOmitsLegacyPopAssignments(t *testing.T) {
	g, _ := game.NewGame("test_player", 1, "#FFFFFF")

	visible, err := g.GetVisibleGameForPlayer(1)
	assert.NoError(t, err)

	payload, err := json.Marshal(visible)
	assert.NoError(t, err)

	var root map[string]any
	err = json.Unmarshal(payload, &root)
	assert.NoError(t, err)

	galaxy := root["galaxy"].(map[string]any)
	stars := galaxy["stars"].([]any)
	star := stars[0].(map[string]any)
	planets := star["planets"].([]any)
	planet := planets[0].(map[string]any)

	_, hasWorkers := planet["workers"]
	_, hasFarmers := planet["farmers"]
	_, hasScientists := planet["scientists"]
	assert.False(t, hasWorkers)
	assert.False(t, hasFarmers)
	assert.False(t, hasScientists)

	industry, hasIndustry := planet["industry"]
	food, hasFood := planet["food"]
	science, hasScience := planet["science"]
	assert.True(t, hasIndustry)
	assert.True(t, hasFood)
	assert.True(t, hasScience)
	assert.NotNil(t, industry)
	assert.NotNil(t, food)
	assert.NotNil(t, science)
}

func TestGame_TurnWithMovePopulationCommand(t *testing.T) {
	g, _ := game.NewGame("test_player", 1, "#FFFFFF")
	planet := g.Galaxy.Stars[0].Planets[0]

	planet.Food.Farmers = 5
	planet.Industry.Workers = 3
	planet.Science.Scientists = 2
	planet.Marines = 0

	commands := model.Commands{
		MovePopulation: []model.MovePopulationCommand{
			{PlanetID: planet.ID, From: model.PopCategoryFarmers, To: model.PopCategoryWorkers, Count: 2},
		},
	}

	err := g.Turn(1, commands)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), planet.Food.Farmers)
	assert.Equal(t, int64(5), planet.Industry.Workers)
}

func TestGame_TurnMovePopulation_PlanetNotOwned(t *testing.T) {
	g, _ := game.NewGame("test_player", 1, "#FFFFFF")
	planet := g.Galaxy.Stars[0].Planets[0]

	planet.Food.Farmers = 5
	planet.Owner = 2

	commands := model.Commands{
		MovePopulation: []model.MovePopulationCommand{
			{PlanetID: planet.ID, From: model.PopCategoryFarmers, To: model.PopCategoryWorkers, Count: 2},
		},
	}

	err := g.Turn(1, commands)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not owned by player")
}

func TestGame_TurnMovePopulation_NotEnoughPop(t *testing.T) {
	g, _ := game.NewGame("test_player", 1, "#FFFFFF")
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

func TestGame_TurnMovePopulation_ExceedsMaxPop(t *testing.T) {
	g, _ := game.NewGame("test_player", 1, "#FFFFFF")
	planet := g.Galaxy.Stars[0].Planets[0]

	planet.Food.Farmers = 5
	planet.Industry.Workers = 5
	planet.Science.Scientists = 5
	planet.Marines = 5
	planet.Population = 20
	planet.Size = model.PlanetSizeTiny

	commands := model.Commands{
		MovePopulation: []model.MovePopulationCommand{
			{PlanetID: planet.ID, From: model.PopCategoryFarmers, To: model.PopCategoryWorkers, Count: 5},
		},
	}

	err := g.Turn(1, commands)
	assert.NoError(t, err)
}
