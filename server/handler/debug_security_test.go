package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"ssg/game"
	"ssg/model"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestGame(t *testing.T, s *Server) *game.Game {
	t.Helper()
	g, err := game.NewGame(s.ResourceManager, game.NewGameOptions{
		PlayerName:  "Ada",
		PlayerColor: model.DefaultPlayerColor(),
		NumStars:    4,
		Race:        game.DefaultRace("Human"),
		NumPlayers:  1,
	})
	require.NoError(t, err)
	s.putGame(g.ID, g)
	return g
}

func postDebugResearchAll(t *testing.T, s *Server, g *game.Game) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/debug/research-all", bytes.NewReader([]byte(`{}`)))
	c.Request.Header.Set(GameIDHeader, g.ID)
	c.Request.Header.Set(PlayerIDHeader, "1")
	c.Set("server", s)
	DebugResearchAllTechs(c)
	return w
}

func getStatus(t *testing.T, s *Server) map[string]any {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/status", nil)
	c.Set("server", s)
	Status(c)
	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body
}

func TestStatusReportsDebugOffByDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s, err := NewServer()
	require.NoError(t, err)
	body := getStatus(t, s)
	assert.Equal(t, true, body["ok"])
	assert.Equal(t, false, body["debug"])
}

func TestStatusReportsDebugOnWhenEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s, err := NewServerWithOptions(ServerOptions{Debug: true})
	require.NoError(t, err)
	body := getStatus(t, s)
	assert.Equal(t, true, body["ok"])
	assert.Equal(t, true, body["debug"])
}

func TestDebugResearchAllUnavailableWithoutDebugFlag(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s, err := NewServer()
	require.NoError(t, err)
	require.False(t, s.DebugEnabled())

	httpServer := httptest.NewServer(s.Handler())
	t.Cleanup(httpServer.Close)

	g := newTestGame(t, s)
	beforeTechs := append([]string(nil), g.Players[0].ResearchedTechs...)
	require.False(t, g.Debug)

	req, err := http.NewRequest(http.MethodPost, httpServer.URL+"/debug/research-all", bytes.NewReader([]byte(`{}`)))
	require.NoError(t, err)
	req.Header.Set(GameIDHeader, g.ID)
	req.Header.Set(PlayerIDHeader, "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	assert.Equal(t, beforeTechs, g.Players[0].ResearchedTechs)
	assert.False(t, g.Debug)
}

func TestDebugResearchAllWorksWhenEnabledAndFlagsGame(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s, err := NewServerWithOptions(ServerOptions{Debug: true})
	require.NoError(t, err)
	require.True(t, s.DebugEnabled())

	g := newTestGame(t, s)
	beforeCount := len(g.Players[0].ResearchedTechs)

	w := postDebugResearchAll(t, s, g)
	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, g.Debug)
	assert.Greater(t, len(g.Players[0].ResearchedTechs), beforeCount)

	listW := httptest.NewRecorder()
	listC, _ := gin.CreateTestContext(listW)
	listC.Request = httptest.NewRequest(http.MethodGet, "/games", nil)
	listC.Set("server", s)
	GetGameList(listC)
	require.Equal(t, http.StatusOK, listW.Code)

	var listBody struct {
		GameList []GameListResponseItem `json:"game_list"`
	}
	require.NoError(t, json.Unmarshal(listW.Body.Bytes(), &listBody))
	var found *GameListResponseItem
	for i := range listBody.GameList {
		if listBody.GameList[i].GameID == g.ID {
			found = &listBody.GameList[i]
			break
		}
	}
	require.NotNil(t, found)
	assert.True(t, found.Debug)
}

func TestDebugFlagSurvivesSaveLoadRoundTrip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	t.Setenv("SSG_SAVES_DIR", dir)

	s, err := NewServerWithOptions(ServerOptions{Debug: true})
	require.NoError(t, err)
	g := newTestGame(t, s)
	w := postDebugResearchAll(t, s, g)
	require.Equal(t, http.StatusOK, w.Code)
	require.True(t, g.Debug)

	snapshot := *g
	snapshot.Resources = nil
	payload, err := json.Marshal(&snapshot)
	require.NoError(t, err)
	path, err := savePath("debug-flag")
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, payload, 0o644))

	s2, err := NewServer()
	require.NoError(t, err)
	loadW := httptest.NewRecorder()
	loadC, _ := gin.CreateTestContext(loadW)
	loadC.Params = gin.Params{{Key: "name", Value: "debug-flag"}}
	loadC.Request = httptest.NewRequest(http.MethodPost, "/scenarios/debug-flag/load", nil)
	loadC.Set("server", s2)
	LoadScenario(loadC)
	require.Equal(t, http.StatusOK, loadW.Code)

	loaded := s2.getGame(scenarioResponseGameID(t, loadW))
	require.NotNil(t, loaded)
	assert.True(t, loaded.Debug, "debug flag must survive save/load")
}

func TestRemovedActionEndpointsReturn404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s, err := NewServer()
	require.NoError(t, err)
	httpServer := httptest.NewServer(s.Handler())
	t.Cleanup(httpServer.Close)

	g := newTestGame(t, s)
	exploredBefore := append([]uint(nil), g.Players[0].ExploredStars...)
	starID := g.Players[0].ExploredStars[0]
	// Pick an unexplored star if possible.
	for _, star := range g.Galaxy.Stars {
		found := false
		for _, id := range exploredBefore {
			if id == star.ID {
				found = true
				break
			}
		}
		if !found {
			starID = star.ID
			break
		}
	}

	exploreURL := fmt.Sprintf("%s/action/1/explore/%d", httpServer.URL, starID)
	req, err := http.NewRequest(http.MethodPost, exploreURL, bytes.NewReader([]byte(fmt.Sprintf(`{"game_id":%q}`, g.ID))))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Equal(t, exploredBefore, g.Players[0].ExploredStars)

	var planetID uint = 1
	for _, star := range g.Galaxy.Stars {
		for _, planet := range star.Planets {
			if planet != nil && planet.Owner == 0 {
				planetID = planet.ID
				break
			}
		}
	}
	colonizeURL := fmt.Sprintf("%s/action/1/colonize/%d", httpServer.URL, planetID)
	req, err = http.NewRequest(http.MethodPost, colonizeURL, bytes.NewReader([]byte(fmt.Sprintf(`{"game_id":%q}`, g.ID))))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Equal(t, exploredBefore, g.Players[0].ExploredStars)
}
