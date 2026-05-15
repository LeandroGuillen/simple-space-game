package game

import (
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGovernmentResourcesHaveDictatorshipDefault(t *testing.T) {
	resources := testResources(t)

	governments := baseGovernmentTechs(&resources.Resources)

	require.Len(t, governments, 4)
	assert.Equal(t, "Dictatorship", defaultGovernmentName(&resources.Resources))
}

func TestCustomRaceDefaultsAndValidatesGovernment(t *testing.T) {
	resources := testResources(t)

	race, err := BuildCustomRace(resources, CustomRaceDesign{Name: "Ada"})
	require.NoError(t, err)
	assert.Equal(t, "Dictatorship", race.Government)

	_, err = BuildCustomRace(resources, CustomRaceDesign{Name: "Ada", Government: "Nope"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown government")

	_, err = BuildCustomRace(resources, CustomRaceDesign{Name: "Ada", Government: "Imperium"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not available at race creation")
}

func TestGovernmentPickCostCountsTowardRaceBudget(t *testing.T) {
	resources := testResources(t)

	_, err := BuildCustomRace(resources, CustomRaceDesign{
		Name:       "Ada",
		Government: "Democracy",
		TraitIDs:   []string{"food_plus_2", "money_plus_1"},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "overspends")
}

func TestAdvancedGovernmentResearchRequiresUpgradePath(t *testing.T) {
	resources := testResources(t)
	democracy, err := BuildCustomRace(resources, CustomRaceDesign{Name: "Ada", Government: "Democracy"})
	require.NoError(t, err)
	g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: democracy})
	require.NoError(t, err)
	player := g.Players[0]

	require.NoError(t, g.selectResearch(player, model.SelectResearchCommand{TechName: "Federation"}))
	err = g.selectResearch(player, model.SelectResearchCommand{TechName: "Imperium"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), `does not upgrade current government "Democracy"`)
}

func TestUnificationCannotResearchMoraleBuildings(t *testing.T) {
	resources := testResources(t)
	unification, err := BuildCustomRace(resources, CustomRaceDesign{Name: "Ada", Government: "Unification"})
	require.NoError(t, err)
	g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: unification})
	require.NoError(t, err)

	err = g.selectResearch(g.Players[0], model.SelectResearchCommand{TechName: "Holo Simulator"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot research morale building")
}

func TestGovernmentMoraleSources(t *testing.T) {
	resources := testResources(t)
	feudal, err := BuildCustomRace(resources, CustomRaceDesign{Name: "Ada", Government: "Feudal"})
	require.NoError(t, err)
	g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: feudal})
	require.NoError(t, err)
	player := g.Players[0]
	planet, err := g.getPlanet(1)
	require.NoError(t, err)
	planet.Buildings = nil

	sources := PlanetMoraleSources(g, planet, player)

	assert.Contains(t, sourceNames(sources), "Feudal: no Barracks")
	assert.Contains(t, sourceNames(sources), "Feudal: no Capitol")
}

func TestUnificationIgnoresMoraleAndUsesLaborMultipliers(t *testing.T) {
	resources := testResources(t)
	unification, err := BuildCustomRace(resources, CustomRaceDesign{Name: "Ada", Government: "Unification"})
	require.NoError(t, err)
	g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: unification})
	require.NoError(t, err)
	planet, err := g.getPlanet(1)
	require.NoError(t, err)
	planet.Buildings = []string{"Holo Simulator"}
	planet.Population = 2000
	planet.Food.Farmers = 1
	planet.Industry.Workers = 1
	planet.Science.Scientists = 0
	hydratePlanetMorale(g, planet, g.Players[0])
	hydratePlanetOutputs(g, planet)

	assert.Zero(t, planet.Morale.Bonus)
	assert.Equal(t, int64(3), GamePlanetFoodProduced(g, planet))
	assert.Equal(t, int64(4), GamePlanetIndustryProduced(g, planet))
}

func TestGovernmentShipCostCommandPointsAndSpyBonus(t *testing.T) {
	resources := testResources(t)
	feudal, err := BuildCustomRace(resources, CustomRaceDesign{Name: "Ada", Government: "Feudal"})
	require.NoError(t, err)
	g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: feudal})
	require.NoError(t, err)
	planet, err := g.getPlanet(1)
	require.NoError(t, err)
	ship, err := getBuildingFromName(g, "Colony Ship")
	require.NoError(t, err)

	assert.Equal(t, int64(200), effectiveBuildingCost(planet, ship, g))
	assert.Equal(t, float64(0), DefensiveSpyBonus(g, g.Players[0]))

	dictatorship := DefaultRace("Ada")
	g, err = NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: dictatorship})
	require.NoError(t, err)
	g.research(g.Players[0], "Imperium")
	g.HydrateCommandPoints()

	assert.Equal(t, 0.2, DefensiveSpyBonus(g, g.Players[0]))
	assert.Equal(t, int64(10), g.Players[0].CommandPointsCapacity)
}

func TestFederationAppliesAdvancedTaxMultiplierInVisibleGame(t *testing.T) {
	resources := testResources(t)
	democracy, err := BuildCustomRace(resources, CustomRaceDesign{Name: "Ada", Government: "Democracy"})
	require.NoError(t, err)
	g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: democracy})
	require.NoError(t, err)
	player := g.Players[0]
	g.research(player, "Federation")

	assert.Equal(t, 1.75, governmentTaxMultiplier(g, player))

	visible, err := g.GetVisibleGameForPlayer(player.ID)
	require.NoError(t, err)
	planet := visible.Galaxy.Stars[0].Planets[0]

	assert.Equal(t, int64(14), planet.Income.Taxes)
	assert.Equal(t, "Federation", planet.Government.Name)
	assert.Equal(t, 1.75, planet.Government.TaxMultiplier)
	assert.Equal(t, 1.75, planet.Government.ScienceLaborMultiplier)
}

func sourceNames(sources []model.PlanetMoraleSource) []string {
	names := make([]string, 0, len(sources))
	for _, source := range sources {
		names = append(names, source.Name)
	}
	return names
}
