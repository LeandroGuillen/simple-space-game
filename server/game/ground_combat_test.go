package game

import (
	"encoding/json"
	"testing"

	. "ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGroundUnit(t *testing.T) {
	marine := NewGroundUnit(MarineUnit)
	assert.Equal(t, MarineUnit, marine.Type)
	assert.Equal(t, int64(10), marine.Strength)
	assert.Equal(t, int64(20), marine.Health)
	assert.Equal(t, int64(20), marine.MaxHP)
	assert.True(t, marine.IsAlive())

	armored := NewGroundUnit(ArmoredUnit)
	assert.Equal(t, ArmoredUnit, armored.Type)
	assert.Equal(t, int64(25), armored.Strength)
	assert.Equal(t, int64(40), armored.Health)
	assert.Equal(t, int64(40), armored.MaxHP)
	assert.True(t, armored.IsAlive())
}

func TestGroundUnit_IsAlive(t *testing.T) {
	unit := NewGroundUnit(MarineUnit)
	assert.True(t, unit.IsAlive())

	unit.Health = 0
	assert.False(t, unit.IsAlive())

	unit.Health = -5
	assert.False(t, unit.IsAlive())
}

func TestGroundForce_TotalStrength(t *testing.T) {
	force := GroundForce{
		Marines: []GroundUnit{
			NewGroundUnit(MarineUnit), // 10 strength
			NewGroundUnit(MarineUnit), // 10 strength
		},
		Armored: []GroundUnit{
			NewGroundUnit(ArmoredUnit), // 25 strength
		},
	}

	assert.Equal(t, int64(45), force.TotalStrength()) // 10 + 10 + 25
}

func TestGroundForce_TotalStrengthWithDeadUnits(t *testing.T) {
	force := GroundForce{
		Marines: []GroundUnit{
			NewGroundUnit(MarineUnit),                              // 10 strength, alive
			{Type: MarineUnit, Strength: 10, Health: 0, MaxHP: 20}, // dead
		},
		Armored: []GroundUnit{
			NewGroundUnit(ArmoredUnit), // 25 strength, alive
		},
	}

	assert.Equal(t, int64(35), force.TotalStrength()) // 10 + 0 + 25
}

func TestGroundForce_TotalUnits(t *testing.T) {
	force := GroundForce{
		Marines: []GroundUnit{
			NewGroundUnit(MarineUnit),
			NewGroundUnit(MarineUnit),
		},
		Armored: []GroundUnit{
			NewGroundUnit(ArmoredUnit),
		},
	}

	assert.Equal(t, 3, force.TotalUnits())
}

func TestGroundForce_HasLivingUnits(t *testing.T) {
	force := GroundForce{
		Marines: []GroundUnit{NewGroundUnit(MarineUnit)},
	}
	assert.True(t, force.HasLivingUnits())

	// All units dead
	force.Marines[0].Health = 0
	assert.False(t, force.HasLivingUnits())

	// No units at all
	force.Marines = []GroundUnit{}
	assert.False(t, force.HasLivingUnits())
}

func TestCalculateGroundDamage(t *testing.T) {
	// Create test players
	attacker := &Player{Race: DefaultRace("Human")}
	defender := &Player{Race: DefaultRace("Klackon")}

	// Create attacking force with marines
	attackerForce := GroundForce{
		Marines: []GroundUnit{
			NewGroundUnit(MarineUnit), // 10 strength
			NewGroundUnit(MarineUnit), // 10 strength
		},
	}

	// Create defending force
	defenderForce := GroundForce{
		Marines: []GroundUnit{NewGroundUnit(MarineUnit)},
	}

	// Calculate damage (this is probabilistic, so we check range)
	damage := calculateGroundDamage(&attackerForce, &defenderForce, attacker, defender)
	assert.GreaterOrEqual(t, damage, int64(0))
	assert.LessOrEqual(t, damage, int64(20)) // Max possible with 20 total strength
}

func TestApplyGroundDamage(t *testing.T) {
	force := GroundForce{
		Marines: []GroundUnit{
			NewGroundUnit(MarineUnit), // 20 HP
			NewGroundUnit(MarineUnit), // 20 HP
		},
		Armored: []GroundUnit{
			NewGroundUnit(ArmoredUnit), // 40 HP
		},
	}

	// Apply 25 damage (should kill first marine and damage second)
	killed := applyGroundDamage(&force, 25)

	assert.Equal(t, 1, killed) // First marine should be killed
	assert.False(t, force.Marines[0].IsAlive())
	assert.Equal(t, int64(0), force.Marines[0].Health)
	assert.Equal(t, int64(15), force.Marines[1].Health) // 20 - 5 remaining damage
	assert.True(t, force.Marines[1].IsAlive())
	assert.True(t, force.Armored[0].IsAlive()) // Armored unit untouched
}

func TestApplyGroundDamage_KillsAllUnits(t *testing.T) {
	force := GroundForce{
		Marines: []GroundUnit{
			NewGroundUnit(MarineUnit), // 20 HP
		},
		Armored: []GroundUnit{
			NewGroundUnit(ArmoredUnit), // 40 HP
		},
	}

	// Apply massive damage
	killed := applyGroundDamage(&force, 100)

	assert.Equal(t, 2, killed) // Both units killed
	assert.False(t, force.Marines[0].IsAlive())
	assert.False(t, force.Armored[0].IsAlive())
}

func TestRunGroundBattle(t *testing.T) {
	// Create test players
	attacker := &Player{Race: DefaultRace("Human")}
	defender := &Player{Race: DefaultRace("Klackon")}

	// Create forces
	attackerForce := GroundForce{
		Marines: []GroundUnit{
			NewGroundUnit(MarineUnit),
			NewGroundUnit(MarineUnit),
			NewGroundUnit(MarineUnit), // 3 marines vs 1 for advantage
		},
	}

	defenderForce := GroundForce{
		Marines: []GroundUnit{NewGroundUnit(MarineUnit)},
	}

	// Run battle
	rounds := runGroundBattle(&attackerForce, &defenderForce, attacker, defender)

	// Battle should complete with some rounds
	assert.Greater(t, len(rounds), 0)
	assert.LessOrEqual(t, len(rounds), maxGroundBattleRounds)

	// One side should be eliminated
	assert.False(t, attackerForce.HasLivingUnits() && defenderForce.HasLivingUnits())
}

func TestGroundUnitProduction(t *testing.T) {
	game := newTestGame(t)
	player := game.Players[0]

	// Get a planet owned by the player
	var planet *Planet
	for _, star := range game.Galaxy.Stars {
		for _, p := range star.Planets {
			if p.Owner == player.ID {
				planet = p
				break
			}
		}
		if planet != nil {
			break
		}
	}
	require.NotNil(t, planet, "Test game should have player-owned planet")

	// Add Marine Barracks
	planet.Buildings = []string{"Marine Barracks"}

	initialMarines := planet.Marines

	// Run ground unit production
	produceGroundUnits(planet, game, player)

	// Should produce 1 marine (from Marine Barracks modifier)
	assert.Equal(t, initialMarines+1, planet.Marines)
}

func TestGroundCombatWithRaceBonus(t *testing.T) {
	resources := testResources(t)

	// Create player with ground combat trait
	groundCombatRace, err := BuildCustomRace(resources, CustomRaceDesign{
		Name:     "Warriors",
		TraitIDs: []string{"ground_combat"},
	})
	require.NoError(t, err)

	attacker := &Player{Race: groundCombatRace}

	// Test ground combat multiplier
	multiplier := GroundCombatMultiplier(attacker)
	assert.Equal(t, 1.5, multiplier) // Ground combat trait gives +50%

	// This is probabilistic, so we can't guarantee exact results in a single test,
	// but we can test the multiplier function works correctly
	assert.Greater(t, multiplier, 1.0)
}

func TestFleetHasTransportMarines(t *testing.T) {
	game := newTestGame(t)
	player := game.Players[0]

	// Create fleet with transport ships
	fleet := &Fleet{
		Owner: player.ID,
		Ships: []FleetShip{
			{Name: "Transport 1", Type: "transport"},
		},
	}

	hasMarines := game.fleetHasTransportMarines(fleet)
	assert.True(t, hasMarines)

	// Fleet without transports
	fleetNoMarines := &Fleet{
		Owner: player.ID,
		Ships: []FleetShip{
			{Name: "Colony Ship", Type: "colony_ship"},
		},
	}

	hasMarinesEmpty := game.fleetHasTransportMarines(fleetNoMarines)
	assert.False(t, hasMarinesEmpty)
}

func TestCopyGroundForce(t *testing.T) {
	original := GroundForce{
		OwnerID: 1,
		Marines: []GroundUnit{NewGroundUnit(MarineUnit)},
		Armored: []GroundUnit{NewGroundUnit(ArmoredUnit)},
	}

	copy := copyGroundForce(original)

	// Should be equal but separate instances
	assert.Equal(t, original.OwnerID, copy.OwnerID)
	assert.Equal(t, len(original.Marines), len(copy.Marines))
	assert.Equal(t, len(original.Armored), len(copy.Armored))
	assert.Equal(t, original.Marines[0], copy.Marines[0])

	// Modifying copy shouldn't affect original
	copy.Marines[0].Health = 0
	assert.NotEqual(t, original.Marines[0].Health, copy.Marines[0].Health)
}

// Integration test for full invasion scenario
func TestFullInvasionScenario(t *testing.T) {
	game := newTestGame(t)
	attacker := game.Players[0]

	// Create a second player manually for this test
	defender := &Player{
		ID:   2,
		Name: "Defender",
		Race: DefaultRace("Klackon"),
	}
	game.Players = append(game.Players, defender)

	// Find a planet to make the defender's and add some defenders
	var defenderPlanet *Planet
	for _, star := range game.Galaxy.Stars {
		for _, planet := range star.Planets {
			if planet.Owner != attacker.ID { // Find unowned planet or make it defender's
				defenderPlanet = planet
				defenderPlanet.Owner = defender.ID
				defenderPlanet.Marines = 5 // Add some defenders
				break
			}
		}
		if defenderPlanet != nil {
			break
		}
	}
	require.NotNil(t, defenderPlanet, "Should find a planet to use for invasion test")

	// Create attacking fleet with transport ships at same star
	attackingFleet := &Fleet{
		ID:                999,
		Owner:             attacker.ID,
		StarID:            defenderPlanet.ParentStar,
		DestinationStarID: 0, // At destination
		TurnsRemaining:    0, // At destination
		Ships: []FleetShip{
			{Name: "Transport 1", Type: "transport"},
			{Name: "Transport 2", Type: "transport"}, // 2 transports = 8 infantry
		},
	}

	game.Fleets = append(game.Fleets, attackingFleet)
	game.PendingInvasions = []PendingInvasion{{
		OwnerID:        attacker.ID,
		FleetID:        attackingFleet.ID,
		TargetPlanetID: defenderPlanet.ID,
	}}

	originalMarines := defenderPlanet.Marines

	// Run invasion resolution
	reports := game.resolveGroundInvasions()
	assert.Empty(t, game.PendingInvasions, "pending invasions clear after resolution")

	// Should generate at least one invasion report
	assert.Greater(t, len(reports), 0, "Should generate invasion reports")

	// Find the report for our specific planet
	var invasionReport *InvasionReport
	for _, report := range reports {
		if report.PlanetID == defenderPlanet.ID {
			invasionReport = report
			break
		}
	}
	require.NotNil(t, invasionReport, "Should find invasion report for test planet")

	// Verify the invasion report details
	assert.Equal(t, attacker.ID, invasionReport.AttackerOwnerID)
	assert.Equal(t, defender.ID, invasionReport.DefenderOwnerID)

	// Battle should have occurred (at least attempt)
	assert.GreaterOrEqual(t, len(invasionReport.Rounds), 0)

	// If battle occurred, verify initial forces make sense
	if len(invasionReport.Rounds) > 0 {
		assert.Equal(t, 8, len(invasionReport.InitialAttacker.Marines), "Should have 8 marines from 2 transports (4 each)")
		assert.Equal(t, int(originalMarines), len(invasionReport.InitialDefender.Marines), "Should match planet defenders")
	}

	// Invasion system ran without errors - that's the main test
	t.Logf("Invasion system successfully processed %d invasions", len(reports))
}

func TestTransportsDoNotAutoCaptureWithoutInvadeOrder(t *testing.T) {
	game, attacker, defenderPlanet := invasionHarness(t)
	fleet := &Fleet{
		ID:     42,
		Owner:  attacker.ID,
		StarID: defenderPlanet.ParentStar,
		Ships: []FleetShip{
			{Name: "Transport 1", Type: "transport"},
			{Name: "Transport 2", Type: "transport"},
		},
	}
	game.Fleets = append(game.Fleets, fleet)
	ownerBefore := defenderPlanet.Owner
	popBefore := defenderPlanet.Population
	marinesBefore := defenderPlanet.Marines

	reports := game.resolveGroundInvasions()

	assert.Empty(t, reports)
	assert.Equal(t, ownerBefore, defenderPlanet.Owner)
	assert.Equal(t, popBefore, defenderPlanet.Population)
	assert.Equal(t, marinesBefore, defenderPlanet.Marines)
	assert.Len(t, fleet.Ships, 2)
}

func TestInvadeOrderResolvesGroundCombat(t *testing.T) {
	SetRandomSeed(1)
	game, attacker, defenderPlanet := invasionHarness(t)
	defenderPlanet.Marines = 0 // undefended so capture is deterministic
	defenderPlanet.Armored = 0
	fleet := &Fleet{
		ID:     42,
		Owner:  attacker.ID,
		StarID: defenderPlanet.ParentStar,
		Ships:  []FleetShip{{Name: "Transport 1", Type: "transport"}},
	}
	game.Fleets = append(game.Fleets, fleet)
	indexes := []uint{0}
	require.NoError(t, game.issueInvadeCommand(attacker.ID, &InvadeCommand{
		FleetID:        fleet.ID,
		TargetPlanetID: defenderPlanet.ID,
		ShipIndexes:    &indexes,
	}))
	require.Len(t, game.PendingInvasions, 1)

	reports := game.resolveGroundInvasions()
	require.Len(t, reports, 1)
	assert.True(t, reports[0].Success)
	assert.True(t, reports[0].PlanetCaptured)
	assert.Equal(t, attacker.ID, defenderPlanet.Owner)
	assert.Empty(t, game.PendingInvasions)
	require.NotEmpty(t, game.InvasionReports)
	assert.Equal(t, defenderPlanet.ID, game.InvasionReports[len(game.InvasionReports)-1].PlanetID)
}

func TestInvadeUncolonizedPlanetRejectedAtSubmit(t *testing.T) {
	game := newTestGame(t)
	attacker := game.Players[0]
	var target *Planet
	for _, star := range game.Galaxy.Stars {
		for _, planet := range star.Planets {
			if planet.Owner == 0 {
				target = planet
				break
			}
		}
		if target != nil {
			break
		}
	}
	require.NotNil(t, target)
	fleet := &Fleet{
		ID:     7,
		Owner:  attacker.ID,
		StarID: target.ParentStar,
		Ships:  []FleetShip{{Name: "Transport", Type: "transport"}},
	}
	game.Fleets = append(game.Fleets, fleet)
	ownerBefore := target.Owner

	err := game.Turn(attacker.ID, Commands{
		Invade: []*InvadeCommand{{
			FleetID:        fleet.ID,
			TargetPlanetID: target.ID,
		}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "uncolonized")
	assert.Empty(t, game.PendingInvasions)
	assert.Equal(t, ownerBefore, target.Owner)
	assert.Empty(t, game.InvasionReports)
}

func TestInvadeShipIndexesLimitParticipatingTransports(t *testing.T) {
	SetRandomSeed(2)
	game, attacker, defenderPlanet := invasionHarness(t)
	defenderPlanet.Marines = 0
	defenderPlanet.Armored = 0
	fleet := &Fleet{
		ID:     42,
		Owner:  attacker.ID,
		StarID: defenderPlanet.ParentStar,
		Ships: []FleetShip{
			{Name: "Transport A", Type: "transport"},
			{Name: "Gunboat", Type: "ship_design"},
			{Name: "Transport B", Type: "transport"},
		},
	}
	game.Fleets = append(game.Fleets, fleet)
	indexes := []uint{0} // only first transport
	game.PendingInvasions = []PendingInvasion{{
		OwnerID:        attacker.ID,
		FleetID:        fleet.ID,
		TargetPlanetID: defenderPlanet.ID,
		ShipIndexes:    &indexes,
	}}

	reports := game.resolveGroundInvasions()
	require.Len(t, reports, 1)
	assert.Equal(t, 4, len(reports[0].InitialAttacker.Marines), "only one transport (4 marines) should fight")
	require.Len(t, fleet.Ships, 2)
	assert.Equal(t, "Gunboat", fleet.Ships[0].Name)
	assert.Equal(t, "Transport B", fleet.Ships[1].Name)
}

func TestPendingInvasionSaveLoadRoundTrip(t *testing.T) {
	indexes := []uint{0, 2}
	original := []PendingInvasion{{
		OwnerID:        1,
		FleetID:        9,
		TargetPlanetID: 3,
		ShipIndexes:    &indexes,
	}}
	g := &Game{PendingInvasions: original}
	data, err := json.Marshal(g)
	require.NoError(t, err)

	var loaded Game
	require.NoError(t, json.Unmarshal(data, &loaded))
	require.Len(t, loaded.PendingInvasions, 1, "raw JSON may still contain pending_invasions")
	assert.Equal(t, original[0].FleetID, loaded.PendingInvasions[0].FleetID)

	resources, err := NewResourceManager()
	require.NoError(t, err)
	HydrateLoadedGame(&loaded, resources)
	assert.Empty(t, loaded.PendingInvasions, "hydrate must clear pending invasions from saves")

	// Old saves omit the field.
	var legacy Game
	require.NoError(t, json.Unmarshal([]byte(`{"id":"x"}`), &legacy))
	HydrateLoadedGame(&legacy, resources)
	assert.Empty(t, legacy.PendingInvasions)
}

func TestStaleInvadeOrderDroppedWithoutPanic(t *testing.T) {
	game, attacker, defenderPlanet := invasionHarness(t)
	ownerBefore := defenderPlanet.Owner
	game.PendingInvasions = []PendingInvasion{
		{OwnerID: attacker.ID, FleetID: 1, TargetPlanetID: defenderPlanet.ID}, // fleet not at star
	}
	game.Fleets = append(game.Fleets, &Fleet{
		ID:     1,
		Owner:  attacker.ID,
		StarID: defenderPlanet.ParentStar + 100, // elsewhere
		Ships:  []FleetShip{{Type: "transport", Name: "T"}},
	})

	require.NotPanics(t, func() {
		reports := game.resolveGroundInvasions()
		assert.Empty(t, reports, "fleet elsewhere is stale, not an orbit-loss cancel")
	})
	assert.Empty(t, game.PendingInvasions)
	assert.Equal(t, ownerBefore, defenderPlanet.Owner)

	// Missing fleet (destroyed) produces a cancelled report, not a panic.
	game.PendingInvasions = []PendingInvasion{
		{OwnerID: attacker.ID, FleetID: 9999, TargetPlanetID: defenderPlanet.ID},
	}
	require.NotPanics(t, func() {
		reports := game.resolveGroundInvasions()
		require.Len(t, reports, 1)
		assert.True(t, reports[0].Cancelled)
	})
	assert.Equal(t, ownerBefore, defenderPlanet.Owner)
}

func invasionHarness(t *testing.T) (*Game, *Player, *Planet) {
	t.Helper()
	game := newTestGame(t)
	attacker := game.Players[0]
	defender := &Player{ID: 2, Name: "Defender", Race: DefaultRace("Klackon")}
	game.Players = append(game.Players, defender)

	var defenderPlanet *Planet
	for _, star := range game.Galaxy.Stars {
		for _, planet := range star.Planets {
			if planet.Owner != attacker.ID {
				defenderPlanet = planet
				defenderPlanet.Owner = defender.ID
				defenderPlanet.Population = 2000
				defenderPlanet.Marines = 2
				break
			}
		}
		if defenderPlanet != nil {
			break
		}
	}
	require.NotNil(t, defenderPlanet)
	return game, attacker, defenderPlanet
}

func TestInvadeRejectsDuplicateShipIndexes(t *testing.T) {
	game, attacker, defenderPlanet := invasionHarness(t)
	fleet := &Fleet{
		ID:     42,
		Owner:  attacker.ID,
		StarID: defenderPlanet.ParentStar,
		Ships:  []FleetShip{{Name: "Transport", Type: "transport"}},
	}
	game.Fleets = append(game.Fleets, fleet)
	dupes := []uint{0, 0, 0}

	err := game.Turn(attacker.ID, Commands{
		Invade: []*InvadeCommand{{
			FleetID:        fleet.ID,
			TargetPlanetID: defenderPlanet.ID,
			ShipIndexes:    &dupes,
		}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate ship index")
	assert.Empty(t, game.PendingInvasions)
	assert.Equal(t, uint(2), defenderPlanet.Owner)
}

func TestGatherMarinesDedupesShipIndexes(t *testing.T) {
	game := newTestGame(t)
	fleet := &Fleet{
		Owner: 1,
		Ships: []FleetShip{{Name: "Transport", Type: "transport"}},
	}
	dupes := []uint{0, 0, 0}
	force := game.gatherMarinesFromInvasionOrder(fleet, &dupes)
	assert.Equal(t, 4, len(force.Marines), "duplicate indexes must not multiply marines")
}

func TestInvadeRejectsMultipleOrdersForSameFleet(t *testing.T) {
	game, attacker, defenderPlanet := invasionHarness(t)
	// Second enemy planet at the same star
	var other *Planet
	for _, planet := range game.Galaxy.Stars[0].Planets {
		if planet.ID != defenderPlanet.ID && planet.ParentStar == defenderPlanet.ParentStar {
			other = planet
			other.Owner = 2
			break
		}
	}
	if other == nil {
		// ensure a second planet on the same star
		star, err := game.getStar(defenderPlanet.ParentStar)
		require.NoError(t, err)
		other = &Planet{ID: 9001, Owner: 2, ParentStar: star.ID, Name: "Other"}
		star.Planets = append(star.Planets, other)
	}
	fleet := &Fleet{
		ID:     42,
		Owner:  attacker.ID,
		StarID: defenderPlanet.ParentStar,
		Ships: []FleetShip{
			{Name: "T0", Type: "transport"},
			{Name: "Escort", Type: "ship_design"},
			{Name: "T2", Type: "transport"},
		},
	}
	game.Fleets = append(game.Fleets, fleet)
	idx0 := []uint{0}
	idx2 := []uint{2}

	err := game.Turn(attacker.ID, Commands{
		Invade: []*InvadeCommand{
			{FleetID: fleet.ID, TargetPlanetID: defenderPlanet.ID, ShipIndexes: &idx0},
			{FleetID: fleet.ID, TargetPlanetID: other.ID, ShipIndexes: &idx2},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already has an invade order")
	assert.Empty(t, game.PendingInvasions)
	assert.Equal(t, uint(2), defenderPlanet.Owner)
	assert.Equal(t, uint(2), other.Owner)
}

func TestInvadeRejectsEmptyShipIndexes(t *testing.T) {
	game, attacker, defenderPlanet := invasionHarness(t)
	fleet := &Fleet{
		ID:     42,
		Owner:  attacker.ID,
		StarID: defenderPlanet.ParentStar,
		Ships:  []FleetShip{{Name: "Transport", Type: "transport"}},
	}
	game.Fleets = append(game.Fleets, fleet)
	empty := []uint{}

	err := game.Turn(attacker.ID, Commands{
		Invade: []*InvadeCommand{{
			FleetID:        fleet.ID,
			TargetPlanetID: defenderPlanet.ID,
			ShipIndexes:    &empty,
		}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one transport")
	assert.Empty(t, game.PendingInvasions)
}

func TestHydrateLoadedGameClearsPendingInvasions(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)
	g := &Game{
		PendingInvasions: []PendingInvasion{{OwnerID: 1, FleetID: 9, TargetPlanetID: 3}},
	}
	HydrateLoadedGame(g, resources)
	assert.Empty(t, g.PendingInvasions)
}

func TestFailedAdvanceTurnClearsPartialPendingInvasions(t *testing.T) {
	game, attacker, defenderPlanet := invasionHarness(t)
	fleet := &Fleet{
		ID:     42,
		Owner:  attacker.ID,
		StarID: defenderPlanet.ParentStar,
		Ships:  []FleetShip{{Name: "Transport", Type: "transport"}},
	}
	game.Fleets = append(game.Fleets, fleet)

	err := game.AdvanceTurn(map[uint]Commands{
		attacker.ID: {
			Invade: []*InvadeCommand{{
				FleetID:        fleet.ID,
				TargetPlanetID: defenderPlanet.ID,
			}},
		},
		2: {
			SelectResearch: &SelectResearchCommand{TechName: "Totally Fake Breakthrough"},
		},
	})
	require.Error(t, err)
	assert.Empty(t, game.PendingInvasions, "failed AdvanceTurn must not leave partial invade orders")
	assert.Equal(t, uint(2), defenderPlanet.Owner, "invasion must not resolve after failed turn")
}

func TestFailedSubmitTurnLeavesNoPendingInvasions(t *testing.T) {
	game, attacker, defenderPlanet := invasionHarness(t)
	fleet := &Fleet{
		ID:     42,
		Owner:  attacker.ID,
		StarID: defenderPlanet.ParentStar,
		Ships:  []FleetShip{{Name: "Transport", Type: "transport"}},
	}
	game.Fleets = append(game.Fleets, fleet)
	dupes := []uint{0, 0}

	_, advanced, err := game.SubmitTurn(attacker.ID, Commands{
		Invade: []*InvadeCommand{{
			FleetID:        fleet.ID,
			TargetPlanetID: defenderPlanet.ID,
			ShipIndexes:    &dupes,
		}},
	})
	require.Error(t, err)
	assert.False(t, advanced)
	assert.Empty(t, game.PendingInvasions)
	assert.NotContains(t, game.TurnStatus.ReadyPlayerIDs, attacker.ID)
}

func TestValidInvadeStillResolvesAfterHardening(t *testing.T) {
	SetRandomSeed(3)
	game, attacker, defenderPlanet := invasionHarness(t)
	defenderPlanet.Marines = 0
	defenderPlanet.Armored = 0
	fleet := &Fleet{
		ID:     42,
		Owner:  attacker.ID,
		StarID: defenderPlanet.ParentStar,
		Ships:  []FleetShip{{Name: "Transport", Type: "transport"}},
	}
	game.Fleets = append(game.Fleets, fleet)
	indexes := []uint{0}

	// Two-player harness: advance both submissions so the invade is issued + resolved.
	require.NoError(t, game.AdvanceTurn(map[uint]Commands{
		attacker.ID: {
			Invade: []*InvadeCommand{{
				FleetID:        fleet.ID,
				TargetPlanetID: defenderPlanet.ID,
				ShipIndexes:    &indexes,
			}},
		},
		2: {},
	}))
	assert.Empty(t, game.PendingInvasions)
	assert.Equal(t, attacker.ID, defenderPlanet.Owner)
	require.NotEmpty(t, game.InvasionReports)
}

func TestInvasionReportSerializesEmptyForcesAsArrays(t *testing.T) {
	// On 12524f8 this fails: armored/marines serialize as null when a force
	// has no units of that type (append onto a zero GroundForce left nil).
	SetRandomSeed(11)
	game, attacker, defenderPlanet := invasionHarness(t)
	defenderPlanet.Marines = 0
	defenderPlanet.Armored = 0
	fleet := &Fleet{
		ID:     42,
		Owner:  attacker.ID,
		StarID: defenderPlanet.ParentStar,
		Ships:  []FleetShip{{Name: "Transport", Type: "transport"}},
	}
	game.Fleets = append(game.Fleets, fleet)
	indexes := []uint{0}

	require.NoError(t, game.AdvanceTurn(map[uint]Commands{
		attacker.ID: {
			Invade: []*InvadeCommand{{
				FleetID:        fleet.ID,
				TargetPlanetID: defenderPlanet.ID,
				ShipIndexes:    &indexes,
			}},
		},
		2: {},
	}))
	require.NotEmpty(t, game.InvasionReports)
	report := game.InvasionReports[len(game.InvasionReports)-1]
	require.NotNil(t, report.InitialAttacker.Marines)
	require.NotNil(t, report.InitialAttacker.Armored)
	require.NotNil(t, report.InitialDefender.Marines)
	require.NotNil(t, report.InitialDefender.Armored)
	require.NotNil(t, report.SurvivingUnits.Marines)
	require.NotNil(t, report.SurvivingUnits.Armored)
	assert.Empty(t, report.InitialAttacker.Armored)
	assert.Empty(t, report.InitialDefender.Marines)
	assert.Empty(t, report.InitialDefender.Armored)

	assertInvasionReportForcesJSONArrays(t, game)

	visible, err := game.GetVisibleGameForPlayer(attacker.ID)
	require.NoError(t, err)
	assertInvasionReportForcesJSONArrays(t, visible)
}

func assertInvasionReportForcesJSONArrays(t *testing.T, g *Game) {
	t.Helper()
	raw, err := json.Marshal(g)
	require.NoError(t, err)
	var decoded struct {
		InvasionReports []struct {
			Rounds          json.RawMessage `json:"rounds"`
			InitialAttacker struct {
				Marines json.RawMessage `json:"marines"`
				Armored json.RawMessage `json:"armored"`
			} `json:"initial_attacker"`
			InitialDefender struct {
				Marines json.RawMessage `json:"marines"`
				Armored json.RawMessage `json:"armored"`
			} `json:"initial_defender"`
			SurvivingUnits struct {
				Marines json.RawMessage `json:"marines"`
				Armored json.RawMessage `json:"armored"`
			} `json:"surviving_units"`
		} `json:"invasion_reports"`
	}
	require.NoError(t, json.Unmarshal(raw, &decoded))
	require.NotEmpty(t, decoded.InvasionReports)
	for i, r := range decoded.InvasionReports {
		for _, field := range []struct {
			name string
			raw  json.RawMessage
		}{
			{"rounds", r.Rounds},
			{"initial_attacker.marines", r.InitialAttacker.Marines},
			{"initial_attacker.armored", r.InitialAttacker.Armored},
			{"initial_defender.marines", r.InitialDefender.Marines},
			{"initial_defender.armored", r.InitialDefender.Armored},
			{"surviving_units.marines", r.SurvivingUnits.Marines},
			{"surviving_units.armored", r.SurvivingUnits.Armored},
		} {
			require.NotEqual(t, "null", string(field.raw), "report[%d].%s must be [], not null", i, field.name)
			require.True(t, len(field.raw) > 0 && field.raw[0] == '[', "report[%d].%s = %s", i, field.name, field.raw)
		}
	}
}

func TestCancelledInvasionReportSerializesRoundsAsArray(t *testing.T) {
	// On b31ecc2 this fails: cancelInvasionOrder leaves Rounds nil, so JSON
	// has "rounds":null. Build via the real lost-orbit cancel path.
	SetRandomSeed(7)
	g := newBattleGame(t)
	star := contestedStar(g)
	planet := &Planet{
		ID: 8910, Name: "Fort", Owner: 2, ParentStar: star.ID,
		Population: 1000, Marines: 5, Armored: 0,
	}
	star.Planets = append(star.Planets, planet)

	defender, err := g.getPlayer(2)
	require.NoError(t, err)
	defender.ShipDesigns = append(defender.ShipDesigns, tacticalGunboatDesign(5))
	invaderFleet := addOrbitFleet(g, 1, star, []FleetShip{{Type: "transport", Name: "Landers"}})
	addOrbitFleet(g, 2, star, []FleetShip{{Type: "ship_design", Name: "Gunboat", DesignName: "Gunboat"}})

	g.PendingInvasions = []PendingInvasion{{
		OwnerID: 1, FleetID: invaderFleet.ID, TargetPlanetID: planet.ID,
	}}
	_ = g.resolveStarBattles()
	assert.False(t, g.holdsOrbit(star.ID, 1))

	reports := g.resolveGroundInvasions()
	require.Len(t, reports, 1)
	require.True(t, reports[0].Cancelled)
	require.NotNil(t, reports[0].Rounds)
	assert.Empty(t, reports[0].Rounds)

	assertInvasionReportForcesJSONArrays(t, g)

	visible, err := g.GetVisibleGameForPlayer(1)
	require.NoError(t, err)
	assertInvasionReportForcesJSONArrays(t, visible)
}

// Test helper functions
func newTestGame(t *testing.T) *Game {
	t.Helper()
	resources := testResources(t)
	race := DefaultRace("Human")
	g, err := NewGame(resources, NewGameOptions{
		PlayerName:  "Test Player",
		PlayerColor: DefaultPlayerColor(),
		NumStars:    6,
		GalaxySize:  "small",
		Race:        race,
	})
	require.NoError(t, err)
	return g
}
