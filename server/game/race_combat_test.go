package game

import (
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCustomRaceAcceptsGroundCombat(t *testing.T) {
	resources := testResources(t)
	for _, traitID := range []string{"ground_combat"} {
		race, err := BuildCustomRace(resources, CustomRaceDesign{Name: "Ada", TraitIDs: []string{traitID}})
		require.NoError(t, err)
		assert.Contains(t, race.TraitIDs, traitID)
	}
}

func TestGroundCombatTraitAppliesModifierAndMultiplier(t *testing.T) {
	resources := testResources(t)
	race, err := BuildCustomRace(resources, CustomRaceDesign{Name: "Marines", TraitIDs: []string{"ground_combat"}})
	require.NoError(t, err)

	player := &model.Player{Race: race}
	assert.Equal(t, 0.5, raceModifierValue(player, "ground_combat_bonus", 0))
	assert.Equal(t, 1.5, GroundCombatMultiplier(player))
	assert.Equal(t, 1.0, GroundCombatMultiplier(nil))
}

func TestSpyingTraitAppliesSpyBonusAndOffensiveModifier(t *testing.T) {
	def := raceTraitDefinitionsByID()["spying"]
	race := model.Race{TraitIDs: []string{"spying"}, Stats: def.Stats, Grants: def.Grants}

	player := &model.Player{Race: race}
	assert.Equal(t, int64(10), RaceSpyBonus(player))
	assert.Equal(t, 0.25, raceModifierValue(player, "offensive_spy_bonus", 0))
	assert.Equal(t, int64(0), RaceSpyBonus(nil))
}

func TestOffensiveSpyBonusWithoutGovernmentModifier(t *testing.T) {
	def := raceTraitDefinitionsByID()["spying"]
	race := model.Race{TraitIDs: []string{"spying"}, Stats: def.Stats, Grants: def.Grants}
	player := &model.Player{Race: race}
	assert.Equal(t, 0.25, OffensiveSpyBonus(nil, player))
}
