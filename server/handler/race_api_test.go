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

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRacesIncludesHumanBuiltIn(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SSG_CUSTOM_RACES_PATH", filepath.Join(t.TempDir(), "custom-races.json"))

	s, err := NewServer()
	require.NoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("server", s)
	GetRaces(c)
	require.Equal(t, http.StatusOK, w.Code)

	var body struct {
		Races []game.RaceDesign `json:"races"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.GreaterOrEqual(t, len(body.Races), 1)
	human := body.Races[0]
	assert.Equal(t, "Human", human.Name)
	assert.Equal(t, "Dictatorship", human.Government)
	assert.True(t, human.BuiltIn)
	assert.True(t, human.Playable)
}

func TestPostCustomRacePersistsAndRoundTrips(t *testing.T) {
	gin.SetMode(gin.TestMode)
	path := filepath.Join(t.TempDir(), "custom-races.json")
	t.Setenv("SSG_CUSTOM_RACES_PATH", path)

	s1, err := NewServer()
	require.NoError(t, err)

	payload := map[string]any{
		"name":       "Synths",
		"government": "Dictatorship",
		"trait_ids":  []string{"research_plus_1"},
	}
	raw, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/races/custom", bytes.NewReader(raw))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("server", s1)
	SaveCustomRace(c)
	require.Equal(t, http.StatusOK, w.Code)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "Synths")

	s2, err := NewServer()
	require.NoError(t, err)
	_, ok := s2.CustomRaces["Synths"]
	assert.True(t, ok)
}

func TestPostCustomRaceRejectsHumanName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SSG_CUSTOM_RACES_PATH", filepath.Join(t.TempDir(), "custom-races.json"))
	s, err := NewServer()
	require.NoError(t, err)

	payload := map[string]any{"name": "Human", "government": "Dictatorship", "trait_ids": []string{}}
	raw, _ := json.Marshal(payload)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/races/custom", bytes.NewReader(raw))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("server", s)
	SaveCustomRace(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestNewGameWithCustomRaceNameUsesPersistedRace(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SSG_CUSTOM_RACES_PATH", filepath.Join(t.TempDir(), "custom-races.json"))
	s, err := NewServer()
	require.NoError(t, err)

	raw := []byte(`{"name":"Synths","government":"Dictatorship","trait_ids":["research_plus_1"]}`)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/races/custom", bytes.NewReader(raw))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("server", s)
	SaveCustomRace(c)
	require.Equal(t, http.StatusOK, w.Code)

	req := NewGameRequest{
		PlayerName:     "Ada",
		PlayerColor:    "#FFFFFF",
		NumStars:       4,
		Race:           "custom",
		CustomRaceName: "Synths",
	}
	race, err := raceFromRequest(s, req)
	require.NoError(t, err)
	assert.Equal(t, "Synths", race.Name)
	assert.Contains(t, race.TraitIDs, "research_plus_1")
}

func TestNewGameRejectsUnplayableCustomRaceName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SSG_CUSTOM_RACES_PATH", filepath.Join(t.TempDir(), "custom-races.json"))
	s, err := NewServer()
	require.NoError(t, err)

	raw := []byte(`{"name":"Broken","government":"Dictatorship","trait_ids":["money_plus_1","food_plus_2"]}`)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/races/custom", bytes.NewReader(raw))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("server", s)
	SaveCustomRace(c)
	require.Equal(t, http.StatusOK, w.Code)
	require.False(t, s.CustomRaces["Broken"].Playable)

	req := NewGameRequest{
		PlayerName:     "Ada",
		PlayerColor:    "#FFFFFF",
		NumStars:       4,
		Race:           "custom",
		CustomRaceName: "Broken",
	}
	_, err = raceFromRequest(s, req)
	require.Error(t, err)
}

func TestDeleteCustomRaceUpdatesFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	path := filepath.Join(t.TempDir(), "custom-races.json")
	t.Setenv("SSG_CUSTOM_RACES_PATH", path)

	s, err := NewServer()
	require.NoError(t, err)

	raw := []byte(`{"name":"X","government":"Dictatorship","trait_ids":[]}`)
	w := httptest.NewRecorder()
	c1, _ := gin.CreateTestContext(w)
	c1.Request, _ = http.NewRequest(http.MethodPost, "/races/custom", bytes.NewReader(raw))
	c1.Request.Header.Set("Content-Type", "application/json")
	c1.Set("server", s)
	SaveCustomRace(c1)
	require.Equal(t, http.StatusOK, w.Code)

	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Params = gin.Params{{Key: "name", Value: "X"}}
	c2.Set("server", s)
	DeleteCustomRace(c2)
	require.Equal(t, http.StatusOK, w2.Code)

	fileRaw, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.NotContains(t, string(fileRaw), `"name": "X"`)
}

func postCustomRace(t *testing.T, s *Server, payload map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/races/custom", bytes.NewReader(raw))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("server", s)
	SaveCustomRace(c)
	return w
}

func TestPostCustomRaceRenameFromReplacesKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SSG_CUSTOM_RACES_PATH", filepath.Join(t.TempDir(), "custom-races.json"))

	s, err := NewServer()
	require.NoError(t, err)

	w1 := postCustomRace(t, s, map[string]any{
		"name": "OldName", "government": "Dictatorship", "trait_ids": []string{"research_plus_1"},
	})
	require.Equal(t, http.StatusOK, w1.Code)

	w2 := postCustomRace(t, s, map[string]any{
		"name":        "NewName",
		"government":  "Dictatorship",
		"trait_ids":   []string{"research_plus_1"},
		"rename_from": "OldName",
	})
	require.Equal(t, http.StatusOK, w2.Code)
	_, hasOld := s.CustomRaces["OldName"]
	assert.False(t, hasOld)
	_, hasNew := s.CustomRaces["NewName"]
	assert.True(t, hasNew)
}

func TestPostCustomRaceSaveAsNewKeepsOriginal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SSG_CUSTOM_RACES_PATH", filepath.Join(t.TempDir(), "custom-races.json"))

	s, err := NewServer()
	require.NoError(t, err)

	w1 := postCustomRace(t, s, map[string]any{
		"name": "Alpha", "government": "Dictatorship", "trait_ids": []string{"research_plus_1"},
	})
	require.Equal(t, http.StatusOK, w1.Code)

	w2 := postCustomRace(t, s, map[string]any{
		"name":        "Beta",
		"government":  "Dictatorship",
		"trait_ids":   []string{"research_plus_1"},
		"save_as_new": true,
	})
	require.Equal(t, http.StatusOK, w2.Code)
	_, hasAlpha := s.CustomRaces["Alpha"]
	assert.True(t, hasAlpha)
	_, hasBeta := s.CustomRaces["Beta"]
	assert.True(t, hasBeta)
}

func TestPostCustomRaceSaveAsNewRejectsDuplicateName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SSG_CUSTOM_RACES_PATH", filepath.Join(t.TempDir(), "custom-races.json"))

	s, err := NewServer()
	require.NoError(t, err)

	require.Equal(t, http.StatusOK, postCustomRace(t, s, map[string]any{
		"name": "Taken", "government": "Dictatorship", "trait_ids": []string{},
	}).Code)

	w := postCustomRace(t, s, map[string]any{
		"name":        "Taken",
		"government":  "Dictatorship",
		"trait_ids":   []string{},
		"save_as_new": true,
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPostCustomRaceRenameRejectsCollision(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SSG_CUSTOM_RACES_PATH", filepath.Join(t.TempDir(), "custom-races.json"))

	s, err := NewServer()
	require.NoError(t, err)

	require.Equal(t, http.StatusOK, postCustomRace(t, s, map[string]any{
		"name": "A", "government": "Dictatorship", "trait_ids": []string{},
	}).Code)
	require.Equal(t, http.StatusOK, postCustomRace(t, s, map[string]any{
		"name": "B", "government": "Dictatorship", "trait_ids": []string{},
	}).Code)

	w := postCustomRace(t, s, map[string]any{
		"name":        "B",
		"government":  "Dictatorship",
		"trait_ids":   []string{},
		"rename_from": "A",
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	_, hasA := s.CustomRaces["A"]
	assert.True(t, hasA)
}

func TestPostCustomRaceRejectsSaveAsNewWithRenameFrom(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SSG_CUSTOM_RACES_PATH", filepath.Join(t.TempDir(), "custom-races.json"))

	s, err := NewServer()
	require.NoError(t, err)

	w := postCustomRace(t, s, map[string]any{
		"name":        "X",
		"government":  "Dictatorship",
		"trait_ids":   []string{},
		"rename_from": "Y",
		"save_as_new": true,
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
