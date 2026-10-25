package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"ssg/game"
	"ssg/model"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestListScenarios(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	t.Setenv("SSG_SAVES_DIR", dir)

	resources, err := game.NewResourceManager()
	require.NoError(t, err)
	createScenarioFile(t, resources, "scenario-alpha")

	server := &Server{
		gameList:        make(map[string]*game.Game),
		ResourceManager: resources,
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("GET", "/scenarios", nil)
	ctx.Set("server", server)

	ListScenarios(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload struct {
		Scenarios []SavedGameInfo `json:"scenarios"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Len(t, payload.Scenarios, 1)
	require.Equal(t, "scenario-alpha", payload.Scenarios[0].Name)
	require.NotEmpty(t, payload.Scenarios[0].GameID)
}

func TestLoadScenarioAddsGame(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	t.Setenv("SSG_SAVES_DIR", dir)

	resources, err := game.NewResourceManager()
	require.NoError(t, err)
	scenario := createScenarioFile(t, resources, "scenario-beta")

	server := &Server{
		gameList:        make(map[string]*game.Game),
		ResourceManager: resources,
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest("POST", "/scenarios/scenario-beta/load", nil)
	ctx.Request = req
	ctx.Params = gin.Params{{Key: "name", Value: "scenario-beta"}}
	ctx.Set("server", server)

	LoadScenario(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.NotNil(t, server.getGame(scenario.ID))
}

func TestLoadUploadedSaveAddsGame(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	t.Setenv("SSG_SAVES_DIR", dir)

	resources, err := game.NewResourceManager()
	require.NoError(t, err)

	gameState, err := game.NewGame(resources, game.NewGameOptions{
		PlayerName:  "Uploader",
		NumStars:    6,
		PlayerColor: model.DefaultPlayerColor(),
		Race:        game.DefaultRace("Human"),
	})
	require.NoError(t, err)

	snapshot := *gameState
	snapshot.Resources = nil
	payload, err := json.Marshal(&snapshot)
	require.NoError(t, err)

	server := &Server{
		gameList:        make(map[string]*game.Game),
		ResourceManager: resources,
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest("POST", "/saves/upload", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req
	ctx.Set("server", server)

	LoadUploadedSave(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		GameID string `json:"gameID"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.NotEmpty(t, response.GameID)
	require.NotEqual(t, gameState.ID, response.GameID)
	require.NotNil(t, server.getGame(response.GameID))
	require.Nil(t, server.getGame(gameState.ID))
}

func createScenarioFile(t *testing.T, resources *game.ResourceManager, name string) *game.Game {
	t.Helper()
	instance, err := game.NewGame(resources, game.NewGameOptions{
		PlayerName:  "Scenario",
		NumStars:    6,
		PlayerColor: model.DefaultPlayerColor(),
		Race:        game.DefaultRace("Human"),
	})
	require.NoError(t, err)

	snapshot := *instance
	snapshot.Resources = nil
	data, err := json.Marshal(&snapshot)
	require.NoError(t, err)

	path, err := savePath(name)
	require.NoError(t, err)
	dir := filepath.Dir(path)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(path, data, 0o644))
	return instance
}
