package game

import (
	"testing"
	"time"

	"ssg/model"

	"github.com/stretchr/testify/require"
)

func TestLastTurnAtNilUntilFirstTurn(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)
	g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: DefaultRace("Human")})
	require.NoError(t, err)
	require.Nil(t, g.LastTurnAt)
	require.False(t, g.CreatedAt.IsZero())
}

func TestLastTurnAtUpdatedEachTurn(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)
	g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: DefaultRace("Human")})
	require.NoError(t, err)
	player := g.Players[0]

	require.NoError(t, g.Turn(player.ID, model.Commands{}))
	require.NotNil(t, g.LastTurnAt)
	first := *g.LastTurnAt

	time.Sleep(2 * time.Millisecond)

	require.NoError(t, g.Turn(player.ID, model.Commands{}))
	require.NotNil(t, g.LastTurnAt)
	require.True(t, g.LastTurnAt.After(first), "last_turn_at should advance after another turn")
}
