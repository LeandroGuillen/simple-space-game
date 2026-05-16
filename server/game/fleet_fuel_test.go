package game

import (
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fuelTestGame(t *testing.T) *Game {
	t.Helper()
	resources, err := NewResourceManager()
	require.NoError(t, err)
	player := &model.Player{
		ID: 1,
		Available: model.AvailableTech{
			FuelCells: []string{"Standard"},
			Drives:    []string{"Nuclear"},
		},
	}
	return &Game{
		Players:   []*model.Player{player},
		Resources: &resources.Resources,
		Fleets: []*model.Fleet{
			{
				ID:     1,
				Owner:  1,
				StarID: 1,
				X:      0,
				Y:      0,
				Ships: []model.FleetShip{
					{Type: "ship_design", Name: "Standard Frigate", FuelCell: "Standard", FuelRangeParsecs: 4},
				},
			},
		},
		Galaxy: model.Galaxy{
			Width:  1000,
			Height: 800,
			Stars: []*model.Star{
				{ID: 1, X: 0, Y: 0, Planets: []*model.Planet{{ID: 1, Owner: 1}}},
				{ID: 2, X: 120, Y: 0, Planets: []*model.Planet{}},
				{ID: 3, X: 180, Y: 0, Planets: []*model.Planet{}},
				{ID: 4, X: 360, Y: 0, Planets: []*model.Planet{}},
				{ID: 5, X: 480, Y: 0, Planets: []*model.Planet{}},
				{ID: 6, X: 1000, Y: 800, Planets: []*model.Planet{}},
			},
		},
	}
}

func TestFleetFuelStandardAllowsDestinationWithinSupplyRange(t *testing.T) {
	g := fuelTestGame(t)

	err := g.Turn(1, model.Commands{
		FleetMove: []*model.FleetMoveCommand{
			{FleetID: 1, DestinationStarID: 2, ShipIndexes: uShips(0)},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, uint(2), g.Fleets[0].DestinationStarID)
}

func TestFleetFuelStandardRejectsDestinationBeyondSupplyRange(t *testing.T) {
	g := fuelTestGame(t)

	err := g.Turn(1, model.Commands{
		FleetMove: []*model.FleetMoveCommand{
			{FleetID: 1, DestinationStarID: 3, ShipIndexes: uShips(0)},
		},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "outside fleet fuel range")
	assert.Equal(t, uint(0), g.Fleets[0].DestinationStarID)
}

func TestFleetFuelHigherTechExtendsSupplyRange(t *testing.T) {
	cases := []struct {
		name        string
		fuelCell    string
		rangeParsec int64
		destStarID  uint
	}{
		{name: "Deuterium", fuelCell: "Deuterium", rangeParsec: 4, destStarID: 3},
		{name: "Iridium", fuelCell: "Iridium", rangeParsec: 6, destStarID: 4},
		{name: "Urridium", fuelCell: "Urridium", rangeParsec: 8, destStarID: 5},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := fuelTestGame(t)
			g.Fleets[0].Ships[0].FuelCell = tc.fuelCell
			g.Fleets[0].Ships[0].FuelRangeParsecs = tc.rangeParsec

			err := g.Turn(1, model.Commands{
				FleetMove: []*model.FleetMoveCommand{
					{FleetID: 1, DestinationStarID: tc.destStarID, ShipIndexes: uShips(0)},
				},
			})

			require.NoError(t, err)
			assert.Equal(t, tc.destStarID, g.Fleets[0].DestinationStarID)
		})
	}
}

func TestFleetFuelThoriumAllowsAnyDestination(t *testing.T) {
	g := fuelTestGame(t)
	g.Fleets[0].Ships[0].FuelCell = "Thorium"
	g.Fleets[0].Ships[0].FuelRangeParsecs = 100
	g.Fleets[0].Ships[0].UnlimitedFuelRange = true

	err := g.Turn(1, model.Commands{
		FleetMove: []*model.FleetMoveCommand{
			{FleetID: 1, DestinationStarID: 6, ShipIndexes: uShips(0)},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, uint(6), g.Fleets[0].DestinationStarID)
}

func TestFleetFuelMixedFleetUsesLowestSelectedRange(t *testing.T) {
	g := fuelTestGame(t)
	g.Fleets[0].Ships = append(g.Fleets[0].Ships, model.FleetShip{
		Type: "ship_design", Name: "Iridium Frigate", FuelCell: "Iridium", FuelRangeParsecs: 9,
	})

	err := g.Turn(1, model.Commands{
		FleetMove: []*model.FleetMoveCommand{
			{FleetID: 1, DestinationStarID: 3, ShipIndexes: uShips(0, 1)},
		},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "outside fleet fuel range")
}

func TestFleetFuelPartialMoveCanSendHighRangeShipsOnly(t *testing.T) {
	g := fuelTestGame(t)
	g.Fleets[0].Ships = append(g.Fleets[0].Ships, model.FleetShip{
		Type: "ship_design", Name: "Iridium Frigate", FuelCell: "Iridium", FuelRangeParsecs: 9,
	})

	err := g.Turn(1, model.Commands{
		FleetMove: []*model.FleetMoveCommand{
			{FleetID: 1, DestinationStarID: 3, ShipIndexes: uShips(1)},
		},
	})

	require.NoError(t, err)
	require.Len(t, g.Fleets, 2)
	var moving *model.Fleet
	for _, fleet := range g.Fleets {
		if fleet.DestinationStarID == 3 {
			moving = fleet
		}
	}
	require.NotNil(t, moving)
	require.Len(t, moving.Ships, 1)
	assert.Equal(t, "Iridium Frigate", moving.Ships[0].Name)
}

func TestFleetFuelGroupedMovesValidateEachDestination(t *testing.T) {
	g := fuelTestGame(t)
	g.Fleets[0].Ships = append(g.Fleets[0].Ships, model.FleetShip{
		Type: "ship_design", Name: "Second Standard Frigate", FuelCell: "Standard", FuelRangeParsecs: 4,
	})

	err := g.Turn(1, model.Commands{
		FleetMove: []*model.FleetMoveCommand{
			{FleetID: 1, DestinationStarID: 2, ShipIndexes: uShips(0)},
			{FleetID: 1, DestinationStarID: 3, ShipIndexes: uShips(1)},
		},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "outside fleet fuel range")
	require.Len(t, g.Fleets, 1)
	assert.Len(t, g.Fleets[0].Ships, 2)
}

func TestFleetFuelRecallToOriginIsAllowedOutOfSupplyRange(t *testing.T) {
	g := fuelTestGame(t)
	g.Fleets[0] = &model.Fleet{
		ID:                1,
		Owner:             1,
		StarID:            3,
		X:                 250,
		Y:                 0,
		OriginX:           0,
		OriginY:           0,
		DestinationStarID: 4,
		TotalTurns:        5,
		TurnsRemaining:    3,
		Ships: []model.FleetShip{
			{Type: "ship_design", Name: "Standard Frigate", FuelCell: "Standard", FuelRangeParsecs: 4},
		},
	}

	err := g.Turn(1, model.Commands{
		FleetMove: []*model.FleetMoveCommand{
			{FleetID: 1, DestinationStarID: 3, ShipIndexes: uShips(0)},
		},
	})

	require.NoError(t, err)
	require.Len(t, g.Fleets, 1)
	assert.Equal(t, uint(0), g.Fleets[0].DestinationStarID)
}

func TestFleetFuelRerouteRequiresSupplyRangeWithHyperspaceCommunications(t *testing.T) {
	g := fuelTestGame(t)
	g.Players[0].ResearchedTechs = []string{"Hyperspace Communications"}
	g.Fleets[0] = &model.Fleet{
		ID:                1,
		Owner:             1,
		StarID:            1,
		X:                 120,
		Y:                 0,
		OriginX:           0,
		OriginY:           0,
		DestinationStarID: 2,
		TotalTurns:        5,
		TurnsRemaining:    3,
		Ships: []model.FleetShip{
			{Type: "ship_design", Name: "Standard Frigate", FuelCell: "Standard", FuelRangeParsecs: 4},
		},
	}

	err := g.Turn(1, model.Commands{
		FleetMove: []*model.FleetMoveCommand{
			{FleetID: 1, DestinationStarID: 3, ShipIndexes: uShips(0)},
		},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "outside fleet fuel range")
	assert.Equal(t, uint(2), g.Fleets[0].DestinationStarID)
}

func TestFleetFuelCivilianShipUsesBestAvailableFuelCell(t *testing.T) {
	g := fuelTestGame(t)
	g.Players[0].Available.FuelCells = []string{"Standard", "Iridium"}
	g.Fleets[0].Ships = []model.FleetShip{
		{Type: "colony_ship", Name: "Colony Ship"},
	}

	err := g.Turn(1, model.Commands{
		FleetMove: []*model.FleetMoveCommand{
			{FleetID: 1, DestinationStarID: 4, ShipIndexes: uShips(0)},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, uint(4), g.Fleets[0].DestinationStarID)
}
