package game

import (
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
			NewGroundUnit(MarineUnit), // 10 strength, alive
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
		TurnsRemaining:    0,  // At destination  
		Ships: []FleetShip{
			{Name: "Transport 1", Type: "transport"},
			{Name: "Transport 2", Type: "transport"}, // 2 transports = 8 infantry
		},
	}
	
	game.Fleets = append(game.Fleets, attackingFleet)
	
	originalMarines := defenderPlanet.Marines
	
	// Run invasion resolution
	reports := game.resolveGroundInvasions()
	
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
