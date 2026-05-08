package game

import (
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaxPopIncludesBuiltBiospheres(t *testing.T) {
	g := &Game{
		Resources: &Resources{
			BuildingCollection: &BuildingCollection{
				Buildings: []model.Building{
					{
						Name: "Biospheres",
						Grants: model.Grants{Modifiers: []model.Modifier{
							{Name: "max_population", Value: 2000},
						}},
					},
				},
			},
		},
	}
	planet := &model.Planet{
		Size:      model.PlanetSizeMedium,
		Buildings: []string{"Biospheres"},
	}

	assert.Equal(t, int64(14000), MaxPop(g, planet, nil))
}

func TestMaxPopIncludesResearchedAdvancedCityPlanning(t *testing.T) {
	player := &model.Player{
		ResearchedTechs: []string{"Advanced City Planning"},
	}
	g := &Game{
		Resources: &Resources{
			TechTree: &model.TechTree{
				AllTechs: []model.Tech{
					{
						Name: "Advanced City Planning",
						Grants: model.Grants{Modifiers: []model.Modifier{
							{Name: "max_population", Value: 5000},
						}},
					},
				},
			},
		},
	}
	planet := &model.Planet{Size: model.PlanetSizeMedium}

	assert.Equal(t, int64(17000), MaxPop(g, planet, player))
}

func TestTurnKillsPopulationOverCapacityWhenBiospheresIsGone(t *testing.T) {
	player := &model.Player{ID: 1}
	g := &Game{
		Players: []*model.Player{player},
		Resources: &Resources{
			BuildingCollection: &BuildingCollection{},
		},
	}
	planet := &model.Planet{
		Owner:      1,
		Size:       model.PlanetSizeMedium,
		Population: 14000,
		Food: model.PlanetFoodOutput{
			Farmers:           14,
			ProducedPerFarmer: 2,
		},
	}

	require.NoError(t, planetTurn(planet, model.Commands{}, player, g))

	assert.Equal(t, int64(12000), planet.Population)
	assert.Equal(t, int64(12000), planet.MaxPopulation)
	assert.Equal(t, int64(12), planet.Food.Farmers+planet.Industry.Workers+planet.Science.Scientists)
}

func TestPopulationGrowthBonusIncludesBuiltCloningCenter(t *testing.T) {
	g := &Game{
		Resources: &Resources{
			BuildingCollection: &BuildingCollection{
				Buildings: []model.Building{
					{
						Name: "Cloning Center",
						Grants: model.Grants{Modifiers: []model.Modifier{
							{Name: "population_growth", Value: 100},
						}},
					},
				},
			},
		},
	}
	planet := &model.Planet{Buildings: []string{"Cloning Center"}}

	assert.Equal(t, int64(100), PopulationGrowthBonus(g, planet))
}

func TestTurnAddsCloningCenterPopulationGrowth(t *testing.T) {
	player := &model.Player{ID: 1}
	g := &Game{
		Players: []*model.Player{player},
		Resources: &Resources{
			BuildingCollection: &BuildingCollection{
				Buildings: []model.Building{
					{
						Name: "Cloning Center",
						Grants: model.Grants{Modifiers: []model.Modifier{
							{Name: "population_growth", Value: 100},
						}},
					},
				},
			},
		},
	}
	planet := &model.Planet{
		Owner:      1,
		Size:       model.PlanetSizeHuge,
		Population: 10000,
		Food: model.PlanetFoodOutput{
			Farmers:           10,
			ProducedPerFarmer: 2,
		},
		Buildings: []string{"Cloning Center"},
	}
	growthWithoutCloningCenter := PopulationGrowth(
		planet.Population,
		planet.Food.ProducedPerBuilding,
		planet.Food.ProducedPerFarmer,
		planet.Food.Farmers,
		MaxPop(g, planet, player),
		0,
	)

	require.NoError(t, planetTurn(planet, model.Commands{}, player, g))

	assert.Equal(t, growthWithoutCloningCenter+100, planet.LastTurnPopGrowth)
}
