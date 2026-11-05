package server

import (
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

// Exercises the same unlocked catalog reads the engine uses while custom designs save.
func TestConcurrentGameShipDesignCatalogReadsWhileCustomSave(t *testing.T) {
	s := newConcurrencyTestServer(t)

	w := serveJSON(s, http.MethodPost, "/games", map[string]any{
		"player_name": "Race Reader",
		"num_stars":   8,
		"players":     1,
		"race":        "human",
	})
	require.Equal(t, http.StatusOK, w.Code)
	var created struct {
		GameID string `json:"gameID"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	require.NotEmpty(t, created.GameID)

	g := s.getGame(created.GameID)
	require.NotNil(t, g)
	player := g.Players[0]

	var stop atomic.Bool
	var wg sync.WaitGroup
	for r := 0; r < 8; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for !stop.Load() {
				for _, d := range g.Resources.ShipDesignCatalog() {
					_ = d.Name
				}
				_, _ = g.GetShipDesignsList(player.ID)
			}
		}()
	}

	for i := 0; i < 40; i++ {
		name := "ConcurrentRead " + string(rune('A'+i%26))
		resp := serveJSON(s, http.MethodPost, "/ship-designs/custom", map[string]any{
			"name":      name,
			"drive":     "Nuclear",
			"computer":  "None",
			"shield":    "None",
			"armor":     "Titanium",
			"hull_size": "Frigate",
			"fuel_cell": "Standard",
		})
		require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	}

	stop.Store(true)
	wg.Wait()

	found := false
	for _, d := range g.Resources.ShipDesignCatalog() {
		if d.Name == "ConcurrentRead A" {
			found = true
			break
		}
	}
	require.True(t, found, "newly saved custom design must appear in game catalog")
}
