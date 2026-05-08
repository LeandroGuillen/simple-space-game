package game

import (
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTurnAddsIncomeToTreasuryEveryTurn(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, "Ada", 4, model.DefaultPlayerColor())
	require.NoError(t, err)

	player := g.Players[0]
	initialTreasury := player.Treasury

	require.NoError(t, g.Turn(player.ID, model.Commands{}))
	firstTurnTreasury := player.Treasury
	require.Greater(t, firstTurnTreasury, initialTreasury)

	colonies, err := g.GetColonyList(player.ID)
	require.NoError(t, err)
	require.NotEmpty(t, colonies)
	require.Greater(t, colonies[0].Income.TradeGoods, int64(0))

	require.NoError(t, g.Turn(player.ID, model.Commands{}))
	require.Greater(t, player.Treasury, firstTurnTreasury)
}

func TestTurnStartsConstructionWithBuildingCostAndNoTradeGoods(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, "Ada", 4, model.DefaultPlayerColor())
	require.NoError(t, err)

	player := g.Players[0]
	require.NoError(t, g.Turn(player.ID, model.Commands{
		Build: []*model.BuildCommand{
			{PlanetID: 1, Building: "Colony Base"},
		},
	}))

	colonies, err := g.GetColonyList(player.ID)
	require.NoError(t, err)
	require.NotEmpty(t, colonies)

	capital := colonies[0]
	assert.NotContains(t, capital.Buildings, "Colony Base")
	assert.True(t, capital.IsBuilding)
	assert.Equal(t, "Colony Base", capital.CurrentProject)
	assert.Equal(t, int64(0), capital.Income.TradeGoods)
	assert.Greater(t, capital.BuildBucket, int64(0))
	assert.Less(t, capital.BuildBucket, int64(200))
}

func TestTurnContinuesExistingConstructionWithoutRepeatedCommand(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, "Ada", 4, model.DefaultPlayerColor())
	require.NoError(t, err)

	player := g.Players[0]
	require.NoError(t, g.Turn(player.ID, model.Commands{
		Build: []*model.BuildCommand{
			{PlanetID: 1, Building: "Colony Base"},
		},
	}))
	colonies, err := g.GetColonyList(player.ID)
	require.NoError(t, err)
	firstBucket := colonies[0].BuildBucket

	require.NoError(t, g.Turn(player.ID, model.Commands{}))
	colonies, err = g.GetColonyList(player.ID)
	require.NoError(t, err)

	assert.True(t, colonies[0].IsBuilding)
	assert.Equal(t, "Colony Base", colonies[0].CurrentProject)
	assert.Equal(t, int64(0), colonies[0].Income.TradeGoods)
	assert.Less(t, colonies[0].BuildBucket, firstBucket)
}

func TestTurnCanSwitchConstructionBackToTradeGoods(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, "Ada", 4, model.DefaultPlayerColor())
	require.NoError(t, err)

	player := g.Players[0]
	require.NoError(t, g.Turn(player.ID, model.Commands{
		Build: []*model.BuildCommand{
			{PlanetID: 1, Building: "Colony Base"},
		},
	}))
	require.NoError(t, g.Turn(player.ID, model.Commands{
		Build: []*model.BuildCommand{
			{PlanetID: 1, Building: "Trade Goods"},
		},
	}))

	colonies, err := g.GetColonyList(player.ID)
	require.NoError(t, err)
	require.NotEmpty(t, colonies)

	assert.False(t, colonies[0].IsBuilding)
	assert.Equal(t, "Trade Goods", colonies[0].CurrentProject)
	assert.Equal(t, int64(0), colonies[0].BuildBucket)
	assert.Greater(t, colonies[0].Income.TradeGoods, int64(0))
}
