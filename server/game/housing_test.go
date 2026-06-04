package game

import (
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func housingTestGame() (*Game, *model.Player, *model.Planet) {
	player := &model.Player{ID: 1}
	planet := &model.Planet{
		ID:           1,
		Owner:        1,
		Size:         model.PlanetSizeHuge,
		MineralClass: model.PlanetMineralPoor,
		Environment:  model.PlanetEnvironmentTerran,
		Population:   10000,
		Food: model.PlanetFoodOutput{
			Farmers: 10,
		},
		Industry: model.PlanetIndustryOutput{
			Workers: 5,
		},
	}
	game := &Game{
		Players:   []*model.Player{player},
		Resources: &Resources{BuildingCollection: &BuildingCollection{}},
	}
	return game, player, planet
}

func TestHousingIncreasesGrowthFromNetIndustryAndPopulationUnits(t *testing.T) {
	game, player, planet := housingTestGame()
	baselineGrowth := PopulationGrowthFromFood(
		player,
		planet.Population,
		20,
		MaxPop(game, planet, player),
		0,
		0,
	)

	require.NoError(t, planetTurn(planet, model.Commands{
		Build: []*model.BuildCommand{{PlanetID: planet.ID, Building: housingProject}},
	}, player, game))

	assert.Equal(t, housingProject, planet.CurrentProject)
	assert.False(t, planet.IsBuilding)
	assert.Equal(t, int64(10), GamePlanetIndustryProduced(game, planet))
	assert.Equal(t, baselineGrowth*140/100, planet.LastTurnPopGrowth)
}

func TestHousingGrowthBonusFallsAsPopulationGrows(t *testing.T) {
	assert.Equal(t, int64(40), housingGrowthBonus(10, 10000))
	assert.Equal(t, int64(20), housingGrowthBonus(10, 20000))
}

func TestHousingUsesPostPollutionIndustry(t *testing.T) {
	game, player, planet := housingTestGame()
	planet.Industry.Workers = 6

	require.NoError(t, planetTurn(planet, model.Commands{
		Build: []*model.BuildCommand{{PlanetID: planet.ID, Building: housingProject}},
	}, player, game))

	assert.Equal(t, int64(11), GamePlanetIndustryProduced(game, planet))
	assert.Equal(t, int64(44), housingGrowthBonus(GamePlanetIndustryProduced(game, planet), 10000))
}

func TestHousingDoesNotProduceTradeGoodsOrStackIndustry(t *testing.T) {
	game, player, planet := housingTestGame()
	planet.IndustryStack = 7

	require.NoError(t, planetTurn(planet, model.Commands{
		Build: []*model.BuildCommand{{PlanetID: planet.ID, Building: housingProject}},
	}, player, game))

	assert.Equal(t, int64(0), planet.Income.TradeGoods)
	assert.Equal(t, int64(7), planet.IndustryStack)
}

func TestHousingPersistsUntilChanged(t *testing.T) {
	game, player, planet := housingTestGame()

	require.NoError(t, planetTurn(planet, model.Commands{
		Build: []*model.BuildCommand{{PlanetID: planet.ID, Building: housingProject}},
	}, player, game))
	require.NoError(t, planetTurn(planet, model.Commands{}, player, game))

	assert.Equal(t, housingProject, planet.CurrentProject)
	assert.False(t, planet.IsBuilding)
}

func TestHousingFallsBackToTradeGoodsAtMaxPopulation(t *testing.T) {
	t.Run("already full when selected", func(t *testing.T) {
		game, player, planet := housingTestGame()
		planet.Population = MaxPop(game, planet, player)

		require.NoError(t, planetTurn(planet, model.Commands{
			Build: []*model.BuildCommand{{PlanetID: planet.ID, Building: housingProject}},
		}, player, game))

		assert.Equal(t, tradeGoodsProject, planet.CurrentProject)
		assert.False(t, planet.IsBuilding)
	})

	t.Run("reaches capacity after growth", func(t *testing.T) {
		game, player, planet := housingTestGame()
		planet.Population = MaxPop(game, planet, player) - 50

		require.NoError(t, planetTurn(planet, model.Commands{
			Build: []*model.BuildCommand{{PlanetID: planet.ID, Building: housingProject}},
		}, player, game))

		assert.Equal(t, MaxPop(game, planet, player), planet.Population)
		assert.Equal(t, tradeGoodsProject, planet.CurrentProject)
	})
}

func TestSanitizeBuildQueueRemovesHousing(t *testing.T) {
	game, player, planet := housingTestGame()
	player.Available.Buildings = []string{"Automated Factory"}
	game.Resources.BuildingCollection.Buildings = []model.Building{{Name: "Automated Factory", Cost: 60}}

	queue := sanitizeBuildQueue(
		planet,
		tradeGoodsProject,
		[]string{housingProject, "Automated Factory", tradeGoodsProject},
		player.Available.Buildings,
		game,
	)

	assert.Equal(t, []string{"Automated Factory"}, queue)
}
