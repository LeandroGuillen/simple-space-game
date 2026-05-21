package game

import (
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCustomRaceAcceptsTolerantAndAquaticTraits(t *testing.T) {
	resources := testResources(t)
	for _, traitID := range []string{"tolerant", "aquatic"} {
		race, err := BuildCustomRace(resources, CustomRaceDesign{Name: "Ada", TraitIDs: []string{traitID}})
		require.NoError(t, err)
		assert.Contains(t, race.TraitIDs, traitID)
	}
}

func TestTolerantIgnoresPollution(t *testing.T) {
	g, planet := gameWithCustomRacePlanet(t, []string{"tolerant"})
	planet.Industry.Workers = 10
	planet.Industry.ProducedPerWorker = 3
	planet.Size = model.PlanetSizeHuge

	hydratePlanetOutputs(g, planet)

	assert.Equal(t, int64(0), planet.Pollution)
	assert.Equal(t, PlanetGrossIndustry(planet), GamePlanetIndustryProduced(g, planet))
}

func TestTolerantUsesTerranCapacityOnHostileWorlds(t *testing.T) {
	g, planet := gameWithCustomRacePlanet(t, []string{"tolerant"})
	planet.Environment = model.PlanetEnvironmentToxic
	planet.Size = model.PlanetSizeMedium

	assert.Equal(t, int64(12000), MaxPop(g, planet, g.Players[0]))
}

func TestHostileWorldHasReducedCapacityWithoutTolerant(t *testing.T) {
	g, planet := gameWithCustomRacePlanet(t, nil)
	planet.Environment = model.PlanetEnvironmentToxic
	planet.Size = model.PlanetSizeMedium

	assert.Equal(t, int64(4000), MaxPop(g, planet, g.Players[0]))
}

func TestAquaticDoublesOceanAndSwampCapacity(t *testing.T) {
	g, planet := gameWithCustomRacePlanet(t, []string{"aquatic"})
	planet.Size = model.PlanetSizeMedium
	planet.Environment = model.PlanetEnvironmentOcean

	assert.Equal(t, int64(12000), MaxPop(g, planet, g.Players[0]))

	planet.Environment = model.PlanetEnvironmentSwamp
	assert.Equal(t, int64(12000), MaxPop(g, planet, g.Players[0]))
}

func TestAquaticHalvesOtherHabitableCapacity(t *testing.T) {
	g, planet := gameWithCustomRacePlanet(t, []string{"aquatic"})
	planet.Size = model.PlanetSizeMedium
	planet.Environment = model.PlanetEnvironmentTerran

	assert.Equal(t, int64(6000), MaxPop(g, planet, g.Players[0]))
}

func TestTolerantIgnoresNegativeEnvironmentBonusOnHostileWorlds(t *testing.T) {
	g := &Game{
		Resources: &Resources{
			BuildingCollection: &BuildingCollection{
				Buildings: []model.Building{
					{
						Name: "Pollution Processor",
						Grants: model.Grants{Modifiers: []model.Modifier{
							{Name: "environment_bonus", Value: -0.5},
						}},
					},
				},
			},
		},
	}
	player := &model.Player{Race: model.Race{TraitIDs: []string{"tolerant"}}}
	planet := &model.Planet{
		Size:        model.PlanetSizeMedium,
		Environment: model.PlanetEnvironmentToxic,
		Buildings:   []string{"Pollution Processor"},
	}

	assert.Equal(t, int64(12000), MaxPop(g, planet, player))

	nonTolerant := &model.Player{Race: model.Race{}}
	assert.Equal(t, int64(2000), MaxPop(g, planet, nonTolerant))
}

func TestRacePopulationCapacityMultiplier(t *testing.T) {
	planet := &model.Planet{Environment: model.PlanetEnvironmentOcean}
	player := &model.Player{Race: model.Race{TraitIDs: []string{"aquatic"}}}

	assert.Equal(t, 2.0, racePopulationCapacityMultiplier(player, planet))
	assert.Equal(t, 1.0, racePopulationCapacityMultiplier(nil, planet))
	assert.Equal(t, 0.5, racePopulationCapacityMultiplier(player, &model.Planet{Environment: model.PlanetEnvironmentTerran}))
}
