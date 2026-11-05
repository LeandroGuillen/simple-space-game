package game

import (
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/require"
)

func TestMovePopulationTransferRequiresRemainingUnits(t *testing.T) {
	for _, tt := range []struct {
		name    string
		farmers int64
		valid   bool
	}{
		{"zero", 0, false}, {"partial", 2, false}, {"original total", 5, false}, {"remaining total", 3, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g := newPopulationTransferHarness()
			g.Players[0].Freighters = 10
			commands := model.Commands{
				MovePopulation: []*model.MovePopulationCommand{{PlanetID: 1, Farmers: tt.farmers}},
				TransferPopulation: []*model.TransferPopulationCommand{
					{SourcePlanetID: 1, DestinationPlanetID: 2, PopulationUnits: 1},
					{SourcePlanetID: 1, DestinationPlanetID: 2, PopulationUnits: 1},
				},
			}
			_, advanced, err := g.SubmitTurn(1, commands)
			if tt.valid {
				require.NoError(t, err)
				require.True(t, advanced)
			} else {
				require.ErrorContains(t, err, "must assign all 3 units")
				require.False(t, advanced)
			}
		})
	}
}

func TestMovePopulationWithOutboundTransfer(t *testing.T) {
	g := newPopulationTransferHarness()
	source := g.Galaxy.Stars[0].Planets[0]
	destination := g.Galaxy.Stars[0].Planets[1]
	commands := model.Commands{
		MovePopulation:     []*model.MovePopulationCommand{{PlanetID: source.ID, Farmers: 3, Workers: 1}},
		TransferPopulation: []*model.TransferPopulationCommand{{SourcePlanetID: source.ID, DestinationPlanetID: destination.ID, PopulationUnits: 1}},
	}
	_, advanced, err := g.SubmitTurn(1, commands)
	require.NoError(t, err)
	require.True(t, advanced)
	require.Len(t, g.PopulationTransfers, 1)
	require.Equal(t, int64(3), source.Food.Farmers)
	require.Equal(t, int64(1), source.Industry.Workers)
	require.Equal(t, source.Population/1000, source.Food.Farmers+source.Industry.Workers+source.Science.Scientists)
}
