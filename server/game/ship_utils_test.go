package game_test

import (
	"fmt"
	. "ssg/game"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFireOn_Damage(t *testing.T) {
	testCases := []struct {
		title             string
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
			10, 200, 0, 50, 50,
			40, 50,
		},
		{
			"destroy armor",
			60, 200, 0, 50, 50,
			0, 40,
		},
		{
			"no armor hull resists",
			45, 200, 0, 0, 50,
			0, 5,
		},
		{
			"full health one hit",
			120, 200, 0, 50, 50,
			0, 0,
		},
		{
			"damaged health one hit",
			120, 200, 0, 0, 40,
			0, 0,
		},
		{
			"armor intact hull damaged",
			35, 200, 0, 50, 30,
			15, 30,
		},
		{
			"armor and hull damaged",
			50, 200, 0, 30, 30,
			0, 10,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s", tc.title), func(t *testing.T) {
			SetRandomSeed(12)
			s1 := NewEmptyBattleShip("S1")
			s2 := NewEmptyBattleShip("S2")
			s1.Computer.BeamAttackBonus = tc.attack
			s2.BeamDefense = tc.defense
			s2.CurrentArmorHP = tc.armor
			s2.CurrentStructureHP = tc.structure
			EquipBeam(&s1, NewBasicBeam(tc.beamDamage))
			_ = FireOn(&s1, &s2, 0)
			assert.Equal(t, tc.expectedArmor, s2.CurrentArmorHP)
			assert.Equal(t, tc.expectedStructure, s2.CurrentStructureHP)
		})
	}
}

func TestFireOn_MultipleWeapons(t *testing.T) {
	testCases := []struct {
		title          string
		beamDamage     int64
		numBeamWeapons int
		attack         int64
		defense        int64
		expectedDamage int64
	}{
		{
			"30 attack 120 defense ~1%",
			1, 100,
			30, 120,
			1,
		},
		{
			"30 attack 60 defense ~25%",
			1, 100,
			30, 60,
			24,
		},
		{
			"same attack and defense ~50%",
			1, 100,
			30, 30,
			49,
		},
		{
			"60 attack 30 defense ~70%",
			1, 100,
			60, 30,
			72,
		},
		{
			"120 attack 30 defense ~99%",
			1, 100,
			120, 30,
			98,
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s", tc.title), func(t *testing.T) {
			SetRandomSeed(12)
			s1 := NewEmptyBattleShip("S1")
			s2 := NewEmptyBattleShip("S2")
			s1.Computer.BeamAttackBonus = tc.attack
			s2.BeamDefense = tc.defense
			EquipBeams(&s1, NewMultipleBasicBeams(tc.numBeamWeapons, tc.beamDamage))
			damageDone := FireOn(&s1, &s2, 0)
			assert.Equal(t, tc.expectedDamage, damageDone)
		})
	}
}

func TestFireOn_Distance(t *testing.T) {
	testCases := []struct {
		title          string
		distance       int64
		damage         int64
		expectedDamage int64
	}{
		// normal cases
		{"distance N=", 0, 100, 100},
		{"distance N=", 2, 100, 100},
		{"distance N=", 5, 100, 90},
		{"distance N=", 9, 100, 70},
		{"distance N=", 13, 100, 60},
		{"distance N=", 17, 100, 50},
		{"distance N=", 23, 100, 35},
		// extreme cases
		{"distance N=", -1, 100, 100},
		{"distance N=", 1000, 100, 0},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s%d", tc.title, tc.distance), func(t *testing.T) {
			SetRandomSeed(1)
			s1 := NewEmptyBattleShip("S1")
			s2 := NewEmptyBattleShip("S2")
			s1.Computer.BeamAttackBonus = 1000
			s2.BeamDefense = 0
			EquipBeam(&s1, NewBasicBeam(tc.damage))
			damageDone := FireOn(&s1, &s2, tc.distance)
			assert.Equal(t, tc.expectedDamage, damageDone)
		})
	}
}
