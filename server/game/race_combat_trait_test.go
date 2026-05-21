package game_test

import (
	. "ssg/game"
	"ssg/model"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCustomRaceShipCombatStats(t *testing.T) {
	resources := testResources(t)
	race, err := BuildCustomRace(resources, CustomRaceDesign{
		Name:     "Hawks",
		TraitIDs: []string{"ship_attack", "ship_defense"},
	})
	require.NoError(t, err)
	assert.Equal(t, int64(20), race.Stats.ShipAttackBonusPercent)
	assert.Equal(t, int64(20), race.Stats.ShipDefenseBonusPercent)
}

func TestFireOnRaceShipAttackBonusIncreasesDamage(t *testing.T) {
	SetRandomSeed(12)
	attacker := NewEmptyBattleShip("A")
	defender := NewEmptyBattleShip("D")
	attacker.Computer.BeamAttackBonus = 30
	defender.BeamDefense = 30
	EquipBeams(&attacker, NewMultipleBasicBeams(100, 1))

	baseline := FireOn(&attacker, &defender, 0, nil, nil)

	SetRandomSeed(12)
	attacker = NewEmptyBattleShip("A")
	defender = NewEmptyBattleShip("D")
	attacker.Computer.BeamAttackBonus = 30
	defender.BeamDefense = 30
	EquipBeams(&attacker, NewMultipleBasicBeams(100, 1))

	resources := testResources(t)
	race, err := BuildCustomRace(resources, CustomRaceDesign{Name: "Hawks", TraitIDs: []string{"ship_attack"}})
	require.NoError(t, err)
	attackerPlayer := &model.Player{Race: race}

	withBonus := FireOn(&attacker, &defender, 0, attackerPlayer, nil)

	assert.Greater(t, withBonus, baseline)
	assert.Equal(t, int64(61), withBonus)
}

func TestFireOnRaceShipDefenseBonusReducesDamage(t *testing.T) {
	SetRandomSeed(12)
	attacker := NewEmptyBattleShip("A")
	defender := NewEmptyBattleShip("D")
	attacker.Computer.BeamAttackBonus = 30
	defender.BeamDefense = 30
	EquipBeams(&attacker, NewMultipleBasicBeams(100, 1))

	baseline := FireOn(&attacker, &defender, 0, nil, nil)

	SetRandomSeed(12)
	attacker = NewEmptyBattleShip("A")
	defender = NewEmptyBattleShip("D")
	attacker.Computer.BeamAttackBonus = 30
	defender.BeamDefense = 30
	EquipBeams(&attacker, NewMultipleBasicBeams(100, 1))

	resources := testResources(t)
	race, err := BuildCustomRace(resources, CustomRaceDesign{Name: "Turtles", TraitIDs: []string{"ship_defense"}})
	require.NoError(t, err)
	defenderPlayer := &model.Player{Race: race}

	withBonus := FireOn(&attacker, &defender, 0, nil, defenderPlayer)

	assert.Less(t, withBonus, baseline)
	assert.Equal(t, int64(47), withBonus)
}

func testResources(t *testing.T) *ResourceManager {
	t.Helper()
	resources, err := NewResourceManager()
	require.NoError(t, err)
	return resources
}

func newGameWithRaceTraits(t *testing.T, traitIDs []string) *Game {
	t.Helper()
	resources := testResources(t)
	race, err := BuildCustomRace(resources, CustomRaceDesign{Name: "Synths", TraitIDs: traitIDs})
	require.NoError(t, err)
	g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: race})
	require.NoError(t, err)
	return g
}
