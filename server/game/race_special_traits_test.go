package game

import (
	"ssg/model"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCustomRaceSpecialShipTraits(t *testing.T) {
	resources := testResources(t)

	stealthy, err := BuildCustomRace(resources, CustomRaceDesign{Name: "Ghost", TraitIDs: []string{"stealthy_ships"}})
	require.NoError(t, err)
	assert.Equal(t, int64(-2), stealthy.Stats.DetectionModifier)

	trans, err := BuildCustomRace(resources, CustomRaceDesign{Name: "Phase", TraitIDs: []string{"trans_dimensional"}})
	require.NoError(t, err)
	assert.Equal(t, int64(2), trans.Stats.CombatSpeedBonus)

	warlord, err := BuildCustomRace(resources, CustomRaceDesign{Name: "Marshal", TraitIDs: []string{"warlord"}})
	require.NoError(t, err)
	assert.Equal(t, int64(25), warlord.Stats.ShipAttackBonusPercent)
	assert.Equal(t, int64(25), warlord.Stats.ShipDefenseBonusPercent)
}

func TestRaceDetectionModifier(t *testing.T) {
	g := newGameWithRaceTraits(t, []string{"stealthy_ships"})
	assert.Equal(t, int64(-2), RaceDetectionModifier(g.Players[0]))
	assert.Equal(t, int64(0), RaceDetectionModifier(nil))
}

func TestFleetShipCombatSpeedIncludesTransDimensionalBonus(t *testing.T) {
	g := newGameWithRaceTraits(t, []string{"trans_dimensional"})
	player := g.Players[0]

	speed, ok := g.FleetShipCombatSpeed(player, model.FleetShip{Drive: "Fusion"})
	require.True(t, ok)
	assert.Equal(t, int64(3), speed) // Fusion combat speed 1 + race bonus 2
}

func TestFireOnWarlordAttackBonusIncreasesDamage(t *testing.T) {
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
	race, err := BuildCustomRace(resources, CustomRaceDesign{Name: "Marshal", TraitIDs: []string{"warlord"}})
	require.NoError(t, err)
	attackerPlayer := &model.Player{Race: race}

	withWarlord := FireOn(&attacker, &defender, 0, attackerPlayer, nil)
	assert.Greater(t, withWarlord, baseline)
	assert.Equal(t, int64(62), withWarlord)
}

func TestFireOnWarlordDefenseBonusReducesDamage(t *testing.T) {
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
	race, err := BuildCustomRace(resources, CustomRaceDesign{Name: "Marshal", TraitIDs: []string{"warlord"}})
	require.NoError(t, err)
	defenderPlayer := &model.Player{Race: race}

	withWarlord := FireOn(&attacker, &defender, 0, nil, defenderPlayer)
	assert.Less(t, withWarlord, baseline)
	assert.Equal(t, int64(46), withWarlord)
}
