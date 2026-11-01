package server

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"ssg/game"
	"ssg/model"
	"strings"
	"testing"
)

func serverAITestGame(t *testing.T) (*Server, *game.Game) {
	t.Helper()
	s, err := NewServer()
	require.NoError(t, err)
	g, err := game.NewGame(s.ResourceManager, game.NewGameOptions{PlayerName: "Ada", PlayerColor: model.DefaultPlayerColor(), NumStars: 12, GalaxySize: "medium", Race: game.DefaultRace("Human"), NumPlayers: 4})
	require.NoError(t, err)
	s.putGame(g.ID, g)
	return s, g
}
func postAITurn(t *testing.T, s *Server, g *game.Game, id uint) map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/turn", strings.NewReader(`{"commands":{}}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(GameIDHeader, g.ID)
	req.Header.Set(PlayerIDHeader, fmt.Sprint(id))
	out := httptest.NewRecorder()
	s.Handler().ServeHTTP(out, req)
	require.Equal(t, 200, out.Code, out.Body.String())
	var result map[string]any
	require.NoError(t, json.Unmarshal(out.Body.Bytes(), &result))
	return result
}
func TestServerAIAdvancesFourPlayerGame(t *testing.T) {
	s, g := serverAITestGame(t)
	before := g.CurrentTurn
	result := postAITurn(t, s, g, 1)
	require.Equal(t, true, result["advanced"])
	require.InDelta(t, before+0.1, g.CurrentTurn, 0.0001)
	for _, p := range g.Players[1:] {
		require.Equal(t, "server_ai", p.Controller)
		require.NotNil(t, p.CurrentResearch)
	}
	v, err := g.GetVisibleGameForPlayer(2)
	require.NoError(t, err)
	for _, star := range v.Galaxy.Stars {
		if star.ID != g.Galaxy.Stars[0].ID && star.Name == "" {
			require.Empty(t, star.Planets)
		}
	}
	data, err := json.Marshal(g)
	require.NoError(t, err)
	var loaded game.Game
	require.NoError(t, json.Unmarshal(data, &loaded))
	require.Equal(t, "server_ai", loaded.Players[1].Controller)
}
func TestExternalAgentTakesOverSlot(t *testing.T) {
	s, g := serverAITestGame(t)
	postAITurn(t, s, g, 2)
	require.Equal(t, "external", g.Players[1].Controller)
	result := postAITurn(t, s, g, 1)
	require.Equal(t, true, result["advanced"])
	result = postAITurn(t, s, g, 1)
	require.Equal(t, false, result["advanced"])
	result = postAITurn(t, s, g, 2)
	require.Equal(t, true, result["advanced"])
}
