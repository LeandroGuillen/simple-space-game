package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ssg/game"
	"ssg/model"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestTurnBroadcastsDroppedStaleSubmission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s, err := NewServer()
	require.NoError(t, err)
	g, err := game.NewGame(s.ResourceManager, game.NewGameOptions{
		PlayerName: "Ada", PlayerColor: model.DefaultPlayerColor(),
		NumStars: 8, Race: game.DefaultRace("Human"), NumPlayers: 2,
	})
	require.NoError(t, err)
	s.putGame(g.ID, g)
	httpServer := httptest.NewServer(s.Handler())
	t.Cleanup(httpServer.Close)

	submit := func(playerID uint, body string) *httptest.ResponseRecorder {
		t.Helper()
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPost, "/turn", strings.NewReader(body))
		ctx.Request.Header.Set("Content-Type", "application/json")
		ctx.Request.Header.Set(GameIDHeader, g.ID)
		ctx.Request.Header.Set(PlayerIDHeader, fmt.Sprint(playerID))
		ctx.Set("server", s)
		Turn(ctx)
		return recorder
	}

	// Model an accepted AI move whose fleet was subsequently destroyed.
	g.TurnSubmissions = map[uint]model.Commands{2: {
		FleetMove: []*model.FleetMoveCommand{{FleetID: 999999, DestinationStarID: 1}},
	}}
	conn := dialGameUpdates(t, httpServer.URL, g.ID, 2)
	drainInitialUpdate(t, conn)

	response := submit(1, `{"commands":{}}`)
	require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
	require.Contains(t, response.Body.String(), g.Players[1].Name)
	require.Contains(t, response.Body.String(), "resubmit")
	msg, err := readWSJSON(t, conn, 2*time.Second)
	require.NoError(t, err, "dropped AI must receive a readiness update even on HTTP 400")
	update := msg["game_update"].(map[string]any)
	status := update["turn_status"].(map[string]any)
	require.Equal(t, []any{float64(2)}, status["waiting_player_ids"])
	require.Equal(t, []any{float64(1)}, status["ready_player_ids"])

	response = submit(2, `{"commands":{}}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.Contains(t, response.Body.String(), `"advanced":true`)
}

func TestTurnRejectsInvalidCallerOrders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resources, err := game.NewResourceManager()
	require.NoError(t, err)
	g, err := game.NewGame(resources, game.NewGameOptions{
		PlayerName: "Ada", PlayerColor: model.DefaultPlayerColor(),
		NumStars: 8, Race: game.DefaultRace("Human"), NumPlayers: 2,
	})
	require.NoError(t, err)
	s := &Server{gameList: map[string]*game.Game{g.ID: g}, ResourceManager: resources}
	// Another player's accepted submission must survive the caller's error.
	_, _, err = g.SubmitTurn(2, model.Commands{})
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/turn", strings.NewReader(
		`{"commands":{"fleet_move":[{"fleet_id":999999,"destination_star_id":1}]}}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Request.Header.Set(GameIDHeader, g.ID)
	ctx.Request.Header.Set(PlayerIDHeader, "1")
	ctx.Set("server", s)
	Turn(ctx)
	require.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
	require.NotContains(t, g.TurnSubmissions, uint(1))
	require.Contains(t, g.TurnSubmissions, uint(2))
	require.Contains(t, g.TurnStatus.WaitingPlayerIDs, uint(1))
}
