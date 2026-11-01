package game

import (
	"encoding/json"
	"testing"

	. "ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// orbitInvasionHarness parks a human transport fleet at a defender planet and
// an enemy armed fleet at the same star so resolveStarBattles creates a
// pending tactical battle. Returns game, attacker, defender planet, transport fleet.
func orbitInvasionHarness(t *testing.T) (*Game, *Player, *Planet, *Fleet) {
	t.Helper()
	g, star := newTacticalGame(t)
	attacker := g.Players[0]
	require.True(t, attacker.IsHuman)

	defenderPlanet := &Planet{
		ID:         8801,
		Name:       "Target World",
		Owner:      2,
		ParentStar: star.ID,
		Population: 2000,
		Marines:    0,
		Armored:    0,
	}
	star.Planets = append(star.Planets, defenderPlanet)

	human, err := g.getPlayer(1)
	require.NoError(t, err)
	human.ShipDesigns = append(human.ShipDesigns, tacticalGunboatDesign(9))

	transport := addOrbitFleet(g, 1, star, []FleetShip{
		{Name: "Transport", Type: "transport"},
		{Name: "Escort", Type: "ship_design", DesignName: "Gunboat"},
	})
	addOrbitFleet(g, 2, star, []FleetShip{{Type: "colony_ship", Name: "Civvie"}})

	return g, attacker, defenderPlanet, transport
}

func TestInvasionPostponedWhileTacticalBattlePending(t *testing.T) {
	// (a) invasion at a star with an active pending battle does not resolve
	// that turn; planet unchanged; order retained as postponed.
	SetRandomSeed(1)
	g, attacker, planet, fleet := orbitInvasionHarness(t)
	ownerBefore := planet.Owner
	marinesBefore := planet.Marines

	require.NoError(t, g.issueInvadeCommand(attacker.ID, &InvadeCommand{
		FleetID:        fleet.ID,
		TargetPlanetID: planet.ID,
	}))
	require.Len(t, g.PendingInvasions, 1)

	_ = g.resolveStarBattles()
	require.Len(t, g.PendingBattles, 1)

	reports := g.resolveGroundInvasions()
	assert.Empty(t, reports)
	assert.Empty(t, g.PendingInvasions)
	require.Len(t, g.PostponedInvasions, 1)
	assert.Equal(t, fleet.ID, g.PostponedInvasions[0].FleetID)
	assert.Equal(t, planet.ID, g.PostponedInvasions[0].TargetPlanetID)
	assert.Equal(t, ownerBefore, planet.Owner)
	assert.Equal(t, marinesBefore, planet.Marines)
	assert.Empty(t, g.InvasionReports)
}

func TestPostponedInvasionResolvesAfterBattleWin(t *testing.T) {
	// (b) battle won by invader -> postponed invasion resolves; planet captured.
	SetRandomSeed(7)
	g, attacker, planet, fleet := orbitInvasionHarness(t)
	indexes := []uint{0}
	require.NoError(t, g.issueInvadeCommand(attacker.ID, &InvadeCommand{
		FleetID:        fleet.ID,
		TargetPlanetID: planet.ID,
		ShipIndexes:    &indexes,
	}))

	battle := startTacticalBattle(t, g, mustStar(t, g, planet.ParentStar))
	reports := g.resolveGroundInvasions()
	assert.Empty(t, reports)
	require.Len(t, g.PostponedInvasions, 1)

	// Fight until human wins (civilian target).
	var target *TacticalShip
	for _, ts := range battle.Ships {
		if ts.OwnerID == 2 {
			target = ts
		}
	}
	require.NotNil(t, target)
	for rounds := 0; rounds < 20 && battle.Status == TacticalBattleActive; rounds++ {
		current := battle.CurrentShip()
		require.NotNil(t, current)
		require.Equal(t, uint(1), current.OwnerID)
		if !current.HasFired {
			_, err := g.ApplyBattleAction(battle.ID, 1, BattleAction{Type: "fire", TargetID: target.ID})
			require.NoError(t, err)
		}
		if battle.Status == TacticalBattleActive {
			_, err := g.ApplyBattleAction(battle.ID, 1, BattleAction{Type: "done"})
			require.NoError(t, err)
		}
	}
	assert.Equal(t, TacticalBattleFinished, battle.Status)
	assert.Equal(t, uint(1), battle.WinnerOwnerID)
	assert.Empty(t, g.PendingBattles)
	assert.Empty(t, g.PostponedInvasions, "postponed invasion must release after win")
	assert.Equal(t, attacker.ID, planet.Owner, "planet must be captured after orbit win")
	require.NotEmpty(t, g.InvasionReports)
	last := g.InvasionReports[len(g.InvasionReports)-1]
	assert.False(t, last.Cancelled)
	assert.True(t, last.PlanetCaptured)
}

func TestPostponedInvasionCancelledAfterBattleLoss(t *testing.T) {
	// (c) battle lost by invader (fleet destroyed) -> invasion cancelled,
	// report produced, planet unchanged.
	SetRandomSeed(11)
	g, attacker, planet, fleet := orbitInvasionHarness(t)
	require.NoError(t, g.issueInvadeCommand(attacker.ID, &InvadeCommand{
		FleetID: fleet.ID, TargetPlanetID: planet.ID,
	}))
	battle := startTacticalBattle(t, g, mustStar(t, g, planet.ParentStar))
	_ = g.resolveGroundInvasions()
	require.Len(t, g.PostponedInvasions, 1)
	ownerBefore := planet.Owner
	marinesBefore := planet.Marines

	// Simulate total loss: destroy the invader's ships and finish for the AI.
	for _, ts := range battle.Ships {
		if ts.OwnerID == attacker.ID {
			ts.Destroyed = true
		}
	}
	g.finishTacticalBattle(battle, 2, false)

	assert.Empty(t, g.PendingBattles)
	assert.Empty(t, g.PostponedInvasions)
	assert.Equal(t, ownerBefore, planet.Owner)
	assert.Equal(t, marinesBefore, planet.Marines)
	require.NotEmpty(t, g.InvasionReports)
	last := g.InvasionReports[len(g.InvasionReports)-1]
	assert.True(t, last.Cancelled)
	assert.False(t, last.Success)
	assert.False(t, last.PlanetCaptured)
	assert.Equal(t, planet.ID, last.PlanetID)
	assert.Equal(t, attacker.ID, last.AttackerOwnerID)
}

func TestAIvsAIInvasionRequiresOrbitControl(t *testing.T) {
	// (d) AI-vs-AI auto-resolved fight: loser does not invade; winner does.
	t.Run("winner invades", func(t *testing.T) {
		SetRandomSeed(7)
		g := newBattleGame(t)
		star := contestedStar(g)
		planet := &Planet{
			ID: 8810, Name: "Prize", Owner: 2, ParentStar: star.ID,
			Population: 1000, Marines: 0, Armored: 0,
		}
		star.Planets = append(star.Planets, planet)

		attacker, err := g.getPlayer(1)
		require.NoError(t, err)
		attacker.ShipDesigns = append(attacker.ShipDesigns, NewShipDesign(
			"Gunboat",
			NewMultipleBasicBeams(4, 50),
			nil,
			HullSize{Name: "BS_1", SizeRating: 5, Space: 250, Marines: 20, BaseSpeed: 5, BaseHP: 50, ComputerHP: 7, DriveHP: 15, ShieldHP: 8},
			Drive{}, Armor{Multiplier: 1}, Computer{BeamAttackBonus: 80}, FuelCell{}, Shield{},
		))
		invaderFleet := addOrbitFleet(g, 1, star, []FleetShip{
			{Type: "ship_design", Name: "Gunboat", DesignName: "Gunboat"},
			{Type: "transport", Name: "Landers"},
		})
		addOrbitFleet(g, 2, star, []FleetShip{{Type: "colony_ship", Name: "Civvie"}})

		g.PendingInvasions = []PendingInvasion{{
			OwnerID: 1, FleetID: invaderFleet.ID, TargetPlanetID: planet.ID,
		}}
		_ = g.resolveStarBattles()
		assert.Empty(t, g.PendingBattles)
		assert.True(t, g.holdsOrbit(star.ID, 1))

		reports := g.resolveGroundInvasions()
		require.NotEmpty(t, reports)
		assert.False(t, reports[0].Cancelled)
		assert.Equal(t, uint(1), planet.Owner)
		assert.Empty(t, g.PostponedInvasions)
	})

	t.Run("loser cancelled", func(t *testing.T) {
		SetRandomSeed(7)
		g := newBattleGame(t)
		star := contestedStar(g)
		planet := &Planet{
			ID: 8811, Name: "Fort", Owner: 2, ParentStar: star.ID,
			Population: 1000, Marines: 5, Armored: 0,
		}
		star.Planets = append(star.Planets, planet)

		defender, err := g.getPlayer(2)
		require.NoError(t, err)
		defender.ShipDesigns = append(defender.ShipDesigns, NewShipDesign(
			"Gunboat",
			NewMultipleBasicBeams(4, 50),
			nil,
			HullSize{Name: "BS_1", SizeRating: 5, Space: 250, Marines: 20, BaseSpeed: 5, BaseHP: 50, ComputerHP: 7, DriveHP: 15, ShieldHP: 8},
			Drive{}, Armor{Multiplier: 1}, Computer{BeamAttackBonus: 80}, FuelCell{}, Shield{},
		))
		// Unarmed invader transports vs armed defender — invader dies.
		invaderFleet := addOrbitFleet(g, 1, star, []FleetShip{{Type: "transport", Name: "Landers"}})
		addOrbitFleet(g, 2, star, []FleetShip{{Type: "ship_design", Name: "Gunboat", DesignName: "Gunboat"}})

		g.PendingInvasions = []PendingInvasion{{
			OwnerID: 1, FleetID: invaderFleet.ID, TargetPlanetID: planet.ID,
		}}
		_ = g.resolveStarBattles()
		assert.False(t, g.holdsOrbit(star.ID, 1))

		reports := g.resolveGroundInvasions()
		require.Len(t, reports, 1)
		assert.True(t, reports[0].Cancelled)
		assert.Equal(t, uint(2), planet.Owner)
		assert.Empty(t, g.PostponedInvasions)
		assert.Empty(t, g.PendingInvasions)
	})
}

func TestPostponedInvasionSurvivesSaveLoad(t *testing.T) {
	// (e) postponed order survives save/load while battle pending; not duplicated.
	SetRandomSeed(1)
	g, attacker, planet, fleet := orbitInvasionHarness(t)
	require.NoError(t, g.issueInvadeCommand(attacker.ID, &InvadeCommand{
		FleetID: fleet.ID, TargetPlanetID: planet.ID,
	}))
	_ = g.resolveStarBattles()
	_ = g.resolveGroundInvasions()
	require.Len(t, g.PostponedInvasions, 1)
	require.Len(t, g.PendingBattles, 1)

	data, err := json.Marshal(g)
	require.NoError(t, err)

	var loaded Game
	require.NoError(t, json.Unmarshal(data, &loaded))
	require.Len(t, loaded.PostponedInvasions, 1)
	require.Len(t, loaded.PendingBattles, 1)

	resources, err := NewResourceManager()
	require.NoError(t, err)
	HydrateLoadedGame(&loaded, resources)
	require.Len(t, loaded.PostponedInvasions, 1, "hydrate must keep postponed while battle lives")
	assert.Empty(t, loaded.PendingInvasions)
	assert.Equal(t, fleet.ID, loaded.PostponedInvasions[0].FleetID)

	// Second hydrate / postpone must not duplicate.
	HydrateLoadedGame(&loaded, resources)
	require.Len(t, loaded.PostponedInvasions, 1)
	loaded.postponeInvasion(loaded.PostponedInvasions[0])
	require.Len(t, loaded.PostponedInvasions, 1)

	// Without a pending battle, hydrate drops the postponed order.
	loaded.PendingBattles = nil
	HydrateLoadedGame(&loaded, resources)
	assert.Empty(t, loaded.PostponedInvasions)
}

func TestNoBattleInvasionStillResolvesSameTurn(t *testing.T) {
	// (f) no-battle case unchanged: normal invasion resolves same turn;
	// PendingInvasions cleared as before.
	SetRandomSeed(3)
	game, attacker, defenderPlanet := invasionHarness(t)
	defenderPlanet.Marines = 0
	defenderPlanet.Armored = 0
	fleet := &Fleet{
		ID: 42, Owner: attacker.ID, StarID: defenderPlanet.ParentStar,
		Ships: []FleetShip{{Name: "Transport 1", Type: "transport"}},
	}
	game.Fleets = append(game.Fleets, fleet)
	indexes := []uint{0}

	require.NoError(t, game.AdvanceTurn(map[uint]Commands{
		attacker.ID: {
			Invade: []*InvadeCommand{{
				FleetID: fleet.ID, TargetPlanetID: defenderPlanet.ID, ShipIndexes: &indexes,
			}},
		},
		2: {},
	}))
	assert.Empty(t, game.PendingInvasions)
	assert.Empty(t, game.PostponedInvasions)
	_, err := game.getFleet(fleet.ID)
	require.Error(t, err, "used-up transport fleet should be removed")
	assert.NotContains(t, attacker.Fleets, fleet.ID)
	assert.Equal(t, attacker.ID, defenderPlanet.Owner)
	require.NotEmpty(t, game.InvasionReports)
	assert.False(t, game.InvasionReports[len(game.InvasionReports)-1].Cancelled)
}

func TestHoldsOrbitDefinition(t *testing.T) {
	g := newBattleGame(t)
	star := contestedStar(g)
	assert.False(t, g.holdsOrbit(star.ID, 1), "no ships → no orbit")

	addOrbitFleet(g, 1, star, []FleetShip{{Type: "transport", Name: "T"}})
	assert.True(t, g.holdsOrbit(star.ID, 1), "alone with transports → holds")

	addOrbitFleet(g, 2, star, []FleetShip{{Type: "colony_ship", Name: "Civ"}})
	assert.True(t, g.holdsOrbit(star.ID, 1), "rival unarmed civilians do not deny orbit")

	defender, err := g.getPlayer(2)
	require.NoError(t, err)
	defender.ShipDesigns = append(defender.ShipDesigns, NewShipDesign(
		"Gunboat",
		NewMultipleBasicBeams(1, 10),
		nil,
		HullSize{Name: "BS_1", SizeRating: 5, Space: 250, Marines: 20, BaseSpeed: 5, BaseHP: 50, ComputerHP: 7, DriveHP: 15, ShieldHP: 8},
		Drive{}, Armor{Multiplier: 1}, Computer{}, FuelCell{}, Shield{},
	))
	addOrbitFleet(g, 2, star, []FleetShip{{Type: "ship_design", Name: "Gunboat", DesignName: "Gunboat"}})
	assert.False(t, g.holdsOrbit(star.ID, 1), "rival armed ships deny orbit")
}

func TestValidateLoadedGameRejectsOrphanPostponedInvasion(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)
	g, err := NewGame(resources, NewGameOptions{
		PlayerName: "Ada", PlayerColor: DefaultPlayerColor(),
		NumStars: 6, Race: DefaultRace("Human"), NumPlayers: 2,
	})
	require.NoError(t, err)
	g.PostponedInvasions = []PendingInvasion{{
		OwnerID: 1, FleetID: 9999, TargetPlanetID: g.Galaxy.Stars[0].Planets[0].ID,
	}}
	err = ValidateLoadedGame(g, resources)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing fleet")
}

func mustStar(t *testing.T, g *Game, starID uint) *Star {
	t.Helper()
	star, err := g.getStar(starID)
	require.NoError(t, err)
	return star
}
