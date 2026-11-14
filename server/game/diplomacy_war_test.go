package game

import (
	"encoding/json"
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests only use the JSON contract (declare_war command, at_war on
// contacts, "declared_war" events) so they also compile against code that
// predates war and fail there by behavior.

// meet gives both empires a contact with each other.
func meet(t *testing.T, g *Game, a, b uint) {
	t.Helper()
	playerByID(t, g, a).Contacts = append(playerByID(t, g, a).Contacts, model.Contact{PlayerID: b})
	playerByID(t, g, b).Contacts = append(playerByID(t, g, b).Contacts, model.Contact{PlayerID: a})
}

func declareWarJSON(t *testing.T, targets ...uint) model.Commands {
	t.Helper()
	orders := []map[string]uint{}
	for _, target := range targets {
		orders = append(orders, map[string]uint{"player_id": target})
	}
	data, err := json.Marshal(map[string]any{"declare_war": orders})
	require.NoError(t, err)
	var cmds model.Commands
	require.NoError(t, json.Unmarshal(data, &cmds))
	return cmds
}

// warsJSON returns, for each contact of v (a player or the visible view of
// one), whether the serialized contact says at_war.
func warsJSON(t *testing.T, v any) map[uint]bool {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	var decoded struct {
		Contacts []struct {
			PlayerID uint `json:"player_id"`
			AtWar    bool `json:"at_war"`
		} `json:"contacts"`
	}
	require.NoError(t, json.Unmarshal(data, &decoded))
	out := map[uint]bool{}
	for _, c := range decoded.Contacts {
		out[c.PlayerID] = c.AtWar
	}
	return out
}

func visiblePlayer(t *testing.T, g *Game, viewer, id uint) *model.Player {
	t.Helper()
	visible, err := g.GetVisibleGameForPlayer(viewer)
	require.NoError(t, err)
	for _, p := range visible.Players {
		if p.ID == id {
			return p
		}
	}
	t.Fatalf("player %d not visible to %d", id, viewer)
	return nil
}

type warEvent struct {
	owner uint
	text  string
}

func warEvents(g *Game) []warEvent {
	out := []warEvent{}
	for _, event := range g.TurnEvents {
		if event.Kind == "declared_war" {
			out = append(out, warEvent{event.Owner, event.Text})
		}
	}
	return out
}

// submitAll submits the given orders and empty orders for everyone else,
// resolving the turn.
func submitAll(t *testing.T, g *Game, orders map[uint]model.Commands) {
	t.Helper()
	for _, id := range g.playerIDs() {
		_, _, err := g.SubmitTurn(id, orders[id])
		require.NoError(t, err)
	}
}

func TestDeclareWarPersistsAndIsVisibleToBoth(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)
	g := multiPlayerGame(t, 3, 30)
	meet(t, g, 1, 2)
	meet(t, g, 1, 3)
	turn := g.TurnsPlayed

	submitAll(t, g, map[uint]model.Commands{1: declareWarJSON(t, 2)})
	require.Equal(t, turn+1, g.TurnsPlayed, "declare war resolves with the turn")

	assert.Equal(t, map[uint]bool{2: true, 3: false}, warsJSON(t, playerByID(t, g, 1)))
	assert.Equal(t, map[uint]bool{1: true}, warsJSON(t, playerByID(t, g, 2)))
	assert.Equal(t, map[uint]bool{1: false}, warsJSON(t, playerByID(t, g, 3)), "a third empire is not involved")
	assert.Equal(t, map[uint]bool{2: true, 3: false}, warsJSON(t, visiblePlayer(t, g, 1, 1)))
	assert.Equal(t, map[uint]bool{1: true}, warsJSON(t, visiblePlayer(t, g, 2, 2)))
	assert.Empty(t, warsJSON(t, visiblePlayer(t, g, 3, 1)), "rival contacts stay private")

	assert.ElementsMatch(t, []warEvent{
		{1, "You declared war on " + playerByID(t, g, 2).Name},
		{2, playerByID(t, g, 1).Name + " declared war on you"},
	}, warEvents(g))
	for _, event := range g.TurnEvents {
		if event.Kind == "declared_war" {
			assert.Zero(t, event.StarID)
		}
	}

	// The war survives a save round trip and later turns emit no new events.
	data, err := g.MarshalSave()
	require.NoError(t, err)
	var loaded Game
	require.NoError(t, json.Unmarshal(data, &loaded))
	HydrateLoadedGame(&loaded, resources)
	require.NoError(t, ValidateLoadedGame(&loaded, resources))
	assert.Equal(t, map[uint]bool{2: true, 3: false}, warsJSON(t, loaded.Players[0]))
	assert.Equal(t, map[uint]bool{1: true}, warsJSON(t, loaded.Players[1]))
	require.NoError(t, loaded.AdvanceTurn(emptySubsFor(&loaded)))
	assert.Empty(t, warEvents(&loaded))
	assert.True(t, warsJSON(t, loaded.Players[0])[2])
}

func TestDeclareWarAgainIsRejected(t *testing.T) {
	g := multiPlayerGame(t, 2, 30)
	meet(t, g, 1, 2)
	submitAll(t, g, map[uint]model.Commands{1: declareWarJSON(t, 2)})
	require.True(t, warsJSON(t, playerByID(t, g, 1))[2])

	for _, declarer := range []uint{1, 2} {
		target := 3 - declarer
		_, _, err := g.SubmitTurn(declarer, declareWarJSON(t, target))
		require.ErrorContains(t, err, "already at war")
		assert.NotContains(t, g.TurnSubmissions, declarer)
	}
	_, _, err := g.SubmitTurn(1, declareWarJSON(t, 2, 2))
	require.Error(t, err, "a duplicate target in one turn is rejected")
}

func TestMutualDeclarationsSameTurnAreSimplyWar(t *testing.T) {
	g := multiPlayerGame(t, 2, 30)
	meet(t, g, 1, 2)
	submitAll(t, g, map[uint]model.Commands{1: declareWarJSON(t, 2), 2: declareWarJSON(t, 1)})

	assert.Equal(t, map[uint]bool{2: true}, warsJSON(t, playerByID(t, g, 1)))
	assert.Equal(t, map[uint]bool{1: true}, warsJSON(t, playerByID(t, g, 2)))
	events := warEvents(g)
	require.Len(t, events, 2)
	assert.ElementsMatch(t, []uint{1, 2}, []uint{events[0].owner, events[1].owner})
	for _, event := range events {
		assert.Contains(t, event.text, "both empires declared war")
	}
}

func TestDeclareWarRejectsUncontactedEliminatedOrUnknownEmpires(t *testing.T) {
	cases := map[string]func(g *Game) uint{
		"uncontacted": func(g *Game) uint { return 3 },
		"self":        func(g *Game) uint { return 1 },
		"unknown":     func(g *Game) uint { return 42 },
		"eliminated": func(g *Game) uint {
			g.Players[1].Eliminated = true
			return 2
		},
		"one-sided contact": func(g *Game) uint {
			g.Players[2].Contacts = nil
			return 3
		},
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			g := multiPlayerGame(t, 3, 30)
			meet(t, g, 1, 2)
			if name == "one-sided contact" {
				meet(t, g, 1, 3)
			}
			target := setup(g)
			_, _, err := g.SubmitTurn(1, declareWarJSON(t, target))
			require.Error(t, err)
			assert.NotContains(t, g.TurnSubmissions, uint(1))
			for _, p := range g.Players {
				for _, atWar := range warsJSON(t, p) {
					assert.False(t, atWar)
				}
			}
		})
	}
}

// saveWithWar returns save JSON for a 3-player game where 1 and 2 have met and
// are at war (built through the JSON contract).
func saveWithWar(t *testing.T) []byte {
	t.Helper()
	g := multiPlayerGame(t, 3, 30)
	meet(t, g, 1, 2)
	data, err := g.MarshalSave()
	require.NoError(t, err)
	var raw map[string]any
	require.NoError(t, json.Unmarshal(data, &raw))
	for _, p := range raw["players"].([]any) {
		contacts, _ := p.(map[string]any)["contacts"].([]any)
		for _, c := range contacts {
			c.(map[string]any)["at_war"] = true
		}
	}
	data, err = json.Marshal(raw)
	require.NoError(t, err)
	return data
}

func loadSave(t *testing.T, data []byte) (*Game, error) {
	t.Helper()
	resources, err := NewResourceManager()
	require.NoError(t, err)
	var loaded Game
	require.NoError(t, json.Unmarshal(data, &loaded))
	HydrateLoadedGame(&loaded, resources)
	return &loaded, ValidateLoadedGame(&loaded, resources)
}

func TestEliminatingAnEmpireEndsItsWars(t *testing.T) {
	g, err := loadSave(t, saveWithWar(t))
	require.NoError(t, err)
	require.True(t, warsJSON(t, playerByID(t, g, 1))[2], "the saved war loads")

	// Player 2 loses every colony and colony ship and is eliminated.
	for _, star := range g.Galaxy.Stars {
		for _, planet := range star.Planets {
			if planet.Owner == 2 {
				planet.Owner = 0
				planet.Population = 0
			}
		}
	}
	kept := g.Fleets[:0]
	for _, f := range g.Fleets {
		if f.Owner != 2 {
			kept = append(kept, f)
		}
	}
	g.Fleets = kept
	require.NoError(t, g.AdvanceTurn(emptySubsFor(g)))
	require.True(t, playerByID(t, g, 2).Eliminated)

	assert.Equal(t, map[uint]bool{2: false}, warsJSON(t, playerByID(t, g, 1)), "the contact stays, the war ends")
	assert.Equal(t, map[uint]bool{1: false}, warsJSON(t, playerByID(t, g, 2)))
}

func TestLoadingWarWithEliminatedEmpireEndsIt(t *testing.T) {
	data := saveWithWar(t)
	var raw map[string]any
	require.NoError(t, json.Unmarshal(data, &raw))
	raw["players"].([]any)[1].(map[string]any)["eliminated"] = true
	data, err := json.Marshal(raw)
	require.NoError(t, err)

	g, err := loadSave(t, data)
	require.NoError(t, err)
	assert.Equal(t, map[uint]bool{2: false}, warsJSON(t, playerByID(t, g, 1)))
}

func TestLoadRejectsOneSidedWar(t *testing.T) {
	data := saveWithWar(t)
	var raw map[string]any
	require.NoError(t, json.Unmarshal(data, &raw))
	contact := raw["players"].([]any)[1].(map[string]any)["contacts"].([]any)[0].(map[string]any)
	delete(contact, "at_war")
	data, err := json.Marshal(raw)
	require.NoError(t, err)

	_, err = loadSave(t, data)
	require.ErrorContains(t, err, "at war")
}

func TestOldSaveContactsHaveNoWars(t *testing.T) {
	// A slice-1 save: contacts without any war field.
	g := multiPlayerGame(t, 2, 30)
	meet(t, g, 1, 2)
	data, err := g.MarshalSave()
	require.NoError(t, err)
	require.NotContains(t, string(data), "at_war")

	loaded, err := loadSave(t, data)
	require.NoError(t, err)
	assert.Equal(t, map[uint]bool{2: false}, warsJSON(t, loaded.Players[0]))
	assert.Equal(t, map[uint]bool{1: false}, warsJSON(t, loaded.Players[1]))

	// Being at peace, the loaded empires can go to war.
	submitAll(t, loaded, map[uint]model.Commands{2: declareWarJSON(t, 1)})
	assert.Equal(t, map[uint]bool{2: true}, warsJSON(t, loaded.Players[0]))
	assert.Len(t, warEvents(loaded), 2)
}
