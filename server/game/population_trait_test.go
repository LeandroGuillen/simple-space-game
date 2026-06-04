package game

import (
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPopulationFoodDemand(t *testing.T) {
	player := &model.Player{Race: model.Race{TraitIDs: []string{"cybernetic"}}}
	assert.Equal(t, 5.0, PopulationFoodDemand(player, 10000))
	assert.Equal(t, 0.0, PopulationFoodDemand(&model.Player{Race: model.Race{TraitIDs: []string{"lithovore"}}}, 10000))
	assert.Equal(t, 10.0, PopulationFoodDemand(&model.Player{}, 10000))
	assert.Equal(t, 10.0, PopulationFoodDemand(nil, 10000))
}

func TestCyberneticHalvesFoodStarvationPenalty(t *testing.T) {
	player := &model.Player{Race: model.Race{TraitIDs: []string{"cybernetic"}}}
	// 10k pop needs 5 food; 4 produced -> 1 lack -> 50 penalty
	withTrait := PopulationGrowthFromFood(player, 10000, 4, 25000, 0, 0)
	withoutTrait := PopulationGrowthFromFood(nil, 10000, 4, 25000, 0, 0)
	assert.Greater(t, withTrait, withoutTrait)
}

func TestLithovoreIgnoresFoodShortage(t *testing.T) {
	player := &model.Player{Race: model.Race{TraitIDs: []string{"lithovore"}}}
	growth := PopulationGrowthFromFood(player, 10000, 0, 25000, 0, 0)
	noFoodNormal := PopulationGrowthFromFood(nil, 10000, 0, 25000, 0, 0)
	assert.Greater(t, growth, noFoodNormal)
	assert.Greater(t, growth, int64(0))
}

func TestBuildCustomRaceAcceptsCyberneticAndLithovore(t *testing.T) {
	resources := testResources(t)
	cyber, err := BuildCustomRace(resources, CustomRaceDesign{Name: "Bots", TraitIDs: []string{"cybernetic"}})
	require.NoError(t, err)
	assert.Contains(t, cyber.TraitIDs, "cybernetic")

	litho, err := BuildCustomRace(resources, CustomRaceDesign{Name: "Rocks", TraitIDs: []string{"lithovore"}})
	require.NoError(t, err)
	assert.Contains(t, litho.TraitIDs, "lithovore")
}

func TestLithovoreFreighterFoodSurplusUsesZeroConsumption(t *testing.T) {
	g := newGameWithRaceTraits(t, []string{"lithovore"})
	player := g.Players[0]
	planet, err := g.getPlanet(1)
	require.NoError(t, err)
	planet.Population = 10000
	planet.Food.Farmers = 5
	planet.Food.ProducedPerFarmer = 2
	hydratePlanetOutputs(g, planet)

	result := g.calculateAutomaticFoodTransfers(player, []*model.Planet{planet}, model.Commands{})
	assert.Empty(t, result.PlanetReceived)
}

func TestCyberneticFreighterDeficitHalved(t *testing.T) {
	g := newGameWithRaceTraits(t, []string{"cybernetic"})
	player := g.Players[0]
	player.Freighters = 10
	hungry, err := g.getPlanet(1)
	require.NoError(t, err)
	star, err := g.getStar(hungry.ParentStar)
	require.NoError(t, err)
	hungry.Population = 10000
	hungry.Food.Farmers = 1
	hungry.Food.ProducedPerFarmer = 2
	fed := &model.Planet{
		ID: 99, Owner: 1, ParentStar: hungry.ParentStar,
		Environment: model.PlanetEnvironmentTerran,
		Population:  1000, Food: model.PlanetFoodOutput{Farmers: 10, ProducedPerFarmer: 2},
	}
	star.Planets = append(star.Planets, fed)
	hydratePlanetOutputs(g, hungry)
	hydratePlanetOutputs(g, fed)

	result := g.calculateAutomaticFoodTransfers(player, []*model.Planet{hungry, fed}, model.Commands{})
	// demand 5, produced 2 -> deficit 3
	assert.Equal(t, int64(3), result.PlanetReceived[hungry.ID])
}
