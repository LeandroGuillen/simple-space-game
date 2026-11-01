package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"ssg/game"
	"ssg/model"
)

func testAutosaveGame(t *testing.T, s *Server, players uint) *game.Game {
	t.Helper()
	g, err := game.NewGame(s.ResourceManager, game.NewGameOptions{PlayerName: "Autosave", NumStars: 8, NumPlayers: players, PlayerColor: model.DefaultPlayerColor(), Race: game.DefaultRace("Human")})
	require.NoError(t, err)
	s.putGame(g.ID, g)
	return g
}

func autosaveRequest(s *Server, method, path, gameID, playerID string, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if gameID != "" {
		req.Header.Set(GameIDHeader, gameID)
	}
	if playerID != "" {
		req.Header.Set(PlayerIDHeader, playerID)
	}
	out := httptest.NewRecorder()
	s.Handler().ServeHTTP(out, req)
	return out
}

func TestAutosaveAfterTurnRestoresOnRestart(t *testing.T) {
	t.Setenv("SSG_SAVES_DIR", t.TempDir())
	s, err := NewServer()
	require.NoError(t, err)
	g := testAutosaveGame(t, s, 1)
	out := autosaveRequest(s, http.MethodPost, "/turn", g.ID, "1", []byte(`{"commands":{}}`))
	require.Equal(t, http.StatusOK, out.Code, out.Body.String())
	var response struct {
		Advanced bool   `json:"advanced"`
		Warning  string `json:"warning"`
	}
	require.NoError(t, json.Unmarshal(out.Body.Bytes(), &response))
	require.True(t, response.Advanced)
	require.Empty(t, response.Warning)
	before, err := g.MarshalSave()
	require.NoError(t, err)
	restarted, err := NewServer()
	require.NoError(t, err)
	restored := restarted.getGame(g.ID)
	require.NotNil(t, restored)
	list := autosaveRequest(restarted, http.MethodGet, "/games", "", "", nil)
	require.Equal(t, http.StatusOK, list.Code)
	require.Contains(t, list.Body.String(), g.ID)
	after, err := restored.MarshalSave()
	require.NoError(t, err)
	var beforeState, afterState map[string]any
	require.NoError(t, json.Unmarshal(before, &beforeState))
	require.NoError(t, json.Unmarshal(after, &afterState))
	// Resources are reloaded catalogs, including a newly shuffled name pool.
	delete(beforeState, "resources")
	delete(afterState, "resources")
	require.Equal(t, beforeState, afterState)
	require.Equal(t, g.CurrentTurn, restored.CurrentTurn)
}

func TestAutosaveRotationAndCorruptNewestFallback(t *testing.T) {
	t.Setenv("SSG_SAVES_DIR", t.TempDir())
	s, err := NewServer()
	require.NoError(t, err)
	g := testAutosaveGame(t, s, 1)
	for i := 0; i < 8; i++ {
		g.Players[0].Treasury = float64(100 + i)
		require.NoError(t, s.autosave(g))
	}
	dir := autosaveGameDir(s.autosaves.dir, g.ID)
	files, err := autosaveFiles(dir)
	require.NoError(t, err)
	require.Len(t, files, 5)
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 5, "no temporary files should remain")
	info, err := os.Stat(filepath.Join(dir, files[0]))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm(), "full game state must remain private")
	require.NoError(t, os.WriteFile(filepath.Join(dir, files[0]), []byte("broken"), 0600))
	restarted, err := NewServer()
	require.NoError(t, err)
	restored := restarted.getGame(g.ID)
	require.NotNil(t, restored)
	require.Equal(t, float64(106), restored.Players[0].Treasury)
}

func TestSaveNowChecksHeadersAndResetsUnsavedOrdersAfterRestart(t *testing.T) {
	t.Setenv("SSG_SAVES_DIR", t.TempDir())
	s, err := NewServer()
	require.NoError(t, err)
	g := testAutosaveGame(t, s, 2)
	_, advanced, err := g.SubmitTurn(1, model.Commands{})
	require.NoError(t, err)
	require.False(t, advanced)
	path := "/games/" + g.ID + "/save"
	for _, tc := range []struct {
		gameID, player string
		status         int
	}{{"", "", 400}, {"wrong", "1", 400}, {g.ID, "999", 400}, {g.ID, "1", 200}} {
		out := autosaveRequest(s, http.MethodPost, path, tc.gameID, tc.player, nil)
		require.Equal(t, tc.status, out.Code, out.Body.String())
		require.NotContains(t, out.Body.String(), "treasury")
	}
	restarted, err := NewServer()
	require.NoError(t, err)
	restored := restarted.getGame(g.ID)
	require.NotNil(t, restored)
	require.Empty(t, restored.TurnSubmissions)
	require.Equal(t, []uint{1, 2}, restored.TurnStatus.WaitingPlayerIDs)
	require.Empty(t, restored.TurnStatus.ReadyPlayerIDs)
}

func TestAutosaveSerializesConcurrentRequests(t *testing.T) {
	t.Setenv("SSG_SAVES_DIR", t.TempDir())
	s, err := NewServer()
	require.NoError(t, err)
	g := testAutosaveGame(t, s, 1)
	var wg sync.WaitGroup
	errs := make(chan error, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- s.autosave(g) }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	files, err := autosaveFiles(autosaveGameDir(s.autosaves.dir, g.ID))
	require.NoError(t, err)
	require.Len(t, files, 5)
	restarted, err := NewServer()
	require.NoError(t, err)
	require.NotNil(t, restarted.getGame(g.ID), fmt.Sprint(files))
}

func TestAutosaveFailureReportsWarningWithoutReplayingTurn(t *testing.T) {
	t.Setenv("SSG_SAVES_DIR", t.TempDir())
	s, err := NewServer()
	require.NoError(t, err)
	g := testAutosaveGame(t, s, 1)
	// A regular file cannot contain the autosave directory, even when tests run as root.
	blocked := filepath.Join(t.TempDir(), "not-a-directory")
	require.NoError(t, os.WriteFile(blocked, []byte("x"), 0600))
	s.autosaves.dir = blocked
	before := g.CurrentTurn
	out := autosaveRequest(s, http.MethodPost, "/turn", g.ID, "1", []byte(`{"commands":{}}`))
	require.Equal(t, http.StatusOK, out.Code, out.Body.String())
	var response struct {
		Advanced bool   `json:"advanced"`
		Warning  string `json:"warning"`
	}
	require.NoError(t, json.Unmarshal(out.Body.Bytes(), &response))
	require.True(t, response.Advanced)
	require.Contains(t, response.Warning, "autosave failed")
	require.Greater(t, g.CurrentTurn, before)
	out = autosaveRequest(s, http.MethodPost, "/games/"+g.ID+"/save", g.ID, "1", nil)
	require.Equal(t, http.StatusInternalServerError, out.Code)
}
