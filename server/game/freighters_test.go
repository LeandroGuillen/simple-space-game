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
	source := g.Galaxy.Stars[0].Planets[0]
	destination := g.Galaxy.Stars[0].Planets[1]
	destination.Size = model.PlanetSizeTiny
	destination.Population = 3500
	sourceBefore := source.Population
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

	// Advance transfers directly so colony growth does not obscure conservation.
	g.advancePopulationTransfers(map[uint]bool{1: true})

	assert.Equal(t, int64(4000), destination.Population)
	assert.Equal(t, sourceBefore+500, source.Population, "overflow should refund to source")
	assert.Equal(t, uint(0), player.FreightersUsed)
}

func TestPopulationTransferSubmitRejectsSameTurnOverfill(t *testing.T) {
	g := newPopulationTransferOverfillHarness()
	player := g.Players[0]
	source := g.Galaxy.Stars[0].Planets[0]
	destination := g.Galaxy.Stars[0].Planets[1]
	sourcePop := source.Population
	destPop := destination.Population
	freightersUsed := player.FreightersUsed

	orders := make([]*model.TransferPopulationCommand, 3)
	for i := range orders {
		orders[i] = &model.TransferPopulationCommand{
			SourcePlanetID:      source.ID,
			DestinationPlanetID: destination.ID,
			PopulationUnits:     6,
		}
	}

	err := g.Turn(player.ID, model.Commands{TransferPopulation: orders})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "destination capacity")
	assert.Empty(t, g.PopulationTransfers)
	assert.Equal(t, sourcePop, source.Population)
	assert.Equal(t, destPop, destination.Population)
	assert.Equal(t, freightersUsed, player.FreightersUsed)
}

func TestPopulationTransferSubmitAcceptsExactCapacityFill(t *testing.T) {
	g := newPopulationTransferOverfillHarness()
	player := g.Players[0]
	source := g.Galaxy.Stars[0].Planets[0]
	destination := g.Galaxy.Stars[0].Planets[1]
	// Medium Terran capacity = 12 units; dest at 1042 → 1 unit occupied → 11 free.
	require.NoError(t, g.Turn(player.ID, model.Commands{
		TransferPopulation: []*model.TransferPopulationCommand{{
			SourcePlanetID:      source.ID,
			DestinationPlanetID: destination.ID,
			PopulationUnits:     11,
		}},
	}))
	require.Len(t, g.PopulationTransfers, 1)
	assert.Equal(t, uint(11), g.PopulationTransfers[0].PopulationUnits)
	hydratePlanetPopulationCapacity(g, destination, player)
	pendingInbound := uint(0)
	for _, transfer := range g.PopulationTransfers {
		if transfer.Owner == player.ID && transfer.DestinationPlanetID == destination.ID {
			pendingInbound += transfer.PopulationUnits
		}
	}
	occupied := uint(destination.Population/1000) + pendingInbound
	assert.Equal(t, uint(destination.MaxPopulation/1000), occupied, "exact fill should commit all remaining room")
}

func TestPopulationTransferRejectsOverfillAgainstInFlight(t *testing.T) {
	g := newPopulationTransferOverfillHarness()
	player := g.Players[0]
	source := g.Galaxy.Stars[0].Planets[0]
	destination := g.Galaxy.Stars[0].Planets[1]

	require.NoError(t, g.Turn(player.ID, model.Commands{
		TransferPopulation: []*model.TransferPopulationCommand{{
			SourcePlanetID:      source.ID,
			DestinationPlanetID: destination.ID,
			PopulationUnits:     6,
		}},
	}))
	require.Len(t, g.PopulationTransfers, 1)

	sourcePop := source.Population
	destPop := destination.Population
	pending := append([]model.PopulationTransfer(nil), g.PopulationTransfers...)

	err := g.Turn(player.ID, model.Commands{
		TransferPopulation: []*model.TransferPopulationCommand{{
			SourcePlanetID:      source.ID,
			DestinationPlanetID: destination.ID,
			PopulationUnits:     6,
		}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "destination capacity")
	assert.Equal(t, sourcePop, source.Population)
	assert.Equal(t, destPop, destination.Population)
	require.Len(t, g.PopulationTransfers, len(pending))
	assert.Equal(t, pending[0].PopulationUnits, g.PopulationTransfers[0].PopulationUnits)
}

func TestPopulationTransferIgnoresOtherPlayerInFlightForCapacity(t *testing.T) {
	g := newPopulationTransferOverfillHarness()
	player := g.Players[0]
	source := g.Galaxy.Stars[0].Planets[0]
	destination := g.Galaxy.Stars[0].Planets[1]
	// Rival transfer aimed at the same planet id must not consume our room.
	// (Owner 2 is not a Players entry so single-player Turn still advances.)
	g.PopulationTransfers = []model.PopulationTransfer{{
		ID:                  9,
		Owner:               2,
		SourcePlanetID:      99,
		DestinationPlanetID: destination.ID,
		PopulationUnits:     11,
		FreightersReserved:  55,
		TurnsRemaining:      3,
	}}

	require.NoError(t, g.Turn(player.ID, model.Commands{
		TransferPopulation: []*model.TransferPopulationCommand{{
			SourcePlanetID:      source.ID,
			DestinationPlanetID: destination.ID,
			PopulationUnits:     11,
		}},
	}))
	require.Len(t, g.PopulationTransfers, 2)
	var ours uint
	for _, transfer := range g.PopulationTransfers {
		if transfer.Owner == player.ID {
			ours = transfer.PopulationUnits
		}
	}
	assert.Equal(t, uint(11), ours)
}

func TestPopulationTransferOverflowRefundsToSource(t *testing.T) {
	g := newPopulationTransferHarness()
	player := g.Players[0]
	source := g.Galaxy.Stars[0].Planets[0]
	destination := g.Galaxy.Stars[0].Planets[1]
	destination.Size = model.PlanetSizeTiny
	destination.Population = 3500
	inTransit := int64(1000)
	sourceBefore := source.Population
	destBefore := destination.Population
	totalBefore := sourceBefore + destBefore + inTransit

	g.PopulationTransfers = []model.PopulationTransfer{{
		ID:                  1,
		Owner:               player.ID,
		SourcePlanetID:      source.ID,
		DestinationPlanetID: destination.ID,
		PopulationUnits:     1,
		FreightersReserved:  5,
		TurnsRemaining:      1,
	}}
	player.FreightersUsed = 5

	require.NotPanics(t, func() {
		g.advancePopulationTransfers(map[uint]bool{1: true})
	})

	assert.Empty(t, g.PopulationTransfers)
	assert.Equal(t, int64(4000), destination.Population)
	assert.Equal(t, sourceBefore+500, source.Population)
	assert.Equal(t, totalBefore, source.Population+destination.Population)
	assert.Equal(t, uint(0), player.FreightersUsed)
}

func TestPopulationTransferOverflowLostWhenSourceGone(t *testing.T) {
	g := newPopulationTransferHarness()
	player := g.Players[0]
	source := g.Galaxy.Stars[0].Planets[0]
	destination := g.Galaxy.Stars[0].Planets[1]
	destination.Size = model.PlanetSizeTiny
	destination.Population = 3500
	source.Owner = 0
	sourceBefore := source.Population
	destBefore := destination.Population

	g.PopulationTransfers = []model.PopulationTransfer{{
		ID:                  1,
		Owner:               player.ID,
		SourcePlanetID:      source.ID,
		DestinationPlanetID: destination.ID,
		PopulationUnits:     1,
		FreightersReserved:  5,
		TurnsRemaining:      1,
	}}
	player.FreightersUsed = 5

	require.NotPanics(t, func() {
		g.advancePopulationTransfers(map[uint]bool{1: true})
	})

	assert.Empty(t, g.PopulationTransfers)
	assert.Equal(t, int64(4000), destination.Population)
	assert.Equal(t, sourceBefore, source.Population, "unowned source must not receive refund")
	assert.Equal(t, destBefore+500, destination.Population)
	assert.Equal(t, uint(0), player.FreightersUsed)
	require.Len(t, g.TransferLosses, 1)
	assert.Equal(t, int64(500), g.TransferLosses[0].PopulationLost)
	assert.Equal(t, uint(1), g.TransferLosses[0].Owner)
	assert.Equal(t, int64(1000), destination.Population-destBefore+g.TransferLosses[0].PopulationLost)
}

func TestPopulationTransferRefundsWhenDestinationCaptured(t *testing.T) {
	// (a) destination captured (owner changed) before arrival -> full cargo refunded.
	g := newPopulationTransferHarness()
	player := g.Players[0]
	source := g.Galaxy.Stars[0].Planets[0]
	destination := g.Galaxy.Stars[0].Planets[1]
	cargo := int64(1000)
	sourceBefore := source.Population
	destBefore := destination.Population
	destination.Owner = 2 // captured by rival

	g.PopulationTransfers = []model.PopulationTransfer{{
		ID: 1, Owner: player.ID,
		SourcePlanetID: source.ID, DestinationPlanetID: destination.ID,
		PopulationUnits: 1, FreightersReserved: 5, TurnsRemaining: 1,
	}}
	player.FreightersUsed = 5

	require.NotPanics(t, func() {
		g.advancePopulationTransfers(map[uint]bool{1: true})
	})

	assert.Empty(t, g.PopulationTransfers)
	assert.Equal(t, sourceBefore+cargo, source.Population, "full cargo refunded to owned source")
	assert.Equal(t, destBefore, destination.Population, "captured destination must not receive cargo")
	assert.Equal(t, uint(0), player.FreightersUsed)
}

func TestPopulationTransferRefundsWhenDestinationMissing(t *testing.T) {
	// (b) destination no longer exists -> refund to source.
	g := newPopulationTransferHarness()
	player := g.Players[0]
	source := g.Galaxy.Stars[0].Planets[0]
	cargo := int64(1000)
	sourceBefore := source.Population
	missingDestID := uint(99)

	g.PopulationTransfers = []model.PopulationTransfer{{
		ID: 1, Owner: player.ID,
		SourcePlanetID: source.ID, DestinationPlanetID: missingDestID,
		PopulationUnits: 1, FreightersReserved: 5, TurnsRemaining: 1,
	}}
	player.FreightersUsed = 5

	require.NotPanics(t, func() {
		g.advancePopulationTransfers(map[uint]bool{1: true})
	})

	assert.Empty(t, g.PopulationTransfers)
	assert.Equal(t, sourceBefore+cargo, source.Population)
	assert.Equal(t, uint(0), player.FreightersUsed)
}

func TestPopulationTransferRefundsWhenDestinationUnowned(t *testing.T) {
	// (c) destination owner 0 -> refund.
	g := newPopulationTransferHarness()
	player := g.Players[0]
	source := g.Galaxy.Stars[0].Planets[0]
	destination := g.Galaxy.Stars[0].Planets[1]
	cargo := int64(1000)
	sourceBefore := source.Population
	destBefore := destination.Population
	destination.Owner = 0

	g.PopulationTransfers = []model.PopulationTransfer{{
		ID: 1, Owner: player.ID,
		SourcePlanetID: source.ID, DestinationPlanetID: destination.ID,
		PopulationUnits: 1, FreightersReserved: 5, TurnsRemaining: 1,
	}}
	player.FreightersUsed = 5

	require.NotPanics(t, func() {
		g.advancePopulationTransfers(map[uint]bool{1: true})
	})

	assert.Empty(t, g.PopulationTransfers)
	assert.Equal(t, sourceBefore+cargo, source.Population)
	assert.Equal(t, destBefore, destination.Population)
	assert.Equal(t, uint(0), player.FreightersUsed)
}

func TestPopulationTransferLostWhenSourceAndDestinationGone(t *testing.T) {
	// (d) source also gone/captured -> no panic, loss only logged, no pop elsewhere.
	g := newPopulationTransferHarness()
	player := g.Players[0]
	source := g.Galaxy.Stars[0].Planets[0]
	destination := g.Galaxy.Stars[0].Planets[1]
	sourceBefore := source.Population
	destBefore := destination.Population
	source.Owner = 2
	destination.Owner = 2

	g.PopulationTransfers = []model.PopulationTransfer{{
		ID: 1, Owner: player.ID,
		SourcePlanetID: source.ID, DestinationPlanetID: destination.ID,
		PopulationUnits: 1, FreightersReserved: 5, TurnsRemaining: 1,
	}}
	player.FreightersUsed = 5

	require.NotPanics(t, func() {
		g.advancePopulationTransfers(map[uint]bool{1: true})
	})

	assert.Empty(t, g.PopulationTransfers)
	assert.Equal(t, sourceBefore, source.Population)
	assert.Equal(t, destBefore, destination.Population)
	assert.Equal(t, uint(0), player.FreightersUsed)

	// Missing source planet id equally safe.
	g2 := newPopulationTransferHarness()
	g2.Players[0].FreightersUsed = 5
	g2.Galaxy.Stars[0].Planets[1].Owner = 0
	source2Before := g2.Galaxy.Stars[0].Planets[0].Population
	dest2Before := g2.Galaxy.Stars[0].Planets[1].Population
	g2.PopulationTransfers = []model.PopulationTransfer{{
		ID: 2, Owner: 1,
		SourcePlanetID: 999, DestinationPlanetID: 2,
		PopulationUnits: 1, FreightersReserved: 5, TurnsRemaining: 1,
	}}
	require.NotPanics(t, func() {
		g2.advancePopulationTransfers(map[uint]bool{2: true})
	})
	assert.Equal(t, source2Before, g2.Galaxy.Stars[0].Planets[0].Population)
	assert.Equal(t, dest2Before, g2.Galaxy.Stars[0].Planets[1].Population)
}

func TestPopulationTransferRefundRespectsSourceCapacity(t *testing.T) {
	// (e) source near capacity -> only what fits refunded; remainder lost.
	// Conservation: refund + lost = cargo.
	g := newPopulationTransferHarness()
	player := g.Players[0]
	source := g.Galaxy.Stars[0].Planets[0]
	destination := g.Galaxy.Stars[0].Planets[1]
	// Tiny Terran MaxPop = 4000; leave 500 room for a 1000-cargo refund.
	source.Size = model.PlanetSizeTiny
	source.Environment = model.PlanetEnvironmentTerran
	source.Population = 3500
	source.Food.Farmers = 3
	destination.Owner = 2 // force full-cargo refund path
	cargo := int64(1000)
	sourceBefore := source.Population
	destBefore := destination.Population

	g.PopulationTransfers = []model.PopulationTransfer{{
		ID: 1, Owner: player.ID,
		SourcePlanetID: source.ID, DestinationPlanetID: destination.ID,
		PopulationUnits: 1, FreightersReserved: 5, TurnsRemaining: 1,
	}}
	player.FreightersUsed = 5

	require.NotPanics(t, func() {
		g.advancePopulationTransfers(map[uint]bool{1: true})
	})

	hydratePlanetPopulationCapacity(g, source, player)
	refunded := source.Population - sourceBefore
	lost := cargo - refunded
	assert.Equal(t, int64(500), refunded, "only capacity room should be refunded")
	assert.Equal(t, int64(500), lost)
	assert.Equal(t, cargo, refunded+lost, "conservation: refund + lost = cargo")
	assert.Equal(t, source.MaxPopulation, source.Population)
	assert.Equal(t, destBefore, destination.Population)
	assert.Equal(t, uint(0), player.FreightersUsed)
}

func TestPopulationTransferOverflowRefundRespectsSourceCapacity(t *testing.T) {
	// (f) overflow path also capacity-caps the source refund.
	g := newPopulationTransferHarness()
	player := g.Players[0]
	source := g.Galaxy.Stars[0].Planets[0]
	destination := g.Galaxy.Stars[0].Planets[1]
	destination.Size = model.PlanetSizeTiny
	destination.Environment = model.PlanetEnvironmentTerran
	destination.Population = 3500 // Tiny Terran cap 4000 → 500 lands, 500 overflow
	source.Size = model.PlanetSizeTiny
	source.Environment = model.PlanetEnvironmentTerran
	source.Population = 3800 // room 200 under Tiny Terran 4000
	source.Food.Farmers = 3
	cargo := int64(1000)
	sourceBefore := source.Population

	g.PopulationTransfers = []model.PopulationTransfer{{
		ID: 1, Owner: player.ID,
		SourcePlanetID: source.ID, DestinationPlanetID: destination.ID,
		PopulationUnits: 1, FreightersReserved: 5, TurnsRemaining: 1,
	}}
	player.FreightersUsed = 5

	require.NotPanics(t, func() {
		g.advancePopulationTransfers(map[uint]bool{1: true})
	})

	assert.Equal(t, int64(4000), destination.Population)
	refunded := source.Population - sourceBefore
	overflow := int64(500)
	lost := overflow - refunded
	assert.Equal(t, int64(200), refunded)
	assert.Equal(t, int64(300), lost)
	assert.Equal(t, overflow, refunded+lost)
	assert.Equal(t, uint(0), player.FreightersUsed)
	require.Len(t, g.TransferLosses, 1)
	assert.Equal(t, int64(300), g.TransferLosses[0].PopulationLost)
	assert.Equal(t, cargo, (destination.Population-3500)+refunded+g.TransferLosses[0].PopulationLost)
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
				Environment:   model.PlanetEnvironmentTerran,
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
				Environment:   model.PlanetEnvironmentTerran,
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
				Environment:   model.PlanetEnvironmentTerran,
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
				Environment:   model.PlanetEnvironmentTerran,
				Food:          model.PlanetFoodOutput{Farmers: 1},
			},
		}}}},
	}
}

// newPopulationTransferOverfillHarness matches the reported overfill scenario:
// dest ~1,042 / 12,000 capacity (11 units free), large source, many freighters.
func newPopulationTransferOverfillHarness() *Game {
	g := newPopulationTransferHarness()
	g.Players[0].Freighters = 90
	g.Galaxy.Stars[0].Planets[0].Population = 20000
	g.Galaxy.Stars[0].Planets[0].Food.Farmers = 20
	g.Galaxy.Stars[0].Planets[1].Population = 1042
	return g
}
