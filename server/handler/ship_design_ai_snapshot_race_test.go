package server

import (
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

// Exercises server AI snapshot construction (GetVisibleGameForPlayer + json.Marshal)
// while custom ship designs save and delete through handler endpoints.
func TestConcurrentCustomShipDesignSaveDeleteWhileAISnapshotMarshal(t *testing.T) {
	s := newConcurrencyTestServer(t)

	w := serveJSON(s, http.MethodPost, "/games", map[string]any{
		"player_name": "AI Snapshot",
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
	playerID := g.Players[0].ID

	var stop atomic.Bool
	var wg sync.WaitGroup
	for r := 0; r < 8; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for !stop.Load() {
				visible, err := g.GetVisibleGameForPlayer(playerID)
				if err != nil {
					continue
				}
				_, _ = json.Marshal(visible)
			}
		}()
	}

	for i := 0; i < 40; i++ {
		name := "AISnap-" + string(rune('A'+i%26))
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
		if i%3 == 0 {
			del := serveJSON(s, http.MethodDelete, "/ship-designs/custom/"+name, nil)
			require.Equal(t, http.StatusOK, del.Code, del.Body.String())
		}
	}

	stop.Store(true)
	wg.Wait()
}
