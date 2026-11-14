package game

// Missile behavior tests that only use APIs and fields that existed before
// missiles (v0.21.0), reading missile state through JSON. They compile on the
// old code and fail there on behavior: missile designs lose their rack,
// missile-only ships cannot fire, and AI/auto-resolve missile ships deal no
// damage.

import (
	"encoding/json"
	"fmt"
	"testing"

	. "ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// missileFrigateDef builds a Titanium frigate definition through JSON, the
// way the editor posts it.
func missileFrigateDef(t *testing.T, name, beam, missile string) ShipDesignsDef {
	t.Helper()
	raw, err := json.Marshal(map[string]string{
		"name": name, "beam": beam, "missile": missile,
		"drive": "Nuclear", "computer": "None", "shield": "None", "armor": "Titanium",
		"hull_size": "Frigate", "fuel_cell": "Standard",
	})
	require.NoError(t, err)
	var def ShipDesignsDef
	require.NoError(t, json.Unmarshal(raw, &def))
	return def
}

func jsonMap(t *testing.T, v any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	out := map[string]any{}
	require.NoError(t, json.Unmarshal(raw, &out))
	return out
}

// shipAmmoLeft reads the first missile rack's ammo_left from a battle ship's
// JSON; -1 means the ship has no missile rack.
func shipAmmoLeft(t *testing.T, ship *TacticalShip) int {
	t.Helper()
	racks, ok := jsonMap(t, ship)["missiles"].([]any)
	if !ok || len(racks) == 0 {
		return -1
	}
	return int(racks[0].(map[string]any)["ammo_left"].(float64))
}

func TestMissileTechsUnlockMissileRacks(t *testing.T) {
	for _, name := range []string{"Nuclear Missile", "Merculite Missile", "Pulson Missile", "Zeon Missile"} {
		t.Run(name, func(t *testing.T) {
			g := newTestGame(t)
			p := g.Players[0]
			before, _ := jsonMap(t, p.Available)["missiles"].([]any)
			assert.NotContains(t, before, name)
			g.research(p, name)
			after, _ := jsonMap(t, p.Available)["missiles"].([]any)
			assert.Contains(t, after, name)
		})
	}
}

func TestMissileDesignTemplateSavesAndRoundTrips(t *testing.T) {
	r := testResources(t)
	var def ShipDesignsDef
	require.NoError(t, json.Unmarshal([]byte(`{"name":"Striker","drive":"Nuclear","computer":"None","shield":"None","armor":"Titanium","hull_size":"Frigate","fuel_cell":"Standard","missile":"Nuclear Missile"}`), &def))
	design, err := SaveableCustomShipDesign(r, def)
	require.NoError(t, err)

	raw, err := json.Marshal(design)
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"missiles":[{"name":"Nuclear Missile","damage":8,"ammo":5`)
	assert.Empty(t, design.Beams)

	var loaded ShipDesign
	require.NoError(t, json.Unmarshal(raw, &loaded))
	again, err := json.Marshal(loaded)
	require.NoError(t, err)
	assert.JSONEq(t, string(raw), string(again))

	parts, err := json.Marshal(ShipDesignPartsDef(loaded))
	require.NoError(t, err)
	assert.Contains(t, string(parts), `"missile":"Nuclear Missile"`)
}

func TestMissileDesignRejectsBeamPlusMissileAndUnknownMissile(t *testing.T) {
	r := testResources(t)
	var both ShipDesignsDef
	require.NoError(t, json.Unmarshal([]byte(`{"name":"Both","drive":"Nuclear","computer":"None","shield":"None","armor":"Titanium","hull_size":"Frigate","fuel_cell":"Standard","beam":"Laser","missile":"Nuclear Missile"}`), &both))
	_, err := SaveableCustomShipDesign(r, both)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "only one weapon slot is supported")

	var unknown ShipDesignsDef
	require.NoError(t, json.Unmarshal([]byte(`{"name":"Dud","drive":"Nuclear","computer":"None","shield":"None","armor":"Titanium","hull_size":"Frigate","fuel_cell":"Standard","missile":"Photon Missile"}`), &unknown))
	_, err = SaveableCustomShipDesign(r, unknown)
	require.Error(t, err)
}

// missileDuel builds a pending two-human battle from JSON: ship 1 carries
// one missile rack and no beams, ship 2 is an unarmed target. Its very low
// missile evasion makes every missile hit, so damage assertions are exact.
func missileDuel(t *testing.T, shooterX, targetX int, missile string, damage int64, shieldReduce int64) (*Game, *TacticalBattle) {
	t.Helper()
	raw := fmt.Sprintf(`{
		"id": 1, "game_turn": 3, "star_id": 1, "status": "active",
		"grid_width": 14, "grid_height": 10, "round": 1,
		"human_owner_ids": [1, 2], "turn_order": [1, 2], "turn_index": 0,
		"ships": [
			{"id": 1, "owner_id": 1, "fleet_id": 1, "name": "Striker", "x": %d, "y": 0,
			 "combat_speed": 9, "initiative": 9, "beam_attack_bonus": 0, "beam_defense": 50,
			 "missiles": [{"name": %q, "damage": %d, "ammo_left": 5, "ammo_max": 5}],
			 "status": {"current_armor_hp": 50, "max_armor_hp": 50, "current_structure_hp": 50, "max_structure_hp": 50}},
			{"id": 2, "owner_id": 2, "fleet_id": 2, "name": "Target", "x": %d, "y": 0,
			 "combat_speed": 1, "initiative": 1, "beam_attack_bonus": 0, "beam_defense": 50,
			 "missile_evasion": -100000, "shield_reduce_damage": %d,
			 "status": {"current_armor_hp": 50, "max_armor_hp": 50, "current_structure_hp": 50, "max_structure_hp": 50}}
		]
	}`, shooterX, missile, damage, targetX, shieldReduce)
	var battle TacticalBattle
	require.NoError(t, json.Unmarshal([]byte(raw), &battle))
	g := &Game{Players: []*Player{{ID: 1, IsHuman: true}, {ID: 2, IsHuman: true}}, PendingBattles: []*TacticalBattle{&battle}}
	return g, &battle
}

func lastFireEvent(t *testing.T, b *TacticalBattle) map[string]any {
	t.Helper()
	for i := len(b.Events) - 1; i >= 0; i-- {
		if b.Events[i].Type == "fire" {
			return jsonMap(t, b.Events[i])
		}
	}
	t.Fatal("no fire event")
	return nil
}

func TestTacticalMissileDamageIgnoresRangeAndTakesShield(t *testing.T) {
	for _, tc := range []struct {
		name         string
		missile      string
		damage       int64
		targetX      int
		shieldReduce int64
		want         int64
	}{
		{"nuclear unshielded distance 1", "Nuclear Missile", 8, 1, 0, 8},
		{"nuclear unshielded distance 13", "Nuclear Missile", 8, 13, 0, 8},
		{"nuclear vs class III", "Nuclear Missile", 8, 1, 3, 5},
		{"nuclear vs class X", "Nuclear Missile", 8, 1, 10, 0},
		{"zeon vs class X", "Zeon Missile", 30, 13, 10, 20},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, battle := missileDuel(t, 0, tc.targetX, tc.missile, tc.damage, tc.shieldReduce)
			_, err := g.ApplyBattleAction(1, 1, BattleAction{Type: "fire", TargetID: 2})
			require.NoError(t, err)
			target := battle.ShipByID(2)
			assert.Equal(t, 50-tc.want, target.CurrentArmorHP)
			assert.Equal(t, int64(50), target.CurrentStructureHP)
			fire := lastFireEvent(t, battle)
			assert.Equal(t, "missile", fire["weapon_kind"])
			assert.Equal(t, tc.missile, fire["weapon_name"])
			assert.Equal(t, 4, shipAmmoLeft(t, battle.ShipByID(1)))
		})
	}
}

// fireAndPassRound fires ship 1 at ship 2 and then ends both activations.
func fireAndPassRound(t *testing.T, g *Game) error {
	t.Helper()
	_, fireErr := g.ApplyBattleAction(1, 1, BattleAction{Type: "fire", TargetID: 2})
	_, err := g.ApplyBattleAction(1, 1, BattleAction{Type: "done"})
	require.NoError(t, err)
	_, err = g.ApplyBattleAction(1, 2, BattleAction{Type: "done"})
	require.NoError(t, err)
	return fireErr
}

func TestTacticalMissileAmmoRunsOutAndSurvivesSaveLoad(t *testing.T) {
	g, battle := missileDuel(t, 0, 13, "Nuclear Missile", 8, 0)
	require.NoError(t, fireAndPassRound(t, g))
	require.NoError(t, fireAndPassRound(t, g))
	assert.Equal(t, 3, shipAmmoLeft(t, battle.ShipByID(1)))

	// Save and reload the pending battle mid-fight: ammo is kept.
	raw, err := json.Marshal(battle)
	require.NoError(t, err)
	var loaded TacticalBattle
	require.NoError(t, json.Unmarshal(raw, &loaded))
	g = &Game{Players: g.Players, PendingBattles: []*TacticalBattle{&loaded}}
	battle = &loaded
	assert.Equal(t, 3, shipAmmoLeft(t, battle.ShipByID(1)))

	for i := 0; i < 3; i++ {
		require.NoError(t, fireAndPassRound(t, g))
	}
	assert.Equal(t, 0, shipAmmoLeft(t, battle.ShipByID(1)))
	assert.Equal(t, int64(100-5*8), battle.ShipByID(2).CurrentArmorHP+battle.ShipByID(2).CurrentStructureHP)

	before, err := json.Marshal(battle)
	require.NoError(t, err)
	_, err = g.ApplyBattleAction(1, 1, BattleAction{Type: "fire", TargetID: 2})
	require.Error(t, err)
	assert.Equal(t, "ship 1 is out of missiles", err.Error())
	after, err := json.Marshal(battle)
	require.NoError(t, err)
	assert.JSONEq(t, string(before), string(after), "an empty magazine changes nothing")
}

func missileTestHull() HullSize {
	return HullSize{Name: "BS_1", SizeRating: 5, Space: 250, Marines: 20, BaseSpeed: 5, BaseHP: 50, ComputerHP: 7, DriveHP: 15, ShieldHP: 8}
}

func TestAIMissileShipFiresFromRangeUntilEmpty(t *testing.T) {
	SetRandomSeed(3)
	g, star := newTacticalGame(t)
	human, err := g.getPlayer(1)
	require.NoError(t, err)
	ai, err := g.getPlayer(2)
	require.NoError(t, err)

	launcher, err := SaveableCustomShipDesign(testResources(t), missileFrigateDef(t, "Launcher", "", "Nuclear Missile"))
	require.NoError(t, err)
	launcher.HullSize = missileTestHull()
	launcher.Drive.CombatSpeed = 1
	ai.ShipDesigns = append(ai.ShipDesigns, launcher)
	human.ShipDesigns = append(human.ShipDesigns, NewShipDesign("Target", nil, nil, missileTestHull(),
		Drive{CombatSpeed: 1}, Armor{Multiplier: 4}, Computer{}, FuelCell{}, Shield{}))
	addOrbitFleet(g, 1, star, []FleetShip{{Type: "ship_design", Name: "Target", DesignName: "Target"}})
	addOrbitFleet(g, 2, star, []FleetShip{{Type: "ship_design", Name: "Launcher", DesignName: "Launcher"}})

	battle := startTacticalBattle(t, g, star)
	var aiShip, target *TacticalShip
	for _, ts := range battle.Ships {
		if ts.OwnerID == 2 {
			aiShip = ts
		} else {
			target = ts
		}
	}
	require.NotNil(t, aiShip)
	require.NotNil(t, target)
	assert.Equal(t, 5, shipAmmoLeft(t, aiShip))

	fires := 0
	seen := 0
	for battle.Round <= 8 && battle.Status == TacticalBattleActive {
		for _, e := range battle.Events[seen:] {
			if e.Type == "fire" && e.ShipID == aiShip.ID {
				fires++
				assert.Greater(t, chebyshev(aiShip.X, aiShip.Y, target.X, target.Y), int64(aiPreferredRange),
					"missile ships fire without closing to beam range")
			}
		}
		seen = len(battle.Events)
		_, err := g.ApplyBattleAction(battle.ID, 1, BattleAction{Type: "done"})
		require.NoError(t, err)
	}
	assert.Equal(t, 5, fires, "the AI fires once per round until its 5 missiles are gone")
	assert.Equal(t, 0, shipAmmoLeft(t, aiShip))
	assert.False(t, target.Destroyed)
}

func autoResolveFrigateDuel(t *testing.T, seed int64, p1Weapon, p2Weapon [2]string) *BattleReport {
	t.Helper()
	SetRandomSeed(seed)
	g := newBattleGame(t)
	star := contestedStar(g)
	r := testResources(t)
	for i, weapon := range [][2]string{p1Weapon, p2Weapon} {
		owner := uint(i + 1)
		p, err := g.getPlayer(owner)
		require.NoError(t, err)
		design, err := SaveableCustomShipDesign(r, missileFrigateDef(t, fmt.Sprintf("Frigate %d", owner), weapon[0], weapon[1]))
		require.NoError(t, err)
		p.ShipDesigns = append(p.ShipDesigns, design)
		addOrbitFleet(g, owner, star, []FleetShip{{Type: "ship_design", Name: design.Name, DesignName: design.Name}})
	}
	reports := g.resolveStarBattles()
	require.Len(t, reports, 1)
	return reports[0]
}

func TestAutoResolveNuclearFrigateBeatsLaserFrigate(t *testing.T) {
	laser := [2]string{"Laser", ""}
	nuclear := [2]string{"", "Nuclear Missile"}
	for seed := int64(1); seed <= 3; seed++ {
		first := autoResolveFrigateDuel(t, seed, nuclear, laser)
		assert.False(t, first.Draw)
		assert.Equal(t, uint(1), first.WinnerOwnerID, "seed %d: missile frigate fires first", seed)

		second := autoResolveFrigateDuel(t, seed, laser, nuclear)
		assert.False(t, second.Draw)
		assert.Equal(t, uint(2), second.WinnerOwnerID, "seed %d: laser frigate fires first", seed)
	}
}
