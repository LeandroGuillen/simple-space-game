package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ssg/game"
	"ssg/model"

	"github.com/stretchr/testify/require"
)

func TestColonyListHeaderAuthorization(t *testing.T) {
	s := newGameTestServer(t)
	g, err := game.NewGame(s.ResourceManager, game.NewGameOptions{
		PlayerName: "Ada", PlayerColor: model.DefaultPlayerColor(), NumStars: 8,
		Race: game.DefaultRace("Human"), NumPlayers: 2,
	})
	require.NoError(t, err)
	s.putGame(g.ID, g)
	own, err := g.GetColonyList(1)
	require.NoError(t, err)
	require.NotEmpty(t, own)
	opponent, err := g.GetColonyList(2)
	require.NoError(t, err)
	require.NotEmpty(t, opponent)

	request := func(gameHeader, playerHeader, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, "/colonies", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set(GameIDHeader, gameHeader)
		r.Header.Set(PlayerIDHeader, playerHeader)
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, r)
		return w
	}
	body := fmt.Sprintf(`{"game_id":%q,"player_id":2}`, g.ID)
	for _, tc := range []struct{ name, gameID, playerID string }{
		{"both missing", "", ""},
		{"game missing", "", "1"},
		{"player missing", g.ID, ""},
		{"player invalid", g.ID, "invalid"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := request(tc.gameID, tc.playerID, body)
			require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
		})
	}
	t.Run("body cannot select opponent", func(t *testing.T) {
		w := request(g.ID, "1", body)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var colonies []model.Planet
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &colonies))
		require.Len(t, colonies, len(own))
		for _, colony := range colonies {
			require.Equal(t, uint(1), colony.Owner)
		}
	})
	t.Run("headers suffice", func(t *testing.T) {
		w := request(g.ID, "2", `{}`)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var colonies []model.Planet
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &colonies))
		require.Len(t, colonies, len(opponent))
		for _, colony := range colonies {
			require.Equal(t, uint(2), colony.Owner)
		}
	})
}
