package game

import (
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func uShips(idxs ...uint) *[]uint {
	s := append([]uint(nil), idxs...)
	return &s
}

func TestTurnAddsIncomeToTreasuryEveryTurn(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: DefaultRace("Human")})
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
	assert.Equal(t, float64(7), g.Players[0].Treasury)
}

func TestTurnStartsConstructionWithBuildingCostAndNoTradeGoods(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: DefaultRace("Human")})
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

func TestTurnUsesPersistedAutobuildForIdleColony(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: DefaultRace("Human")})
	require.NoError(t, err)

	player := g.Players[0]
	player.Available.Buildings = append(player.Available.Buildings, "Automated Factory")
	require.NoError(t, g.SetPlanetAutobuild(player.ID, 1, true))

	require.NoError(t, g.Turn(player.ID, model.Commands{}))

	colonies, err := g.GetColonyList(player.ID)
	require.NoError(t, err)
	require.NotEmpty(t, colonies)
	assert.True(t, colonies[0].Autobuild)
	assert.True(t, colonies[0].IsBuilding)
	assert.Equal(t, "Automated Factory", colonies[0].CurrentProject)
}

func TestTurnContinuesExistingConstructionWithoutRepeatedCommand(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: DefaultRace("Human")})
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

	g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: DefaultRace("Human")})
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

func TestMovePopulationOnlyAppliesToCommandPlanet(t *testing.T) {
	g := &Game{
		Players: []*model.Player{
			{ID: 1},
		},
		Resources: &Resources{},
		Galaxy: model.Galaxy{
			Stars: []*model.Star{
				{
					Planets: []*model.Planet{
						{
							ID:           1,
							Owner:        1,
							Population:   2000,
							Size:         model.PlanetSizeMedium,
							MineralClass: model.PlanetMineralAbundant,
							Environment:  model.PlanetEnvironmentTerran,
							Food: model.PlanetFoodOutput{
								Farmers: 2,
							},
						},
						{
							ID:           2,
							Owner:        1,
							Population:   2000,
							Size:         model.PlanetSizeMedium,
							MineralClass: model.PlanetMineralAbundant,
							Environment:  model.PlanetEnvironmentTerran,
							Industry: model.PlanetIndustryOutput{
								Workers: 2,
							},
						},
					},
				},
			},
		},
	}

	require.NoError(t, g.Turn(1, model.Commands{
		MovePopulation: []*model.MovePopulationCommand{{
			PlanetID:   2,
			Farmers:    1,
			Workers:    1,
			Scientists: 0,
		}},
	}))

	colonies, err := g.GetColonyList(1)
	require.NoError(t, err)
	require.Len(t, colonies, 2)
	assert.Equal(t, int64(2), colonies[0].Food.Farmers)
	assert.Equal(t, int64(0), colonies[0].Industry.Workers)
	assert.Equal(t, int64(0), colonies[0].Science.Scientists)
	assert.Equal(t, int64(1), colonies[1].Food.Farmers)
	assert.Equal(t, int64(1), colonies[1].Industry.Workers)
	assert.Equal(t, int64(0), colonies[1].Science.Scientists)
}

func TestMovePopulationCanApplyToMultipleColonies(t *testing.T) {
	g := &Game{
		Players: []*model.Player{
			{ID: 1},
		},
		Resources: &Resources{},
		Galaxy: model.Galaxy{
			Stars: []*model.Star{
				{
					Planets: []*model.Planet{
						{
							ID:           1,
							Owner:        1,
							Population:   3000,
							Size:         model.PlanetSizeMedium,
							MineralClass: model.PlanetMineralAbundant,
							Environment:  model.PlanetEnvironmentTerran,
							Food: model.PlanetFoodOutput{
								Farmers: 3,
							},
						},
						{
							ID:           2,
							Owner:        1,
							Population:   3000,
							Size:         model.PlanetSizeMedium,
							MineralClass: model.PlanetMineralAbundant,
							Environment:  model.PlanetEnvironmentTerran,
							Industry: model.PlanetIndustryOutput{
								Workers: 3,
							},
						},
					},
				},
			},
		},
	}

	require.NoError(t, g.Turn(1, model.Commands{
		MovePopulation: []*model.MovePopulationCommand{
			{PlanetID: 1, Farmers: 2, Workers: 1, Scientists: 0},
			{PlanetID: 2, Farmers: 1, Workers: 1, Scientists: 1},
		},
	}))

	colonies, err := g.GetColonyList(1)
	require.NoError(t, err)
	require.Len(t, colonies, 2)
	assert.Equal(t, int64(2), colonies[0].Food.Farmers)
	assert.Equal(t, int64(1), colonies[0].Industry.Workers)
	assert.Equal(t, int64(0), colonies[0].Science.Scientists)
	assert.Equal(t, int64(1), colonies[1].Food.Farmers)
	assert.Equal(t, int64(1), colonies[1].Industry.Workers)
	assert.Equal(t, int64(1), colonies[1].Science.Scientists)
}

func TestTurnStoresBuildQueueAndStartsFirstProject(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: DefaultRace("Human")})
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
						{
							ID:           2,
							Owner:        0,
							Environment:  model.PlanetEnvironmentTerran,
							Size:         model.PlanetSizeMedium,
							MineralClass: model.PlanetMineralAbundant,
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
	assert.Equal(t, int64(8), colonies[0].BuildBucket)
}

func TestTurnCarriesOverflowProductionThroughBuildQueue(t *testing.T) {
	g := &Game{
		Players: []*model.Player{
			{
				ID: 1,
				Available: model.AvailableTech{
					Buildings: []string{"Cheap Building", "Next Building", "Final Building"},
				},
			},
		},
		Resources: &Resources{
			BuildingCollection: &BuildingCollection{
				Buildings: []model.Building{
					{Name: "Cheap Building", Cost: 1},
					{Name: "Next Building", Cost: 2},
					{Name: "Final Building", Cost: 10},
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
							BuildQueue:     []string{"Next Building", "Final Building"},
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
	assert.Contains(t, colonies[0].Buildings, "Next Building")
	assert.Equal(t, "Final Building", colonies[0].CurrentProject)
	assert.Empty(t, colonies[0].BuildQueue)
	assert.True(t, colonies[0].IsBuilding)
	assert.Equal(t, int64(10), colonies[0].BuildBucket)
}

func TestTurnDoesNotClearQueueWhenSwitchingToTradeGoods(t *testing.T) {
	g := &Game{
		Players: []*model.Player{
			{
				ID: 1,
				Available: model.AvailableTech{
					Buildings: []string{"Queued Building"},
				},
			},
		},
		Resources: &Resources{
			BuildingCollection: &BuildingCollection{
				Buildings: []model.Building{
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
							CurrentProject: "Trade Goods",
							BuildQueue:     []string{"Queued Building"},
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
			{PlanetID: 1, Building: "Trade Goods"},
		},
	}))

	colonies, err := g.GetColonyList(1)
	require.NoError(t, err)
	require.Len(t, colonies, 1)
	assert.Equal(t, "Trade Goods", colonies[0].CurrentProject)
	assert.Equal(t, []string{"Queued Building"}, colonies[0].BuildQueue)
	assert.False(t, colonies[0].IsBuilding)
}

func TestCompletedColonyBaseCreatesPendingBaseWithoutBuiltBuilding(t *testing.T) {
	g := &Game{
		Players: []*model.Player{
			{
				ID: 1,
				Available: model.AvailableTech{
					Buildings: []string{"Colony Base"},
				},
			},
		},
		Resources: &Resources{
			BuildingCollection: &BuildingCollection{
				Buildings: []model.Building{
					{Name: "Colony Base", Cost: 1},
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
							CurrentProject: "Colony Base",
							IsBuilding:     true,
							BuildBucket:    1,
							Industry: model.PlanetIndustryOutput{
								Workers:           1,
								ProducedPerWorker: 1,
							},
						},
						{
							ID:           2,
							Owner:        0,
							Environment:  model.PlanetEnvironmentTerran,
							Size:         model.PlanetSizeMedium,
							MineralClass: model.PlanetMineralAbundant,
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
	assert.NotContains(t, colonies[0].Buildings, "Colony Base")
	assert.Equal(t, []uint{1}, g.PendingColonyBases[1])
}

func TestTurnClearsPendingColonyBaseWhenNoColonizablePlanetsRemain(t *testing.T) {
	g := &Game{
		Players: []*model.Player{
			{ID: 1},
		},
		Resources:          &Resources{},
		PendingColonyBases: map[uint][]uint{1: []uint{1}},
		Galaxy: model.Galaxy{
			Stars: []*model.Star{
				{
					Planets: []*model.Planet{
						{
							ID:           1,
							Owner:        1,
							Population:   1000,
							Size:         model.PlanetSizeMedium,
							MineralClass: model.PlanetMineralAbundant,
						},
						{
							ID:           2,
							Owner:        0,
							Environment:  model.PlanetEnvironmentGasGiant,
							Size:         model.PlanetSizeHuge,
							MineralClass: model.PlanetMineralAbundant,
						},
					},
				},
			},
		},
	}

	require.NoError(t, g.Turn(1, model.Commands{}))

	assert.Empty(t, g.PendingColonyBases[1])
}

func TestTurnClearsPendingColonyBaseWhenOnlyOtherSystemsHaveColonizablePlanets(t *testing.T) {
	g := &Game{
		Players: []*model.Player{
			{ID: 1},
		},
		Resources:          &Resources{},
		PendingColonyBases: map[uint][]uint{1: []uint{1}},
		Galaxy: model.Galaxy{
			Stars: []*model.Star{
				{
					ID: 1,
					Planets: []*model.Planet{
						{
							ID:           1,
							Owner:        1,
							ParentStar:   1,
							Population:   1000,
							Size:         model.PlanetSizeMedium,
							MineralClass: model.PlanetMineralAbundant,
						},
						{
							ID:           2,
							Owner:        0,
							ParentStar:   1,
							Environment:  model.PlanetEnvironmentGasGiant,
							Size:         model.PlanetSizeHuge,
							MineralClass: model.PlanetMineralAbundant,
						},
					},
				},
				{
					ID: 2,
					Planets: []*model.Planet{
						{
							ID:           3,
							Owner:        0,
							ParentStar:   2,
							Environment:  model.PlanetEnvironmentTerran,
							Size:         model.PlanetSizeMedium,
							MineralClass: model.PlanetMineralAbundant,
						},
					},
				},
			},
		},
	}

	require.NoError(t, g.Turn(1, model.Commands{}))

	assert.Empty(t, g.PendingColonyBases[1])
}

func TestColonyBaseCannotColonizeDifferentSystem(t *testing.T) {
	g := &Game{
		Players: []*model.Player{
			{ID: 1},
		},
		Resources: &Resources{},
		Galaxy: model.Galaxy{
			Stars: []*model.Star{
				{
					ID: 1,
					Planets: []*model.Planet{
						{
							ID:           1,
							Owner:        1,
							ParentStar:   1,
							Population:   1000,
							Size:         model.PlanetSizeMedium,
							MineralClass: model.PlanetMineralAbundant,
						},
					},
				},
				{
					ID: 2,
					Planets: []*model.Planet{
						{
							ID:           2,
							Owner:        0,
							ParentStar:   2,
							Environment:  model.PlanetEnvironmentTerran,
							Size:         model.PlanetSizeMedium,
							MineralClass: model.PlanetMineralAbundant,
						},
					},
				},
			},
		},
	}

	err := g.ColonizePlanet(1, 1, 2)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "same system")
}

func TestLegacyBuiltColonyBaseDoesNotBlockFutureColonyBases(t *testing.T) {
	g := &Game{
		Players: []*model.Player{
			{
				ID: 1,
				Available: model.AvailableTech{
					Buildings: []string{"Colony Base"},
				},
			},
		},
		Resources: &Resources{
			BuildingCollection: &BuildingCollection{
				Buildings: []model.Building{
					{Name: "Colony Base", Cost: 10},
				},
			},
		},
		Galaxy: model.Galaxy{
			Stars: []*model.Star{
				{
					Planets: []*model.Planet{
						{
							ID:           1,
							Owner:        1,
							Population:   1000,
							Size:         model.PlanetSizeMedium,
							MineralClass: model.PlanetMineralAbundant,
							Buildings:    []string{"Colony Base"},
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
			{PlanetID: 1, Building: "Colony Base"},
		},
	}))

	colonies, err := g.GetColonyList(1)
	require.NoError(t, err)
	require.Len(t, colonies, 1)
	assert.Equal(t, "Colony Base", colonies[0].CurrentProject)
	assert.True(t, colonies[0].IsBuilding)
	assert.Greater(t, colonies[0].BuildBucket, int64(0))
	assert.Less(t, colonies[0].BuildBucket, int64(10))
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

func TestTurnCanReplaceCurrentProjectAndKeepInvestedProduction(t *testing.T) {
	g := &Game{
		Players: []*model.Player{
			{
				ID: 1,
				Available: model.AvailableTech{
					Buildings: []string{"Current Building", "Replacement Building", "Queued Building"},
				},
			},
		},
		Resources: &Resources{
			BuildingCollection: &BuildingCollection{
				Buildings: []model.Building{
					{Name: "Current Building", Cost: 20},
					{Name: "Replacement Building", Cost: 30},
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
							BuildBucket:    15,
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
				Building:   "Replacement Building",
				BuildQueue: []string{"Queued Building"},
			},
		},
	}))

	colonies, err := g.GetColonyList(1)
	require.NoError(t, err)
	require.Len(t, colonies, 1)
	assert.Equal(t, "Replacement Building", colonies[0].CurrentProject)
	assert.Equal(t, int64(22), colonies[0].BuildBucket)
	assert.Equal(t, []string{"Queued Building"}, colonies[0].BuildQueue)
	assert.True(t, colonies[0].IsBuilding)
}

func TestTurnStacksReplacementProductionThroughCheapQueue(t *testing.T) {
	g := &Game{
		Players: []*model.Player{
			{
				ID: 1,
				Available: model.AvailableTech{
					Buildings: []string{
						"Expensive Building",
						"Cheap One",
						"Cheap Two",
						"Cheap Three",
						"Cheap Four",
					},
				},
			},
		},
		Resources: &Resources{
			BuildingCollection: &BuildingCollection{
				Buildings: []model.Building{
					{Name: "Expensive Building", Cost: 1000},
					{Name: "Cheap One", Cost: 100},
					{Name: "Cheap Two", Cost: 100},
					{Name: "Cheap Three", Cost: 100},
					{Name: "Cheap Four", Cost: 100},
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
							CurrentProject: "Expensive Building",
							IsBuilding:     true,
							BuildBucket:    400,
						},
					},
				},
			},
		},
	}

	require.NoError(t, g.Turn(1, model.Commands{
		Build: []*model.BuildCommand{
			{
				PlanetID: 1,
				Building: "Cheap One",
				BuildQueue: []string{
					"Cheap Two",
					"Cheap Three",
					"Cheap Four",
				},
			},
		},
	}))

	colonies, err := g.GetColonyList(1)
	require.NoError(t, err)
	require.Len(t, colonies, 1)
	assert.Contains(t, colonies[0].Buildings, "Cheap One")
	assert.Contains(t, colonies[0].Buildings, "Cheap Two")
	assert.Contains(t, colonies[0].Buildings, "Cheap Three")
	assert.Contains(t, colonies[0].Buildings, "Cheap Four")
	assert.False(t, colonies[0].IsBuilding)
	assert.Empty(t, colonies[0].BuildQueue)
	assert.Equal(t, int64(200), colonies[0].IndustryStack)
}

func TestTurnKeepsNextQueuedProjectWhenCompletedTopWasSentWithFullQueue(t *testing.T) {
	g := &Game{
		Players: []*model.Player{
			{
				ID: 1,
				Available: model.AvailableTech{
					Buildings: []string{"Cheap One", "Cheap Two", "Cheap Three"},
				},
			},
		},
		Resources: &Resources{
			BuildingCollection: &BuildingCollection{
				Buildings: []model.Building{
					{Name: "Cheap One", Cost: 100},
					{Name: "Cheap Two", Cost: 100},
					{Name: "Cheap Three", Cost: 100},
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
							CurrentProject: "Cheap One",
							IsBuilding:     true,
							BuildBucket:    1,
							IndustryStack:  1,
						},
					},
				},
			},
		},
	}

	require.NoError(t, g.Turn(1, model.Commands{
		Build: []*model.BuildCommand{
			{
				PlanetID: 1,
				Building: "Cheap One",
				BuildQueue: []string{
					"Cheap One",
					"Cheap Two",
					"Cheap Three",
				},
			},
		},
	}))

	colonies, err := g.GetColonyList(1)
	require.NoError(t, err)
	require.Len(t, colonies, 1)
	assert.Contains(t, colonies[0].Buildings, "Cheap One")
	assert.Equal(t, "Cheap Two", colonies[0].CurrentProject)
	assert.Equal(t, []string{"Cheap Three"}, colonies[0].BuildQueue)
	assert.True(t, colonies[0].IsBuilding)
}

func TestTurnKeepsOldCurrentProjectWhenReorderedBehindNewTop(t *testing.T) {
	g := &Game{
		Players: []*model.Player{
			{
				ID: 1,
				Available: model.AvailableTech{
					Buildings: []string{"Current Building", "Next Building"},
				},
			},
		},
		Resources: &Resources{
			BuildingCollection: &BuildingCollection{
				Buildings: []model.Building{
					{Name: "Current Building", Cost: 100},
					{Name: "Next Building", Cost: 100},
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
							BuildBucket:    1,
							IndustryStack:  1,
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
				Building:   "Next Building",
				BuildQueue: []string{"Current Building"},
			},
		},
	}))

	colonies, err := g.GetColonyList(1)
	require.NoError(t, err)
	require.Len(t, colonies, 1)
	assert.Contains(t, colonies[0].Buildings, "Next Building")
	assert.Equal(t, "Current Building", colonies[0].CurrentProject)
	assert.Empty(t, colonies[0].BuildQueue)
	assert.True(t, colonies[0].IsBuilding)
}

func TestBuildRejectsUniqueBuildingWhenOwnedElsewhere(t *testing.T) {
	g := &Game{
		Resources: &Resources{
			BuildingCollection: &BuildingCollection{
				Buildings: []model.Building{
					{Name: "Capitol", Cost: 100, Unique: true},
				},
			},
		},
		Galaxy: model.Galaxy{
			Stars: []*model.Star{
				{
					Planets: []*model.Planet{
						{ID: 1, Owner: 1, Buildings: []string{"Capitol"}},
						{ID: 2, Owner: 1},
					},
				},
			},
		},
	}
	target := g.Galaxy.Stars[0].Planets[1]

	built := Build(target, "Capitol", []string{"Capitol"}, g)

	assert.False(t, built)
	assert.False(t, target.IsBuilding)
	assert.Empty(t, target.CurrentProject)
}

func TestBuildAllowsUniqueBuildingAfterItIsRemoved(t *testing.T) {
	g := &Game{
		Resources: &Resources{
			BuildingCollection: &BuildingCollection{
				Buildings: []model.Building{
					{Name: "Capitol", Cost: 100, Unique: true},
				},
			},
		},
		Galaxy: model.Galaxy{
			Stars: []*model.Star{
				{
					Planets: []*model.Planet{
						{ID: 1, Owner: 1},
						{ID: 2, Owner: 1},
					},
				},
			},
		},
	}
	target := g.Galaxy.Stars[0].Planets[1]

	built := Build(target, "Capitol", []string{"Capitol"}, g)

	assert.True(t, built)
	assert.True(t, target.IsBuilding)
	assert.Equal(t, "Capitol", target.CurrentProject)
	assert.Equal(t, int64(100), target.BuildBucket)
}

func TestBuildUpgradeUsesLowerTierCostAsCredit(t *testing.T) {
	g := &Game{
		Resources: &Resources{
			BuildingCollection: &BuildingCollection{
				Buildings: []model.Building{
					{Name: "Planetary Radiation Shield", Cost: 80},
					{Name: "Planetary Flux Shield", Cost: 200, Upgrades: []string{"Planetary Radiation Shield"}},
					{Name: "Planetary Barrier Shield", Cost: 500, Upgrades: []string{"Planetary Radiation Shield", "Planetary Flux Shield"}},
				},
			},
		},
	}
	planet := &model.Planet{Buildings: []string{"Planetary Radiation Shield"}}

	assert.True(t, Build(planet, "Planetary Flux Shield", []string{"Planetary Flux Shield"}, g))
	assert.Equal(t, int64(120), planet.BuildBucket)

	assert.True(t, Build(planet, "Planetary Barrier Shield", []string{"Planetary Barrier Shield"}, g))
	assert.Equal(t, int64(420), planet.BuildBucket)
}

func TestBuildRejectsLowerTierWhenSuperiorBuildingExists(t *testing.T) {
	g := &Game{
		Resources: &Resources{
			BuildingCollection: &BuildingCollection{
				Buildings: []model.Building{
					{Name: "Planetary Radiation Shield", Cost: 80},
					{Name: "Planetary Flux Shield", Cost: 200, Upgrades: []string{"Planetary Radiation Shield"}},
				},
			},
		},
	}
	planet := &model.Planet{Buildings: []string{"Planetary Flux Shield"}}

	assert.False(t, Build(planet, "Planetary Radiation Shield", []string{"Planetary Radiation Shield"}, g))
}

func TestCompletingBuildingUpgradeRemovesLowerTier(t *testing.T) {
	g := &Game{
		Players: []*model.Player{
			{
				ID: 1,
				Available: model.AvailableTech{
					Buildings: []string{"Planetary Radiation Shield", "Planetary Flux Shield"},
				},
			},
		},
		Resources: &Resources{
			BuildingCollection: &BuildingCollection{
				Buildings: []model.Building{
					{Name: "Planetary Radiation Shield", Cost: 80},
					{Name: "Planetary Flux Shield", Cost: 200, Upgrades: []string{"Planetary Radiation Shield"}},
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
							Buildings:      []string{"Planetary Radiation Shield"},
							CurrentProject: "Planetary Flux Shield",
							IsBuilding:     true,
							BuildBucket:    1,
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
	assert.NotContains(t, colonies[0].Buildings, "Planetary Radiation Shield")
	assert.Contains(t, colonies[0].Buildings, "Planetary Flux Shield")
}

func TestTurnCompletingColonyShipCreatesOrbitFleet(t *testing.T) {
	g := &Game{
		Players: []*model.Player{
			{
				ID: 1,
				Available: model.AvailableTech{
					Buildings: []string{"Colony Ship"},
				},
			},
		},
		Resources: &Resources{
			BuildingCollection: &BuildingCollection{
				Buildings: []model.Building{
					{Name: "Colony Ship", Cost: 0},
				},
			},
		},
		Galaxy: model.Galaxy{
			Stars: []*model.Star{
				{
					ID: 1,
					Planets: []*model.Planet{
						{
							ID:         1,
							Owner:      1,
							ParentStar: 1,
							Population: 1000,
							Size:       model.PlanetSizeMedium,
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
			{PlanetID: 1, Building: "Colony Ship"},
		},
	}))

	require.Len(t, g.Fleets, 1)
	assert.Equal(t, uint(1), g.Fleets[0].Owner)
	assert.Equal(t, uint(1), g.Fleets[0].StarID)
	require.Len(t, g.Fleets[0].Ships, 1)
	assert.Equal(t, "Colony Ship", g.Fleets[0].Ships[0].Name)
	assert.Equal(t, []uint{g.Fleets[0].ID}, g.Players[0].Fleets)
}

func TestColonyShipCommandQueuesPendingFleet(t *testing.T) {
	g := &Game{
		Players:   []*model.Player{{ID: 1, Fleets: []uint{1}}},
		Resources: &Resources{},
		Fleets: []*model.Fleet{
			{
				ID:     1,
				Owner:  1,
				StarID: 1,
				Ships:  []model.FleetShip{{Type: "colony_ship", Name: "Colony Ship"}},
			},
		},
		Galaxy: model.Galaxy{
			Stars: []*model.Star{
				{
					ID: 1,
					Planets: []*model.Planet{
						{ID: 1, Owner: 1, ParentStar: 1, Population: 1000},
						{ID: 2, Owner: 0, ParentStar: 1, Environment: model.PlanetEnvironmentTerran},
					},
				},
			},
		},
	}

	require.NoError(t, g.Turn(1, model.Commands{
		ColonyShip: &model.PrepareColonyShipCommand{FleetID: 1},
	}))

	assert.Equal(t, []uint{1}, g.PendingColonyShips[1])
	require.Len(t, g.Fleets, 1)
	require.Len(t, g.Fleets[0].Ships, 1)
}

func TestPendingColonyShipColonizesAndConsumesOneShip(t *testing.T) {
	g := &Game{
		Players:   []*model.Player{{ID: 1, Fleets: []uint{1}}},
		Resources: &Resources{},
		Fleets: []*model.Fleet{
			{
				ID:     1,
				Owner:  1,
				StarID: 1,
				Ships: []model.FleetShip{
					{Type: "colony_ship", Name: "Colony Ship"},
					{Type: "frigate", Name: "Guard"},
				},
			},
		},
		Galaxy: model.Galaxy{
			Stars: []*model.Star{
				{
					ID: 1,
					Planets: []*model.Planet{
						{ID: 1, Owner: 1, ParentStar: 1, Population: 1000},
						{ID: 2, Owner: 0, ParentStar: 1, Environment: model.PlanetEnvironmentTerran},
					},
				},
			},
		},
	}

	require.NoError(t, g.Turn(1, model.Commands{
		Colonize: &model.ColonizeCommand{FleetID: 1, PlanetID: 2},
	}))

	planet, err := g.getPlanet(2)
	require.NoError(t, err)
	assert.Equal(t, uint(1), planet.Owner)
	require.Len(t, g.Fleets, 1)
	require.Len(t, g.Fleets[0].Ships, 1)
	assert.Equal(t, "Guard", g.Fleets[0].Ships[0].Name)
	assert.Empty(t, g.PendingColonyShips[1])
}

func TestColonyShipColonizeWithStaleFleetIDUsesOrbitingShipInTargetSystem(t *testing.T) {
	g := &Game{
		Players:   []*model.Player{{ID: 1, Fleets: []uint{7}}},
		Resources: &Resources{},
		Fleets: []*model.Fleet{
			{
				ID:     7,
				Owner:  1,
				StarID: 1,
				Ships:  []model.FleetShip{{Type: "colony_ship", Name: "Colony Ship"}},
			},
		},
		Galaxy: model.Galaxy{
			Stars: []*model.Star{
				{
					ID: 1,
					Planets: []*model.Planet{
						{ID: 1, Owner: 1, ParentStar: 1, Population: 1000},
						{ID: 2, Owner: 0, ParentStar: 1, Environment: model.PlanetEnvironmentTerran},
					},
				},
			},
		},
	}

	require.NoError(t, g.Turn(1, model.Commands{
		Colonize: &model.ColonizeCommand{FleetID: 1, PlanetID: 2},
	}))

	planet, err := g.getPlanet(2)
	require.NoError(t, err)
	assert.Equal(t, uint(1), planet.Owner)
	assert.Empty(t, g.Fleets)
	assert.Empty(t, g.Players[0].Fleets)
}

func TestMultipleColonyShipsCanColonizeInSameTurn(t *testing.T) {
	g := &Game{
		Players:   []*model.Player{{ID: 1, Fleets: []uint{1, 2}}},
		Resources: &Resources{},
		Fleets: []*model.Fleet{
			{
				ID:     1,
				Owner:  1,
				StarID: 1,
				Ships:  []model.FleetShip{{Type: "colony_ship", Name: "Colony Ship A"}},
			},
			{
				ID:     2,
				Owner:  1,
				StarID: 2,
				Ships:  []model.FleetShip{{Type: "colony_ship", Name: "Colony Ship B"}},
			},
		},
		Galaxy: model.Galaxy{
			Stars: []*model.Star{
				{
					ID: 1,
					Planets: []*model.Planet{
						{ID: 1, Owner: 1, ParentStar: 1, Population: 1000},
						{ID: 2, Owner: 0, ParentStar: 1, Environment: model.PlanetEnvironmentTerran},
					},
				},
				{
					ID: 2,
					Planets: []*model.Planet{
						{ID: 3, Owner: 0, ParentStar: 2, Environment: model.PlanetEnvironmentTerran},
					},
				},
			},
		},
	}

	require.NoError(t, g.Turn(1, model.Commands{
		ColonizeMany: []*model.ColonizeCommand{
			{FleetID: 1, PlanetID: 2},
			{FleetID: 2, PlanetID: 3},
		},
	}))

	first, err := g.getPlanet(2)
	require.NoError(t, err)
	second, err := g.getPlanet(3)
	require.NoError(t, err)
	assert.Equal(t, uint(1), first.Owner)
	assert.Equal(t, uint(1), second.Owner)
	assert.Empty(t, g.Fleets)
}

func TestColonyShipCommandWithoutEligiblePlanetKeepsShip(t *testing.T) {
	g := &Game{
		Players:   []*model.Player{{ID: 1, Fleets: []uint{1}}},
		Resources: &Resources{},
		Fleets: []*model.Fleet{
			{
				ID:     1,
				Owner:  1,
				StarID: 1,
				Ships:  []model.FleetShip{{Type: "colony_ship", Name: "Colony Ship"}},
			},
		},
		Galaxy: model.Galaxy{
			Stars: []*model.Star{
				{
					ID: 1,
					Planets: []*model.Planet{
						{ID: 1, Owner: 1, ParentStar: 1, Population: 1000},
						{ID: 2, Owner: 0, ParentStar: 1, Environment: model.PlanetEnvironmentGasGiant},
					},
				},
			},
		},
	}

	require.NoError(t, g.Turn(1, model.Commands{
		ColonyShip: &model.PrepareColonyShipCommand{FleetID: 1},
	}))

	assert.Empty(t, g.PendingColonyShips[1])
	require.Len(t, g.Fleets, 1)
	require.Len(t, g.Fleets[0].Ships, 1)
}

func TestTurnCompletingMultipleColonyShipsReusesSystemFleet(t *testing.T) {
	g := &Game{
		Players: []*model.Player{
			{
				ID: 1,
				Available: model.AvailableTech{
					Buildings: []string{"Colony Ship"},
				},
			},
		},
		Resources: &Resources{
			BuildingCollection: &BuildingCollection{
				Buildings: []model.Building{
					{Name: "Colony Ship", Cost: 0},
				},
			},
		},
		Galaxy: model.Galaxy{
			Stars: []*model.Star{
				{
					ID: 1,
					Planets: []*model.Planet{
						{
							ID:         1,
							Owner:      1,
							ParentStar: 1,
							Population: 1000,
							Size:       model.PlanetSizeMedium,
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

	for i := 0; i < 2; i++ {
		require.NoError(t, g.Turn(1, model.Commands{
			Build: []*model.BuildCommand{
				{PlanetID: 1, Building: "Colony Ship"},
			},
		}))
	}

	require.Len(t, g.Fleets, 1)
	require.Len(t, g.Fleets[0].Ships, 2)
	assert.Equal(t, []uint{g.Fleets[0].ID}, g.Players[0].Fleets)
}

func TestTurnQueuesFleetMoveCommand(t *testing.T) {
	g := &Game{
		Players: []*model.Player{
			{ID: 1},
		},
		Fleets: []*model.Fleet{
			{
				ID:     1,
				Owner:  1,
				StarID: 1,
				Ships: []model.FleetShip{
					{Type: "colony_ship", Name: "Colony Ship"},
				},
			},
		},
		Galaxy: model.Galaxy{
			Stars: []*model.Star{
				{ID: 1, X: 10, Y: 10, Planets: []*model.Planet{}},
				{ID: 2, X: 310, Y: 10, Planets: []*model.Planet{}},
			},
		},
		Resources: &Resources{
			BuildingCollection: &BuildingCollection{
				Buildings: []model.Building{},
			},
		},
	}

	require.NoError(t, g.Turn(1, model.Commands{
		FleetMove: []*model.FleetMoveCommand{
			{FleetID: 1, DestinationStarID: 2, ShipIndexes: uShips(0)},
		},
	}))

	require.Len(t, g.Fleets, 1)
	assert.Equal(t, uint(2), g.Fleets[0].DestinationStarID)
	assert.Greater(t, g.Fleets[0].TurnsRemaining, uint(0))
	assert.Less(t, g.Fleets[0].TurnsRemaining, g.Fleets[0].TotalTurns)
	assert.Greater(t, g.Fleets[0].X, float64(10))
}

func TestFleetArrivalExploresDestinationStar(t *testing.T) {
	g := &Game{
		Players: []*model.Player{
			{ID: 1, ExploredStars: []uint{1}},
		},
		Fleets: []*model.Fleet{
			{
				ID:     1,
				Owner:  1,
				StarID: 1,
				Ships: []model.FleetShip{
					{Type: "colony_ship", Name: "Colony Ship"},
				},
			},
		},
		Galaxy: model.Galaxy{
			Stars: []*model.Star{
				{ID: 1, X: 10, Y: 10, Planets: []*model.Planet{}},
				{ID: 2, X: 80, Y: 10, Planets: []*model.Planet{}},
			},
		},
		Resources: &Resources{
			BuildingCollection: &BuildingCollection{
				Buildings: []model.Building{},
			},
		},
	}

	require.NoError(t, g.Turn(1, model.Commands{
		FleetMove: []*model.FleetMoveCommand{
			{FleetID: 1, DestinationStarID: 2, ShipIndexes: uShips(0)},
		},
	}))

	require.Len(t, g.Fleets, 1)
	assert.Equal(t, uint(2), g.Fleets[0].StarID)
	assert.Equal(t, uint(0), g.Fleets[0].DestinationStarID)
	assert.Contains(t, g.Players[0].ExploredStars, uint(2))
}

func TestTurnRejectsRerouteWithoutHyperspaceCommunications(t *testing.T) {
	g := &Game{
		Players: []*model.Player{
			{ID: 1},
		},
		Fleets: []*model.Fleet{
			{
				ID:                1,
				Owner:             1,
				StarID:            1,
				X:                 100,
				Y:                 100,
				OriginX:           10,
				OriginY:           10,
				DestinationStarID: 2,
				TotalTurns:        5,
				TurnsRemaining:    3,
				Ships:             []model.FleetShip{{Type: "colony_ship", Name: "Colony Ship"}},
			},
		},
		Galaxy: model.Galaxy{
			Stars: []*model.Star{
				{ID: 1, X: 10, Y: 10, Planets: []*model.Planet{}},
				{ID: 2, X: 310, Y: 10, Planets: []*model.Planet{}},
				{ID: 3, X: 610, Y: 10, Planets: []*model.Planet{}},
			},
		},
		Resources: &Resources{BuildingCollection: &BuildingCollection{Buildings: []model.Building{}}},
	}

	err := g.Turn(1, model.Commands{
		FleetMove: []*model.FleetMoveCommand{
			{FleetID: 1, DestinationStarID: 3, ShipIndexes: uShips(0)},
		},
	})

	require.Error(t, err)
}

func TestTurnAllowsRerouteWithHyperspaceCommunications(t *testing.T) {
	g := &Game{
		Players: []*model.Player{
			{ID: 1, ResearchedTechs: []string{"Hyperspace Communications"}},
		},
		Fleets: []*model.Fleet{
			{
				ID:                1,
				Owner:             1,
				StarID:            1,
				X:                 100,
				Y:                 100,
				OriginX:           10,
				OriginY:           10,
				DestinationStarID: 2,
				TotalTurns:        5,
				TurnsRemaining:    3,
				Ships:             []model.FleetShip{{Type: "colony_ship", Name: "Colony Ship"}},
			},
		},
		Galaxy: model.Galaxy{
			Stars: []*model.Star{
				{ID: 1, X: 10, Y: 10, Planets: []*model.Planet{}},
				{ID: 2, X: 310, Y: 10, Planets: []*model.Planet{}},
				{ID: 3, X: 610, Y: 10, Planets: []*model.Planet{}},
			},
		},
		Resources: &Resources{BuildingCollection: &BuildingCollection{Buildings: []model.Building{}}},
	}

	require.NoError(t, g.Turn(1, model.Commands{
		FleetMove: []*model.FleetMoveCommand{
			{FleetID: 1, DestinationStarID: 3, ShipIndexes: uShips(0)},
		},
	}))
	assert.Equal(t, uint(3), g.Fleets[0].DestinationStarID)
}

func TestFleetPartialMoveSplitsOrbitStack(t *testing.T) {
	g := &Game{
		Players: []*model.Player{
			{ID: 1},
		},
		Fleets: []*model.Fleet{
			{
				ID:     1,
				Owner:  1,
				StarID: 1,
				X:      10,
				Y:      10,
				Ships: []model.FleetShip{
					{Type: "colony_ship", Name: "Colony Ship A"},
					{Type: "colony_ship", Name: "Colony Ship B"},
				},
			},
		},
		Galaxy: model.Galaxy{
			Stars: []*model.Star{
				{ID: 1, X: 10, Y: 10, Planets: []*model.Planet{}},
				{ID: 2, X: 310, Y: 10, Planets: []*model.Planet{}},
			},
		},
		Resources: &Resources{BuildingCollection: &BuildingCollection{Buildings: []model.Building{}}},
	}
	g.Players[0].Fleets = []uint{1}

	require.NoError(t, g.Turn(1, model.Commands{
		FleetMove: []*model.FleetMoveCommand{
			{FleetID: 1, DestinationStarID: 2, ShipIndexes: uShips(0)},
		},
	}))

	require.Len(t, g.Fleets, 2)
	var orbit, moving *model.Fleet
	for _, f := range g.Fleets {
		if f.DestinationStarID == 0 && f.TurnsRemaining == 0 {
			orbit = f
		} else {
			moving = f
		}
	}
	require.NotNil(t, orbit)
	require.NotNil(t, moving)
	require.Len(t, orbit.Ships, 1)
	require.Equal(t, "Colony Ship B", orbit.Ships[0].Name)
	require.Len(t, moving.Ships, 1)
	require.Equal(t, "Colony Ship A", moving.Ships[0].Name)
	require.Equal(t, uint(2), moving.DestinationStarID)
}

func TestFleetMultiplePartialMovesUseOriginalShipIndexes(t *testing.T) {
	g := &Game{
		Players: []*model.Player{
			{ID: 1},
		},
		Fleets: []*model.Fleet{
			{
				ID:     1,
				Owner:  1,
				StarID: 1,
				X:      10,
				Y:      10,
				Ships: []model.FleetShip{
					{Type: "colony_ship", Name: "A"},
					{Type: "colony_ship", Name: "B"},
					{Type: "colony_ship", Name: "C"},
				},
			},
		},
		Galaxy: model.Galaxy{
			Stars: []*model.Star{
				{ID: 1, X: 10, Y: 10, Planets: []*model.Planet{}},
				{ID: 2, X: 310, Y: 10, Planets: []*model.Planet{}},
				{ID: 3, X: 610, Y: 10, Planets: []*model.Planet{}},
			},
		},
		Resources: &Resources{BuildingCollection: &BuildingCollection{Buildings: []model.Building{}}},
	}
	g.Players[0].Fleets = []uint{1}

	require.NoError(t, g.Turn(1, model.Commands{
		FleetMove: []*model.FleetMoveCommand{
			{FleetID: 1, DestinationStarID: 2, ShipIndexes: uShips(0)},
			{FleetID: 1, DestinationStarID: 3, ShipIndexes: uShips(1)},
		},
	}))

	require.Len(t, g.Fleets, 3)
	shipsByDestination := map[uint]string{}
	for _, fleet := range g.Fleets {
		require.Len(t, fleet.Ships, 1)
		shipsByDestination[fleet.DestinationStarID] = fleet.Ships[0].Name
	}
	assert.Equal(t, "C", shipsByDestination[0])
	assert.Equal(t, "A", shipsByDestination[2])
	assert.Equal(t, "B", shipsByDestination[3])
}

func TestFleetRecallWhileMovingMergesOrbitWithoutHyperspace(t *testing.T) {
	g := &Game{
		Players: []*model.Player{
			{ID: 1},
		},
		Fleets: []*model.Fleet{
			{
				ID:                1,
				Owner:             1,
				StarID:            1,
				X:                 100,
				Y:                 10,
				OriginX:           10,
				OriginY:           10,
				DestinationStarID: 2,
				TotalTurns:        5,
				TurnsRemaining:    3,
				Ships: []model.FleetShip{
					{Type: "colony_ship", Name: "Returning"},
				},
			},
		},
		Galaxy: model.Galaxy{
			Stars: []*model.Star{
				{ID: 1, X: 10, Y: 10, Planets: []*model.Planet{}},
				{ID: 2, X: 310, Y: 10, Planets: []*model.Planet{}},
			},
		},
		Resources: &Resources{BuildingCollection: &BuildingCollection{Buildings: []model.Building{}}},
	}
	g.Players[0].Fleets = []uint{1}

	require.NoError(t, g.Turn(1, model.Commands{
		FleetMove: []*model.FleetMoveCommand{
			{FleetID: 1, DestinationStarID: 1, ShipIndexes: uShips(0)},
		},
	}))

	require.Len(t, g.Fleets, 1)
	require.Equal(t, uint(0), g.Fleets[0].DestinationStarID)
	require.Len(t, g.Fleets[0].Ships, 1)
	require.Equal(t, "Returning", g.Fleets[0].Ships[0].Name)
}

func TestFleetArrivalMergesIntoExistingOrbitFleet(t *testing.T) {
	g := &Game{
		Players: []*model.Player{
			{ID: 1, ExploredStars: []uint{1, 2}},
		},
		Fleets: []*model.Fleet{
			{
				ID:     1,
				Owner:  1,
				StarID: 2,
				X:      80,
				Y:      10,
				Ships:  []model.FleetShip{{Type: "frigate", Name: "Guard"}},
			},
			{
				ID:                2,
				Owner:             1,
				StarID:            1,
				X:                 80,
				Y:                 10,
				OriginX:           10,
				OriginY:           10,
				DestinationStarID: 2,
				TotalTurns:        1,
				TurnsRemaining:    1,
				Ships:             []model.FleetShip{{Type: "colony_ship", Name: "Arrival"}},
			},
		},
		Galaxy: model.Galaxy{
			Stars: []*model.Star{
				{ID: 1, X: 10, Y: 10, Planets: []*model.Planet{}},
				{ID: 2, X: 80, Y: 10, Planets: []*model.Planet{}},
			},
		},
		Resources: &Resources{BuildingCollection: &BuildingCollection{Buildings: []model.Building{}}},
	}
	g.Players[0].Fleets = []uint{1, 2}

	require.NoError(t, g.Turn(1, model.Commands{}))

	require.Len(t, g.Fleets, 1)
	require.Len(t, g.Fleets[0].Ships, 2)
}
