package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"ssg/game"
	"ssg/model"

	"github.com/stretchr/testify/require"
)

func TestGetBattleHeaderAuthorization(t *testing.T) {
	s := newGameTestServer(t)
	g, err := game.NewGame(s.ResourceManager, game.NewGameOptions{
		PlayerName: "Ada", PlayerColor: model.DefaultPlayerColor(), NumStars: 8,
		Race: game.DefaultRace("Human"), NumPlayers: 3,
	})
	require.NoError(t, err)
	battle := &model.TacticalBattle{
		ID: 1, Status: model.TacticalBattleActive, HumanOwnerIDs: []uint{1},
		Ships: []*model.TacticalShip{
			{ID: 1, OwnerID: 1, Name: "Ada's ship", DesignName: "Gunboat", BeamAttackBonus: 80, X: 1, Y: 2},
			{ID: 2, OwnerID: 2, Name: "Opponent's ship", BeamDefense: 20, Destroyed: true},
		},
		Events: []model.BattleEvent{{Seq: 1, Type: "destroyed", TargetID: 2}},
	}
	g.PendingBattles = []*model.TacticalBattle{battle}
	s.putGame(g.ID, g)
	visible, err := g.GetVisibleGameForPlayer(3)
	require.NoError(t, err)
	require.Empty(t, visible.PendingBattles)

	for _, tc := range []struct {
		name     string
		playerID string
		battleID uint
		status   int
	}{
		{"human participant", "1", 1, http.StatusOK},
		{"participant with destroyed ship", "2", 1, http.StatusOK},
		{"non-participant", "3", 1, http.StatusNotFound},
		{"unknown player", "999", 1, http.StatusNotFound},
		{"zero player", "0", 1, http.StatusNotFound},
		{"missing player", "", 1, http.StatusBadRequest},
		{"invalid player", "invalid", 1, http.StatusBadRequest},
		{"nonexistent battle", "1", 999, http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/battles/%d", tc.battleID), nil)
			r.Header.Set(GameIDHeader, g.ID)
			if tc.playerID != "" {
				r.Header.Set(PlayerIDHeader, tc.playerID)
			}
			w := httptest.NewRecorder()
			s.Handler().ServeHTTP(w, r)
			require.Equal(t, tc.status, w.Code, w.Body.String())
			if tc.status == http.StatusOK {
				var response struct {
					Battle *model.TacticalBattle `json:"battle"`
				}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
				require.Equal(t, battle, response.Battle, "participants receive the full state and event log")
			} else if tc.status == http.StatusNotFound {
				require.JSONEq(t, `{"message":"battle not found"}`, w.Body.String())
			}
		})
	}
}
