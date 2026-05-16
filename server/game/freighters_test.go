package game

import (
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResearchFreightersUnlocksBuildProject(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: DefaultRace("Human")})
	require.NoError(t, err)

	player := g.Players[0]
	g.research(player, "Freighters")

	assert.Contains(t, player.Available.Buildings, "Freighters")
}

func TestCompletedFreightersAddsCapacityWithoutBuildingOrFleet(t *testing.T) {
	g := newFreighterHarness()
	player := g.Players[0]
	planet := g.Galaxy.Stars[0].Planets[0]
	planet.CurrentProject = "Freighters"
	planet.IsBuilding = true
	planet.BuildBucket = 1
	planet.IndustryStack = 1

	require.NoError(t, g.Turn(player.ID, model.Commands{}))

	assert.Equal(t, uint(5), player.Freighters)
	assert.NotContains(t, planet.Buildings, "Freighters")
	assert.Empty(t, g.Fleets)
}

func TestFreightersCanBeBuiltRepeatedly(t *testing.T) {
	g := newFreighterHarness()
	player := g.Players[0]
	planet := g.Galaxy.Stars[0].Planets[0]
	planet.CurrentProject = "Freighters"
	planet.IsBuilding = true
	planet.BuildBucket = 1
	planet.BuildQueue = []string{"Freighters"}
	planet.IndustryStack = 2

	require.NoError(t, g.Turn(player.ID, model.Commands{}))

	assert.Equal(t, uint(5), player.Freighters)
	assert.Equal(t, "Freighters", planet.CurrentProject)
	assert.True(t, planet.IsBuilding)
	assert.NotContains(t, planet.Buildings, "Freighters")
}

func TestAutomaticFoodTransferPreventsStarvationAndChargesHalfBc(t *testing.T) {
	g := newFoodTransferHarness(2)
	player := g.Players[0]
	deficit := g.Galaxy.Stars[0].Planets[1]

	require.NoError(t, g.Turn(player.ID, model.Commands{}))

	assert.Equal(t, int64(44), deficit.LastTurnPopGrowth)
	assert.Equal(t, float64(14), player.Treasury)
	assert.Equal(t, uint(0), player.FreightersUsed)
}

func TestAutomaticFoodTransferIsCappedByAvailableFreighters(t *testing.T) {
	g := newFoodTransferHarness(1)
	player := g.Players[0]
	deficit := g.Galaxy.Stars[0].Planets[1]

	require.NoError(t, g.Turn(player.ID, model.Commands{}))

	assert.Equal(t, int64(-6), deficit.LastTurnPopGrowth)
	assert.Equal(t, float64(13.5), player.Treasury)
}

func TestPopulationTransferReservesFreightersAndArrivesNextTurn(t *testing.T) {
	g := newPopulationTransferHarness()
	player := g.Players[0]
	source := g.Galaxy.Stars[0].Planets[0]
	destination := g.Galaxy.Stars[0].Planets[1]

	require.NoError(t, g.Turn(player.ID, model.Commands{
		TransferPopulation: []*model.TransferPopulationCommand{{
			SourcePlanetID:      source.ID,
			DestinationPlanetID: destination.ID,
			PopulationUnits:     1,
		}},
	}))

	require.Len(t, g.PopulationTransfers, 1)
	assert.Equal(t, int64(4073), source.Population)
	assert.Equal(t, int64(1042), destination.Population)
	assert.Equal(t, uint(5), player.FreightersUsed)
	assert.Equal(t, uint(1), g.PopulationTransfers[0].TurnsRemaining)

	require.NoError(t, g.Turn(player.ID, model.Commands{}))

	assert.Empty(t, g.PopulationTransfers)
	assert.Equal(t, int64(2084), destination.Population)
	assert.Equal(t, uint(0), player.FreightersUsed)
}

func TestPopulationTransferRejectsInvalidCommands(t *testing.T) {
	t.Run("insufficient source population", func(t *testing.T) {
		g := newPopulationTransferHarness()
		source := g.Galaxy.Stars[0].Planets[0]
		destination := g.Galaxy.Stars[0].Planets[1]
		source.Population = 1000

		err := g.Turn(1, model.Commands{TransferPopulation: []*model.TransferPopulationCommand{{
			SourcePlanetID:      source.ID,
			DestinationPlanetID: destination.ID,
			PopulationUnits:     1,
		}}})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "retain at least 1")
	})

	t.Run("insufficient freighters", func(t *testing.T) {
		g := newPopulationTransferHarness()
		g.Players[0].Freighters = 4
		source := g.Galaxy.Stars[0].Planets[0]
		destination := g.Galaxy.Stars[0].Planets[1]

		err := g.Turn(1, model.Commands{TransferPopulation: []*model.TransferPopulationCommand{{
			SourcePlanetID:      source.ID,
			DestinationPlanetID: destination.ID,
			PopulationUnits:     1,
		}}})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "needs 5 freighters")
	})

	t.Run("destination capacity", func(t *testing.T) {
		g := newPopulationTransferHarness()
		source := g.Galaxy.Stars[0].Planets[0]
		destination := g.Galaxy.Stars[0].Planets[1]
		destination.Size = model.PlanetSizeTiny
		destination.Population = 4000

		err := g.Turn(1, model.Commands{TransferPopulation: []*model.TransferPopulationCommand{{
			SourcePlanetID:      source.ID,
			DestinationPlanetID: destination.ID,
			PopulationUnits:     1,
		}}})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "destination capacity")
	})
}

func TestPopulationTransferArrivalCapsOverflow(t *testing.T) {
	g := newPopulationTransferHarness()
	player := g.Players[0]
	destination := g.Galaxy.Stars[0].Planets[1]
	destination.Size = model.PlanetSizeTiny
	destination.Population = 3500
	g.PopulationTransfers = []model.PopulationTransfer{{
		ID:                  1,
		Owner:               player.ID,
		SourcePlanetID:      1,
		DestinationPlanetID: destination.ID,
		PopulationUnits:     1,
		FreightersReserved:  5,
		TurnsRemaining:      1,
	}}
	player.FreightersUsed = 5

	require.NoError(t, g.Turn(player.ID, model.Commands{}))

	assert.Equal(t, int64(4000), destination.Population)
	assert.Equal(t, uint(0), player.FreightersUsed)
}

func newFreighterHarness() *Game {
	return &Game{
		Players: []*model.Player{{
			ID:        1,
			Treasury:  10,
			Available: model.AvailableTech{Buildings: []string{"Freighters"}},
		}},
		Resources: &Resources{BuildingCollection: &BuildingCollection{Buildings: []model.Building{{Name: "Freighters", Cost: 50}}}},
		Galaxy: model.Galaxy{Stars: []*model.Star{{ID: 1, Planets: []*model.Planet{{
			ID:            1,
			Owner:         1,
			ParentStar:    1,
			Population:    1000,
			MaxPopulation: 4000,
			Size:          model.PlanetSizeTiny,
			MineralClass:  model.PlanetMineralAbundant,
		}}}}},
	}
}

func newFoodTransferHarness(freighters uint) *Game {
	return &Game{
		Players:   []*model.Player{{ID: 1, Treasury: 10, Freighters: freighters}},
		Resources: &Resources{BuildingCollection: &BuildingCollection{Buildings: []model.Building{}}},
		Galaxy: model.Galaxy{Stars: []*model.Star{{ID: 1, Planets: []*model.Planet{
			{
				ID:            1,
				Owner:         1,
				ParentStar:    1,
				Population:    2000,
				MaxPopulation: 2000,
				Size:          model.PlanetSizeTiny,
				MineralClass:  model.PlanetMineralAbundant,
				Food:          model.PlanetFoodOutput{Farmers: 2},
			},
			{
				ID:            2,
				Owner:         1,
				ParentStar:    1,
				Population:    2000,
				MaxPopulation: 2000,
				Size:          model.PlanetSizeTiny,
				MineralClass:  model.PlanetMineralAbundant,
			},
		}}}},
	}
}

func newPopulationTransferHarness() *Game {
	return &Game{
		Players:   []*model.Player{{ID: 1, Treasury: 10, Freighters: 5}},
		Resources: &Resources{BuildingCollection: &BuildingCollection{Buildings: []model.Building{}}},
		Galaxy: model.Galaxy{Stars: []*model.Star{{ID: 1, X: 0, Y: 0, Planets: []*model.Planet{
			{
				ID:            1,
				Owner:         1,
				ParentStar:    1,
				Population:    5000,
				MaxPopulation: 8000,
				Size:          model.PlanetSizeMedium,
				MineralClass:  model.PlanetMineralAbundant,
				Food:          model.PlanetFoodOutput{Farmers: 5},
			},
			{
				ID:            2,
				Owner:         1,
				ParentStar:    1,
				Population:    1000,
				MaxPopulation: 8000,
				Size:          model.PlanetSizeMedium,
				MineralClass:  model.PlanetMineralAbundant,
				Food:          model.PlanetFoodOutput{Farmers: 1},
			},
		}}}},
	}
}
