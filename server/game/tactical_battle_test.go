package game

import (
	"encoding/json"
	"fmt"
	"testing"

	. "ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTacticalGame builds a two-player game where player 1 is human (the
// NewGame default) and both sides have a fleet at the same neutral star.
func newTacticalGame(t *testing.T) (*Game, *Star) {
	t.Helper()
	resources, err := NewResourceManager()
	require.NoError(t, err)
	g, err := NewGame(resources, NewGameOptions{
		PlayerName:  "Ada",
		PlayerColor: DefaultPlayerColor(),
		NumStars:    8,
		Race:        DefaultRace("Human"),
		NumPlayers:  2,
	})
	require.NoError(t, err)
	return g, contestedStar(g)
}

func tacticalGunboatDesign(combatSpeed int64) ShipDesign {
	return NewShipDesign(
		"Gunboat",
		NewMultipleBasicBeams(4, 50),
		nil,
		HullSize{Name: "BS_1", SizeRating: 5, Space: 250, Marines: 20, BaseSpeed: 5, BaseHP: 50, ComputerHP: 7, DriveHP: 15, ShieldHP: 8},
		Drive{CombatSpeed: combatSpeed}, Armor{Multiplier: 1}, Computer{BeamAttackBonus: 80}, FuelCell{}, Shield{},
	)
}

func startTacticalBattle(t *testing.T, g *Game, star *Star) *TacticalBattle {
	t.Helper()
	reports := g.resolveStarBattles()
	assert.Empty(t, reports, "human battles must pause, not auto-resolve")
	require.Len(t, g.PendingBattles, 1)
	return g.PendingBattles[0]
}

func TestGetTacticalBattleVisibility(t *testing.T) {
	battle := &TacticalBattle{
		ID: 1, HumanOwnerIDs: []uint{1, 999},
		Ships: []*TacticalShip{
			nil,
			{ID: 1, OwnerID: 2, Destroyed: true},
			{ID: 2, OwnerID: 998},
		},
	}
	g := &Game{
		Players:        []*Player{{ID: 1}, {ID: 2}, {ID: 3}},
		PendingBattles: []*TacticalBattle{nil, battle},
	}
	for _, tc := range []struct {
		name     string
		playerID uint
		battleID uint
		allowed  bool
	}{
		{"human owner without ship", 1, 1, true},
		{"owner of destroyed ship", 2, 1, true},
		{"non-participant", 3, 1, false},
		{"unknown human owner", 999, 1, false},
		{"unknown ship owner", 998, 1, false},
		{"nonexistent battle", 1, 999, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := g.GetTacticalBattle(tc.battleID, tc.playerID)
			if !tc.allowed {
				require.EqualError(t, err, "battle not found")
				require.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, battle, got)
			require.NotSame(t, battle, got)
			require.NotSame(t, battle.Ships[1], got.Ships[1])
		})
	}
}

// armHumanGunboat registers the fast gunboat design for player 1 and parks a
// one-ship fleet at the star; speed 9 guarantees it tops initiative.
func armHumanGunboat(t *testing.T, g *Game, star *Star) {
	t.Helper()
	human, err := g.getPlayer(1)
	require.NoError(t, err)
	human.ShipDesigns = append(human.ShipDesigns, tacticalGunboatDesign(9))
	addOrbitFleet(g, 1, star, []FleetShip{{Type: "ship_design", Name: "Gunboat", DesignName: "Gunboat"}})
}

func TestHumanBattleBecomesPendingInsteadOfAutoResolving(t *testing.T) {
	g, star := newTacticalGame(t)
	armHumanGunboat(t, g, star)
	addOrbitFleet(g, 2, star, []FleetShip{{Type: "colony_ship", Name: "Civvie"}})

	battle := startTacticalBattle(t, g, star)
	assert.Equal(t, TacticalBattleActive, battle.Status)
	assert.Equal(t, star.ID, battle.StarID)
	assert.Equal(t, []uint{1}, battle.HumanOwnerIDs)
	assert.Len(t, battle.Ships, 2)
	assert.Empty(t, g.BattleReports)
	// Fleets stay untouched until the battle resolves.
	assert.Len(t, g.Fleets, 2)

	// The fast human gunboat acts first.
	current := battle.CurrentShip()
	require.NotNil(t, current)
	assert.Equal(t, uint(1), current.OwnerID)
	assert.Equal(t, 9, current.MovesLeft)
}

func TestTacticalBattleDeploymentUsesUniqueSquares(t *testing.T) {
	for _, tc := range []struct {
		name   string
		counts []int
	}{
		{"two sides", []int{1, 1}},
		{"three sides", []int{1, 1, 1}},
		{"four sides", []int{1, 1, 1, 1}},
		{"four sides with full left edge", []int{10, 1, 1, 1}},
		{"four sides with inward overflow", []int{21, 21, 29, 29}},
		{"full grid", []int{35, 35, 35, 35}},
		{"large first side", []int{137, 1, 1, 1}},
		{"beyond default grid capacity", []int{36, 36, 36, 36}},
		{"large last side", []int{1, 1, 1, 150}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resources, err := NewResourceManager()
			require.NoError(t, err)
			g, err := NewGame(resources, NewGameOptions{
				PlayerName:  "Ada",
				PlayerColor: DefaultPlayerColor(),
				NumStars:    8,
				Race:        DefaultRace("Human"),
				NumPlayers:  uint(len(tc.counts)),
			})
			require.NoError(t, err)
			star := contestedStar(g)
			total := 0
			for side, count := range tc.counts {
				// Keep every side human so AI actions cannot move ships before
				// we inspect their starting squares.
				g.Players[side].IsHuman = true
				ships := make([]FleetShip, count)
				for i := range ships {
					ships[i] = FleetShip{Type: "colony_ship", Name: fmt.Sprintf("Ship %d", i)}
				}
				addOrbitFleet(g, g.Players[side].ID, star, ships)
				total += count
			}

			battle := startTacticalBattle(t, g, star)
			require.Len(t, battle.Ships, total)
			require.Len(t, battle.TurnOrder, total)
			occupied := map[[2]int]uint{}
			for _, ship := range battle.Ships {
				require.False(t, ship.Destroyed)
				require.True(t, ship.X >= 0 && ship.X < battle.GridWidth && ship.Y >= 0 && ship.Y < battle.GridHeight,
					"ship %d starts off grid at (%d,%d)", ship.ID, ship.X, ship.Y)
				position := [2]int{ship.X, ship.Y}
				require.NotContains(t, occupied, position,
					"ship %d (owner %d) overlaps ship %d at %v", ship.ID, ship.OwnerID, occupied[position], position)
				occupied[position] = ship.ID
				require.Same(t, ship, battle.ShipAt(ship.X, ship.Y), "every ship must be targetable by its square")
			}
			if total <= TacticalGridWidth*TacticalGridHeight {
				assert.Equal(t, TacticalGridWidth, battle.GridWidth)
				assert.Equal(t, TacticalGridHeight, battle.GridHeight)
			}
		})
	}
}

func TestAIOnlyBattleStillAutoResolves(t *testing.T) {
	g, star := newTacticalGame(t)
	g.Players[0].IsHuman = false
	addOrbitFleet(g, 1, star, []FleetShip{{Type: "colony_ship", Name: "A"}})
	addOrbitFleet(g, 2, star, []FleetShip{{Type: "colony_ship", Name: "B"}})

	reports := g.resolveStarBattles()
	require.Len(t, reports, 1)
	assert.Empty(t, g.PendingBattles)
}

func TestBattleMoveValidation(t *testing.T) {
	g, star := newTacticalGame(t)
	armHumanGunboat(t, g, star)
	addOrbitFleet(g, 2, star, []FleetShip{{Type: "colony_ship", Name: "Civvie"}})
	battle := startTacticalBattle(t, g, star)
	ship := battle.CurrentShip()
	require.NotNil(t, ship)

	_, err := g.ApplyBattleAction(battle.ID, 1, BattleAction{Type: "move", X: -1, Y: 0})
	assert.ErrorContains(t, err, "off the grid")

	_, err = g.ApplyBattleAction(battle.ID, 1, BattleAction{Type: "move", X: 13, Y: 9})
	assert.ErrorContains(t, err, "not enough movement")

	// Enemy civilian sits on the right edge at (13, 0).
	_, err = g.ApplyBattleAction(battle.ID, 1, BattleAction{Type: "move", X: 13, Y: 0})
	assert.ErrorContains(t, err, "occupied")

	_, err = g.ApplyBattleAction(battle.ID, 1, BattleAction{Type: "move", X: 5, Y: 4})
	require.NoError(t, err)
	assert.Equal(t, 5, ship.X)
	assert.Equal(t, 4, ship.Y)
	assert.Equal(t, 9-5, ship.MovesLeft)

	// Acting out of turn is rejected.
	_, err = g.ApplyBattleAction(battle.ID, 2, BattleAction{Type: "done"})
	assert.ErrorContains(t, err, "not player 2's turn")
}

func TestBattleFireDestroysTargetAndFinishes(t *testing.T) {
	SetRandomSeed(7)
	g, star := newTacticalGame(t)
	armHumanGunboat(t, g, star)
	addOrbitFleet(g, 2, star, []FleetShip{{Type: "colony_ship", Name: "Civvie"}})
	battle := startTacticalBattle(t, g, star)
	ship := battle.CurrentShip()
	require.NotNil(t, ship)

	var target *TacticalShip
	for _, ts := range battle.Ships {
		if ts.OwnerID == 2 {
			target = ts
		}
	}
	require.NotNil(t, target)

	// Close in and fire until the fragile civilian shell (10 HP) pops. The
	// gunboat fires 4x50 beams with +80 attack so this ends within a few
	// rounds regardless of seed; the loop guards against test hangs.
	for rounds := 0; rounds < 20 && battle.Status == TacticalBattleActive; rounds++ {
		current := battle.CurrentShip()
		require.NotNil(t, current)
		require.Equal(t, uint(1), current.OwnerID)
		if !current.HasFired {
			if _, err := g.ApplyBattleAction(battle.ID, 1, BattleAction{Type: "fire", TargetID: target.ID}); err != nil {
				t.Fatalf("fire failed: %v", err)
			}
		}
		if battle.Status == TacticalBattleActive {
			_, err := g.ApplyBattleAction(battle.ID, 1, BattleAction{Type: "done"})
			require.NoError(t, err)
		}
	}

	assert.Equal(t, TacticalBattleFinished, battle.Status)
	assert.Equal(t, uint(1), battle.WinnerOwnerID)
	assert.True(t, target.Destroyed)

	// Battle resolved: pending list cleared, report recorded, fleets pruned.
	assert.Empty(t, g.PendingBattles)
	require.Len(t, g.BattleReports, 1)
	report := g.BattleReports[0]
	assert.Equal(t, uint(1), report.WinnerOwnerID)
	owners := map[uint]int{}
	for _, fleet := range g.Fleets {
		owners[fleet.Owner] += len(fleet.Ships)
	}
	assert.Equal(t, 1, owners[1])
	assert.Equal(t, 0, owners[2])

	// The event log tells the full story for client playback.
	types := map[string]int{}
	for _, e := range battle.Events {
		types[e.Type]++
	}
	assert.Greater(t, types["fire"], 0)
	assert.Equal(t, 1, types["destroyed"])
	assert.Equal(t, 1, types["finished"])
}

func TestBattleWaitDefersActivation(t *testing.T) {
	g, star := newTacticalGame(t)
	armHumanGunboat(t, g, star)
	// Two human ships so wait has somewhere to rotate to; enemy is slower.
	addOrbitFleet(g, 1, star, []FleetShip{{Type: "ship_design", Name: "Gunboat 2", DesignName: "Gunboat"}})
	addOrbitFleet(g, 2, star, []FleetShip{{Type: "colony_ship", Name: "Civvie"}})
	battle := startTacticalBattle(t, g, star)

	first := battle.CurrentShip()
	require.NotNil(t, first)
	_, err := g.ApplyBattleAction(battle.ID, 1, BattleAction{Type: "wait"})
	require.NoError(t, err)
	second := battle.CurrentShip()
	require.NotNil(t, second)
	assert.NotEqual(t, first.ID, second.ID)
	assert.Equal(t, first.ID, battle.TurnOrder[len(battle.TurnOrder)-1])
	// Still round 1.
	assert.Equal(t, 1, battle.Round)
}

func TestSubmitTurnBlockedWhileBattlePending(t *testing.T) {
	g, star := newTacticalGame(t)
	armHumanGunboat(t, g, star)
	addOrbitFleet(g, 2, star, []FleetShip{{Type: "colony_ship", Name: "Civvie"}})
	startTacticalBattle(t, g, star)

	_, _, err := g.SubmitTurn(1, Commands{})
	assert.ErrorContains(t, err, "resolve pending battle")

	// Non-participant AI players are not blocked.
	_, _, err = g.SubmitTurn(2, Commands{})
	assert.NoError(t, err)
}

func TestAdvanceTurnAutoResolvesLeftoverBattles(t *testing.T) {
	SetRandomSeed(11)
	g, star := newTacticalGame(t)
	armHumanGunboat(t, g, star)
	addOrbitFleet(g, 2, star, []FleetShip{{Type: "colony_ship", Name: "Civvie"}})
	startTacticalBattle(t, g, star)

	err := g.AdvanceTurn(map[uint]Commands{1: {}, 2: {}})
	require.NoError(t, err)
	assert.Empty(t, g.PendingBattles)
	require.NotEmpty(t, g.BattleReports)
}

func TestBattleAutoActionPlaysActivation(t *testing.T) {
	SetRandomSeed(3)
	g, star := newTacticalGame(t)
	armHumanGunboat(t, g, star)
	addOrbitFleet(g, 2, star, []FleetShip{{Type: "colony_ship", Name: "Civvie"}})
	battle := startTacticalBattle(t, g, star)

	// Repeated autos must finish the battle (gunboat hunts down civilian).
	for i := 0; i < 200 && battle.Status == TacticalBattleActive; i++ {
		_, err := g.ApplyBattleAction(battle.ID, 1, BattleAction{Type: "auto"})
		require.NoError(t, err)
	}
	assert.Equal(t, TacticalBattleFinished, battle.Status)
	assert.Equal(t, uint(1), battle.WinnerOwnerID)
}

func TestTacticalBattleDestroyedSideSerializesEmptySurvivors(t *testing.T) {
	g, star := newTacticalGame(t)
	armHumanGunboat(t, g, star)
	addOrbitFleet(g, 2, star, []FleetShip{{Type: "colony_ship", Name: "Civvie"}})
	battle := startTacticalBattle(t, g, star)
	for _, ship := range battle.Ships {
		if ship.OwnerID == 2 {
			ship.Destroyed = true
		}
	}
	g.finishTacticalBattle(battle, 1, false)
	check := func(t *testing.T, reports []*BattleReport) {
		t.Helper()
		require.Len(t, reports, 1)
		var defeated *BattleSide
		for i := range reports[0].Sides {
			if reports[0].Sides[i].OwnerID == 2 {
				defeated = &reports[0].Sides[i]
			}
		}
		require.NotNil(t, defeated)
		data, err := json.Marshal(defeated)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"surviving_ships":[]`)
		assert.Contains(t, string(data), `"original_ships":[`)
	}
	t.Run("stored", func(t *testing.T) { check(t, g.BattleReports) })
	for _, playerID := range []uint{1, 2} {
		visible, err := g.GetVisibleGameForPlayer(playerID)
		require.NoError(t, err)
		t.Run(fmt.Sprintf("player %d", playerID), func(t *testing.T) { check(t, visible.BattleReports) })
	}
}

func TestLastShipWaitDoesNotRepeatActivation(t *testing.T) {
	g, star := newTacticalGame(t)
	armHumanGunboat(t, g, star)
	addOrbitFleet(g, 2, star, []FleetShip{{Type: "colony_ship", Name: "Civvie"}})
	battle := startTacticalBattle(t, g, star)
	ship := battle.CurrentShip()
	// Simulate the last activation in the round after other ships have acted.
	battle.TurnOrder = []uint{ship.ID}
	battle.TurnIndex = 0
	initial := len(battle.Events)
	for i := 0; i < 3; i++ {
		_, err := g.ApplyBattleAction(battle.ID, 1, BattleAction{Type: "wait"})
		require.NoError(t, err)
	}
	assert.Len(t, battle.Events, initial)
}

func TestBattleTurnMatchesCompletedTurn(t *testing.T) {
	g, star := newTacticalGame(t)
	armHumanGunboat(t, g, star)
	addOrbitFleet(g, 2, star, []FleetShip{{Type: "colony_ship", Name: "Civvie"}})
	require.NoError(t, g.AdvanceTurn(emptySubsFor(g)))
	require.Len(t, g.PendingBattles, 1)
	assert.Equal(t, g.CurrentTurn, g.PendingBattles[0].GameTurn)
}
