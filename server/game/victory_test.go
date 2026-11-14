package game

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"ssg/model"
)

func removeColoniesFor(g *Game, owner uint) {
	for _, star := range g.Galaxy.Stars {
		for _, planet := range star.Planets {
			if planet.Owner == owner {
				planet.Owner = 0
			}
		}
	}
}

func TestEliminationRemovesPlayerFromTurns(t *testing.T) {
	g := multiPlayerGame(t, 3, 30)
	removeColoniesFor(g, 2)
	require.NoError(t, g.AdvanceTurn(emptySubsFor(g)))
	require.True(t, g.Players[1].Eliminated)
	require.Equal(t, "active", g.Status)
	require.Equal(t, []uint{1, 3}, g.TurnStatus.RequiredPlayerIDs)
	_, _, err := g.SubmitTurn(2, model.Commands{})
	require.ErrorContains(t, err, "eliminated")
	_, advanced, err := g.SubmitTurn(1, model.Commands{})
	require.NoError(t, err)
	require.False(t, advanced)
	_, advanced, err = g.SubmitTurn(3, model.Commands{})
	require.NoError(t, err)
	require.True(t, advanced)
	// Rivals appear in the visible player list only after first contact; an
	// eliminated contact stays listed.
	g.Players[0].Contacts = []model.Contact{{PlayerID: 2}}
	view, err := g.GetVisibleGameForPlayer(1)
	require.NoError(t, err)
	require.True(t, view.Players[1].Eliminated)
}

func TestColonyShipPreventsEliminationAndConquest(t *testing.T) {
	g := multiPlayerGame(t, 2, 30)
	removeColoniesFor(g, 2)
	g.Fleets = append(g.Fleets, &model.Fleet{ID: 900, Owner: 2, DestinationStarID: g.Galaxy.Stars[0].ID, TotalTurns: 100, TurnsRemaining: 100, Ships: []model.FleetShip{{Type: "colony_ship"}}})
	require.NoError(t, g.AdvanceTurn(emptySubsFor(g)))
	require.False(t, g.Players[1].Eliminated)
	require.Equal(t, "active", g.Status)
	g.Fleets = nil
	require.NoError(t, g.AdvanceTurn(emptySubsFor(g)))
	require.True(t, g.Players[1].Eliminated)
	require.Equal(t, "finished", g.Status)
	require.Equal(t, uint(1), g.WinnerID)
	require.Empty(t, g.TurnStatus.RequiredPlayerIDs)
	_, _, err := g.SubmitTurn(1, model.Commands{})
	require.ErrorContains(t, err, "finished")
	require.ErrorContains(t, g.AdvanceTurn(nil), "finished")
	view, err := g.GetVisibleGameForPlayer(2)
	require.NoError(t, err)
	require.Equal(t, "finished", view.Status)
	require.Equal(t, uint(1), view.WinnerID)
	require.Equal(t, uint(2), view.TurnsPlayed)
	data, err := g.MarshalSave()
	require.NoError(t, err)
	var restored Game
	require.NoError(t, json.Unmarshal(data, &restored))
	require.Equal(t, "finished", restored.Status)
	require.Equal(t, uint(1), restored.WinnerID)
	require.True(t, restored.Players[1].Eliminated)
}

func TestPendingBattleDefersElimination(t *testing.T) {
	g := multiPlayerGame(t, 2, 30)
	removeColoniesFor(g, 2)
	g.PendingBattles = []*model.TacticalBattle{{Status: model.TacticalBattleActive}}
	g.evaluateConquest()
	require.False(t, g.Players[1].Eliminated)
	require.Equal(t, "active", g.Status)
	g.PendingBattles = nil
	g.evaluateConquest()
	require.True(t, g.Players[1].Eliminated)
	require.Equal(t, uint(1), g.WinnerID)
}

func TestOnePlayerGameRemainsPlayable(t *testing.T) {
	g := multiPlayerGame(t, 1, 10)
	require.NoError(t, g.AdvanceTurn(emptySubsFor(g)))
	require.Equal(t, "active", g.Status)
}

func TestEliminationWaitsForBattleResolution(t *testing.T) {
	g, star := newTacticalGame(t)
	removeColoniesFor(g, 2)
	armHumanGunboat(t, g, star)
	addOrbitFleet(g, 2, star, []model.FleetShip{{Type: "colony_ship", Name: "Civvie"}})
	battle := startTacticalBattle(t, g, star)
	g.evaluateConquest()
	require.Equal(t, "active", g.Status)
	var target *model.TacticalShip
	for _, ship := range battle.Ships {
		if ship.OwnerID == 2 {
			target = ship
		}
	}
	require.NotNil(t, target)
	for rounds := 0; rounds < 30 && battle.Status == model.TacticalBattleActive; rounds++ {
		current := battle.CurrentShip()
		require.NotNil(t, current)
		require.Equal(t, uint(1), current.OwnerID)
		if !current.HasFired {
			_, err := g.ApplyBattleAction(battle.ID, 1, model.BattleAction{Type: "fire", TargetID: target.ID})
			require.NoError(t, err)
		}
		if battle.Status == model.TacticalBattleActive {
			_, err := g.ApplyBattleAction(battle.ID, 1, model.BattleAction{Type: "done"})
			require.NoError(t, err)
		}
	}
	require.Equal(t, model.TacticalBattleFinished, battle.Status)
	require.True(t, g.Players[1].Eliminated)
	require.Equal(t, "finished", g.Status)
	require.Equal(t, uint(1), g.WinnerID)
}

func TestLegacySaveLoadsAsActive(t *testing.T) {
	g := multiPlayerGame(t, 2, 30)
	g.Status = ""
	require.NoError(t, ValidateLoadedGame(g, testResources(t)))
	HydrateLoadedGame(g, nil)
	require.Equal(t, "active", g.Status)
}
