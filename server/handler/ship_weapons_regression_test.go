package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShipWeaponTemplateRoundTrip(t *testing.T) {
	t.Setenv("SSG_SAVES_DIR", t.TempDir())
	t.Setenv("SSG_CUSTOM_RACES_PATH", filepath.Join(t.TempDir(), "races.json"))
	t.Setenv("SSG_CUSTOM_SHIP_DESIGNS_PATH", filepath.Join(t.TempDir(), "ships.json"))
	s, err := NewServer()
	require.NoError(t, err)
	payload := sampleCustomShipPayload("Armed scout")
	payload["beam"] = "Laser"
	w := postCustomShipDesign(t, s, payload)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body map[string]map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "Laser", body["custom_ship_design"]["beam"])
	s2, err := NewServer()
	require.NoError(t, err)
	d := s2.CustomShipDesigns["Armed scout"]
	require.Len(t, d.Beams, 1)
	require.Equal(t, "Laser", d.Beams[0].Name)
	payload["beam"] = "invented"
	w = postCustomShipDesign(t, s2, payload)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Equal(t, "Laser", s2.CustomShipDesigns["Armed scout"].Beams[0].Name)
}

func TestShipWeaponPlayerCreateUpdateRoundTrip(t *testing.T) {
	t.Setenv("SSG_SAVES_DIR", t.TempDir())
	s := newConcurrencyTestServer(t)
	w := serveJSON(s, http.MethodPost, "/games", map[string]any{"player_name": "Weapon tester", "num_stars": 8, "players": 1, "race": "human"})
	require.Equal(t, http.StatusOK, w.Code)
	var created struct {
		GameID string `json:"gameID"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	g := s.getGame(created.GameID)
	g.Players[0].Available.Beams = []string{"Laser"}
	request := func(method string, payload any) *httptest.ResponseRecorder {
		raw, err := json.Marshal(payload)
		require.NoError(t, err)
		req := httptest.NewRequest(method, "/resources/shipdesigns", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Game-Id", created.GameID)
		req.Header.Set("Player-Id", "1")
		response := httptest.NewRecorder()
		s.Handler().ServeHTTP(response, req)
		return response
	}
	payload := sampleCustomShipPayload("Player frigate")
	payload["beam"] = "Laser"
	w = request(http.MethodPost, payload)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	require.Len(t, g.Players[0].ShipDesigns, 1)
	require.Len(t, g.Players[0].ShipDesigns[0].Beams, 1)
	require.Equal(t, "Laser", g.Players[0].ShipDesigns[0].Beams[0].Name)
	payload["beam"] = "" // explicit utility conversion
	w = request(http.MethodPut, payload)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	require.Empty(t, g.Players[0].ShipDesigns[0].Beams)
	payload["beam"] = "Laser"
	w = request(http.MethodPut, payload)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	w = request(http.MethodGet, nil)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"weapons":[{"name":"Laser"`)
	g.Players[0].Available.Beams = nil
	w = request(http.MethodPut, payload)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Len(t, g.Players[0].ShipDesigns[0].Beams, 1)
}
