package game

import (
	"testing"

	. "ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newBattleGame(t *testing.T) *Game {
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
	return g
}

func contestedStar(g *Game) *Star {
	for _, star := range g.Galaxy.Stars {
		busy := false
		for _, planet := range star.Planets {
			if planet.Owner != 0 {
				busy = true
				break
			}
		}
		if !busy {
			return star
		}
	}
	return g.Galaxy.Stars[len(g.Galaxy.Stars)-1]
}

func addOrbitFleet(g *Game, ownerID uint, star *Star, ships []FleetShip) *Fleet {
	fleet, err := g.newFleetAtStar(ownerID, star.ID, ships)
	if err != nil {
		panic(err)
	}
	return fleet
}

func TestResolveStarBattlesIgnoresSingleOwnerStars(t *testing.T) {
	g := newBattleGame(t)
	star := contestedStar(g)
	addOrbitFleet(g, 1, star, []FleetShip{{Type: "colony_ship", Name: "CS"}})
	reports := g.resolveStarBattles()
	assert.Empty(t, reports)
	assert.Len(t, g.Fleets, 1)
}

func TestResolveStarBattlesDetectsMultiOwnerEncounter(t *testing.T) {
	g := newBattleGame(t)
	star := contestedStar(g)
	addOrbitFleet(g, 1, star, []FleetShip{{Type: "colony_ship", Name: "CS1"}})
	addOrbitFleet(g, 2, star, []FleetShip{{Type: "colony_ship", Name: "CS2"}})

	reports := g.resolveStarBattles()
	require.Len(t, reports, 1)
	assert.Equal(t, star.ID, reports[0].StarID)
	assert.Len(t, reports[0].Sides, 2)
}

func TestResolveStarBattlesUnarmedFleetsDrawAndKeepShips(t *testing.T) {
	SetRandomSeed(1)
	g := newBattleGame(t)
	star := contestedStar(g)
	addOrbitFleet(g, 1, star, []FleetShip{{Type: "colony_ship", Name: "A"}})
	addOrbitFleet(g, 2, star, []FleetShip{{Type: "colony_ship", Name: "B"}})

	reports := g.resolveStarBattles()
	require.Len(t, reports, 1)
	assert.True(t, reports[0].Draw)
	assert.Equal(t, uint(0), reports[0].WinnerOwnerID)
	assert.Equal(t, maxBattleRounds, reports[0].Rounds)
	assert.Len(t, g.Fleets, 2)
	for _, fleet := range g.Fleets {
		assert.Len(t, fleet.Ships, 1)
	}
}

func TestResolveStarBattlesArmedFleetDestroysCivilians(t *testing.T) {
	SetRandomSeed(7)
	g := newBattleGame(t)
	star := contestedStar(g)

	attacker, err := g.getPlayer(1)
	require.NoError(t, err)
	armedDesign := NewShipDesign(
		"Gunboat",
		NewMultipleBasicBeams(4, 50),
		nil,
		HullSize{Name: "BS_1", SizeRating: 5, Space: 250, Marines: 20, BaseSpeed: 5, BaseHP: 50, ComputerHP: 7, DriveHP: 15, ShieldHP: 8},
		Drive{}, Armor{Multiplier: 1}, Computer{BeamAttackBonus: 80}, FuelCell{}, Shield{},
	)
	attacker.ShipDesigns = append(attacker.ShipDesigns, armedDesign)

	addOrbitFleet(g, 1, star, []FleetShip{{Type: "ship_design", Name: "Gunboat", DesignName: "Gunboat"}})
	addOrbitFleet(g, 2, star, []FleetShip{{Type: "colony_ship", Name: "Civvie"}})

	reports := g.resolveStarBattles()
	require.Len(t, reports, 1)
	report := reports[0]
	assert.False(t, report.Draw)
	assert.Equal(t, uint(1), report.WinnerOwnerID)
	require.Len(t, report.Sides, 2)
	for _, side := range report.Sides {
		if side.OwnerID == 2 {
			assert.Empty(t, side.SurvivingShips)
			assert.Equal(t, 1, side.DestroyedCount)
		} else {
			assert.Len(t, side.SurvivingShips, 1)
		}
	}

	// Loser fleet pruned, winner fleet retained.
	owners := map[uint]int{}
	for _, fleet := range g.Fleets {
		owners[fleet.Owner] += len(fleet.Ships)
	}
	assert.Equal(t, 1, owners[1])
	assert.Equal(t, 0, owners[2])
}

func TestResolveStarBattlesAppendsReportsToGame(t *testing.T) {
	g := newBattleGame(t)
	star := contestedStar(g)
	addOrbitFleet(g, 1, star, []FleetShip{{Type: "colony_ship", Name: "A"}})
	addOrbitFleet(g, 2, star, []FleetShip{{Type: "colony_ship", Name: "B"}})

	g.resolveStarBattles()
	require.Len(t, g.BattleReports, 1)
	assert.Equal(t, uint(1), g.BattleReports[0].ID)

	// IDs are stable and monotonic across subsequent resolutions.
	g.resolveStarBattles()
	require.GreaterOrEqual(t, len(g.BattleReports), 2)
	assert.Greater(t, g.BattleReports[len(g.BattleReports)-1].ID, g.BattleReports[0].ID)
}
