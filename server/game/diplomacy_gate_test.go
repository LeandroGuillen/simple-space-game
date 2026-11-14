package game

import (
	"encoding/json"
	"testing"

	. "ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests set diplomacy state through the JSON contract (peace_treaty and
// at_war on contacts) so they also compile against code that predates the
// war/peace gate and fail there by behavior.

// setContactFieldJSON sets a JSON field on a's contact with b (round-tripping
// the contact through JSON).
func setContactFieldJSON(t *testing.T, g *Game, a, b uint, field string, value any) {
	t.Helper()
	p := playerByID(t, g, a)
	for i := range p.Contacts {
		if p.Contacts[i].PlayerID != b {
			continue
		}
		data, err := json.Marshal(p.Contacts[i])
		require.NoError(t, err)
		var raw map[string]any
		require.NoError(t, json.Unmarshal(data, &raw))
		raw[field] = value
		data, err = json.Marshal(raw)
		require.NoError(t, err)
		var c Contact
		require.NoError(t, json.Unmarshal(data, &c))
		p.Contacts[i] = c
		return
	}
	t.Fatalf("player %d has no contact with %d", a, b)
}

// signPeace records a peace treaty on both sides (what slice 4 will do when
// peace is accepted).
func signPeace(t *testing.T, g *Game, a, b uint) {
	t.Helper()
	setContactFieldJSON(t, g, a, b, "peace_treaty", true)
	setContactFieldJSON(t, g, b, a, "peace_treaty", true)
}

func goToWar(t *testing.T, g *Game, a, b uint) {
	t.Helper()
	setContactFieldJSON(t, g, a, b, "at_war", true)
	setContactFieldJSON(t, g, b, a, "at_war", true)
	setContactFieldJSON(t, g, a, b, "war_since", g.CurrentTurn)
	setContactFieldJSON(t, g, b, a, "war_since", g.CurrentTurn)
}

// contactJSON returns the serialized contact of a with b.
func contactJSON(t *testing.T, g *Game, a, b uint) map[string]any {
	t.Helper()
	data, err := json.Marshal(playerByID(t, g, a))
	require.NoError(t, err)
	var decoded struct {
		Contacts []map[string]any `json:"contacts"`
	}
	require.NoError(t, json.Unmarshal(data, &decoded))
	for _, c := range decoded.Contacts {
		if uint(c["player_id"].(float64)) == b {
			return c
		}
	}
	t.Fatalf("player %d has no contact with %d", a, b)
	return nil
}

// gateGame returns a two-player game with a fleet of each empire sharing
// orbit at a neutral star. Player 1 is human when human is true, so a fight
// becomes a pending tactical battle; otherwise it auto-resolves.
func gateGame(t *testing.T, human bool) (*Game, *Star) {
	t.Helper()
	g, star := newTacticalGame(t)
	g.Players[0].IsHuman = human
	if human {
		armHumanGunboat(t, g, star)
	} else {
		addOrbitFleet(g, 1, star, []FleetShip{{Type: "colony_ship", Name: "A"}})
	}
	addOrbitFleet(g, 2, star, []FleetShip{{Type: "colony_ship", Name: "B"}})
	meet(t, g, 1, 2)
	return g, star
}

var battleModes = map[string]bool{"auto-resolve": false, "pending tactical": true}

func TestPeaceTreatyFleetsShareOrbitWithoutBattle(t *testing.T) {
	for name, human := range battleModes {
		t.Run(name, func(t *testing.T) {
			g, _ := gateGame(t, human)
			signPeace(t, g, 1, 2)

			reports := g.resolveStarBattles()
			assert.Empty(t, reports, "no auto-resolve battle under a peace treaty")
			assert.Empty(t, g.BattleReports)
			assert.Empty(t, g.PendingBattles, "no pending tactical battle under a peace treaty")
			assert.Empty(t, warEvents(g), "a treaty does not auto-declare war")
			assert.Len(t, g.Fleets, 2, "both fleets keep orbit")
			assert.Equal(t, true, contactJSON(t, g, 1, 2)["peace_treaty"])
			assert.Nil(t, contactJSON(t, g, 1, 2)["at_war"])
		})
	}
}

func TestAtWarFleetsFightWithoutNewDeclaration(t *testing.T) {
	for name, human := range battleModes {
		t.Run(name, func(t *testing.T) {
			g, _ := gateGame(t, human)
			goToWar(t, g, 1, 2)

			reports := g.resolveStarBattles()
			if human {
				assert.Empty(t, reports)
				assert.Len(t, g.PendingBattles, 1)
			} else {
				assert.Len(t, reports, 1)
				assert.Empty(t, g.PendingBattles)
			}
			assert.Empty(t, warEvents(g), "already at war: no extra declaration")
		})
	}
}

func TestNoTreatyBattleAutoDeclaresWarThenFights(t *testing.T) {
	for name, human := range battleModes {
		t.Run(name, func(t *testing.T) {
			g, star := gateGame(t, human)

			reports := g.resolveStarBattles()
			if human {
				assert.Empty(t, reports)
				require.Len(t, g.PendingBattles, 1, "the tactical battle still starts")
			} else {
				require.Len(t, reports, 1, "the battle still auto-resolves")
				assert.Empty(t, g.PendingBattles)
			}
			assert.Equal(t, map[uint]bool{2: true}, warsJSON(t, playerByID(t, g, 1)))
			assert.Equal(t, map[uint]bool{1: true}, warsJSON(t, playerByID(t, g, 2)))
			events := warEvents(g)
			require.Len(t, events, 2, "both sides get a declared_war event")
			assert.ElementsMatch(t, []uint{1, 2}, []uint{events[0].owner, events[1].owner})
			for _, event := range g.TurnEvents {
				if event.Kind == "declared_war" {
					assert.Equal(t, star.ID, event.StarID)
					assert.Contains(t, event.Text, star.Name)
				}
			}
		})
	}
}

func TestAutoDeclaredWarIsDatedThisTurn(t *testing.T) {
	// A full turn: first contact, then the fight auto-declares after the
	// stardate advanced. Old saves without any diplomacy field behave the same.
	g, star := newTacticalGame(t)
	g.Players[0].IsHuman = false
	addOrbitFleet(g, 1, star, []FleetShip{{Type: "colony_ship", Name: "A"}})
	addOrbitFleet(g, 2, star, []FleetShip{{Type: "colony_ship", Name: "B"}})
	before := g.CurrentTurn

	require.NoError(t, g.AdvanceTurn(emptySubsFor(g)))
	require.Greater(t, g.CurrentTurn, before)
	require.NotEmpty(t, g.BattleReports)
	for _, pair := range [][2]uint{{1, 2}, {2, 1}} {
		contact := contactJSON(t, g, pair[0], pair[1])
		assert.Equal(t, true, contact["at_war"])
		assert.Equal(t, g.CurrentTurn, contact["war_since"])
	}
	assert.Len(t, warEvents(g), 2)
}

func TestTreatyPartnerSitsOutThreeWayBattle(t *testing.T) {
	g := multiPlayerGame(t, 3, 30)
	for _, p := range g.Players {
		p.IsHuman = false
	}
	star := emptyStarAwayFrom(t, g)
	for owner := uint(1); owner <= 3; owner++ {
		addOrbitFleet(g, owner, star, []FleetShip{{Type: "colony_ship", Name: "Civvie"}})
	}
	meet(t, g, 1, 2)
	meet(t, g, 1, 3)
	meet(t, g, 2, 3)
	signPeace(t, g, 1, 2)

	reports := g.resolveStarBattles()
	require.Len(t, reports, 1)
	sides := []uint{}
	for _, side := range reports[0].Sides {
		sides = append(sides, side.OwnerID)
	}
	assert.Equal(t, []uint{1, 3}, sides, "player 2 has a treaty with player 1 and sits out")
	assert.Equal(t, map[uint]bool{2: false, 3: true}, warsJSON(t, playerByID(t, g, 1)))
	assert.Equal(t, map[uint]bool{1: false, 3: false}, warsJSON(t, playerByID(t, g, 2)))
	assert.Equal(t, true, contactJSON(t, g, 1, 2)["peace_treaty"], "the treaty is kept")
}

// invadeSetup parks an attacker transport over a defender colony with no
// marines, so a landing captures it.
func invadeSetup(t *testing.T) (*Game, *Player, *Planet, *Fleet) {
	t.Helper()
	SetRandomSeed(1)
	g, attacker, planet := invasionHarness(t)
	planet.Marines = 0
	planet.Armored = 0
	fleet := &Fleet{ID: 42, Owner: attacker.ID, StarID: planet.ParentStar, Ships: []FleetShip{{Name: "Transport 1", Type: "transport"}}}
	g.Fleets = append(g.Fleets, fleet)
	meet(t, g, attacker.ID, planet.Owner)
	// A second colony keeps the defender alive after the capture (elimination
	// would end the war).
	for _, star := range g.Galaxy.Stars {
		for _, other := range star.Planets {
			if other.Owner == 0 && star.ID != planet.ParentStar {
				other.Owner = planet.Owner
				other.Population = 2000
				return g, attacker, planet, fleet
			}
		}
	}
	t.Fatal("no spare planet for the defender")
	return nil, nil, nil, nil
}

func invadeCommands(fleet *Fleet, planet *Planet) Commands {
	return Commands{Invade: []*InvadeCommand{{FleetID: fleet.ID, TargetPlanetID: planet.ID}}}
}

// submitInvasionTurn submits the attacker's orders (and empty orders for
// everyone else), resolving the turn, and returns the invasion reports it
// produced.
func submitInvasionTurn(t *testing.T, g *Game, attacker *Player, commands Commands) []*InvasionReport {
	t.Helper()
	before := len(g.InvasionReports)
	submitAll(t, g, map[uint]Commands{attacker.ID: commands})
	return g.InvasionReports[before:]
}

func TestInvadeRejectedUnderPeaceTreaty(t *testing.T) {
	g, attacker, planet, fleet := invadeSetup(t)
	defender := planet.Owner
	signPeace(t, g, attacker.ID, defender)

	err := g.Turn(attacker.ID, invadeCommands(fleet, planet))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "peace treaty")
	assert.Empty(t, g.PendingInvasions)
	assert.Equal(t, defender, planet.Owner)
	assert.Empty(t, warEvents(g))
}

func TestInvadeAcceptedAtWar(t *testing.T) {
	g, attacker, planet, fleet := invadeSetup(t)
	goToWar(t, g, attacker.ID, planet.Owner)

	reports := submitInvasionTurn(t, g, attacker, invadeCommands(fleet, planet))
	require.Len(t, reports, 1)
	assert.True(t, reports[0].PlanetCaptured)
	assert.Empty(t, warEvents(g), "already at war: no extra declaration")
}

func TestInvadeWithoutTreatyAutoDeclaresThenProceeds(t *testing.T) {
	g, attacker, planet, fleet := invadeSetup(t)
	defender := planet.Owner

	reports := submitInvasionTurn(t, g, attacker, invadeCommands(fleet, planet))
	require.Len(t, reports, 1)
	assert.True(t, reports[0].PlanetCaptured, "the invasion proceeds")
	assert.Equal(t, attacker.ID, planet.Owner)
	for _, pair := range [][2]uint{{attacker.ID, defender}, {defender, attacker.ID}} {
		contact := contactJSON(t, g, pair[0], pair[1])
		assert.Equal(t, true, contact["at_war"])
		assert.Equal(t, g.CurrentTurn, contact["war_since"], "war dates from this turn")
	}
	events := warEvents(g)
	require.Len(t, events, 2)
	assert.ElementsMatch(t, []uint{attacker.ID, defender}, []uint{events[0].owner, events[1].owner})
}

func TestTreatyPartnerWarshipsDoNotContestOrbit(t *testing.T) {
	// Player 3 has a treaty with the invader and an armed ship in orbit: it
	// does not fight, so the invasion of player 2 still lands.
	g, attacker, planet, fleet := invadeSetup(t)
	third := &Player{ID: 3, Name: "Third", Race: DefaultRace("Klackon")}
	third.ShipDesigns = append(third.ShipDesigns, tacticalGunboatDesign(5))
	g.Players = append(g.Players, third)
	g.Fleets = append(g.Fleets, &Fleet{ID: 77, Owner: 3, StarID: planet.ParentStar, Ships: []FleetShip{{Type: "ship_design", Name: "Gunboat", DesignName: "Gunboat"}}})
	meet(t, g, attacker.ID, 3)
	signPeace(t, g, attacker.ID, 3)

	reports := submitInvasionTurn(t, g, attacker, invadeCommands(fleet, planet))
	require.Len(t, reports, 1)
	assert.False(t, reports[0].Cancelled)
	assert.True(t, reports[0].PlanetCaptured)
}

func TestDeclaringWarClearsPeaceTreaty(t *testing.T) {
	g := multiPlayerGame(t, 2, 30)
	meet(t, g, 1, 2)
	signPeace(t, g, 1, 2)
	require.Equal(t, true, contactJSON(t, g, 2, 1)["peace_treaty"])

	submitAll(t, g, map[uint]Commands{1: declareWarJSON(t, 2)})
	for _, pair := range [][2]uint{{1, 2}, {2, 1}} {
		contact := contactJSON(t, g, pair[0], pair[1])
		assert.Equal(t, true, contact["at_war"])
		assert.Nil(t, contact["peace_treaty"], "declaring war ends the treaty")
	}
}

func saveWithPeace(t *testing.T) []byte {
	t.Helper()
	g := multiPlayerGame(t, 3, 30)
	meet(t, g, 1, 2)
	signPeace(t, g, 1, 2)
	data, err := g.MarshalSave()
	require.NoError(t, err)
	return data
}

func TestPeaceTreatySurvivesSaveAndStaysPrivate(t *testing.T) {
	g, err := loadSave(t, saveWithPeace(t))
	require.NoError(t, err)
	assert.Equal(t, true, contactJSON(t, g, 1, 2)["peace_treaty"])
	assert.Equal(t, true, contactJSON(t, g, 2, 1)["peace_treaty"])

	data, err := json.Marshal(visiblePlayer(t, g, 1, 1))
	require.NoError(t, err)
	assert.Contains(t, string(data), `"peace_treaty":true`, "own contacts show the treaty")
	data, err = json.Marshal(visiblePlayer(t, g, 2, 1))
	require.NoError(t, err)
	assert.NotContains(t, string(data), "peace_treaty", "rival contacts stay private")

	snap := snapshotGameState(g)
	playerByID(t, g, 1).Contacts = nil
	restoreGameState(g, snap)
	assert.Equal(t, true, contactJSON(t, g, 1, 2)["peace_treaty"], "turn snapshot keeps the treaty")
}

func TestLoadRejectsOneSidedOrContradictoryPeaceTreaty(t *testing.T) {
	edit := func(t *testing.T, mutate func(contact map[string]any)) []byte {
		t.Helper()
		var raw map[string]any
		require.NoError(t, json.Unmarshal(saveWithPeace(t), &raw))
		mutate(raw["players"].([]any)[1].(map[string]any)["contacts"].([]any)[0].(map[string]any))
		data, err := json.Marshal(raw)
		require.NoError(t, err)
		return data
	}
	_, err := loadSave(t, edit(t, func(c map[string]any) { delete(c, "peace_treaty") }))
	require.ErrorContains(t, err, "peace treaty")

	_, err = loadSave(t, edit(t, func(c map[string]any) { c["at_war"] = true }))
	require.ErrorContains(t, err, "peace treaty")
}

func TestDeclareWarInvalidAtResolveIsSkippedNotFatal(t *testing.T) {
	g := multiPlayerGame(t, 3, 30)
	meet(t, g, 1, 2)
	meet(t, g, 1, 3)
	turn := g.TurnsPlayed

	_, _, err := g.SubmitTurn(1, declareWarJSON(t, 2, 3))
	require.NoError(t, err, "both targets are valid at submit")
	// Player 2 is eliminated before the turn resolves.
	playerByID(t, g, 2).Eliminated = true
	_, _, err = g.SubmitTurn(3, Commands{})
	require.NoError(t, err, "a stale declaration must not fail the turn")

	require.Equal(t, turn+1, g.TurnsPlayed, "the turn resolved")
	assert.Equal(t, map[uint]bool{2: false, 3: true}, warsJSON(t, playerByID(t, g, 1)), "the valid declaration still applies")
	assert.Equal(t, map[uint]bool{1: true}, warsJSON(t, playerByID(t, g, 3)))
	failed := []string{}
	for _, event := range g.TurnEvents {
		if event.Kind == "declare_war_failed" {
			assert.Equal(t, uint(1), event.Owner)
			failed = append(failed, event.Text)
		}
	}
	require.Len(t, failed, 1, "the declarer is told the declaration could not be applied")
	assert.Contains(t, failed[0], playerByID(t, g, 2).Name)
	assert.Contains(t, failed[0], "could not be applied")
}
