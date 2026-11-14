package game

import (
	"encoding/json"
	"testing"

	. "ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubMissileRoll makes every missile hit roll return roll for this test.
func stubMissileRoll(t *testing.T, roll float64) {
	t.Helper()
	previous := missileRoll
	missileRoll = func() float64 { return roll }
	t.Cleanup(func() { missileRoll = previous })
}

func catalogMissile(t *testing.T, name string) Missile {
	t.Helper()
	missile, ok := testResources(t).missileByName(name)
	require.True(t, ok, name)
	return missile
}

func TestMissileCatalogValues(t *testing.T) {
	r := testResources(t)
	require.Len(t, r.MissileCollection.Missiles, 4)
	require.Len(t, r.ShipPartsCollection.MissileOptions, 4)
	for name, damage := range map[string]int64{
		"Nuclear Missile": 8, "Merculite Missile": 14, "Pulson Missile": 20, "Zeon Missile": 30,
	} {
		m := catalogMissile(t, name)
		assert.Equal(t, damage, m.Damage, name)
		assert.Equal(t, 5, m.Ammo, name)
		assert.Equal(t, int64(12), m.Speed, name)
	}
}

func TestMissileHitChanceIsMoO2EvasionRoll(t *testing.T) {
	assert.InDelta(t, 0.897, missileHitChance(0), 0.001)
	assert.Equal(t, int64(50), missileAttack)
	assert.InDelta(t, 0.5, missileHitChance(50), 1e-9)
}

func TestLaunchMissilesShieldReductionPerHit(t *testing.T) {
	stubMissileRoll(t, 0)
	for _, tc := range []struct {
		missile string
		shield  string
		want    int64
	}{
		{"Nuclear Missile", "None", 8},
		{"Nuclear Missile", "Class III", 5},
		{"Nuclear Missile", "Class X", 0},
		{"Merculite Missile", "Class X", 4},
		{"Zeon Missile", "Class X", 20},
	} {
		racks := newMissileRacks([]Missile{catalogMissile(t, tc.missile)})
		volley := launchMissiles(racks, 0, catalogShield(t, tc.shield).ReduceDamage)
		assert.Equal(t, tc.want, volley.damage, "%s vs %s", tc.missile, tc.shield)
		assert.Equal(t, 1, volley.hits)
		assert.Equal(t, 4, racks[0].AmmoLeft)
	}
}

func TestTacticalMissileHitChanceIgnoresComputerAndDistance(t *testing.T) {
	for _, tc := range []struct {
		roll    float64
		hit     bool
		targetX int
		bonus   int64
	}{
		{0.896, true, 1, 0},
		{0.896, true, 13, 0},
		{0.898, false, 1, 100000},
		{0.898, false, 13, 100000},
		{0.95, false, 1, 0},
	} {
		stubMissileRoll(t, tc.roll)
		g, battle := missileDuel(t, 0, tc.targetX, "Nuclear Missile", 8, 0)
		target := battle.ShipByID(2)
		target.MissileEvasion = 0
		battle.ShipByID(1).BeamAttackBonus = tc.bonus

		_, err := g.ApplyBattleAction(1, 1, BattleAction{Type: "fire", TargetID: 2})
		require.NoError(t, err)
		fire := battle.Events[len(battle.Events)-1]
		require.Equal(t, "fire", fire.Type)
		assert.Equal(t, "missile", fire.WeaponKind)
		if tc.hit {
			assert.Equal(t, 1, fire.Hits)
			assert.Equal(t, int64(42), target.CurrentArmorHP)
		} else {
			assert.Equal(t, 1, fire.Misses, "roll %.3f misses", tc.roll)
			assert.Equal(t, int64(50), target.CurrentArmorHP)
		}
		assert.Equal(t, 4, battle.ShipByID(1).Missiles[0].AmmoLeft, "a miss still spends ammo")
		assert.True(t, battle.ShipByID(1).HasFired)
	}
}

func TestPlayerMissileDesignNeedsResearchAndOneWeaponSlot(t *testing.T) {
	g := newTestGame(t)
	p := g.Players[0]
	def := missileFrigateDef(t, "Striker", "", "Nuclear Missile")

	p.Available.Missiles = nil
	err := g.CreateShipDesignFromDef(p.ID, def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `missile "Nuclear Missile" has not been researched`)

	g.research(p, "Nuclear Missile")
	require.NoError(t, g.CreateShipDesignFromDef(p.ID, def))
	require.Len(t, p.ShipDesigns, 1)
	design := p.ShipDesigns[0]
	require.Equal(t, []Missile{catalogMissile(t, "Nuclear Missile")}, design.Missiles)
	assert.True(t, g.playerCanBuildShipDesign(p, design))

	// The design list returns the rack, and it survives a JSON round trip.
	list, err := g.GetShipDesignsList(p.ID)
	require.NoError(t, err)
	var found *ShipDesign
	for _, d := range list {
		if d.Name == "Striker" {
			found = d
		}
	}
	require.NotNil(t, found)
	raw, err := json.Marshal(found)
	require.NoError(t, err)
	var loaded ShipDesign
	require.NoError(t, json.Unmarshal(raw, &loaded))
	assert.Equal(t, design.Missiles, loaded.Missiles)

	both := missileFrigateDef(t, "Both", "Laser", "Nuclear Missile")
	g.research(p, "Laser Cannon")
	err = g.CreateShipDesignFromDef(p.ID, both)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "only one weapon slot is supported")
	assert.Error(t, g.CreateShipDesign(p.ID, "Two Beams", "Nuclear", "None", "None", "Titanium", "Frigate", "Standard", "Laser", "Laser"))

	// Swapping the missile for a beam goes through the update path.
	require.NoError(t, g.UpdateShipDesignFromDef(p.ID, missileFrigateDef(t, "Striker", "Laser", "")))
	assert.Empty(t, p.ShipDesigns[0].Missiles)
	assert.Len(t, p.ShipDesigns[0].Beams, 1)

	// A design whose missile is no longer available cannot be built.
	p.ShipDesigns[0] = design
	p.Available.Missiles = nil
	assert.False(t, g.playerCanBuildShipDesign(p, design))
}

func TestAutoResolveMissilesStopAfterFiveRounds(t *testing.T) {
	stubMissileRoll(t, 0)
	SetRandomSeed(1)
	launcherDesign := NewShipDesign("Launcher", nil, nil, missileTestHull(),
		Drive{CombatSpeed: 1}, Armor{Multiplier: 1}, Computer{}, FuelCell{}, Shield{})
	launcherDesign.Missiles = []Missile{catalogMissile(t, "Nuclear Missile")}
	launcher := NewShip("Launcher", launcherDesign)
	targetDesign := NewShipDesign("Hulk", nil, nil, missileTestHull(),
		Drive{CombatSpeed: 1}, Armor{Multiplier: 10}, Computer{}, FuelCell{}, Shield{})
	target := NewShip("Hulk", targetDesign)
	launcherShip := &battleShip{ownerID: 1, ship: &launcher, missiles: newMissileRacks(launcher.Missiles)}
	sides := []*battleSide{
		{ownerID: 1, ships: []*battleShip{launcherShip}},
		{ownerID: 2, ships: []*battleShip{{ownerID: 2, ship: &target}}},
	}

	rounds, winner, draw := runBattle(sides)
	assert.Equal(t, maxBattleRounds, rounds)
	assert.Equal(t, uint(0), winner)
	assert.True(t, draw)
	assert.Equal(t, target.MaxArmorHP-5*8, target.CurrentArmorHP, "only 5 missiles hit in 24 rounds")
	assert.Equal(t, 0, launcherShip.missiles[0].AmmoLeft)

	// The design and the next battle's magazine are untouched.
	assert.Equal(t, 5, launcher.Missiles[0].Ammo)
	assert.Equal(t, 5, newMissileRacks(launcher.Missiles)[0].AmmoLeft)
}

func TestAutoResolveMissileFrigateBeatsLaserFrigateWithSureHits(t *testing.T) {
	stubMissileRoll(t, 0)
	laser := [2]string{"Laser", ""}
	nuclear := [2]string{"", "Nuclear Missile"}
	assert.Equal(t, uint(1), autoResolveFrigateDuel(t, 7, nuclear, laser).WinnerOwnerID)
	assert.Equal(t, uint(2), autoResolveFrigateDuel(t, 7, laser, nuclear).WinnerOwnerID)
}

func TestTacticalBattleRefillsAmmoForEachBattle(t *testing.T) {
	stubMissileRoll(t, 0.99) // all misses: nobody dies, the fleets stay put
	g, star := newTacticalGame(t)
	human, err := g.getPlayer(1)
	require.NoError(t, err)
	ai, err := g.getPlayer(2)
	require.NoError(t, err)
	launcher := NewShipDesign("Launcher", nil, nil, missileTestHull(),
		Drive{CombatSpeed: 1}, Armor{Multiplier: 1}, Computer{}, FuelCell{}, Shield{})
	launcher.Missiles = []Missile{catalogMissile(t, "Nuclear Missile")}
	human.ShipDesigns = append(human.ShipDesigns, launcher)
	ai.ShipDesigns = append(ai.ShipDesigns, NewShipDesign("Target", nil, nil, missileTestHull(),
		Drive{CombatSpeed: 1}, Armor{Multiplier: 1}, Computer{}, FuelCell{}, Shield{}))
	humanFleet := addOrbitFleet(g, 1, star, []FleetShip{{Type: "ship_design", Name: "Launcher", DesignName: "Launcher"}})
	aiFleet := addOrbitFleet(g, 2, star, []FleetShip{{Type: "ship_design", Name: "Target", DesignName: "Target"}})

	first := g.createTacticalBattle(star.ID, []*Fleet{humanFleet, aiFleet})
	var shooter *TacticalShip
	for _, ts := range first.Ships {
		if ts.OwnerID == 1 {
			shooter = ts
		}
	}
	require.NotNil(t, shooter)
	for i := 0; i < 5 && first.Status == TacticalBattleActive; i++ {
		for first.CurrentShip() != shooter {
			_, err := g.ApplyBattleAction(first.ID, 1, BattleAction{Type: "done"})
			require.NoError(t, err)
		}
		_, err := g.ApplyBattleAction(first.ID, 1, BattleAction{Type: "fire", TargetID: 2})
		require.NoError(t, err)
		_, err = g.ApplyBattleAction(first.ID, 1, BattleAction{Type: "done"})
		require.NoError(t, err)
	}
	assert.Equal(t, 0, shooter.Missiles[0].AmmoLeft)

	g.removePendingBattle(first.ID)
	second := g.createTacticalBattle(star.ID, []*Fleet{humanFleet, aiFleet})
	for _, ts := range second.Ships {
		if ts.OwnerID == 1 {
			assert.Equal(t, []MissileRack{{Name: "Nuclear Missile", Damage: 8, AmmoLeft: 5, AmmoMax: 5}}, ts.Missiles)
		}
	}
}

func TestLegacyDataWithoutMissileFieldsLoadsUnchanged(t *testing.T) {
	// Designs saved before missiles have no rack.
	var design ShipDesign
	require.NoError(t, json.Unmarshal([]byte(`{"name":"Old","weapons":[{"name":"Laser","minDamage":1,"maxDamage":4}],"hull_size":{"name":"BS_1","base_hp":50},"drive":{"combat_speed":3},"armor":{"multiplier":1}}`), &design))
	assert.Nil(t, design.Missiles)
	ship := NewShip("Old", design)
	ts := newTacticalShip(1, &Fleet{ID: 1, Owner: 1}, 0, FleetShip{Name: "Old"}, &ship)
	raw, err := json.Marshal(ts)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), `"missiles"`)
	assert.NotContains(t, string(raw), `"missile_evasion"`)

	// Beam fire events keep their old shape.
	g, battle := missileDuel(t, 0, 1, "Nuclear Missile", 8, 0)
	shooter := battle.ShipByID(1)
	shooter.Missiles = nil
	shooter.Beams = NewMultipleBasicBeams(1, 4)
	shooter.BeamAttackBonus = 100000
	_, err = g.ApplyBattleAction(1, 1, BattleAction{Type: "fire", TargetID: 2})
	require.NoError(t, err)
	raw, err = json.Marshal(battle.Events[len(battle.Events)-1])
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "weapon_kind")
	assert.Equal(t, int64(46), battle.ShipByID(2).CurrentArmorHP)

	// Saves from before missiles get racks for missile techs already researched.
	sg := newTestGame(t)
	p := sg.Players[0]
	sg.research(p, "Nuclear Missile")
	p.Available.Missiles = nil
	data, err := sg.MarshalSave()
	require.NoError(t, err)
	var loaded Game
	require.NoError(t, json.Unmarshal(data, &loaded))
	resources := testResources(t)
	require.NoError(t, ValidateLoadedGame(&loaded, resources))
	HydrateLoadedGame(&loaded, resources)
	assert.Contains(t, loaded.Players[0].Available.Missiles, "Nuclear Missile")
	assert.Equal(t, p.Available.Beams, loaded.Players[0].Available.Beams)
}
