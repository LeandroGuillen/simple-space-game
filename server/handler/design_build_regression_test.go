package server

import (
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"ssg/game"
	"ssg/model"
	"testing"
)

func designBuildServer(t *testing.T) (*Server, *game.Game) {
	t.Helper()
	t.Setenv("SSG_SAVES_DIR", t.TempDir())
	t.Setenv("SSG_CUSTOM_RACES_PATH", filepath.Join(t.TempDir(), "races.json"))
	t.Setenv("SSG_CUSTOM_SHIP_DESIGNS_PATH", filepath.Join(t.TempDir(), "ships.json"))
	s, err := NewServer()
	require.NoError(t, err)
	g, err := game.NewGame(s.ResourceManager, game.NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: game.DefaultRace("Ada")})
	require.NoError(t, err)
	s.putGame(g.ID, g)
	return s, g
}

func designBuildRequest(t *testing.T, s *Server, g *game.Game, method, path string, payload any) *httptest.ResponseRecorder {
	t.Helper()
	data, err := json.Marshal(payload)
	require.NoError(t, err)
	req := httptest.NewRequest(method, path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Game-Id", g.ID)
	req.Header.Set("Player-Id", "1")
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	return w
}

func TestCustomShipDesignChangeDuringProduction(t *testing.T) {
	for _, operation := range []string{"delete", "replace", "rename"} {
		for _, queued := range []bool{false, true} {
			t.Run(operation+map[bool]string{false: "/active", true: "/queued"}[queued], func(t *testing.T) {
				s, g := designBuildServer(t)
				require.Equal(t, http.StatusOK, postCustomShipDesign(t, s, sampleCustomShipPayload("Lancer")).Code)
				command := &model.BuildCommand{PlanetID: 1, Building: "Lancer"}
				if queued {
					command.Building = "Flunker I"
					command.BuildQueue = []string{"Lancer"}
				}
				require.NoError(t, g.Turn(1, model.Commands{Build: []*model.BuildCommand{command}}))
				payload := sampleCustomShipPayload("Lancer")
				payload["hull_size"] = "Destroyer"
				method, path := http.MethodPost, "/ship-designs/custom"
				if operation == "delete" {
					method = http.MethodDelete
					path += "/Lancer"
				}
				if operation == "rename" {
					payload["name"] = "Renamed"
					payload["rename_from"] = "Lancer"
				}
				w := designBuildRequest(t, s, g, method, path, payload)
				assert.Equal(t, http.StatusConflict, w.Code, w.Body.String())
				assert.Contains(t, w.Body.String(), "in use")
				assert.Equal(t, "Frigate", s.CustomShipDesigns["Lancer"].HullSize.Name)
				for i := 0; i < 50; i++ {
					require.NoError(t, g.Turn(1, model.Commands{}))
					for _, event := range g.TurnEvents {
						if event.Kind == "building" {
							assert.NotContains(t, event.Text, "Lancer")
						}
					}
				}
				found := false
				for _, fleet := range g.Fleets {
					for _, ship := range fleet.Ships {
						if ship.DesignName == "Lancer" {
							found = true
							assert.Equal(t, "Frigate", ship.HullSize)
						}
					}
				}
				assert.True(t, found, "original ship must complete")
				for _, star := range g.Galaxy.Stars {
					for _, p := range star.Planets {
						assert.NotContains(t, p.Buildings, "Lancer")
					}
				}
				assert.NoError(t, game.ValidateLoadedGame(g, s.ResourceManager))
				// Exercise the actual Download/Upload paths and autosave restart, not just JSON.
				saved := designBuildRequest(t, s, g, http.MethodGet, "/games/"+g.ID+"/save", nil)
				require.Equal(t, http.StatusOK, saved.Code, saved.Body.String())
				var downloaded any
				require.NoError(t, json.Unmarshal(saved.Body.Bytes(), &downloaded))
				uploaded := designBuildRequest(t, s, g, http.MethodPost, "/saves/upload", downloaded)
				assert.Equal(t, http.StatusOK, uploaded.Code, uploaded.Body.String())
				require.NoError(t, s.autosave(g))
				restarted, err := NewServer()
				require.NoError(t, err)
				restored := restarted.getGame(g.ID)
				require.NotNil(t, restored)
				assert.Equal(t, g.CurrentTurn, restored.CurrentTurn)
				assert.NoError(t, game.ValidateLoadedGame(restored, restarted.ResourceManager))
				// Existing ships still depend on the catalog for combat and save validation.
				w = designBuildRequest(t, restarted, restored, http.MethodDelete, "/ship-designs/custom/Lancer", nil)
				assert.Equal(t, http.StatusConflict, w.Code, w.Body.String())
			})
		}
	}
}

func TestPlayerShipDesignUpdateReturnsConflictDuringProduction(t *testing.T) {
	for _, queued := range []bool{false, true} {
		t.Run(map[bool]string{false: "active", true: "queued"}[queued], func(t *testing.T) {
			s, g := designBuildServer(t)
			payload := sampleCustomShipPayload("Raider")
			w := designBuildRequest(t, s, g, http.MethodPost, "/resources/shipdesigns", payload)
			require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
			command := &model.BuildCommand{PlanetID: 1, Building: "Raider"}
			if queued {
				command.Building = "Flunker I"
				command.BuildQueue = []string{"Raider"}
			}
			require.NoError(t, g.Turn(1, model.Commands{Build: []*model.BuildCommand{command}}))
			payload["hull_size"] = "Destroyer"
			w = designBuildRequest(t, s, g, http.MethodPut, "/resources/shipdesigns", payload)
			assert.Equal(t, http.StatusConflict, w.Code, w.Body.String())
			assert.Contains(t, w.Body.String(), "in use")
			assert.Equal(t, "Frigate", g.Players[0].ShipDesigns[0].HullSize.Name)
		})
	}
}
