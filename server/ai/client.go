package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"ssg/model"

	"github.com/gorilla/websocket"
)

const (
	GameIDHeader   = "Game-Id"
	PlayerIDHeader = "Player-Id"
)

// VisibleGame is the slim shape of the game-state payload the AI consumes.
// Fields we don't act on are left out; the original JSON is preserved as Raw
// so the UI can display it verbatim.
type VisibleGame struct {
	ID                 string            `json:"id"`
	CurrentTurn        float64           `json:"current_turn"`
	Galaxy             Galaxy            `json:"galaxy"`
	Players            []*model.Player   `json:"players"`
	Fleets             []*model.Fleet    `json:"fleets"`
	PendingColonyBases map[uint][]uint   `json:"pending_colony_bases"`
	PendingColonyShips map[uint][]uint   `json:"pending_colony_ships"`
	TurnStatus         TurnStatus        `json:"turn_status"`
	Resources          *VisibleResources `json:"resources,omitempty"`

	Raw json.RawMessage `json:"-"`
}

type VisibleResources struct {
	TechTree             *model.TechTree              `json:"tech_tree,omitempty"`
	ShipParts            *VisibleShipParts            `json:"ship_parts,omitempty"`
	ShipDesignCollection *VisibleShipDesignCollection `json:"ship_design_collection,omitempty"`
	BeamCollection       *VisibleBeams                `json:"beam_collection,omitempty"`
}

type VisibleBeams struct {
	Beams []model.Beam `json:"beams"`
}

// VisibleShipParts carries the fuel-cell catalog the AI needs to match
// submit-time fuel-range validation (FuelCell autonomy over stamps).
type VisibleShipParts struct {
	FuelCells []model.FuelCell `json:"fuel_cells,omitempty"`
	HullSizes []model.HullSize `json:"hull_sizes,omitempty"`
	Drives    []model.Drive    `json:"drives,omitempty"`
	Armors    []model.Armor    `json:"armors,omitempty"`
	Computers []model.Computer `json:"computers,omitempty"`
	Shields   []model.Shield   `json:"shields,omitempty"`
}

// VisibleShipDesignCollection carries stock designs so design-only fleet ships
// (design_name, no fuel stamp) resolve the same way as Game.fleetShipFuelRange.
type VisibleShipDesignCollection struct {
	ShipDesigns []model.ShipDesign `json:"ship_designs,omitempty"`
}

type Galaxy struct {
	Stars []*model.Star `json:"stars"`
}

type TurnStatus struct {
	Mode              string `json:"mode"`
	ReadyPlayerIDs    []uint `json:"ready_player_ids"`
	WaitingPlayerIDs  []uint `json:"waiting_player_ids"`
	RequiredPlayerIDs []uint `json:"required_player_ids"`
}

type TurnResponse struct {
	Message    string          `json:"message,omitempty"`
	OK         bool            `json:"ok"`
	Advanced   bool            `json:"advanced"`
	TurnStatus TurnStatus      `json:"turn_status"`
	Raw        json.RawMessage `json:"-"`
}

// Client is the per-agent client to the game server.
type Client struct {
	BaseURL  string
	GameID   string
	PlayerID uint

	HTTP *http.Client
}

func NewClient(baseURL, gameID string, playerID uint) *Client {
	return &Client{
		BaseURL:  strings.TrimRight(baseURL, "/"),
		GameID:   gameID,
		PlayerID: playerID,
		HTTP:     &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) headers(req *http.Request) {
	req.Header.Set(GameIDHeader, c.GameID)
	req.Header.Set(PlayerIDHeader, strconv.FormatUint(uint64(c.PlayerID), 10))
}

func (c *Client) FetchGame(ctx context.Context) (*VisibleGame, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/games/"+url.PathEscape(c.GameID), nil)
	if err != nil {
		return nil, err
	}
	c.headers(req)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch game: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return parseVisibleGame(body)
}

func parseVisibleGame(body []byte) (*VisibleGame, error) {
	var g VisibleGame
	if err := json.Unmarshal(body, &g); err != nil {
		return nil, fmt.Errorf("decode game payload: %w", err)
	}
	g.Raw = append(json.RawMessage(nil), body...)
	return &g, nil
}

type turnRequestBody struct {
	Commands model.Commands `json:"commands"`
}

func (c *Client) SubmitTurn(ctx context.Context, cmds model.Commands) (*TurnResponse, error) {
	payload, err := json.Marshal(turnRequestBody{Commands: cmds})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/turn", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	c.headers(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("submit turn: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var tr TurnResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("decode turn response: %w", err)
	}
	tr.Raw = append(json.RawMessage(nil), body...)
	return &tr, nil
}

// ListGames calls the server's GET /games (no headers required). Used by the
// UI to populate the agent-creation form.
func ListGames(ctx context.Context, baseURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/games", nil)
	if err != nil {
		return nil, err
	}
	hc := &http.Client{Timeout: 10 * time.Second}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list games: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

// Subscribe opens the WS update channel for this client's game/player and
// returns a channel of incoming VisibleGame snapshots. The goroutine driving
// the channel exits when ctx is cancelled or the connection drops; the caller
// is expected to call Subscribe again to reconnect.
func (c *Client) Subscribe(ctx context.Context) (<-chan *VisibleGame, error) {
	wsURL, err := buildWSURL(c.BaseURL, c.GameID, c.PlayerID)
	if err != nil {
		return nil, err
	}
	dialer := websocket.DefaultDialer
	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return nil, err
	}

	out := make(chan *VisibleGame, 8)
	go func() {
		defer conn.Close()
		defer close(out)
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var env struct {
				GameUpdate json.RawMessage `json:"game_update"`
			}
			if err := json.Unmarshal(raw, &env); err != nil || len(env.GameUpdate) == 0 {
				continue
			}
			vg, err := parseVisibleGame(env.GameUpdate)
			if err != nil {
				continue
			}
			select {
			case <-ctx.Done():
				return
			case out <- vg:
			}
		}
	}()
	return out, nil
}

func buildWSURL(baseURL, gameID string, playerID uint) (string, error) {
	u, err := url.Parse(strings.TrimRight(baseURL, "/") + "/gameupdates")
	if err != nil {
		return "", err
	}
	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	}
	q := u.Query()
	q.Set("game_id", gameID)
	q.Set("player_id", strconv.FormatUint(uint64(playerID), 10))
	u.RawQuery = q.Encode()
	return u.String(), nil
}
