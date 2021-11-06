package combat_test

import (
	"fmt"
	"math/rand"
	"ssg/game/combat"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFireOn(t *testing.T) {
	testCases := []struct {
		title             string
		attacker          combat.Ship
		defender          combat.Ship
		beamDamage        int64
		attack            int64
		defense           int64
		armor             int64
		structure         int64
		expectedArmor     int64
		expectedStructure int64
	}{
		{
			"damage armor",
			combat.NewEmptyBattleShip("S1"),
			combat.NewEmptyBattleShip("S2"),
			10, 200, 0, 50, 50,
			40, 50,
		},
		{
			"destroy armor",
			combat.NewEmptyBattleShip("S1"),
			combat.NewEmptyBattleShip("S2"),
			60, 200, 0, 50, 50,
			0, 40,
		},
		{
			"no armor hull resists",
			combat.NewEmptyBattleShip("S1"),
			combat.NewEmptyBattleShip("S2"),
			45, 200, 0, 0, 50,
			0, 5,
		},
		{
			"full health one hit",
			combat.NewEmptyBattleShip("S1"),
			combat.NewEmptyBattleShip("S2"),
			120, 200, 0, 50, 50,
			0, 0,
		},
		{
			"damaged health one hit",
			combat.NewEmptyBattleShip("S1"),
			combat.NewEmptyBattleShip("S2"),
			120, 200, 0, 0, 40,
			0, 0,
		},
		{
			"armor intact hull damaged",
			combat.NewEmptyBattleShip("S1"),
			combat.NewEmptyBattleShip("S2"),
			35, 200, 0, 50, 30,
			15, 30,
		},
		{
			"armor and hull damaged",
			combat.NewEmptyBattleShip("S1"),
			combat.NewEmptyBattleShip("S2"),
			50, 200, 0, 30, 30,
			0, 10,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s", tc.title), func(t *testing.T) {
			rand.Seed(12)
			tc.attacker.BeamAttack = tc.attack
			tc.defender.BeamDefense = tc.defense
			tc.defender.CurrentArmorHP = tc.armor
			tc.defender.CurrentStructureHP = tc.structure
			tc.attacker.Weapons = []combat.Beam{
				{"Test Laser I", 1, tc.beamDamage},
			}

			// before attack
			//log.Println(tc.defender)
			_ = combat.FireOn(&tc.attacker, &tc.defender, 5)
			//log.Printf("total damage: %d\n", damageDone)
			// after attack
			//log.Println(tc.defender)

			assert.Equal(t, tc.expectedArmor, tc.defender.CurrentArmorHP)
			assert.Equal(t, tc.expectedStructure, tc.defender.CurrentStructureHP)
		})
	}
}

func TestFireOn_RandomResults(t *testing.T) {
	// seed=12,
	// 7 hits, 3 misses
	// expected total damage=4*7
	rand.Seed(12)
	expectedDamage := int64(28)
	s1 := combat.NewEmptyShip("S1",
		combat.HullSize{"BS_1", 250, 20, 5, 50, 7, 15, 8},
	)
	s2 := combat.NewEmptyShip("S2",
		combat.HullSize{"BS_1", 250, 20, 5, 50, 7, 15, 8},
	)
	s1.Weapons = []combat.Beam{
		{"Laser I", 1, 4},
		{"Laser I", 1, 4},
		{"Laser I", 1, 4},
		{"Laser I", 1, 4},
		{"Laser I", 1, 4},
		{"Laser I", 1, 4},
		{"Laser I", 1, 4},
		{"Laser I", 1, 4},
		{"Laser I", 1, 4},
		{"Laser I", 1, 4},
	}

	damageDone := combat.FireOn(&s1, &s2, 5)
	assert.Equal(t, expectedDamage, damageDone)
}
