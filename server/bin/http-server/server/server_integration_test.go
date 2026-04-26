package server_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	serverpkg "ssg/bin/http-server/server"
	"ssg/bin/http-server/server/testharness"
	"ssg/game"
	"ssg/game/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testPlayerID uint = 1

func TestScenario_CreateGameAndFetchState(t *testing.T) {
	ts := testharness.NewTestServer(t)
	client := testharness.NewClient(ts.BaseURL)

	status, err := client.Status()
	require.NoError(t, err)
	assert.True(t, status.OK)

	gameID, err := client.CreateGame("integration-player", 2)
	require.NoError(t, err)
	assert.NotEmpty(t, gameID)

	gameList, err := client.GetGameList()
	require.NoError(t, err)
	assert.Contains(t, extractGameIDs(gameList), gameID)

	gotGame, err := client.GetGame(gameID)
	require.NoError(t, err)
	require.Len(t, gotGame.Players, 1)
	assert.Equal(t, "integration-player", gotGame.Players[0].Name)
	assert.Equal(t, int64(3500), gotGame.CurrentTurn)
	assert.Equal(t, gameID, gotGame.ID)
	assert.NotEmpty(t, gotGame.Galaxy.Stars)
	assert.NotNil(t, gotGame.Resources.ShipPartsCollection)
}

func TestScenario_BuildOnPlanet_BroadcastsUpdate(t *testing.T) {
	ts := testharness.NewTestServer(t)
	client := testharness.NewClient(ts.BaseURL)

	gameID, err := client.CreateGame("builder", 2)
	require.NoError(t, err)

	colonies, err := client.GetColonyList(gameID, testPlayerID)
	require.NoError(t, err)
	require.NotEmpty(t, colonies)

	targetPlanet := colonies[0]
	updates := testharness.ConnectGameUpdates(t, ts.WSURL)

	err = client.BuildOnPlanet(gameID, testPlayerID, targetPlanet.ID, 1)
	require.NoError(t, err)

	update := updates.WaitForGameUpdate(t, func(g game.Game) bool {
		return g.ID == gameID && g.CurrentTurn == 3500
	})

	updatedPlanet := findPlanet(t, update.GameUpdate, targetPlanet.ID)
	assert.True(t, updatedPlanet.IsBuilding)
	assert.Equal(t, int64(60), updatedPlanet.BuildBucket)
	assert.Equal(t, "Automated Factory", updatedPlanet.CurrentProject.Name)

	liveGame, err := client.GetGame(gameID)
	require.NoError(t, err)
	assert.Equal(t, updatedPlanet.CurrentProject.Name, findPlanet(t, *liveGame, targetPlanet.ID).CurrentProject.Name)
}

func TestScenario_Turn_AdvancesState_AndBroadcasts(t *testing.T) {
	ts := testharness.NewTestServer(t)
	client := testharness.NewClient(ts.BaseURL)

	gameID, err := client.CreateGame("governor", 2)
	require.NoError(t, err)

	updates := testharness.ConnectGameUpdates(t, ts.WSURL)

	err = client.Turn(gameID, testPlayerID)
	require.NoError(t, err)

	update := updates.WaitForGameUpdate(t, func(g game.Game) bool {
		return g.ID == gameID && g.CurrentTurn == 3501
	})

	require.Len(t, update.GameUpdate.Players, 1)
	assert.Equal(t, int64(3501), update.GameUpdate.CurrentTurn)
	assert.Equal(t, int64(2), update.GameUpdate.Players[0].Treasury)
}

func TestScenario_ShipDesign_CreateAndUpdate(t *testing.T) {
	ts := testharness.NewTestServer(t)
	client := testharness.NewClient(ts.BaseURL)

	gameID, err := client.CreateGame("designer", 2)
	require.NoError(t, err)

	parts, err := client.GetShipParts(gameID, testPlayerID)
	require.NoError(t, err)

	createReq := testharness.CreateShipDesignRequest{
		Name:     "Scout Mk I",
		Drive:    parts.Drives[0].Name,
		Computer: parts.Computers[0].Name,
		Shield:   parts.Shields[0].Name,
		Armor:    parts.Armors[0].Name,
		HullSize: parts.HullSizes[0].Name,
		FuelCell: parts.FuelCells[0].Name,
	}

	updates := testharness.ConnectGameUpdates(t, ts.WSURL)

	err = client.CreateShipDesign(gameID, testPlayerID, createReq)
	require.NoError(t, err)

	createUpdate := updates.WaitForGameUpdate(t, func(g game.Game) bool {
		return g.ID == gameID && gameHasShipDesign(g, createReq.Name)
	})
	assert.True(t, gameHasShipDesign(createUpdate.GameUpdate, createReq.Name))

	designs, err := client.GetShipDesigns(gameID, testPlayerID)
	require.NoError(t, err)
	assert.True(t, containsShipDesign(designs, createReq.Name))

	updateReq := createReq
	updateReq.Drive = parts.Drives[1].Name
	updateReq.Computer = parts.Computers[1].Name

	err = client.UpdateShipDesign(gameID, testPlayerID, updateReq)
	require.NoError(t, err)

	updateMsg := updates.WaitForGameUpdate(t, func(g game.Game) bool {
		d, ok := findShipDesign(g, updateReq.Name)
		return ok && d.Drive.Name == updateReq.Drive && d.Computer.Name == updateReq.Computer
	})

	updatedDesign, ok := findShipDesign(updateMsg.GameUpdate, updateReq.Name)
	require.True(t, ok)
	assert.Equal(t, updateReq.Drive, updatedDesign.Drive.Name)
	assert.Equal(t, updateReq.Computer, updatedDesign.Computer.Name)
}

func TestScenario_InvalidHeadersOrIDs_ReturnErrors(t *testing.T) {
	ts := testharness.NewTestServer(t)
	client := testharness.NewClient(ts.BaseURL)

	gameID, err := client.CreateGame("negative-paths", 2)
	require.NoError(t, err)

	resp, body := doRequest(t, http.MethodGet, ts.BaseURL+"/turn", nil, nil)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, body, serverpkg.GameIDHeader+" header missing")

	missingPlayerHeaders := http.Header{}
	missingPlayerHeaders.Set(serverpkg.GameIDHeader, gameID)
	resp, body = doRequest(t, http.MethodGet, ts.BaseURL+"/turn", missingPlayerHeaders, nil)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, body, serverpkg.PlayerIDHeader+" header missing")

	unknownGameHeaders := http.Header{}
	unknownGameHeaders.Set(serverpkg.GameIDHeader, "missing-game")
	unknownGameHeaders.Set(serverpkg.PlayerIDHeader, "1")
	resp, body = doRequest(t, http.MethodGet, ts.BaseURL+"/turn", unknownGameHeaders, nil)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Contains(t, body, "game with id missing-game was not found")

	resp, body = doRequest(t, http.MethodPost, ts.BaseURL+"/action/not-a-number/colonize/1", nil, map[string]any{
		"game_id": gameID,
	})
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, body, "owner ID is invalid or not present")

	invalidShipHeaders := http.Header{}
	invalidShipHeaders.Set(serverpkg.GameIDHeader, gameID)
	invalidShipHeaders.Set(serverpkg.PlayerIDHeader, "1")
	resp, body = doRequest(t, http.MethodPost, ts.BaseURL+"/resources/shipdesigns", invalidShipHeaders, map[string]any{
		"name":      "Broken",
		"drive":     "Nope",
		"computer":  "None",
		"shield":    "None",
		"armor":     "Titanium",
		"hull_size": "Frigate",
		"fuel_cell": "Standard",
	})
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, body, `"message":"drive with name \"Nope\" does not exist"`)
}

func TestScenario_ColonizePlanet_DoesNotBroadcastUpdate(t *testing.T) {
	ts := testharness.NewTestServer(t)
	client := testharness.NewClient(ts.BaseURL)

	gameID, err := client.CreateGame("colonizer", 2)
	require.NoError(t, err)

	colonies, err := client.GetColonyList(gameID, testPlayerID)
	require.NoError(t, err)
	require.NotEmpty(t, colonies)

	updates := testharness.ConnectGameUpdates(t, ts.WSURL)

	resp, body := doRequest(t, http.MethodPost, ts.BaseURL+"/action/1/colonize/1", nil, map[string]any{
		"game_id": gameID,
	})
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, body, `"id":"`+gameID+`"`)

	updates.AssertNoGameUpdate(t, 250*time.Millisecond)
}

func extractGameIDs(items []serverpkg.GameListResponseItem) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.GameID)
	}
	return ids
}

func findPlanet(t *testing.T, g game.Game, planetID uint) model.Planet {
	t.Helper()

	for _, star := range g.Galaxy.Stars {
		for _, planet := range star.Planets {
			if planet.ID == planetID {
				return *planet
			}
		}
	}

	require.FailNowf(t, "planet not found", "planet %d was not found in game %s", planetID, g.ID)
	return model.Planet{}
}

func containsShipDesign(designs []model.ShipDesign, name string) bool {
	for _, design := range designs {
		if design.Name == name {
			return true
		}
	}
	return false
}

func gameHasShipDesign(g game.Game, name string) bool {
	_, ok := findShipDesign(g, name)
	return ok
}

func findShipDesign(g game.Game, name string) (model.ShipDesign, bool) {
	for _, player := range g.Players {
		for _, design := range player.ShipDesigns {
			if design.Name == name {
				return design, true
			}
		}
	}
	return model.ShipDesign{}, false
}

func doRequest(t *testing.T, method, url string, headers http.Header, body any) (*http.Response, string) {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		require.NoError(t, err)
		bodyReader = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = resp.Body.Close()
	})

	rawBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp, string(rawBody)
}
