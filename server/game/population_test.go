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
		0,
	)

	require.NoError(t, planetTurn(planet, model.Commands{}, player, g))

	assert.Equal(t, growthWithoutCloningCenter+100, planet.LastTurnPopGrowth)
}

func TestTurnAssignsNewPopulationToFarmingWhenPlanetWouldStarve(t *testing.T) {
	player := &model.Player{ID: 1}
	g := &Game{
		Players: []*model.Player{player},
		Resources: &Resources{
			BuildingCollection: &BuildingCollection{
				Buildings: []model.Building{
					{
						Name: "Growth Pod",
						Grants: model.Grants{Modifiers: []model.Modifier{
							{Name: "population_growth", Value: 200},
						}},
					},
				},
			},
		},
	}
	planet := &model.Planet{
		Owner:      1,
		Size:       model.PlanetSizeHuge,
		Population: 1900,
		Industry: model.PlanetIndustryOutput{
			Workers: 1,
		},
		Buildings: []string{"Growth Pod"},
	}

	require.NoError(t, planetTurn(planet, model.Commands{}, player, g))

	assert.Equal(t, int64(1), planet.Food.Farmers)
	assert.Equal(t, int64(1), planet.Industry.Workers)
	assert.Equal(t, int64(0), planet.Science.Scientists)
}

func TestTurnAssignsNewPopulationToLargerIndustryOrScienceWorkforceWhenFed(t *testing.T) {
	tests := []struct {
		name           string
		workers        int64
		scientists     int64
		wantWorkers    int64
		wantScientists int64
	}{
		{
			name:           "industry",
			workers:        1,
			scientists:     0,
			wantWorkers:    2,
			wantScientists: 0,
		},
		{
			name:           "science",
			workers:        0,
			scientists:     1,
			wantWorkers:    0,
			wantScientists: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			player := &model.Player{ID: 1}
			g := &Game{
				Players: []*model.Player{player},
				Resources: &Resources{
					BuildingCollection: &BuildingCollection{
						Buildings: []model.Building{
							{
								Name: "Growth Pod",
								Grants: model.Grants{Modifiers: []model.Modifier{
									{Name: "population_growth", Value: 300},
								}},
							},
							{
								Name: "Hydroponic Farms",
								Grants: model.Grants{Modifiers: []model.Modifier{
									{Name: "food_flat", Value: 10},
								}},
							},
						},
					},
				},
			}
			planet := &model.Planet{
				Owner:      1,
				Size:       model.PlanetSizeHuge,
				Population: 2900,
				Food: model.PlanetFoodOutput{
					Farmers: 1,
				},
				Industry: model.PlanetIndustryOutput{
					Workers: tt.workers,
				},
				Science: model.PlanetScienceOutput{
					Scientists: tt.scientists,
				},
				Buildings: []string{"Growth Pod", "Hydroponic Farms"},
			}

			require.NoError(t, planetTurn(planet, model.Commands{}, player, g))

			assert.Equal(t, int64(1), planet.Food.Farmers)
			assert.Equal(t, tt.wantWorkers, planet.Industry.Workers)
			assert.Equal(t, tt.wantScientists, planet.Science.Scientists)
		})
	}
}
