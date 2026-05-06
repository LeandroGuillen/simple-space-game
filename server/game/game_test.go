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
	assert.Equal(t, int64(4), vPlanet.Industry.ProducedPerWorker)
	assert.Equal(t, int64(5), vPlanet.Industry.ProducedPerBuilding)

	assert.Equal(t, int64(3), vPlanet.Food.Farmers)
	assert.Equal(t, int64(2), vPlanet.Food.ProducedPerFarmer)
	assert.Equal(t, int64(2), vPlanet.Food.ProducedPerBuilding)

	assert.Equal(t, int64(2), vPlanet.Science.Scientists)
	assert.Equal(t, int64(6), vPlanet.Science.ProducedPerScientist)
	assert.Equal(t, int64(4), vPlanet.Science.ProducedPerBuilding)
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

func TestGame_ConstructingBuilding_CompletesAfterProductionCostIsCovered(t *testing.T) {
	g, _ := game.NewGame("test_player", 1, "#FFFFFF")
	planet := g.Galaxy.Stars[0].Planets[0]
	planet.Food.Farmers = 4
	planet.Industry.Workers = 10
	planet.Science.Scientists = 0
	planet.MineralClass = model.PlanetMineralAbundant

	automatedFactory := g.Resources.Buildings[1]
	err := g.BuildOnPlanet(planet.ID, 1, automatedFactory.ID)
	assert.NoError(t, err)
	assert.True(t, planet.IsBuilding)
	assert.Equal(t, "Automated Factory", planet.CurrentProject.Name)
	assert.Equal(t, automatedFactory.Cost, planet.BuildBucket)

	err = g.Turn(1, model.Commands{})
	assert.NoError(t, err)
	assert.True(t, planet.IsBuilding)
	assert.Equal(t, int64(30), planet.BuildBucket)
	assert.Empty(t, planet.Buildings)

	err = g.Turn(1, model.Commands{})
	assert.NoError(t, err)
	assert.False(t, planet.IsBuilding)
	assert.Equal(t, int64(0), planet.BuildBucket)
	assert.Len(t, planet.Buildings, 1)
	assert.Equal(t, "Automated Factory", planet.Buildings[0].Name)
}

func TestGame_BuildOnPlanet_TradeGoodsClearsCurrentProject(t *testing.T) {
	g, _ := game.NewGame("test_player", 1, "#FFFFFF")
	planet := g.Galaxy.Stars[0].Planets[0]

	automatedFactory := g.Resources.Buildings[1]
	err := g.BuildOnPlanet(planet.ID, 1, automatedFactory.ID)
	assert.NoError(t, err)

	err = g.BuildOnPlanet(planet.ID, 1, 0)
	assert.NoError(t, err)
	assert.False(t, planet.IsBuilding)
	assert.Equal(t, "", planet.CurrentProject.Name)
	assert.Equal(t, int64(0), planet.BuildBucket)
}

func TestGame_BuildOnPlanet_RejectsAlreadyBuiltBuilding(t *testing.T) {
	g, _ := game.NewGame("test_player", 1, "#FFFFFF")
	planet := g.Galaxy.Stars[0].Planets[0]
	automatedFactory := g.Resources.Buildings[1]
	planet.Buildings = append(planet.Buildings, automatedFactory)

	err := g.BuildOnPlanet(planet.ID, 1, automatedFactory.ID)

	assert.Error(t, err)
	assert.False(t, planet.IsBuilding)
	assert.Equal(t, "", planet.CurrentProject.Name)
	assert.Equal(t, int64(0), planet.BuildBucket)
}
