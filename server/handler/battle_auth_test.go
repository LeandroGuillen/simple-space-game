package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestPostBattleActionHeaderAuthorization(t *testing.T) {
	newFixture := func(t *testing.T) (*gameTestServerFixture, *model.TacticalBattle, *model.TacticalBattle) {
		s := newGameTestServer(t)
		g, err := game.NewGame(s.ResourceManager, game.NewGameOptions{
			PlayerName: "Ada", PlayerColor: model.DefaultPlayerColor(), NumStars: 8,
			Race: game.DefaultRace("Human"), NumPlayers: 3,
		})
		require.NoError(t, err)
		active := &model.TacticalBattle{
			ID: 1, Status: model.TacticalBattleActive, HumanOwnerIDs: []uint{1},
			GridWidth: 10, GridHeight: 8, Round: 1,
			Ships: []*model.TacticalShip{
				{ID: 1, OwnerID: 1, Name: "Ada's ship", DesignName: "Scout", X: 1, Y: 2, CombatSpeed: 2, MovesLeft: 2},
				{ID: 2, OwnerID: 2, Name: "Opponent's ship", DesignName: "Gunboat", X: 8, Y: 5, CombatSpeed: 2, MovesLeft: 2},
			},
			TurnOrder: []uint{1, 2},
			Events:    []model.BattleEvent{{Seq: 1, Type: "round_start", Round: 1}},
		}
		finished := &model.TacticalBattle{
			ID: 2, Status: model.TacticalBattleFinished, HumanOwnerIDs: []uint{1},
			GridWidth: 10, GridHeight: 8, Round: 3, WinnerOwnerID: 1,
			Ships: []*model.TacticalShip{
				{ID: 1, OwnerID: 1, Name: "Ada's ship", DesignName: "Scout", X: 4, Y: 4},
				{ID: 2, OwnerID: 2, Name: "Opponent's ship", DesignName: "Gunboat", Destroyed: true},
			},
			TurnOrder: []uint{1, 2},
			Events:    []model.BattleEvent{{Seq: 1, Type: "destroyed", TargetID: 2}},
		}
		g.PendingBattles = []*model.TacticalBattle{active, finished}
		s.putGame(g.ID, g)
		return &gameTestServerFixture{s: s, gameID: g.ID}, active, finished
	}

	post := func(t *testing.T, f *gameTestServerFixture, playerID string, battleID uint, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/battles/%d/action", battleID), strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set(GameIDHeader, f.gameID)
		if playerID != "" {
			r.Header.Set(PlayerIDHeader, playerID)
		}
		w := httptest.NewRecorder()
		f.s.Handler().ServeHTTP(w, r)
		return w
	}

	actions := map[string]string{
		"wait":    `{"type":"wait"}`,
		"move":    `{"type":"move","x":2,"y":2}`,
		"fire":    `{"type":"fire","target_id":2}`,
		"unknown": `{"type":"dance"}`,
	}

	t.Run("non-participants get 404 without battle state", func(t *testing.T) {
		for _, playerID := range []string{"3", "999", "0"} {
			for _, battleID := range []uint{1, 2, 999} {
				for name, body := range actions {
					t.Run(fmt.Sprintf("player %s battle %d %s", playerID, battleID, name), func(t *testing.T) {
						f, active, finished := newFixture(t)
						activeBefore := mustJSON(t, active)
						finishedBefore := mustJSON(t, finished)
						w := post(t, f, playerID, battleID, body)
						require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
						require.JSONEq(t, `{"message":"battle not found"}`, w.Body.String())
						require.JSONEq(t, activeBefore, mustJSON(t, active), "non-participant must not change the battle")
						require.JSONEq(t, finishedBefore, mustJSON(t, finished))
					})
				}
			}
		}
	})

	t.Run("missing battle for participant matches GET", func(t *testing.T) {
		f, _, _ := newFixture(t)
		w := post(t, f, "1", 999, actions["wait"])
		require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
		require.JSONEq(t, `{"message":"battle not found"}`, w.Body.String())

		r := httptest.NewRequest(http.MethodGet, "/battles/999", nil)
		r.Header.Set(GameIDHeader, f.gameID)
		r.Header.Set(PlayerIDHeader, "1")
		get := httptest.NewRecorder()
		f.s.Handler().ServeHTTP(get, r)
		require.Equal(t, http.StatusNotFound, get.Code)
		require.JSONEq(t, get.Body.String(), w.Body.String(), "POST and GET must be indistinguishable")
	})

	t.Run("participant action errors still return the battle", func(t *testing.T) {
		for _, tc := range []struct {
			name     string
			playerID string
			battleID uint
			body     string
		}{
			{"wrong turn", "2", 1, actions["wait"]},
			{"finished battle", "1", 2, actions["wait"]},
			{"finished battle for destroyed participant", "2", 2, actions["fire"]},
			{"move off grid", "1", 1, `{"type":"move","x":50,"y":50}`},
			{"fire without weapons", "1", 1, actions["fire"]},
			{"unknown action", "1", 1, actions["unknown"]},
		} {
			t.Run(tc.name, func(t *testing.T) {
				f, active, finished := newFixture(t)
				want := active
				if tc.battleID == finished.ID {
					want = finished
				}
				w := post(t, f, tc.playerID, tc.battleID, tc.body)
				require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
				var response struct {
					Message string                `json:"message"`
					Battle  *model.TacticalBattle `json:"battle"`
				}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
				require.NotEmpty(t, response.Message)
				require.NotEqual(t, "battle not found", response.Message)
				require.Equal(t, want, response.Battle, "participants receive the battle to resync")
			})
		}
	})

	t.Run("participant action succeeds", func(t *testing.T) {
		f, _, _ := newFixture(t)
		w := post(t, f, "1", 1, `{"type":"move","x":2,"y":2}`)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		require.Contains(t, w.Body.String(), `"battle"`)
	})

	t.Run("missing or invalid player header stays 400", func(t *testing.T) {
		for _, playerID := range []string{"", "invalid"} {
			f, _, _ := newFixture(t)
			w := post(t, f, playerID, 1, actions["wait"])
			require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
			require.NotContains(t, w.Body.String(), `"battle"`)
		}
	})
}

type gameTestServerFixture struct {
	s      *Server
	gameID string
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}
