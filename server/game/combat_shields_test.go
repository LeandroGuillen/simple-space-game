package game

import (
	"encoding/json"
	"testing"

	. "ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sureHitComputer makes the chance to hit round to 1.0, so every beam hits
// regardless of the random seed and damage assertions are exact.
var sureHitComputer = Computer{Name: "Test Sure Hit", BeamAttackBonus: 100000}

func catalogShield(t *testing.T, name string) Shield {
	t.Helper()
	resources, err := NewResourceManager()
	require.NoError(t, err)
	shield, err := ShieldFromName(name, resources.Shields)
	require.NoError(t, err)
	return shield
}

func shieldTestHull() HullSize {
	return HullSize{Name: "BS_1", SizeRating: 5, Space: 250, Marines: 20, BaseSpeed: 5, BaseHP: 50, ComputerHP: 7, DriveHP: 15, ShieldHP: 8}
}

func shieldTestDefender(shield Shield) *Ship {
	design := NewShipDesign("Target", nil, nil, shieldTestHull(),
		Drive{CombatSpeed: 1}, Armor{Multiplier: 1}, Computer{}, FuelCell{}, shield)
	ship := NewShip("Target", design)
	return &ship
}

func shieldTestAttacker(beams []Beam) *Ship {
	design := NewShipDesign("Shooter", beams, nil, shieldTestHull(),
		Drive{CombatSpeed: 9}, Armor{Multiplier: 1}, sureHitComputer, FuelCell{}, Shield{})
	ship := NewShip("Shooter", design)
	return &ship
}

func TestCatalogShieldReduceDamageValues(t *testing.T) {
	for name, want := range map[string]int64{
		"None": 0, "Class I": 1, "Class III": 3, "Class V": 5, "Class VII": 7, "Class X": 10,
	} {
		assert.Equal(t, want, catalogShield(t, name).ReduceDamage, name)
	}
}

func TestFireOnShieldReducesEachHit(t *testing.T) {
	// Three 4-damage beams at point-blank range against Class III (reduce 3):
	// each hit is reduced separately, 1+1+1 = 3, not max(0, 12-3) = 9.
	attacker := shieldTestAttacker(NewMultipleBasicBeams(3, 4))
	defender := shieldTestDefender(catalogShield(t, "Class III"))
	armorBefore := defender.CurrentArmorHP

	dealt := FireOn(attacker, defender, 0, nil, nil)

	assert.Equal(t, int64(3), dealt)
	assert.Equal(t, armorBefore-3, defender.CurrentArmorHP)
	assert.Equal(t, defender.MaxStructureHP, defender.CurrentStructureHP)
}

func TestFireOnShieldAppliesAfterRangeDissipation(t *testing.T) {
	// A 4-damage beam at range 3 dissipates to round(3.6) = 4; Class III
	// leaves 1 per hit. A shield stronger than the hit absorbs it entirely.
	attacker := shieldTestAttacker(NewMultipleBasicBeams(3, 4))
	require.Equal(t, int64(4), GetDamage(attacker.Beams[0], 3))

	defender := shieldTestDefender(catalogShield(t, "Class III"))
	assert.Equal(t, int64(3), FireOn(attacker, defender, 3, nil, nil))

	strong := shieldTestDefender(catalogShield(t, "Class V"))
	armorBefore := strong.CurrentArmorHP
	assert.Equal(t, int64(0), FireOn(attacker, strong, 3, nil, nil))
	assert.Equal(t, armorBefore, strong.CurrentArmorHP)
}

func TestFireOnWithoutShieldIsUnchanged(t *testing.T) {
	// Control: the None shield (and a zero-value shield from old data) keep
	// today's full damage.
	for _, shield := range []Shield{catalogShield(t, "None"), {}} {
		attacker := shieldTestAttacker(NewMultipleBasicBeams(3, 4))
		defender := shieldTestDefender(shield)
		armorBefore := defender.CurrentArmorHP
		assert.Equal(t, int64(12), FireOn(attacker, defender, 0, nil, nil))
		assert.Equal(t, armorBefore-12, defender.CurrentArmorHP)
	}
}

// shieldDuelResult runs an AI-only auto-resolve fight between an armed
// design and an unarmed design using the given shield, both built through
// player ship designs the way the ship editor saves them.
func shieldDuelResult(t *testing.T, shieldName string) *BattleReport {
	t.Helper()
	SetRandomSeed(11)
	g := newBattleGame(t)
	star := contestedStar(g)
	p1, err := g.getPlayer(1)
	require.NoError(t, err)
	p2, err := g.getPlayer(2)
	require.NoError(t, err)

	// 3-damage beams dissipate to round(2.7) = 3 at battleStartRange, which
	// Class III blocks completely.
	p1.ShipDesigns = append(p1.ShipDesigns, NewShipDesign("Shooter", NewMultipleBasicBeams(4, 3), nil,
		shieldTestHull(), Drive{CombatSpeed: 9}, Armor{Multiplier: 1}, sureHitComputer, FuelCell{}, Shield{}))
	p2.ShipDesigns = append(p2.ShipDesigns, NewShipDesign("Target", nil, nil,
		shieldTestHull(), Drive{CombatSpeed: 1}, Armor{Multiplier: 1}, Computer{}, FuelCell{}, catalogShield(t, shieldName)))
	addOrbitFleet(g, 1, star, []FleetShip{{Type: "ship_design", Name: "Shooter", DesignName: "Shooter"}})
	addOrbitFleet(g, 2, star, []FleetShip{{Type: "ship_design", Name: "Target", DesignName: "Target"}})

	reports := g.resolveStarBattles()
	require.Len(t, reports, 1)
	return reports[0]
}

func TestAutoResolveShieldChoiceChangesBattleResult(t *testing.T) {
	unshielded := shieldDuelResult(t, "None")
	assert.False(t, unshielded.Draw)
	assert.Equal(t, uint(1), unshielded.WinnerOwnerID, "unshielded target is destroyed")

	shielded := shieldDuelResult(t, "Class III")
	assert.True(t, shielded.Draw, "Class III blocks every 3-damage hit")
	for _, side := range shielded.Sides {
		assert.Equal(t, 0, side.DestroyedCount)
	}
}

// tacticalShieldVolley starts a human-vs-AI tactical battle where the human
// gunboat fires one volley of three 10-damage beams at an unarmed AI target
// carrying the given shield. It returns the fire event and the expected
// unshielded per-hit damage at the firing distance.
func tacticalShieldVolley(t *testing.T, shieldName string) (fire BattleEvent, perHit int64, target *TacticalShip) {
	t.Helper()
	SetRandomSeed(5)
	g, star := newTacticalGame(t)
	human, err := g.getPlayer(1)
	require.NoError(t, err)
	ai, err := g.getPlayer(2)
	require.NoError(t, err)
	human.ShipDesigns = append(human.ShipDesigns, NewShipDesign("Gunboat", NewMultipleBasicBeams(3, 10), nil,
		shieldTestHull(), Drive{CombatSpeed: 9}, Armor{Multiplier: 1}, sureHitComputer, FuelCell{}, Shield{}))
	ai.ShipDesigns = append(ai.ShipDesigns, NewShipDesign("Target", nil, nil,
		shieldTestHull(), Drive{CombatSpeed: 1}, Armor{Multiplier: 1}, Computer{}, FuelCell{}, catalogShield(t, shieldName)))
	addOrbitFleet(g, 1, star, []FleetShip{{Type: "ship_design", Name: "Gunboat", DesignName: "Gunboat"}})
	addOrbitFleet(g, 2, star, []FleetShip{{Type: "ship_design", Name: "Target", DesignName: "Target"}})

	battle := startTacticalBattle(t, g, star)
	shooter := battle.CurrentShip()
	require.NotNil(t, shooter)
	require.Equal(t, uint(1), shooter.OwnerID)
	for _, ts := range battle.Ships {
		if ts.OwnerID == 2 {
			target = ts
		}
	}
	require.NotNil(t, target)
	armorBefore := target.CurrentArmorHP

	_, err = g.ApplyBattleAction(battle.ID, 1, BattleAction{Type: "fire", TargetID: target.ID})
	require.NoError(t, err)
	for _, e := range battle.Events {
		if e.Type == "fire" && e.ShipID == shooter.ID {
			fire = e
		}
	}
	require.Equal(t, 3, fire.Hits)
	perHit = GetDamage(NewBasicBeam(10), chebyshev(shooter.X, shooter.Y, target.X, target.Y))
	require.Greater(t, perHit, int64(3))
	assert.Equal(t, armorBefore-fire.Damage, target.CurrentArmorHP)
	return fire, perHit, target
}

func TestTacticalFireShieldReducesEachHit(t *testing.T) {
	fire, perHit, _ := tacticalShieldVolley(t, "Class III")
	assert.Equal(t, 3*(perHit-3), fire.Damage)
}

func TestTacticalFireWithoutShieldIsUnchanged(t *testing.T) {
	fire, perHit, _ := tacticalShieldVolley(t, "None")
	assert.Equal(t, 3*perHit, fire.Damage)
}

func TestTacticalShipCarriesShieldForHUD(t *testing.T) {
	_, _, target := tacticalShieldVolley(t, "Class III")
	assert.Equal(t, "Class III", target.ShieldName)
	assert.Equal(t, int64(3), target.ShieldReduceDamage)

	raw, err := json.Marshal(target)
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"shield_name":"Class III"`)
	assert.Contains(t, string(raw), `"shield_reduce_damage":3`)
}

func TestLegacyPendingBattleWithoutShieldFieldsTakesFullDamage(t *testing.T) {
	// A battle saved before shields mattered has no shield fields on its
	// ships. It must still load, and its ships take unreduced damage.
	legacy := `{
		"id": 1, "game_turn": 3, "star_id": 1, "status": "active",
		"grid_width": 14, "grid_height": 10, "round": 1,
		"human_owner_ids": [1], "turn_order": [1, 2], "turn_index": 0,
		"ships": [
			{"id": 1, "owner_id": 1, "fleet_id": 1, "name": "Gunboat", "x": 0, "y": 0,
			 "combat_speed": 9, "initiative": 9, "beam_attack_bonus": 100000, "beam_defense": 0,
			 "beams": [{"name": "Basic", "minDamage": 1, "maxDamage": 4},
			           {"name": "Basic", "minDamage": 1, "maxDamage": 4}],
			 "status": {"current_armor_hp": 50, "max_armor_hp": 50, "current_structure_hp": 50, "max_structure_hp": 50}},
			{"id": 2, "owner_id": 2, "fleet_id": 2, "name": "Target", "x": 1, "y": 0,
			 "combat_speed": 1, "initiative": 1, "beam_attack_bonus": 0, "beam_defense": 0,
			 "status": {"current_armor_hp": 50, "max_armor_hp": 50, "current_structure_hp": 50, "max_structure_hp": 50, "current_shield_hp": 8}}
		]
	}`
	var battle TacticalBattle
	require.NoError(t, json.Unmarshal([]byte(legacy), &battle))
	target := battle.ShipByID(2)
	require.NotNil(t, target)
	assert.Equal(t, int64(8), target.CurrentShieldHP, "current_shield_hp keeps its equipment-HP meaning")

	g := &Game{Players: []*Player{{ID: 1}, {ID: 2}}, PendingBattles: []*TacticalBattle{&battle}}
	_, err := g.ApplyBattleAction(1, 1, BattleAction{Type: "fire", TargetID: 2})
	require.NoError(t, err)
	assert.Equal(t, int64(42), target.CurrentArmorHP)
}

func TestLegacyDesignWithoutReduceDamageHasNoReduction(t *testing.T) {
	// Saved designs from before shields mattered may lack reduce_damage.
	var design ShipDesign
	require.NoError(t, json.Unmarshal([]byte(`{"name": "Old", "shield": {"name": "Class III"}}`), &design))
	attacker := shieldTestAttacker(NewMultipleBasicBeams(2, 4))
	ship := NewShip("Old", design)
	ship.CurrentArmorHP, ship.CurrentStructureHP = 50, 50
	assert.Equal(t, int64(8), FireOn(attacker, &ship, 0, nil, nil))

	var noShield ShipDesign
	require.NoError(t, json.Unmarshal([]byte(`{"name": "Older"}`), &noShield))
	old := NewShip("Older", noShield)
	old.CurrentArmorHP, old.CurrentStructureHP = 50, 50
	assert.Equal(t, int64(8), FireOn(attacker, &old, 0, nil, nil))
}
