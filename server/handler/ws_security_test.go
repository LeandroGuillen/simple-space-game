package server

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"ssg/game"
	"ssg/model"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func dialGameUpdates(t *testing.T, serverURL, gameID string, playerID uint) *websocket.Conn {
	t.Helper()
	u, err := url.Parse(serverURL)
	require.NoError(t, err)
	u.Scheme = "ws"
	u.Path = "/gameupdates"
	q := u.Query()
	q.Set("game_id", gameID)
	q.Set("player_id", fmt.Sprintf("%d", playerID))
	u.RawQuery = q.Encode()

	conn, resp, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		if resp != nil {
			t.Fatalf("dial %s: %v (HTTP %d)", u.String(), err, resp.StatusCode)
		}
		t.Fatalf("dial %s: %v", u.String(), err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func readWSJSON(t *testing.T, conn *websocket.Conn, timeout time.Duration) (map[string]any, error) {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	defer func() { _ = conn.SetReadDeadline(time.Time{}) }()
	var msg map[string]any
	err := conn.ReadJSON(&msg)
	return msg, err
}

func drainInitialUpdate(t *testing.T, conn *websocket.Conn) {
	t.Helper()
	msg, err := readWSJSON(t, conn, 2*time.Second)
	require.NoError(t, err, "expected initial game_update on connect")
	_, hasUpdate := msg["game_update"]
	require.True(t, hasUpdate, "initial message must be a game_update, got %#v", msg)
}

func gameUpdateID(t *testing.T, msg map[string]any) string {
	t.Helper()
	raw, ok := msg["game_update"]
	require.True(t, ok, "missing game_update in %#v", msg)
	update, ok := raw.(map[string]any)
	require.True(t, ok, "game_update is not an object: %#v", raw)
	id, _ := update["id"].(string)
	return id
}

func TestWebsocketClientMessagesAreNotRebroadcast(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s, err := NewServer()
	require.NoError(t, err)

	resources := s.ResourceManager
	g1, err := game.NewGame(resources, game.NewGameOptions{
		PlayerName:  "Ada",
		PlayerColor: model.DefaultPlayerColor(),
		NumStars:    4,
		Race:        game.DefaultRace("Human"),
		NumPlayers:  2,
	})
	require.NoError(t, err)
	g2, err := game.NewGame(resources, game.NewGameOptions{
		PlayerName:  "Bob",
		PlayerColor: model.DefaultPlayerColor(),
		NumStars:    4,
		Race:        game.DefaultRace("Human"),
		NumPlayers:  1,
	})
	require.NoError(t, err)
	s.putGame(g1.ID, g1)
	s.putGame(g2.ID, g2)

	httpServer := httptest.NewServer(s.Handler())
	t.Cleanup(httpServer.Close)

	connA := dialGameUpdates(t, httpServer.URL, g1.ID, 1)
	connB := dialGameUpdates(t, httpServer.URL, g1.ID, 2)
	connOtherGame := dialGameUpdates(t, httpServer.URL, g2.ID, 1)

	drainInitialUpdate(t, connA)
	drainInitialUpdate(t, connB)
	drainInitialUpdate(t, connOtherGame)

	spoof := map[string]any{
		"game_update": map[string]any{
			"id":   "X",
			"turn": 999,
		},
	}
	require.NoError(t, connA.WriteJSON(spoof))

	// Do not ReadJSON-with-deadline to assert "no message" — a timed-out gorilla
	// read permanently breaks the connection. Wait, then require the next
	// message (from a real server broadcast) is not the spoof.
	time.Sleep(300 * time.Millisecond)

	s.hub.BroadcastGameUpdate(g1)

	msg, err := readWSJSON(t, connB, 2*time.Second)
	require.NoError(t, err, "client B must receive real gameUpdates broadcast")
	require.Equal(t, g1.ID, gameUpdateID(t, msg), "B must not receive A's spoofed game_update")

	// Other-game client must not have received the spoof or g1's update; the
	// next message after a g2 server broadcast must be g2 itself.
	s.hub.BroadcastGameUpdate(g2)
	otherMsg, err := readWSJSON(t, connOtherGame, 2*time.Second)
	require.NoError(t, err, "other-game client must still receive its own updates")
	require.Equal(t, g2.ID, gameUpdateID(t, otherMsg), "other-game client must not see spoof or g1 updates first")
}

func TestWebsocketOversizedFrameClosesSenderWithoutForwarding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s, err := NewServer()
	require.NoError(t, err)

	g, err := game.NewGame(s.ResourceManager, game.NewGameOptions{
		PlayerName:  "Ada",
		PlayerColor: model.DefaultPlayerColor(),
		NumStars:    4,
		Race:        game.DefaultRace("Human"),
		NumPlayers:  2,
	})
	require.NoError(t, err)
	s.putGame(g.ID, g)

	httpServer := httptest.NewServer(s.Handler())
	t.Cleanup(httpServer.Close)

	connA := dialGameUpdates(t, httpServer.URL, g.ID, 1)
	connB := dialGameUpdates(t, httpServer.URL, g.ID, 2)
	drainInitialUpdate(t, connA)
	drainInitialUpdate(t, connB)

	oversized := strings.Repeat("x", 2048)
	payload, err := json.Marshal(map[string]any{
		"game_update": map[string]any{"id": "X", "blob": oversized},
	})
	require.NoError(t, err)
	require.Greater(t, len(payload), 512)

	require.NoError(t, connA.WriteMessage(websocket.TextMessage, payload))

	// Sender should be closed by the server read-limit enforcement.
	_ = connA.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, readErr := connA.ReadMessage()
	require.Error(t, readErr, "oversized frame must close sender connection")

	// Give the hub a moment to discard/close A; do not deadline-read on B.
	time.Sleep(300 * time.Millisecond)

	// Hub still delivers real updates to the surviving client, and that update
	// must not be the oversized spoof payload.
	s.hub.BroadcastGameUpdate(g)
	msg, err := readWSJSON(t, connB, 2*time.Second)
	require.NoError(t, err, "client B must still receive real updates")
	require.Equal(t, g.ID, gameUpdateID(t, msg))
}
