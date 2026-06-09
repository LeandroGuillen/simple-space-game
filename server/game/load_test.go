package game

import (
	"encoding/json"
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/require"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)
	original, err := NewGame(resources, NewGameOptions{
		PlayerName:  "Ada",
		NumStars:    6,
		PlayerColor: model.DefaultPlayerColor(),
		Race:        DefaultRace("Human"),
	})
	require.NoError(t, err)

	snapshot := *original
	snapshot.Resources = nil
	data, err := json.Marshal(&snapshot)
	require.NoError(t, err)

	var loaded Game
	require.NoError(t, json.Unmarshal(data, &loaded))
	HydrateLoadedGame(&loaded, resources)

	require.Equal(t, original.ID, loaded.ID)
	require.Equal(t, original.CurrentTurn, loaded.CurrentTurn)
	require.Equal(t, len(original.Galaxy.Stars), len(loaded.Galaxy.Stars))
	require.Equal(t, len(original.Players), len(loaded.Players))
	require.Equal(t, original.Players[0].Name, loaded.Players[0].Name)
	require.NotNil(t, loaded.Resources)
	require.NotNil(t, loaded.TurnSubmissions)

	require.NoError(t, loaded.Turn(loaded.Players[0].ID, model.Commands{}))
	require.NotNil(t, loaded.LastTurnAt)
}
