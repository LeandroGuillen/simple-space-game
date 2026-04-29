package testharness

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	serverpkg "ssg/handler"
	"ssg/game"
	"ssg/model"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

const waitTimeout = 2 * time.Second

type TestServer struct {
	BaseURL string
	WSURL   string
	Server  *serverpkg.Server

	httpServer *httptest.Server
}

func NewTestServer(t *testing.T) *TestServer {
	t.Helper()

	s := serverpkg.NewServer()
	httpServer := httptest.NewServer(s.Handler())
	t.Cleanup(httpServer.Close)

	return &TestServer{
		BaseURL:    httpServer.URL,
		WSURL:      strings.Replace(httpServer.URL, "http://", "ws://", 1),
		Server:     s,
		httpServer: httpServer,
	}
}

type APIClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient(baseURL string) *APIClient {
	return &APIClient{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{Timeout: waitTimeout},
	}
}

type NewGameResponse struct {
	OK     bool   `json:"ok"`
	GameID string `json:"gameID"`
}

type StatusResponse struct {
	OK bool `json:"ok"`
}

type GameListResponse struct {
	GameList []serverpkg.GameListResponseItem `json:"game_list"`
}

type CreateShipDesignRequest struct {
	Name     string `json:"name"`
	Drive    string `json:"drive"`
	Computer string `json:"computer"`
	Shield   string `json:"shield"`
	Armor    string `json:"armor"`
	HullSize string `json:"hull_size"`
	FuelCell string `json:"fuel_cell"`
}

type GameUpdateMessage struct {
	GameUpdate game.Game `json:"game_update"`
}

func (c *APIClient) Status() (*StatusResponse, error) {
	var out StatusResponse
	if _, err := c.doJSON(http.MethodGet, "/status", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *APIClient) CreateGame(playerName string, numStars uint) (string, error) {
	req := serverpkg.NewGameRequest{
		PlayerName:  playerName,
		NumStars:    numStars,
		PlayerColor: "#FFFFFF",
	}

	var out NewGameResponse
	if _, err := c.doJSON(http.MethodPost, "/games", nil, req, &out); err != nil {
		return "", err
	}
	return out.GameID, nil
}

func (c *APIClient) GetGameList() ([]serverpkg.GameListResponseItem, error) {
	var out GameListResponse
	if _, err := c.doJSON(http.MethodGet, "/games", nil, nil, &out); err != nil {
		return nil, err
	}
	return out.GameList, nil
}

func (c *APIClient) GetGame(gameID string) (*game.Game, error) {
	headers := http.Header{}
	headers.Set(serverpkg.GameIDHeader, gameID)
	headers.Set(serverpkg.PlayerIDHeader, "1")

	var out game.Game
	if _, err := c.doJSON(http.MethodGet, "/games/"+gameID, headers, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *APIClient) GetColonyList(gameID string, playerID uint) ([]model.Planet, error) {
	req := serverpkg.GetColonyListRequest{
		GameID:   gameID,
		PlayerID: playerID,
	}

	var out []model.Planet
	if _, err := c.doJSON(http.MethodPost, "/colonies", nil, req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *APIClient) BuildOnPlanet(gameID string, ownerID, planetID, buildingID uint) error {
	req := serverpkg.BuildOnPlanetRequest{
		GameID:     gameID,
		OwnerID:    ownerID,
		PlanetID:   planetID,
		BuildingID: buildingID,
	}

	_, err := c.doJSON(http.MethodPost, "/colonies/build", nil, req, &StatusResponse{})
	return err
}

func (c *APIClient) Turn(gameID string, playerID uint) error {
	headers := http.Header{}
	headers.Set(serverpkg.GameIDHeader, gameID)
	headers.Set(serverpkg.PlayerIDHeader, fmt.Sprintf("%d", playerID))

	_, err := c.doJSON(http.MethodGet, "/turn", headers, nil, &StatusResponse{})
	return err
}

func (c *APIClient) GetShipParts(gameID string, playerID uint) (*game.ShipPartsCollection, error) {
	headers := http.Header{}
	headers.Set(serverpkg.GameIDHeader, gameID)
	headers.Set(serverpkg.PlayerIDHeader, fmt.Sprintf("%d", playerID))

	var out game.ShipPartsCollection
	if _, err := c.doJSON(http.MethodGet, "/resources/shipparts", headers, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *APIClient) GetShipDesigns(gameID string, playerID uint) ([]model.ShipDesign, error) {
	headers := http.Header{}
	headers.Set(serverpkg.GameIDHeader, gameID)
	headers.Set(serverpkg.PlayerIDHeader, fmt.Sprintf("%d", playerID))

	var out []model.ShipDesign
	if _, err := c.doJSON(http.MethodGet, "/resources/shipdesigns", headers, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *APIClient) CreateShipDesign(gameID string, playerID uint, req CreateShipDesignRequest) error {
	headers := http.Header{}
	headers.Set(serverpkg.GameIDHeader, gameID)
	headers.Set(serverpkg.PlayerIDHeader, fmt.Sprintf("%d", playerID))

	_, err := c.doJSON(http.MethodPost, "/resources/shipdesigns", headers, req, &StatusResponse{})
	return err
}

func (c *APIClient) UpdateShipDesign(gameID string, playerID uint, req CreateShipDesignRequest) error {
	headers := http.Header{}
	headers.Set(serverpkg.GameIDHeader, gameID)
	headers.Set(serverpkg.PlayerIDHeader, fmt.Sprintf("%d", playerID))

	_, err := c.doJSON(http.MethodPut, "/resources/shipdesigns", headers, req, &StatusResponse{})
	return err
}

func (c *APIClient) doJSON(method, path string, headers http.Header, body any, out any) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return resp, fmt.Errorf("request %s %s failed with status %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(rawBody)))
	}

	if out != nil && len(rawBody) > 0 {
		if err := json.Unmarshal(rawBody, out); err != nil {
			return resp, err
		}
	}

	return resp, nil
}

type GameUpdatesClient struct {
	conn     *websocket.Conn
	messages chan GameUpdateMessage
	errs     chan error
}

func ConnectGameUpdates(t *testing.T, wsBaseURL, gameID string, playerID uint) *GameUpdatesClient {
	t.Helper()

	wsURL := fmt.Sprintf("%s/gameupdates?game_id=%s&player_id=%d", wsBaseURL, gameID, playerID)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	client := &GameUpdatesClient{
		conn:     conn,
		messages: make(chan GameUpdateMessage, 8),
		errs:     make(chan error, 1),
	}

	go client.readLoop()
	t.Cleanup(func() {
		_ = client.conn.Close()
	})

	return client
}

func (c *GameUpdatesClient) readLoop() {
	for {
		var message GameUpdateMessage
		if err := c.conn.ReadJSON(&message); err != nil {
			select {
			case c.errs <- err:
			default:
			}
			close(c.messages)
			return
		}
		c.messages <- message
	}
}

func (c *GameUpdatesClient) WaitForGameUpdate(t *testing.T, predicate func(game.Game) bool) GameUpdateMessage {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)
	defer cancel()

	for {
		select {
		case msg, ok := <-c.messages:
			require.True(t, ok, "game updates websocket closed before delivering message")
			if predicate == nil || predicate(msg.GameUpdate) {
				return msg
			}
		case err := <-c.errs:
			require.NoError(t, err)
		case <-ctx.Done():
			require.FailNow(t, "timed out waiting for matching game update")
		}
	}
}

func (c *GameUpdatesClient) AssertNoGameUpdate(t *testing.T, timeout time.Duration) {
	t.Helper()

	select {
	case msg := <-c.messages:
		require.FailNowf(t, "unexpected websocket update", "received update for game %s", msg.GameUpdate.ID)
	case err := <-c.errs:
		if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
			return
		}
		require.NoError(t, err)
	case <-time.After(timeout):
	}
}
