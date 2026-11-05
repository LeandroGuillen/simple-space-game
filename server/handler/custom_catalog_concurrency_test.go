package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	concurrentWriters      = 4
	concurrentReaders      = 4
	concurrentSavesPerUser = 8
)

func newConcurrencyTestServer(t *testing.T) *Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	t.Setenv("SSG_CUSTOM_RACES_PATH", filepath.Join(dir, "custom-races.json"))
	t.Setenv("SSG_CUSTOM_SHIP_DESIGNS_PATH", filepath.Join(dir, "custom-ship-designs.json"))
	s, err := NewServer()
	require.NoError(t, err)
	return s
}

func serveJSON(s *Server, method, path string, body any) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	return w
}

// runSaveListLoad runs writers (save then delete every other entry) against
// readers listing the same catalog, and returns non-200 responses seen.
func runSaveListLoad(s *Server, save func(name string) *httptest.ResponseRecorder, del func(name string) *httptest.ResponseRecorder, reads []func() *httptest.ResponseRecorder, prefix string) []string {
	var (
		writers sync.WaitGroup
		readers sync.WaitGroup
		done    atomic.Bool
		mu      sync.Mutex
		bad     []string
	)
	record := func(what string, w *httptest.ResponseRecorder) {
		if w.Code != http.StatusOK {
			mu.Lock()
			bad = append(bad, fmt.Sprintf("%s: %d %s", what, w.Code, w.Body.String()))
			mu.Unlock()
		}
	}
	for r := 0; r < concurrentReaders; r++ {
		readers.Add(1)
		go func(r int) {
			defer readers.Done()
			read := reads[r%len(reads)]
			for !done.Load() {
				record("list", read())
			}
		}(r)
	}
	for w := 0; w < concurrentWriters; w++ {
		writers.Add(1)
		go func(w int) {
			defer writers.Done()
			for i := 0; i < concurrentSavesPerUser; i++ {
				name := fmt.Sprintf("%s%d-%d", prefix, w, i)
				record("save "+name, save(name))
				if i%2 == 1 {
					record("delete "+name, del(name))
				}
			}
		}(w)
	}
	writers.Wait()
	done.Store(true)
	readers.Wait()
	return bad
}

func expectedSurvivors(prefix string) []string {
	var names []string
	for w := 0; w < concurrentWriters; w++ {
		for i := 0; i < concurrentSavesPerUser; i += 2 {
			names = append(names, fmt.Sprintf("%s%d-%d", prefix, w, i))
		}
	}
	sort.Strings(names)
	return names
}

func TestConcurrentCustomRaceSaveAndList(t *testing.T) {
	s := newConcurrencyTestServer(t)

	bad := runSaveListLoad(s,
		func(name string) *httptest.ResponseRecorder {
			return serveJSON(s, http.MethodPost, "/races/custom", map[string]any{
				"name":       name,
				"government": "Dictatorship",
				"trait_ids":  []string{},
			})
		},
		func(name string) *httptest.ResponseRecorder {
			return serveJSON(s, http.MethodDelete, "/races/custom/"+name, nil)
		},
		[]func() *httptest.ResponseRecorder{
			func() *httptest.ResponseRecorder { return serveJSON(s, http.MethodGet, "/races", nil) },
			func() *httptest.ResponseRecorder { return serveJSON(s, http.MethodGet, "/races/custom", nil) },
		},
		"R",
	)
	assert.Empty(t, bad)

	w := serveJSON(s, http.MethodGet, "/races/custom", nil)
	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		CustomRaces []struct {
			Name string `json:"name"`
		} `json:"custom_races"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	got := make([]string, 0, len(body.CustomRaces))
	for _, r := range body.CustomRaces {
		got = append(got, r.Name)
	}
	assert.Equal(t, expectedSurvivors("R"), got)

	reloaded, err := NewServer()
	require.NoError(t, err)
	assert.Len(t, reloaded.CustomRaces, len(expectedSurvivors("R")))
}

func TestConcurrentCustomShipDesignSaveAndList(t *testing.T) {
	s := newConcurrencyTestServer(t)

	bad := runSaveListLoad(s,
		func(name string) *httptest.ResponseRecorder {
			return serveJSON(s, http.MethodPost, "/ship-designs/custom", map[string]any{
				"name":      name,
				"drive":     "Nuclear",
				"computer":  "None",
				"shield":    "None",
				"armor":     "Titanium",
				"hull_size": "Frigate",
				"fuel_cell": "Standard",
			})
		},
		func(name string) *httptest.ResponseRecorder {
			return serveJSON(s, http.MethodDelete, "/ship-designs/custom/"+name, nil)
		},
		[]func() *httptest.ResponseRecorder{
			func() *httptest.ResponseRecorder { return serveJSON(s, http.MethodGet, "/ship-designs", nil) },
			func() *httptest.ResponseRecorder { return serveJSON(s, http.MethodGet, "/ship-designs/custom", nil) },
		},
		"S",
	)
	assert.Empty(t, bad)

	w := serveJSON(s, http.MethodGet, "/ship-designs/custom", nil)
	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		CustomShipDesigns []ShipDesignTemplate `json:"custom_ship_designs"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	got := make([]string, 0, len(body.CustomShipDesigns))
	for _, d := range body.CustomShipDesigns {
		got = append(got, d.Name)
	}
	assert.Equal(t, expectedSurvivors("S"), got)

	// The merged catalog used by games must hold built-ins plus exactly the survivors.
	catalog := s.ResourceManager.ShipDesigns
	require.Len(t, catalog, len(s.ResourceManager.BuiltinShipDesigns)+len(expectedSurvivors("S")))
	for i, name := range expectedSurvivors("S") {
		assert.Equal(t, name, catalog[len(s.ResourceManager.BuiltinShipDesigns)+i].Name)
	}

	reloaded, err := NewServer()
	require.NoError(t, err)
	assert.Len(t, reloaded.CustomShipDesigns, len(expectedSurvivors("S")))
}
