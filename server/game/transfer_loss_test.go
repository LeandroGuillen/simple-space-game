package game

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"ssg/model"
)

func TestTransferLossForMissingSourceIsPrivate(t *testing.T) {
	g := multiPlayerGame(t, 2, 30)
	var destination *model.Planet
	for _, star := range g.Galaxy.Stars {
		for _, p := range star.Planets {
			if p.Owner == 1 {
				destination = p
			}
		}
	}
	require.NotNil(t, destination)
	destination.Size = model.PlanetSizeTiny
	destination.Environment = model.PlanetEnvironmentTerran
	destination.Population = 3500
	before := destination.Population
	g.PopulationTransfers = []model.PopulationTransfer{{ID: 4, Owner: 1, SourcePlanetID: 987654, DestinationPlanetID: destination.ID, PopulationUnits: 1, TurnsRemaining: 1}}
	g.advancePopulationTransfers(map[uint]bool{4: true})
	require.Len(t, g.TransferLosses, 1)
	loss := g.TransferLosses[0]
	require.Equal(t, int64(1000), destination.Population-before+loss.PopulationLost)
	require.Equal(t, int64(500), loss.PopulationLost)
	require.Equal(t, uint(987654), loss.SourcePlanetID)
	require.Equal(t, destination.Name, loss.DestinationPlanetName)
	ownerView, err := g.GetVisibleGameForPlayer(1)
	require.NoError(t, err)
	require.Equal(t, g.TransferLosses, ownerView.TransferLosses)
	rivalView, err := g.GetVisibleGameForPlayer(2)
	require.NoError(t, err)
	require.Empty(t, rivalView.TransferLosses)
	data, err := g.MarshalSave()
	require.NoError(t, err)
	var restored Game
	require.NoError(t, json.Unmarshal(data, &restored))
	resources := testResources(t)
	require.NoError(t, ValidateLoadedGame(&restored, resources))
	HydrateLoadedGame(&restored, resources)
	restoredView, err := restored.GetVisibleGameForPlayer(1)
	require.NoError(t, err)
	require.Equal(t, ownerView.TransferLosses, restoredView.TransferLosses)
}

func TestTransferLossNoticeExpiresOnlyAfterSuccessfulTurn(t *testing.T) {
	g := multiPlayerGame(t, 2, 30)
	g.TransferLosses = []model.TransferLossNotice{{Owner: 1, TransferID: 4, PopulationLost: 500}}
	before := append([]model.TransferLossNotice(nil), g.TransferLosses...)
	t.Cleanup(func() { advanceTurnTestHook = nil })
	advanceTurnTestHook = func(stage string) error {
		if stage == "before_fleet_movements" {
			return errors.New("injected failure")
		}
		return nil
	}
	require.Error(t, g.AdvanceTurn(emptySubsFor(g)))
	require.Equal(t, before, g.TransferLosses)
	advanceTurnTestHook = nil
	require.NoError(t, g.AdvanceTurn(emptySubsFor(g)))
	require.Empty(t, g.TransferLosses)
}
