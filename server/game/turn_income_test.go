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

func TestTurnSubtractsNegativeIncomeFromTreasury(t *testing.T) {
	g := &Game{
		Players: []*model.Player{
			{
				ID:       1,
				Treasury: 10,
			},
		},
		Resources: &Resources{
			BuildingCollection: &BuildingCollection{
				Buildings: []model.Building{
					{Name: "High Maintenance Building", Maintenance: 4},
				},
			},
		},
		Galaxy: model.Galaxy{
			Stars: []*model.Star{
				{
					Planets: []*model.Planet{
						{
							ID:          1,
							Owner:       1,
							Population:  1000,
							Size:        model.PlanetSizeMedium,
							Buildings:   []string{"High Maintenance Building"},
							IsBuilding:  true,
							BuildBucket: 100,
						},
					},
				},
			},
		},
	}

	require.NoError(t, g.Turn(1, model.Commands{}))

	colonies, err := g.GetColonyList(1)
	require.NoError(t, err)
	require.Len(t, colonies, 1)
	assert.Equal(t, int64(-3), colonies[0].Income.Total)
	assert.Equal(t, int64(7), g.Players[0].Treasury)
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

func TestTurnStoresBuildQueueAndStartsFirstProject(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, "Ada", 4, model.DefaultPlayerColor())
	require.NoError(t, err)

	player := g.Players[0]
	player.Available.Buildings = append(player.Available.Buildings, "Automated Factory")

	require.NoError(t, g.Turn(player.ID, model.Commands{
		Build: []*model.BuildCommand{
			{
				PlanetID:   1,
				Building:   "Colony Base",
				BuildQueue: []string{"Colony Base", "Automated Factory"},
			},
		},
	}))

	colonies, err := g.GetColonyList(player.ID)
	require.NoError(t, err)
	require.NotEmpty(t, colonies)

	assert.Equal(t, "Colony Base", colonies[0].CurrentProject)
	assert.Equal(t, []string{"Automated Factory"}, colonies[0].BuildQueue)
}

func TestTurnStartsNextQueuedProjectAfterCompletion(t *testing.T) {
	g := &Game{
		Players: []*model.Player{
			{
				ID: 1,
				Available: model.AvailableTech{
					Buildings: []string{"Cheap Building", "Next Building"},
				},
			},
		},
		Resources: &Resources{
			BuildingCollection: &BuildingCollection{
				Buildings: []model.Building{
					{Name: "Cheap Building", Cost: 1},
					{Name: "Next Building", Cost: 10},
				},
			},
		},
		Galaxy: model.Galaxy{
			Stars: []*model.Star{
				{
					Planets: []*model.Planet{
						{
							ID:             1,
							Owner:          1,
							Population:     1000,
							Size:           model.PlanetSizeMedium,
							MineralClass:   model.PlanetMineralAbundant,
							CurrentProject: "Cheap Building",
							IsBuilding:     true,
							BuildBucket:    1,
							BuildQueue:     []string{"Next Building"},
							Industry: model.PlanetIndustryOutput{
								Workers:           1,
								ProducedPerWorker: 1,
							},
						},
					},
				},
			},
		},
	}

	require.NoError(t, g.Turn(1, model.Commands{}))

	colonies, err := g.GetColonyList(1)
	require.NoError(t, err)
	require.Len(t, colonies, 1)
	assert.Contains(t, colonies[0].Buildings, "Cheap Building")
	assert.Equal(t, "Next Building", colonies[0].CurrentProject)
	assert.Empty(t, colonies[0].BuildQueue)
	assert.True(t, colonies[0].IsBuilding)
}

func TestTurnCanUpdateQueueWithoutSwitchingCurrentProject(t *testing.T) {
	g := &Game{
		Players: []*model.Player{
			{
				ID: 1,
				Available: model.AvailableTech{
					Buildings: []string{"Current Building", "Queued Building"},
				},
			},
		},
		Resources: &Resources{
			BuildingCollection: &BuildingCollection{
				Buildings: []model.Building{
					{Name: "Current Building", Cost: 20},
					{Name: "Queued Building", Cost: 10},
				},
			},
		},
		Galaxy: model.Galaxy{
			Stars: []*model.Star{
				{
					Planets: []*model.Planet{
						{
							ID:             1,
							Owner:          1,
							Population:     1000,
							Size:           model.PlanetSizeMedium,
							MineralClass:   model.PlanetMineralAbundant,
							CurrentProject: "Current Building",
							IsBuilding:     true,
							BuildBucket:    20,
							Industry: model.PlanetIndustryOutput{
								Workers:           1,
								ProducedPerWorker: 1,
							},
						},
					},
				},
			},
		},
	}

	require.NoError(t, g.Turn(1, model.Commands{
		Build: []*model.BuildCommand{
			{
				PlanetID:   1,
				Building:   "Current Building",
				BuildQueue: []string{"Queued Building"},
			},
		},
	}))

	colonies, err := g.GetColonyList(1)
	require.NoError(t, err)
	require.Len(t, colonies, 1)
	assert.Equal(t, "Current Building", colonies[0].CurrentProject)
	assert.Equal(t, []string{"Queued Building"}, colonies[0].BuildQueue)
	assert.True(t, colonies[0].IsBuilding)
}
