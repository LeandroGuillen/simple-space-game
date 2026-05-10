package game

import (
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCustomRaceRejectsUnknownTrait(t *testing.T) {
	_, err := BuildCustomRace(CustomRaceDesign{Name: "Ada", TraitIDs: []string{"nope"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown race trait")
}

func TestBuildCustomRaceRejectsDisabledTrait(t *testing.T) {
	_, err := BuildCustomRace(CustomRaceDesign{Name: "Ada", TraitIDs: []string{"creative"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
}

func TestBuildCustomRaceRejectsDuplicateTrait(t *testing.T) {
	_, err := BuildCustomRace(CustomRaceDesign{Name: "Ada", TraitIDs: []string{"food_plus_1", "food_plus_1"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate")
}

func TestBuildCustomRaceRejectsOverspentPicks(t *testing.T) {
	_, err := BuildCustomRace(CustomRaceDesign{Name: "Ada", TraitIDs: []string{"money_plus_1", "food_plus_2"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "overspends")
}

func TestBuildCustomRaceRejectsTooManyNegativePicks(t *testing.T) {
	_, err := BuildCustomRace(CustomRaceDesign{Name: "Ada", TraitIDs: []string{"growth_minus_50", "food_minus_1", "production_minus_1", "money_minus_half"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "negative picks")
}

func TestBuildCustomRaceRejectsExclusiveGroupConflict(t *testing.T) {
	_, err := BuildCustomRace(CustomRaceDesign{Name: "Ada", TraitIDs: []string{"research_plus_1", "research_plus_2"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")
}

func TestNewGameAppliesCustomRaceNameAndTraitIDs(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)
	race, err := BuildCustomRace(CustomRaceDesign{Name: "Synths", TraitIDs: []string{"research_plus_1"}})
	require.NoError(t, err)

	g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: race})
	require.NoError(t, err)

	assert.Equal(t, "Synths", g.Players[0].Race.Name)
	assert.Equal(t, []string{"research_plus_1"}, g.Players[0].Race.TraitIDs)
}

func TestRaceFoodProductionModifierAffectsPlanetOutput(t *testing.T) {
	g, planet := gameWithCustomRacePlanet(t, []string{"food_plus_1"})
	hydratePlanetOutputs(g, planet)

	assert.Equal(t, int64(3), planet.Food.ProducedPerFarmer)
}

func TestRaceIndustryModifierAffectsPlanetOutput(t *testing.T) {
	g, planet := gameWithCustomRacePlanet(t, []string{"production_plus_1"})
	hydratePlanetOutputs(g, planet)

	assert.Equal(t, int64(4), planet.Industry.ProducedPerWorker)
}

func TestRaceResearchModifierAffectsPlanetOutput(t *testing.T) {
	g, planet := gameWithCustomRacePlanet(t, []string{"research_plus_2"})
	hydratePlanetOutputs(g, planet)

	assert.Equal(t, int64(5), planet.Science.ProducedPerScientist)
}

func TestRaceIncomeBonusAffectsPlanetIncome(t *testing.T) {
	g, planet := gameWithCustomRacePlanet(t, []string{"money_plus_half"})
	hydratePlanetMorale(g, planet, g.Players[0])
	hydratePlanetOutputs(g, planet)
	income := PlanetIncome(g, planet, 0)

	assert.Equal(t, 0.5, income.Bonus)
	assert.Equal(t, int64(15), income.Total)
}

func TestRacePopulationGrowthPercentAffectsTurnGrowth(t *testing.T) {
	g, planet := gameWithCustomRacePlanet(t, []string{"growth_plus_100"})
	baseGrowth := PopulationGrowthFromFood(planet.Population, PlanetFoodProduced(planet), MaxPop(g, planet, g.Players[0]), 0, 0)

	require.NoError(t, planetTurn(planet, model.Commands{}, g.Players[0], g))

	assert.Greater(t, planet.LastTurnPopGrowth, baseGrowth)
}

func TestSubterraneanIncreasesMaxPopulation(t *testing.T) {
	g, planet := gameWithCustomRacePlanet(t, []string{"subterranean"})

	assert.Equal(t, baseMaxPop(planet)*2, MaxPop(g, planet, g.Players[0]))
}

func TestLargeHomeworldChangesCapitalSize(t *testing.T) {
	g := newGameWithRaceTraits(t, []string{"large_homeworld"})
	planet, err := g.getPlanet(1)
	require.NoError(t, err)

	assert.Equal(t, model.PlanetSizeLarge, planet.Size)
}

func TestRichAndPoorHomeworldChangeCapitalMinerals(t *testing.T) {
	rich := newGameWithRaceTraits(t, []string{"rich_homeworld"})
	richPlanet, err := rich.getPlanet(1)
	require.NoError(t, err)
	assert.Equal(t, model.PlanetMineralRich, richPlanet.MineralClass)

	poor := newGameWithRaceTraits(t, []string{"poor_homeworld"})
	poorPlanet, err := poor.getPlanet(1)
	require.NoError(t, err)
	assert.Equal(t, model.PlanetMineralPoor, poorPlanet.MineralClass)
}

func TestArtifactsHomeworldIncreasesCapitalScience(t *testing.T) {
	g := newGameWithRaceTraits(t, []string{"artifacts_homeworld"})
	planet, err := g.getPlanet(1)
	require.NoError(t, err)
	hydratePlanetOutputs(g, planet)

	assert.Equal(t, int64(5), planet.Science.ProducedPerScientist)
}

func gameWithCustomRacePlanet(t *testing.T, traitIDs []string) (*Game, *model.Planet) {
	t.Helper()
	g := newGameWithRaceTraits(t, traitIDs)
	planet, err := g.getPlanet(1)
	require.NoError(t, err)
	planet.Buildings = nil
	planet.Population = 10000
	planet.Food.Farmers = 5
	planet.Industry.Workers = 3
	planet.Science.Scientists = 2
	planet.MineralClass = model.PlanetMineralAbundant
	return g, planet
}

func newGameWithRaceTraits(t *testing.T, traitIDs []string) *Game {
	t.Helper()
	resources, err := NewResourceManager()
	require.NoError(t, err)
	race, err := BuildCustomRace(CustomRaceDesign{Name: "Synths", TraitIDs: traitIDs})
	require.NoError(t, err)
	g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: race})
	require.NoError(t, err)
	return g
}
