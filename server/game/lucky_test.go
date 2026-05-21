package game

import (
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLuckyEventBonus(t *testing.T) {
	assert.Equal(t, 0.0, LuckyEventBonus(nil))
	assert.Equal(t, 0.0, LuckyEventBonus(&model.Player{}))
	assert.Equal(t, 0.25, LuckyEventBonus(&model.Player{Race: model.Race{TraitIDs: []string{"lucky"}}}))
}

func TestLuckyTurnResearchBonusNoTrait(t *testing.T) {
	SetRandomSeed(1)
	assert.Equal(t, uint64(0), luckyTurnResearchBonus(&model.Player{}))
}

func TestBuildCustomRaceAcceptsLuckyTrait(t *testing.T) {
	resources := testResources(t)
	race, err := BuildCustomRace(resources, CustomRaceDesign{Name: "Fortunate", TraitIDs: []string{"lucky"}})
	require.NoError(t, err)
	assert.Contains(t, race.TraitIDs, "lucky")
}

func TestLuckyTurnGrantsBonusResearchWithSeededRoll(t *testing.T) {
	SetRandomSeed(1)
	g := newGameWithRaceTraits(t, []string{"lucky"})
	player := g.Players[0]
	player.ResearchPoints = 0

	require.NoError(t, g.Turn(player.ID, model.Commands{}))

	assert.Equal(t, uint64(4), player.ResearchPoints)
}

func TestLuckyTurnSkipsBonusResearchWithSeededRoll(t *testing.T) {
	SetRandomSeed(99)
	g := newGameWithRaceTraits(t, []string{"lucky"})
	player := g.Players[0]
	player.ResearchPoints = 0

	require.NoError(t, g.Turn(player.ID, model.Commands{}))

	assert.Equal(t, uint64(3), player.ResearchPoints)
}

func TestNonLuckyTurnNeverGrantsBonusResearch(t *testing.T) {
	for seed := int64(0); seed < 20; seed++ {
		SetRandomSeed(seed)
		g := newGameWithRaceTraits(t, nil)
		player := g.Players[0]
		player.ResearchPoints = 0
		require.NoError(t, g.Turn(player.ID, model.Commands{}))
		assert.Equal(t, uint64(3), player.ResearchPoints, "seed %d", seed)
	}
}
