package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func TestLoadScenarioRejectsInvalidGame(t *testing.T) {
	// On 5d21dc9 this fails by assertion: old LoadScenario returns 200 and
	// registers the game despite a fleet pointing at a missing star.
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	t.Setenv("SSG_SAVES_DIR", dir)

	resources, err := game.NewResourceManager()
	require.NoError(t, err)
	scenario := createScenarioFile(t, resources, "bad-fleet-star")
	path, err := savePath("bad-fleet-star")
	require.NoError(t, err)
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var loaded game.Game
	require.NoError(t, json.Unmarshal(raw, &loaded))
	require.NotEmpty(t, loaded.Players)
	// Ensure a fleet exists, then point it at a star that does not.
	loaded.Fleets = []*model.Fleet{{
		ID:     1,
		Owner:  loaded.Players[0].ID,
		StarID: 999999,
		Ships:  []model.FleetShip{{Name: "Scout", Type: "scout"}},
	}}
	loaded.Players[0].Fleets = []uint{1}
	rewritten, err := json.Marshal(&loaded)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, rewritten, 0o644))

	server := &Server{
		gameList:        make(map[string]*game.Game),
		ResourceManager: resources,
	}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("POST", "/scenarios/bad-fleet-star/load", nil)
	ctx.Params = gin.Params{{Key: "name", Value: "bad-fleet-star"}}
	ctx.Set("server", server)
	LoadScenario(ctx)

	require.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
	var payload struct {
		Error string `json:"error"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Contains(t, payload.Error, "missing star")
	require.Nil(t, server.getGame(scenario.ID))
	require.Empty(t, server.gameList)
}

func TestLoadScenarioAcceptsShippedScenarios(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resources, err := game.NewResourceManager()
	require.NoError(t, err)

	scenarioDir := filepath.Join("..", "data", "scenarios")
	entries, err := os.ReadDir(scenarioDir)
	require.NoError(t, err)

	dir := t.TempDir()
	t.Setenv("SSG_SAVES_DIR", dir)
	found := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		found++
		name := strings.TrimSuffix(entry.Name(), ".json")
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(scenarioDir, entry.Name()))
			require.NoError(t, err)
			path, err := savePath(name)
			require.NoError(t, err)
			require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
			require.NoError(t, os.WriteFile(path, data, 0o644))

			var header struct {
				ID string `json:"id"`
			}
			require.NoError(t, json.Unmarshal(data, &header))
			require.NotEmpty(t, header.ID)

			server := &Server{
				gameList:        make(map[string]*game.Game),
				ResourceManager: resources,
			}
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest("POST", "/scenarios/"+name+"/load", nil)
			ctx.Params = gin.Params{{Key: "name", Value: name}}
			ctx.Set("server", server)
			LoadScenario(ctx)

			require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
			require.NotNil(t, server.getGame(header.ID))
		})
	}
	require.GreaterOrEqual(t, found, 3, "expected shipped scenarios under data/scenarios")
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

func TestLoadUploadedSaveClearsPendingInvasions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resources, err := game.NewResourceManager()
	require.NoError(t, err)

	gameState, err := game.NewGame(resources, game.NewGameOptions{
		PlayerName:  "Uploader",
		NumStars:    6,
		PlayerColor: model.DefaultPlayerColor(),
		Race:        game.DefaultRace("Human"),
	})
	require.NoError(t, err)
	gameState.PendingInvasions = []game.PendingInvasion{{
		OwnerID: 1, FleetID: 9, TargetPlanetID: 3,
	}}

	snapshot := *gameState
	snapshot.Resources = nil
	payload, err := json.Marshal(&snapshot)
	require.NoError(t, err)

	server := &Server{gameList: make(map[string]*game.Game), ResourceManager: resources}
	recorder := postUpload(t, server, bytes.NewReader(payload), "application/json")
	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		GameID string `json:"gameID"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	loaded := server.getGame(response.GameID)
	require.NotNil(t, loaded)
	require.Empty(t, loaded.PendingInvasions)
}

func TestLoadScenarioClearsPendingInvasions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	t.Setenv("SSG_SAVES_DIR", dir)

	resources, err := game.NewResourceManager()
	require.NoError(t, err)
	instance := createScenarioFile(t, resources, "with-pending-invade")
	// Inject pending invasions into the on-disk scenario JSON.
	path, err := savePath("with-pending-invade")
	require.NoError(t, err)
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var encoded map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &encoded))
	pending, err := json.Marshal([]game.PendingInvasion{{OwnerID: 1, FleetID: 3, TargetPlanetID: 2}})
	require.NoError(t, err)
	encoded["pending_invasions"] = pending
	rewritten, err := json.Marshal(encoded)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, rewritten, 0o644))

	server := &Server{gameList: make(map[string]*game.Game), ResourceManager: resources}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("POST", "/scenarios/with-pending-invade/load", nil)
	ctx.Params = gin.Params{{Key: "name", Value: "with-pending-invade"}}
	ctx.Set("server", server)
	LoadScenario(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	loaded := server.getGame(instance.ID)
	require.NotNil(t, loaded)
	require.Empty(t, loaded.PendingInvasions)
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

func postUpload(t *testing.T, server *Server, body io.Reader, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodPost, "/saves/upload", body)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	ctx.Request = req
	ctx.Set("server", server)
	LoadUploadedSave(ctx)
	return recorder
}

func TestLoadUploadedSaveRejectsOversizedBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resources, err := game.NewResourceManager()
	require.NoError(t, err)
	server := &Server{gameList: make(map[string]*game.Game), ResourceManager: resources}

	// Just over the limit; MaxBytesReader must reject before full buffering.
	payload := bytes.Repeat([]byte("a"), MaxUploadedSaveBytes+1)
	recorder := postUpload(t, server, bytes.NewReader(payload), "application/json")
	require.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)
	require.Contains(t, recorder.Body.String(), "limit")
	require.Empty(t, server.gameList)
}

func TestLoadUploadedSaveRejectsMalformedJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resources, err := game.NewResourceManager()
	require.NoError(t, err)
	server := &Server{gameList: make(map[string]*game.Game), ResourceManager: resources}

	recorder := postUpload(t, server, strings.NewReader("{not-json"), "application/json")
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "cannot decode upload")
	require.Empty(t, server.gameList)
}

func TestLoadUploadedSaveRejectsInvalidGames(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resources, err := game.NewResourceManager()
	require.NoError(t, err)

	cases := []struct {
		name   string
		mutate func(*game.Game)
		reason string
	}{
		{
			name: "negative population",
			mutate: func(g *game.Game) {
				for _, star := range g.Galaxy.Stars {
					for _, p := range star.Planets {
						if p != nil && p.Owner != 0 {
							p.Population = -5
							return
						}
					}
				}
			},
			reason: "negative population",
		},
		{
			name: "unknown tech",
			mutate: func(g *game.Game) {
				g.Players[0].ResearchedTechs = append(g.Players[0].ResearchedTechs, "Bogus Tech Name")
			},
			reason: "unknown tech",
		},
		{
			name: "duplicate player id",
			mutate: func(g *game.Game) {
				dup := *g.Players[0]
				g.Players = append(g.Players, &dup)
			},
			reason: "duplicate player id",
		},
		{
			name: "no players",
			mutate: func(g *game.Game) {
				g.Players = nil
			},
			reason: "no players",
		},
		{
			name: "negative current turn",
			mutate: func(g *game.Game) {
				g.CurrentTurn = -10
			},
			reason: "current turn",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g, err := game.NewGame(resources, game.NewGameOptions{
				PlayerName:  "Bad",
				NumStars:    6,
				PlayerColor: model.DefaultPlayerColor(),
				Race:        game.DefaultRace("Human"),
			})
			require.NoError(t, err)
			tc.mutate(g)
			snapshot := *g
			snapshot.Resources = nil
			payload, err := json.Marshal(&snapshot)
			require.NoError(t, err)

			server := &Server{gameList: make(map[string]*game.Game), ResourceManager: resources}
			recorder := postUpload(t, server, bytes.NewReader(payload), "application/json")
			require.Equal(t, http.StatusBadRequest, recorder.Code)
			require.Contains(t, recorder.Body.String(), tc.reason)
			require.Empty(t, server.gameList)
		})
	}
}

func TestLoadUploadedSaveRoundTripsFreshExportAndScenarios(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resources, err := game.NewResourceManager()
	require.NoError(t, err)

	t.Run("fresh export", func(t *testing.T) {
		g, err := game.NewGame(resources, game.NewGameOptions{
			PlayerName:  "RoundTrip",
			NumStars:    6,
			PlayerColor: model.DefaultPlayerColor(),
			Race:        game.DefaultRace("Human"),
		})
		require.NoError(t, err)
		snapshot := *g
		snapshot.Resources = nil
		payload, err := json.Marshal(&snapshot)
		require.NoError(t, err)

		server := &Server{gameList: make(map[string]*game.Game), ResourceManager: resources}
		recorder := postUpload(t, server, bytes.NewReader(payload), "application/json")
		require.Equal(t, http.StatusOK, recorder.Code)
		var response struct {
			GameID string `json:"gameID"`
		}
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
		require.NotNil(t, server.getGame(response.GameID))
	})

	scenarioDir := filepath.Join("..", "data", "scenarios")
	entries, err := os.ReadDir(scenarioDir)
	require.NoError(t, err)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		name := entry.Name()
		t.Run("scenario "+name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(scenarioDir, name))
			require.NoError(t, err)
			server := &Server{gameList: make(map[string]*game.Game), ResourceManager: resources}
			recorder := postUpload(t, server, bytes.NewReader(data), "application/json")
			require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
			var response struct {
				GameID string `json:"gameID"`
			}
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
			require.NotNil(t, server.getGame(response.GameID))
		})
	}
}

func TestStatusVersionHasNoNewline(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s, err := NewServer()
	require.NoError(t, err)
	body := getStatus(t, s)
	version, ok := body["version"].(string)
	require.True(t, ok)
	require.NotEmpty(t, version)
	require.Equal(t, strings.TrimSpace(version), version)
	require.NotContains(t, version, "\n")
}
