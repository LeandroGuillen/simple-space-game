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

func TestLoadUploadedAndScenarioRejectOutOfRangeFields(t *testing.T) {
	// On 2ea7ae1 these fail by assertion: upload/scenario load return 200 for
	// tax_rate 999, unknown buildings, and out-of-range size/environment.
	gin.SetMode(gin.TestMode)
	resources, err := game.NewResourceManager()
	require.NoError(t, err)

	type corruptFn func(g *game.Game)
	cases := []struct {
		name    string
		corrupt corruptFn
		want    string
	}{
		{
			name: "tax_rate",
			corrupt: func(g *game.Game) {
				g.Players[0].TaxRate = 999
			},
			want: "tax_rate",
		},
		{
			name: "unknown_building",
			corrupt: func(g *game.Game) {
				for _, star := range g.Galaxy.Stars {
					for _, p := range star.Planets {
						if p != nil && p.Owner != 0 {
							p.Buildings = []string{"NoSuchBuilding"}
							return
						}
					}
				}
			},
			want: "unknown building",
		},
		{
			name: "planet_size",
			corrupt: func(g *game.Game) {
				for _, star := range g.Galaxy.Stars {
					for _, p := range star.Planets {
						if p != nil && p.Owner != 0 {
							p.Size = 99
							return
						}
					}
				}
			},
			want: "size",
		},
		{
			name: "planet_environment",
			corrupt: func(g *game.Game) {
				for _, star := range g.Galaxy.Stars {
					for _, p := range star.Planets {
						if p != nil && p.Owner != 0 {
							p.Environment = 999
							return
						}
					}
				}
			},
			want: "environment",
		},
	}

	for _, tc := range cases {
		t.Run("upload_"+tc.name, func(t *testing.T) {
			gameState, err := game.NewGame(resources, game.NewGameOptions{
				PlayerName:  "Uploader",
				NumStars:    6,
				PlayerColor: model.DefaultPlayerColor(),
				Race:        game.DefaultRace("Human"),
			})
			require.NoError(t, err)
			tc.corrupt(gameState)
			snapshot := *gameState
			snapshot.Resources = nil
			payload, err := json.Marshal(&snapshot)
			require.NoError(t, err)

			server := &Server{gameList: make(map[string]*game.Game), ResourceManager: resources}
			recorder := postUpload(t, server, bytes.NewReader(payload), "application/json")
			require.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
			require.Contains(t, recorder.Body.String(), tc.want)
			require.Empty(t, server.gameList)
		})

		t.Run("scenario_"+tc.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("SSG_SAVES_DIR", dir)
			scenario := createScenarioFile(t, resources, "bad-"+tc.name)
			path, err := savePath("bad-" + tc.name)
			require.NoError(t, err)
			raw, err := os.ReadFile(path)
			require.NoError(t, err)
			var loaded game.Game
			require.NoError(t, json.Unmarshal(raw, &loaded))
			tc.corrupt(&loaded)
			rewritten, err := json.Marshal(&loaded)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(path, rewritten, 0o644))

			server := &Server{gameList: make(map[string]*game.Game), ResourceManager: resources}
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest("POST", "/scenarios/bad-"+tc.name+"/load", nil)
			ctx.Params = gin.Params{{Key: "name", Value: "bad-" + tc.name}}
			ctx.Set("server", server)
			LoadScenario(ctx)

			require.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
			require.Contains(t, recorder.Body.String(), tc.want)
			require.Nil(t, server.getGame(scenario.ID))
			require.Empty(t, server.gameList)
		})
	}
}

func TestLoadUploadedSaveRejectsUnknownCurrentResearchLevel(t *testing.T) {
	// On 2108d9b this fails by assertion: upload returns 200 despite Bogus Level.
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
	gameState.Players[0].CurrentResearch = &model.ResearchProject{
		Subject:   "Bogus",
		TechName:  "Bogus Tech",
		TechNames: []string{"Bogus Tech"},
		LevelName: "Bogus Level",
	}

	snapshot := *gameState
	snapshot.Resources = nil
	payload, err := json.Marshal(&snapshot)
	require.NoError(t, err)

	server := &Server{gameList: make(map[string]*game.Game), ResourceManager: resources}
	recorder := postUpload(t, server, bytes.NewReader(payload), "application/json")
	require.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
	require.Contains(t, recorder.Body.String(), "Bogus Level")
	require.Empty(t, server.gameList)
}

func TestLoadUploadedSaveRejectsUnknownHyperAdvancedSubject(t *testing.T) {
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
	gameState.Players[0].CurrentResearch = &model.ResearchProject{
		Subject:   "NoSuchSubject",
		TechName:  "Hyper Advanced NoSuchSubject",
		LevelName: "Hyper Advanced 1",
	}

	snapshot := *gameState
	snapshot.Resources = nil
	payload, err := json.Marshal(&snapshot)
	require.NoError(t, err)

	server := &Server{gameList: make(map[string]*game.Game), ResourceManager: resources}
	recorder := postUpload(t, server, bytes.NewReader(payload), "application/json")
	require.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
	require.Contains(t, recorder.Body.String(), "hyper advanced subject")
	require.Empty(t, server.gameList)
}

func TestLoadUploadedSaveWithValidResearchAdvancesTurn(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resources, err := game.NewResourceManager()
	require.NoError(t, err)

	gameState, err := game.NewGame(resources, game.NewGameOptions{
		PlayerName:  "Uploader",
		NumStars:    6,
		PlayerColor: model.DefaultPlayerColor(),
		Race:        game.DefaultRace("Human"),
		NumPlayers:  1,
	})
	require.NoError(t, err)
	gameState.Players[0].CurrentResearch = &model.ResearchProject{
		Subject:   "Electronic Computer",
		TechName:  "Electronic Computer",
		TechNames: []string{"Electronic Computer"},
		LevelName: "Electronics",
		Progress:  3,
	}
	turnBefore := gameState.CurrentTurn

	snapshot := *gameState
	snapshot.Resources = nil
	payload, err := json.Marshal(&snapshot)
	require.NoError(t, err)

	server := &Server{gameList: make(map[string]*game.Game), ResourceManager: resources}
	recorder := postUpload(t, server, bytes.NewReader(payload), "application/json")
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	var response struct {
		GameID string `json:"gameID"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	loaded := server.getGame(response.GameID)
	require.NotNil(t, loaded)
	require.NotNil(t, loaded.Players[0].CurrentResearch)
	require.Equal(t, "Electronics", loaded.Players[0].CurrentResearch.LevelName)

	_, advanced, err := loaded.SubmitTurn(loaded.Players[0].ID, model.Commands{})
	require.NoError(t, err)
	require.True(t, advanced)
	require.Greater(t, loaded.CurrentTurn, turnBefore)
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

func TestLoadUploadedAndScenarioAcceptNegativeTreasury(t *testing.T) {
	// On e0ef594 this fails by assertion: LoadUploadedSave / LoadScenario
	// return 400 for games with negative treasury (reachable via CP overdraw).
	gin.SetMode(gin.TestMode)
	resources, err := game.NewResourceManager()
	require.NoError(t, err)

	makeNegativeTreasuryGame := func(t *testing.T) *game.Game {
		t.Helper()
		g, err := game.NewGame(resources, game.NewGameOptions{
			PlayerName:  "Overdrawn",
			NumStars:    4,
			PlayerColor: model.DefaultPlayerColor(),
			Race:        game.DefaultRace("Human"),
		})
		require.NoError(t, err)
		g.Players[0].Treasury = -5
		return g
	}

	t.Run("upload", func(t *testing.T) {
		g := makeNegativeTreasuryGame(t)
		snapshot := *g
		snapshot.Resources = nil
		payload, err := json.Marshal(&snapshot)
		require.NoError(t, err)

		server := &Server{gameList: make(map[string]*game.Game), ResourceManager: resources}
		recorder := postUpload(t, server, bytes.NewReader(payload), "application/json")
		require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
		var response struct {
			GameID string `json:"gameID"`
		}
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
		require.NotEmpty(t, response.GameID)
		require.NotNil(t, server.getGame(response.GameID))
	})

	t.Run("scenario", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("SSG_SAVES_DIR", dir)
		scenario := createScenarioFile(t, resources, "neg-treasury")
		path, err := savePath("neg-treasury")
		require.NoError(t, err)

		g := makeNegativeTreasuryGame(t)
		snapshot := *g
		snapshot.Resources = nil
		snapshot.ID = scenario.ID
		rewritten, err := json.Marshal(&snapshot)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(path, rewritten, 0o644))

		server := &Server{gameList: make(map[string]*game.Game), ResourceManager: resources}
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest("POST", "/scenarios/neg-treasury/load", nil)
		ctx.Params = gin.Params{{Key: "name", Value: "neg-treasury"}}
		ctx.Set("server", server)
		LoadScenario(ctx)

		require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
		require.NotNil(t, server.getGame(scenario.ID))
	})
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

func TestGameSaveUploadRoundTripPreservesFullState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server, err := NewServer()
	require.NoError(t, err)

	g, err := game.NewGame(server.ResourceManager, game.NewGameOptions{
		PlayerName:  "RoundTrip",
		NumStars:    8,
		NumPlayers:  3,
		PlayerColor: model.DefaultPlayerColor(),
		Race:        game.DefaultRace("Human"),
	})
	require.NoError(t, err)

	for i, player := range g.Players {
		player.Treasury = 1000.5 + float64(i)*250
		player.ResearchedTechs = append(player.ResearchedTechs, "Anti-Missile Rockets")
		var homeStarID uint
		for _, star := range g.Galaxy.Stars {
			for _, planet := range star.Planets {
				if planet.Owner == player.ID {
					homeStarID = star.ID
					break
				}
			}
			if homeStarID != 0 {
				break
			}
		}
		require.NotZero(t, homeStarID)
		fleetID := uint(i + 1)
		g.Fleets = append(g.Fleets, &model.Fleet{
			ID:     fleetID,
			Owner:  player.ID,
			StarID: homeStarID,
			Ships:  []model.FleetShip{{Type: "scout", Name: "Scout"}},
		})
		player.Fleets = append(player.Fleets, fleetID)
	}
	server.putGame(g.ID, g)

	performRequest := func(method, path string, body io.Reader) *httptest.ResponseRecorder {
		t.Helper()
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, body)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		server.Handler().ServeHTTP(recorder, req)
		return recorder
	}

	exported := performRequest(http.MethodGet, "/games/"+g.ID+"/save", nil)
	require.Equal(t, http.StatusOK, exported.Code, exported.Body.String())
	expected, err := json.Marshal(g)
	require.NoError(t, err)
	require.JSONEq(t, string(expected), exported.Body.String())

	uploaded := performRequest(http.MethodPost, "/saves/upload", bytes.NewReader(exported.Body.Bytes()))
	require.Equal(t, http.StatusOK, uploaded.Code, uploaded.Body.String())
	var uploadResponse struct {
		GameID string `json:"gameID"`
	}
	require.NoError(t, json.Unmarshal(uploaded.Body.Bytes(), &uploadResponse))
	require.NotEmpty(t, uploadResponse.GameID)
	require.NotEqual(t, g.ID, uploadResponse.GameID)

	reexported := performRequest(http.MethodGet, "/games/"+uploadResponse.GameID+"/save", nil)
	require.Equal(t, http.StatusOK, reexported.Code, reexported.Body.String())
	var before, after map[string]any
	require.NoError(t, json.Unmarshal(exported.Body.Bytes(), &before))
	require.NoError(t, json.Unmarshal(reexported.Body.Bytes(), &after))
	before["id"] = ""
	after["id"] = ""
	require.Equal(t, before, after, "only the newly assigned game ID may change")
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
