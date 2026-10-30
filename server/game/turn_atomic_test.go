package game

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"

	. "ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func captureGameJSON(t *testing.T, g *Game) []byte {
	t.Helper()
	raw, err := json.Marshal(g)
	require.NoError(t, err)
	return raw
}

func cloneTurnSubmissions(src map[uint]Commands) map[uint]Commands {
	return copyTurnSubmissions(src)
}

func assertGameUnchanged(t *testing.T, g *Game, beforeJSON []byte, beforeSubs map[uint]Commands) {
	t.Helper()
	afterJSON := captureGameJSON(t, g)
	if !bytes.Equal(beforeJSON, afterJSON) {
		t.Fatalf("game JSON changed after failed advance (%d vs %d bytes)", len(beforeJSON), len(afterJSON))
	}
	require.Equal(t, beforeSubs, g.TurnSubmissions, "TurnSubmissions (json:\"-\") must match pre-failure state")
}

func multiPlayerGame(t *testing.T, numPlayers, numStars uint) *Game {
	t.Helper()
	resources, err := NewResourceManager()
	require.NoError(t, err)
	g, err := NewGame(resources, NewGameOptions{
		PlayerName:  "Human",
		PlayerColor: DefaultPlayerColor(),
		NumStars:    numStars,
		GalaxySize:  "huge",
		Race:        DefaultRace("Human"),
		NumPlayers:  numPlayers,
	})
	require.NoError(t, err)
	return g
}

func TestAdvanceTurnFailureRestoresByteIdenticalState(t *testing.T) {
	// On cae88ff this fails by assertion: mid-advance mutation sticks.
	g := multiPlayerGame(t, 2, 40)
	beforeSubs := cloneTurnSubmissions(g.TurnSubmissions)
	beforeJSON := captureGameJSON(t, g)
	treasuryBefore := g.Players[0].Treasury

	t.Cleanup(func() { advanceTurnTestHook = nil })
	advanceTurnTestHook = func(stage string) error {
		if stage == "before_fleet_movements" {
			// Mutate something visible so a missing restore is obvious.
			g.Players[0].Treasury += 999
			g.CurrentTurn += 1
			return errors.New("injected advance failure")
		}
		return nil
	}

	err := g.AdvanceTurn(map[uint]Commands{
		g.Players[0].ID: {},
		g.Players[1].ID: {},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "injected advance failure")
	assert.Equal(t, treasuryBefore, g.Players[0].Treasury)
	assertGameUnchanged(t, g, beforeJSON, beforeSubs)
}

func TestSubmitTurnFailureRemovesOnlyTriggeringSubmission(t *testing.T) {
	// Failed SubmitTurn restores world state, drops the triggering player's
	// submission, and leaves other players' submissions intact.
	g := multiPlayerGame(t, 2, 30)
	p1, p2 := g.Players[0].ID, g.Players[1].ID

	status, advanced, err := g.SubmitTurn(p1, Commands{})
	require.NoError(t, err)
	assert.False(t, advanced)
	require.Contains(t, status.ReadyPlayerIDs, p1)
	require.Contains(t, g.TurnSubmissions, p1)

	beforeJSON := captureGameJSON(t, g)
	beforeSubs := cloneTurnSubmissions(g.TurnSubmissions)

	t.Cleanup(func() { advanceTurnTestHook = nil })
	advanceTurnTestHook = func(stage string) error {
		if stage == fmt.Sprintf("after_apply:%d", p2) {
			return errors.New("injected after second player")
		}
		return nil
	}

	_, advanced, err = g.SubmitTurn(p2, Commands{})
	require.Error(t, err)
	assert.False(t, advanced)
	assert.Contains(t, err.Error(), "injected after second player")

	// Triggering player (p2) submission removed; p1 stays. World JSON matches
	// the post-p1 / pre-p2-advance state (p2 never remains ready).
	require.Contains(t, g.TurnSubmissions, p1)
	require.NotContains(t, g.TurnSubmissions, p2)
	assert.Contains(t, g.TurnStatus.WaitingPlayerIDs, p2)
	assert.Contains(t, g.TurnStatus.ReadyPlayerIDs, p1)
	assert.Equal(t, beforeSubs, g.TurnSubmissions)
	require.Equal(t, beforeJSON, captureGameJSON(t, g), "game JSON must match pre-advance")
}

func TestAdvanceTurnFailureRestoresPostponedInvasionsAndPendingBattle(t *testing.T) {
	SetRandomSeed(1)
	g, attacker, planet, fleet := orbitInvasionHarness(t)
	require.NoError(t, g.issueInvadeCommand(attacker.ID, &InvadeCommand{
		FleetID:        fleet.ID,
		TargetPlanetID: planet.ID,
	}))
	_ = g.resolveStarBattles()
	_ = g.resolveGroundInvasions()
	require.Len(t, g.PostponedInvasions, 1)
	require.Len(t, g.PendingBattles, 1)
	require.Empty(t, g.PendingInvasions)

	beforeJSON := captureGameJSON(t, g)
	beforeSubs := cloneTurnSubmissions(g.TurnSubmissions)
	postponedFleet := g.PostponedInvasions[0].FleetID
	battleID := g.PendingBattles[0].ID

	t.Cleanup(func() { advanceTurnTestHook = nil })
	advanceTurnTestHook = func(stage string) error {
		if stage == "after_auto_resolve" {
			return errors.New("injected after auto-resolve")
		}
		return nil
	}

	err := g.AdvanceTurn(map[uint]Commands{
		g.Players[0].ID: {},
		g.Players[1].ID: {},
	})
	require.Error(t, err)
	assertGameUnchanged(t, g, beforeJSON, beforeSubs)
	require.Len(t, g.PostponedInvasions, 1)
	assert.Equal(t, postponedFleet, g.PostponedInvasions[0].FleetID)
	require.Len(t, g.PendingBattles, 1)
	assert.Equal(t, battleID, g.PendingBattles[0].ID)
}

func TestAdvanceTurnFailureRestoresPendingInvasionsFromPriorOrders(t *testing.T) {
	// Seed PendingInvasions as if recorded mid-turn, then fail before they are
	// consumed — restore must bring them back (not leave nil).
	g := multiPlayerGame(t, 2, 20)
	g.PendingInvasions = []PendingInvasion{{
		OwnerID: g.Players[0].ID, FleetID: 9, TargetPlanetID: 3,
	}}
	beforeJSON := captureGameJSON(t, g)
	beforeSubs := cloneTurnSubmissions(g.TurnSubmissions)

	t.Cleanup(func() { advanceTurnTestHook = nil })
	advanceTurnTestHook = func(stage string) error {
		if stage == "before_fleet_movements" {
			return errors.New("injected before fleets")
		}
		return nil
	}

	err := g.AdvanceTurn(map[uint]Commands{
		g.Players[0].ID: {},
		g.Players[1].ID: {},
	})
	require.Error(t, err)
	assertGameUnchanged(t, g, beforeJSON, beforeSubs)
	require.Len(t, g.PendingInvasions, 1)
	assert.Equal(t, uint(9), g.PendingInvasions[0].FleetID)
}

func TestAdvanceTurnFailureAfterFirstPlayerAppliedRollsBack(t *testing.T) {
	g := multiPlayerGame(t, 2, 25)
	p1, p2 := g.Players[0].ID, g.Players[1].ID
	g.Players[0].Treasury = 100
	beforeJSON := captureGameJSON(t, g)
	beforeSubs := cloneTurnSubmissions(g.TurnSubmissions)

	t.Cleanup(func() { advanceTurnTestHook = nil })
	calls := 0
	advanceTurnTestHook = func(stage string) error {
		if stage == fmt.Sprintf("after_apply:%d", p1) {
			calls++
			// First player fully applied; mutate treasury to prove rollback.
			g.Players[0].Treasury = 1
		}
		if stage == fmt.Sprintf("after_apply:%d", p2) {
			calls++
			return errors.New("injected after second apply")
		}
		return nil
	}

	err := g.AdvanceTurn(map[uint]Commands{p1: {}, p2: {}})
	require.Error(t, err)
	assert.Equal(t, 2, calls)
	assert.Equal(t, float64(100), g.Players[0].Treasury)
	assertGameUnchanged(t, g, beforeJSON, beforeSubs)
}

func TestSuccessfulAdvanceTurnStillAdvances(t *testing.T) {
	g := multiPlayerGame(t, 2, 20)
	turnBefore := g.CurrentTurn
	require.NoError(t, g.AdvanceTurn(map[uint]Commands{
		g.Players[0].ID: {},
		g.Players[1].ID: {},
	}))
	assert.Greater(t, g.CurrentTurn, turnBefore)
	assert.Empty(t, g.TurnSubmissions)
	assert.Empty(t, g.TurnStatus.ReadyPlayerIDs)
}

func TestGameTurnSnapshotCoversAllMutableFields(t *testing.T) {
	handled := map[string]bool{}
	for _, name := range gameTurnSnapshotFieldNames {
		handled[name] = true
	}
	for _, name := range gameTurnSnapshotSharedFieldNames {
		handled[name] = true
	}
	rt := reflect.TypeOf(Game{})
	var missing []string
	for i := 0; i < rt.NumField(); i++ {
		name := rt.Field(i).Name
		if !handled[name] {
			missing = append(missing, name)
		}
	}
	assert.Empty(t, missing, "new Game fields must be added to the turn snapshot (or the shared-field list)")
}

func TestSuccessfulAdvanceMatchesApplyWithoutSnapshot(t *testing.T) {
	// Same starting world: success-path AdvanceTurn (snapshot kept) must match
	// advanceTurnApplying alone. Clone via snapshot so RNG/name generation is
	// not involved twice.
	g := multiPlayerGame(t, 2, 30)
	subs := map[uint]Commands{}
	for _, p := range g.Players {
		subs[p.ID] = Commands{}
	}

	applyOnly := &Game{Resources: g.Resources}
	restoreGameState(applyOnly, snapshotGameState(g))
	withSnap := &Game{Resources: g.Resources}
	restoreGameState(withSnap, snapshotGameState(g))

	require.NoError(t, applyOnly.advanceTurnApplying(subs))
	require.NoError(t, withSnap.AdvanceTurn(subs))

	applyOnly.LastTurnAt = nil
	withSnap.LastTurnAt = nil
	assert.Equal(t, captureGameJSON(t, applyOnly), captureGameJSON(t, withSnap))
}

func BenchmarkAdvanceTurnLargeGalaxy(b *testing.B) {
	resources, err := NewResourceManager()
	if err != nil {
		b.Fatal(err)
	}
	makeGame := func() *Game {
		g, err := NewGame(resources, NewGameOptions{
			PlayerName:  "Human",
			PlayerColor: DefaultPlayerColor(),
			NumStars:    250,
			GalaxySize:  "huge",
			Race:        DefaultRace("Human"),
			NumPlayers:  6,
		})
		if err != nil {
			b.Fatal(err)
		}
		return g
	}

	b.Run("with_snapshot", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			g := makeGame()
			subs := map[uint]Commands{}
			for _, p := range g.Players {
				subs[p.ID] = Commands{}
			}
			if err := g.AdvanceTurn(subs); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("clone_only", func(b *testing.B) {
		g := makeGame()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = snapshotGameState(g)
		}
	})

	b.Run("apply_without_outer_snapshot", func(b *testing.B) {
		// Measures advanceTurnApplying alone (no clone/restore) for comparison.
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			g := makeGame()
			subs := map[uint]Commands{}
			for _, p := range g.Players {
				subs[p.ID] = Commands{}
			}
			if err := g.advanceTurnApplying(subs); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func TestSubmitTurnDropsStaleAIOrdersAfterBattle(t *testing.T) {
	SetRandomSeed(7)
	g, star := newTacticalGame(t)
	armHumanGunboat(t, g, star)
	fleet := addOrbitFleet(g, 2, star, []FleetShip{{Type: "colony_ship", Name: "Civvie"}})
	battle := startTacticalBattle(t, g, star)
	aiName := g.Players[1].Name
	turn := g.CurrentTurn
	var destination uint
	for _, candidate := range g.Galaxy.Stars {
		for _, planet := range candidate.Planets {
			if planet.Owner == 2 {
				destination = candidate.ID
			}
		}
	}
	require.NotZero(t, destination)
	require.NotEqual(t, star.ID, destination)

	_, advanced, err := g.SubmitTurn(2, Commands{FleetMove: []*FleetMoveCommand{{FleetID: fleet.ID, DestinationStarID: destination}}})
	require.NoError(t, err)
	require.False(t, advanced)
	for i := 0; i < 200 && battle.Status == TacticalBattleActive; i++ {
		_, err = g.ApplyBattleAction(battle.ID, 1, BattleAction{Type: "auto"})
		require.NoError(t, err)
	}
	require.Equal(t, TacticalBattleFinished, battle.Status)
	require.Equal(t, uint(1), battle.WinnerOwnerID)
	_, err = g.getFleet(fleet.ID)
	require.Error(t, err)

	// Even a real stale-order failure must restore every world byte.
	beforeJSON := captureGameJSON(t, g)
	beforeSubs := cloneTurnSubmissions(g.TurnSubmissions)
	err = g.AdvanceTurn(map[uint]Commands{1: {}, 2: g.TurnSubmissions[2]})
	require.Error(t, err)
	assertGameUnchanged(t, g, beforeJSON, beforeSubs)

	status, advanced, err := g.SubmitTurn(1, Commands{})
	require.Error(t, err)
	require.ErrorContains(t, err, aiName)
	require.ErrorContains(t, err, "2")
	require.ErrorContains(t, err, "resubmit")
	require.False(t, advanced)
	require.NotContains(t, g.TurnSubmissions, uint(2))
	require.Contains(t, g.TurnSubmissions, uint(1))
	require.Equal(t, []uint{2}, status.WaitingPlayerIDs)
	require.Equal(t, turn, g.CurrentTurn)

	_, advanced, err = g.SubmitTurn(2, Commands{})
	require.NoError(t, err)
	require.True(t, advanced)
	require.Greater(t, g.CurrentTurn, turn)
}
