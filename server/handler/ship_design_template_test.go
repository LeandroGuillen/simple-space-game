package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sampleCustomShipPayload(name string) map[string]any {
	return map[string]any{
		"name":      name,
		"drive":     "Nuclear",
		"computer":  "None",
		"shield":    "None",
		"armor":     "Titanium",
		"hull_size": "Frigate",
		"fuel_cell": "Standard",
	}
}

func postCustomShipDesign(t *testing.T, s *Server, payload map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/ship-designs/custom", bytes.NewReader(raw))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("server", s)
	SaveCustomShipDesign(c)
	return w
}

func TestGetShipDesignTemplatesIncludesBuiltinFlunker(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SSG_CUSTOM_RACES_PATH", filepath.Join(t.TempDir(), "custom-races.json"))
	t.Setenv("SSG_CUSTOM_SHIP_DESIGNS_PATH", filepath.Join(t.TempDir(), "custom-ship-designs.json"))

	s, err := NewServer()
	require.NoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("server", s)
	GetShipDesignTemplates(c)
	require.Equal(t, http.StatusOK, w.Code)

	var body struct {
		ShipDesigns []ShipDesignTemplate `json:"ship_designs"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	found := false
	for _, row := range body.ShipDesigns {
		if row.Name == "Flunker I" {
			assert.True(t, row.BuiltIn)
			found = true
		}
	}
	assert.True(t, found)
}

func TestPostCustomShipDesignPersistsAndRoundTrips(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SSG_CUSTOM_RACES_PATH", filepath.Join(t.TempDir(), "custom-races.json"))
	path := filepath.Join(t.TempDir(), "custom-ship-designs.json")
	t.Setenv("SSG_CUSTOM_SHIP_DESIGNS_PATH", path)

	s1, err := NewServer()
	require.NoError(t, err)

	w := postCustomShipDesign(t, s1, sampleCustomShipPayload("My Scout"))
	require.Equal(t, http.StatusOK, w.Code)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "My Scout")

	s2, err := NewServer()
	require.NoError(t, err)
	_, ok := s2.CustomShipDesigns["My Scout"]
	assert.True(t, ok)
}

func TestPostCustomShipDesignRejectsBuiltinName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SSG_CUSTOM_RACES_PATH", filepath.Join(t.TempDir(), "custom-races.json"))
	t.Setenv("SSG_CUSTOM_SHIP_DESIGNS_PATH", filepath.Join(t.TempDir(), "custom-ship-designs.json"))

	s, err := NewServer()
	require.NoError(t, err)

	payload := sampleCustomShipPayload("Flunker I")
	w := postCustomShipDesign(t, s, payload)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteCustomShipDesignUpdatesFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SSG_CUSTOM_RACES_PATH", filepath.Join(t.TempDir(), "custom-races.json"))
	path := filepath.Join(t.TempDir(), "custom-ship-designs.json")
	t.Setenv("SSG_CUSTOM_SHIP_DESIGNS_PATH", path)

	s, err := NewServer()
	require.NoError(t, err)

	require.Equal(t, http.StatusOK, postCustomShipDesign(t, s, sampleCustomShipPayload("Zapper")).Code)

	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Params = gin.Params{{Key: "name", Value: "Zapper"}}
	c2.Set("server", s)
	DeleteCustomShipDesign(c2)
	require.Equal(t, http.StatusOK, w2.Code)

	fileRaw, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.NotContains(t, string(fileRaw), `"name": "Zapper"`)
}

func TestPostCustomShipDesignRenameFromReplacesKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SSG_CUSTOM_RACES_PATH", filepath.Join(t.TempDir(), "custom-races.json"))
	t.Setenv("SSG_CUSTOM_SHIP_DESIGNS_PATH", filepath.Join(t.TempDir(), "custom-ship-designs.json"))

	s, err := NewServer()
	require.NoError(t, err)

	w1 := postCustomShipDesign(t, s, sampleCustomShipPayload("OldHull"))
	require.Equal(t, http.StatusOK, w1.Code)

	w2 := postCustomShipDesign(t, s, map[string]any{
		"name":        "NewHull",
		"drive":       "Nuclear",
		"computer":    "None",
		"shield":      "None",
		"armor":       "Titanium",
		"hull_size":   "Frigate",
		"fuel_cell":   "Standard",
		"rename_from": "OldHull",
	})
	require.Equal(t, http.StatusOK, w2.Code)
	_, hasOld := s.CustomShipDesigns["OldHull"]
	assert.False(t, hasOld)
	_, hasNew := s.CustomShipDesigns["NewHull"]
	assert.True(t, hasNew)
}

func TestPostCustomShipDesignSaveAsNewKeepsOriginal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SSG_CUSTOM_RACES_PATH", filepath.Join(t.TempDir(), "custom-races.json"))
	t.Setenv("SSG_CUSTOM_SHIP_DESIGNS_PATH", filepath.Join(t.TempDir(), "custom-ship-designs.json"))

	s, err := NewServer()
	require.NoError(t, err)

	require.Equal(t, http.StatusOK, postCustomShipDesign(t, s, sampleCustomShipPayload("Alpha Ship")).Code)

	w2 := postCustomShipDesign(t, s, map[string]any{
		"name":        "Beta Ship",
		"drive":       "Nuclear",
		"computer":    "None",
		"shield":      "None",
		"armor":       "Titanium",
		"hull_size":   "Frigate",
		"fuel_cell":   "Standard",
		"save_as_new": true,
	})
	require.Equal(t, http.StatusOK, w2.Code)
	_, hasAlpha := s.CustomShipDesigns["Alpha Ship"]
	assert.True(t, hasAlpha)
	_, hasBeta := s.CustomShipDesigns["Beta Ship"]
	assert.True(t, hasBeta)
}

func TestGetShipPartsWithoutGameHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("SSG_CUSTOM_RACES_PATH", filepath.Join(t.TempDir(), "custom-races.json"))
	t.Setenv("SSG_CUSTOM_SHIP_DESIGNS_PATH", filepath.Join(t.TempDir(), "custom-ship-designs.json"))

	s, err := NewServer()
	require.NoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/resources/shipparts", nil)
	c.Set("server", s)
	GetShipPartsList(c)
	require.Equal(t, http.StatusOK, w.Code)

	var body struct {
		HullSizes []struct {
			Name string `json:"name"`
		} `json:"hull_sizes"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.NotEmpty(t, body.HullSizes)
}
