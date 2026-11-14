package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"ssg/game"
	"ssg/model"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const suggestShipDesignPath = "/resources/shipdesigns/suggest"

type suggestedDesignBody struct {
	Suggestion struct {
		Name     string `json:"name"`
		Beam     string `json:"beam"`
		Missile  string `json:"missile"`
		Drive    string `json:"drive"`
		Computer string `json:"computer"`
		Shield   string `json:"shield"`
		Armor    string `json:"armor"`
		HullSize string `json:"hull_size"`
		FuelCell string `json:"fuel_cell"`
	} `json:"suggestion"`
	Message string `json:"message"`
}

func suggestServer(t *testing.T) (*Server, *game.Game) {
	t.Helper()
	t.Setenv("SSG_SAVES_DIR", t.TempDir())
	t.Setenv("SSG_CUSTOM_RACES_PATH", filepath.Join(t.TempDir(), "races.json"))
	t.Setenv("SSG_CUSTOM_SHIP_DESIGNS_PATH", filepath.Join(t.TempDir(), "ships.json"))
	s, err := NewServer()
	require.NoError(t, err)
	g, err := game.NewGame(s.ResourceManager, game.NewGameOptions{PlayerName: "Ada", NumStars: 8, NumPlayers: 2, PlayerColor: model.DefaultPlayerColor(), Race: game.DefaultRace("Ada")})
	require.NoError(t, err)
	require.Len(t, g.Players, 2)
	s.putGame(g.ID, g)
	return s, g
}

func requestSuggestion(t *testing.T, s *Server, g *game.Game, playerID uint) (*httptest.ResponseRecorder, suggestedDesignBody) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, suggestShipDesignPath, nil)
	req.Header.Set("Game-Id", g.ID)
	req.Header.Set("Player-Id", strconv.FormatUint(uint64(playerID), 10))
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	var body suggestedDesignBody
	if w.Code == http.StatusOK || w.Header().Get("Content-Type") == "application/json; charset=utf-8" {
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body), w.Body.String())
	}
	return w, body
}

// grant mirrors research unlocks directly on the player's available tech.
func grant(p *model.Player, unlocks model.AvailableTech) {
	p.Available.Drives = append(p.Available.Drives, unlocks.Drives...)
	p.Available.FuelCells = append(p.Available.FuelCells, unlocks.FuelCells...)
	p.Available.Armors = append(p.Available.Armors, unlocks.Armors...)
	p.Available.Computers = append(p.Available.Computers, unlocks.Computers...)
	p.Available.Shields = append(p.Available.Shields, unlocks.Shields...)
	p.Available.Beams = append(p.Available.Beams, unlocks.Beams...)
	p.Available.Missiles = append(p.Available.Missiles, unlocks.Missiles...)
}

func TestSuggestShipDesignWithStarterTechnology(t *testing.T) {
	s, g := suggestServer(t)
	p := g.Players[0]
	w, body := requestSuggestion(t, s, g, p.ID)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	d := body.Suggestion
	assert.Equal(t, "Suggested Frigate 1", d.Name)
	assert.Equal(t, "Frigate", d.HullSize)
	assert.Equal(t, "Nuclear", d.Drive)
	assert.Equal(t, "Standard", d.FuelCell)
	assert.Equal(t, "Titanium", d.Armor)
	assert.Equal(t, "None", d.Computer)
	assert.Equal(t, "None", d.Shield)
	// No weapon is researched at the start, so the slot stays empty rather
	// than offering unavailable technology.
	assert.Empty(t, d.Beam)
	assert.Empty(t, d.Missile)
	assert.Empty(t, p.ShipDesigns, "a suggestion must not save a design")
}

func TestSuggestShipDesignWithUpgradedTechnologyPicksBestPartsAndWeapon(t *testing.T) {
	s, g := suggestServer(t)
	p := g.Players[0]
	grant(p, model.AvailableTech{
		Drives:    []string{"Fusion", "Ion"},
		FuelCells: []string{"Deuterium"},
		Armors:    []string{"Tritanium"},
		Computers: []string{"Electronic", "Optronic"},
		Shields:   []string{"Class I", "Class III"},
		Beams:     []string{"Laser"},
		Missiles:  []string{"Nuclear Missile"},
	})

	w, body := requestSuggestion(t, s, g, p.ID)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	d := body.Suggestion
	assert.Equal(t, "Frigate", d.HullSize)
	assert.Equal(t, "Ion", d.Drive)
	assert.Equal(t, "Deuterium", d.FuelCell)
	assert.Equal(t, "Tritanium", d.Armor)
	assert.Equal(t, "Optronic", d.Computer)
	assert.Equal(t, "Class III", d.Shield)
	// Nuclear Missile (8 damage) beats Laser (max 4): one weapon slot only.
	assert.Equal(t, "Nuclear Missile", d.Missile)
	assert.Empty(t, d.Beam)

	// Fusion Beam ties the missile at 8; the beam wins (unlimited shots).
	grant(p, model.AvailableTech{Beams: []string{"Fusion Beam"}})
	w, body = requestSuggestion(t, s, g, p.ID)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "Fusion Beam", body.Suggestion.Beam)
	assert.Empty(t, body.Suggestion.Missile)

	// Merculite (14) outranks Fusion Beam (8); Phasors (max 24) outranks both.
	grant(p, model.AvailableTech{Missiles: []string{"Merculite Missile"}})
	_, body = requestSuggestion(t, s, g, p.ID)
	assert.Equal(t, "Merculite Missile", body.Suggestion.Missile)
	assert.Empty(t, body.Suggestion.Beam)
	grant(p, model.AvailableTech{Beams: []string{"Phasors"}})
	_, body = requestSuggestion(t, s, g, p.ID)
	assert.Equal(t, "Phasors", body.Suggestion.Beam)
	assert.Empty(t, body.Suggestion.Missile)

	// The draft passes the same validation as saving it.
	save := designBuildRequestAs(t, s, g, p.ID, http.MethodPost, "/resources/shipdesigns", body.Suggestion)
	require.Equal(t, http.StatusCreated, save.Code, save.Body.String())
	require.Len(t, p.ShipDesigns, 1)
	assert.Equal(t, "Phasors", p.ShipDesigns[0].Beams[0].Name)
}

func TestSuggestShipDesignUsesOnlyTheRequestingPlayersTechnology(t *testing.T) {
	s, g := suggestServer(t)
	a, b := g.Players[0], g.Players[1]
	grant(a, model.AvailableTech{
		Drives:    []string{"Hyper"},
		FuelCells: []string{"Thorium"},
		Armors:    []string{"Zortrium"},
		Computers: []string{"Positronic"},
		Shields:   []string{"Class V"},
		Beams:     []string{"Plasma Web"},
		Missiles:  []string{"Zeon Missile"},
	})

	_, bodyA := requestSuggestion(t, s, g, a.ID)
	assert.Equal(t, "Hyper", bodyA.Suggestion.Drive)
	assert.Equal(t, "Plasma Web", bodyA.Suggestion.Beam)

	w, bodyB := requestSuggestion(t, s, g, b.ID)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	d := bodyB.Suggestion
	assert.Equal(t, "Nuclear", d.Drive)
	assert.Equal(t, "Standard", d.FuelCell)
	assert.Equal(t, "Titanium", d.Armor)
	assert.Equal(t, "None", d.Computer)
	assert.Equal(t, "None", d.Shield)
	assert.Empty(t, d.Beam)
	assert.Empty(t, d.Missile)
}

func TestSuggestShipDesignDoesNotOverwriteExistingDesigns(t *testing.T) {
	s, g := suggestServer(t)
	p := g.Players[0]
	grant(p, model.AvailableTech{Beams: []string{"Laser"}})
	existing := map[string]any{"name": "Suggested Frigate 1", "beam": "Laser", "drive": "Nuclear", "computer": "None", "shield": "None", "armor": "None", "hull_size": "Frigate", "fuel_cell": "Standard"}
	save := designBuildRequestAs(t, s, g, p.ID, http.MethodPost, "/resources/shipdesigns", existing)
	require.Equal(t, http.StatusCreated, save.Code, save.Body.String())
	before := append([]model.ShipDesign(nil), p.ShipDesigns...)

	w, body := requestSuggestion(t, s, g, p.ID)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "Suggested Frigate 2", body.Suggestion.Name)
	assert.Equal(t, "Titanium", body.Suggestion.Armor)
	assert.Equal(t, before, p.ShipDesigns)
	assert.Equal(t, "None", p.ShipDesigns[0].Armor.Name)
}

func TestSuggestShipDesignFailsClearlyWithoutSuitableTechnology(t *testing.T) {
	s, g := suggestServer(t)
	p := g.Players[0]
	p.Available.Drives = []string{"None"}

	w, body := requestSuggestion(t, s, g, p.ID)
	require.Equal(t, http.StatusUnprocessableEntity, w.Code, w.Body.String())
	assert.Contains(t, body.Message, "interstellar drive")
	assert.Empty(t, p.ShipDesigns)

	missing, _ := requestSuggestion(t, s, g, 99)
	assert.Equal(t, http.StatusBadRequest, missing.Code)

	req := httptest.NewRequest(http.MethodPost, suggestShipDesignPath, nil)
	req.Header.Set("Game-Id", g.ID)
	noPlayer := httptest.NewRecorder()
	s.Handler().ServeHTTP(noPlayer, req)
	assert.Equal(t, http.StatusBadRequest, noPlayer.Code)
}

func designBuildRequestAs(t *testing.T, s *Server, g *game.Game, playerID uint, method, path string, payload any) *httptest.ResponseRecorder {
	t.Helper()
	data, err := json.Marshal(payload)
	require.NoError(t, err)
	req := httptest.NewRequest(method, path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Game-Id", g.ID)
	req.Header.Set("Player-Id", strconv.FormatUint(uint64(playerID), 10))
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	return w
}

func TestSuggestShipDesignFallsBackToSmallestAvailableHull(t *testing.T) {
	s, g := suggestServer(t)
	p := g.Players[0]
	p.Available.HullSizes = []string{"Cruiser", "Destroyer"}

	w, body := requestSuggestion(t, s, g, p.ID)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "Destroyer", body.Suggestion.HullSize)
	assert.Equal(t, "Suggested Destroyer 1", body.Suggestion.Name)
}
