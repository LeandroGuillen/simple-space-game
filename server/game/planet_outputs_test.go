package game

import (
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/assert"
)

func TestHydratePlanetOutputsIncludesIndustryPerColonist(t *testing.T) {
	g := &Game{
		Resources: &Resources{
			BuildingCollection: &BuildingCollection{
				Buildings: []model.Building{
					{
						Name: "Recyclotron",
						Grants: model.Grants{Modifiers: []model.Modifier{
							{Name: "industry_per_colonist", Value: 1},
						}},
					},
				},
			},
		},
	}
	planet := &model.Planet{
		Population:   8000,
		MineralClass: model.PlanetMineralAbundant,
		Size:         model.PlanetSizeMedium,
		Industry:     model.PlanetIndustryOutput{Workers: 2},
		Buildings:    []string{"Recyclotron"},
	}

	hydratePlanetOutputs(g, planet)

	assert.Equal(t, int64(1), planet.Industry.ProducedPerColonist)
	assert.Equal(t, int64(14), PlanetGrossIndustry(planet))
	assert.Equal(t, int64(14), PlanetIndustryProduced(g, planet))
}

func TestHydratePlanetOutputsIncludesEmpireTechLaborModifiers(t *testing.T) {
	g := &Game{
		Players: []*model.Player{
			{
				ID:              1,
				ResearchedTechs: []string{"Microlite Construction", "Heightened Intelligence"},
			},
		},
		Resources: &Resources{
			TechTree: &model.TechTree{AllTechs: []model.Tech{
				{
					Name: "Microlite Construction",
					Grants: model.Grants{Modifiers: []model.Modifier{
						{Name: "industry_per_worker", Value: 1},
					}},
				},
				{
					Name: "Heightened Intelligence",
					Grants: model.Grants{Modifiers: []model.Modifier{
						{Name: "science_per_scientist", Value: 1},
					}},
				},
			}},
		},
	}
	planet := &model.Planet{
		Owner:        1,
		MineralClass: model.PlanetMineralAbundant,
		Industry:     model.PlanetIndustryOutput{Workers: 2},
		Science:      model.PlanetScienceOutput{Scientists: 2},
	}

	hydratePlanetOutputs(g, planet)

	assert.Equal(t, int64(4), planet.Industry.ProducedPerWorker)
	assert.Equal(t, int64(4), planet.Science.ProducedPerScientist)
}

func TestIndustryPerColonistDoesNotGeneratePollution(t *testing.T) {
	g := &Game{}
	planet := &model.Planet{
		Population: 20000,
		Size:       model.PlanetSizeTiny,
		Industry: model.PlanetIndustryOutput{
			ProducedPerColonist: 1,
		},
	}

	assert.Equal(t, int64(20), PlanetGrossIndustry(planet))
	assert.Equal(t, int64(0), Pollution(g, planet))
	assert.Equal(t, int64(20), PlanetIndustryProduced(g, planet))
}

func TestNanoDisassemblersDoubleEmpirePollutionTolerance(t *testing.T) {
	g := &Game{Players: []*model.Player{{ID: 1}}}
	planet := &model.Planet{
		Owner: 1,
		Size:  model.PlanetSizeMedium,
		Industry: model.PlanetIndustryOutput{
			Workers:           8,
			ProducedPerWorker: 1,
		},
	}

	assert.Equal(t, int64(1), Pollution(g, planet))

	g.Players[0].ResearchedTechs = []string{"Nano Disassemblers"}

	assert.Equal(t, int64(0), Pollution(g, planet))
}

func TestMoraleBoostsLaborOutputsOnly(t *testing.T) {
	planet := &model.Planet{
		Population: 10000,
		Size:       model.PlanetSizeMedium,
		Morale:     model.PlanetMoraleOutput{Bonus: 0.25, Multiplier: 1.25},
		Food: model.PlanetFoodOutput{
			Farmers:             3,
			ProducedPerFarmer:   2,
			ProducedPerBuilding: 4,
		},
		Industry: model.PlanetIndustryOutput{
			Workers:             3,
			ProducedPerWorker:   3,
			ProducedPerBuilding: 5,
			ProducedPerColonist: 1,
		},
		Science: model.PlanetScienceOutput{
			Scientists:           3,
			ProducedPerScientist: 3,
			ProducedPerBuilding:  5,
		},
	}

	assert.Equal(t, int64(11), PlanetFoodProduced(planet))
	assert.Equal(t, int64(26), PlanetGrossIndustry(planet))
	assert.Equal(t, int64(16), PlanetScienceProduced(planet))
}

func TestFoodReplicatorsConvertProductionToFoodWhenNeeded(t *testing.T) {
	g := &Game{
		Resources: &Resources{
			BuildingCollection: &BuildingCollection{
				Buildings: []model.Building{
					{Name: "Food Replicators"},
				},
			},
		},
	}
	planet := &model.Planet{
		Buildings: []string{"Food Replicators"},
		Food: model.PlanetFoodOutput{
			Farmers:           0,
			ProducedPerFarmer: 2,
		},
		Industry: model.PlanetIndustryOutput{
			Workers:             10,
			ProducedPerWorker:   2,
			ProducedPerBuilding: 0,
		},
	}

	// 10 pop units need 10 food, but we produce 0 food (no farmers)
	// Food Replicators should convert up to 20 production into 10 food
	result := CalculateFoodWithReplicators(g, planet, 10, 20)

	assert.Equal(t, int64(10), result.FoodProduced)
	assert.Equal(t, int64(20), result.IndustryConsumed)
	assert.Equal(t, int64(10), result.FoodFromReplicators)
}

func TestFoodReplicatorsPartialConversionWhenNotEnoughIndustry(t *testing.T) {
	g := &Game{
		Resources: &Resources{
			BuildingCollection: &BuildingCollection{
				Buildings: []model.Building{
					{Name: "Food Replicators"},
				},
			},
		},
	}
	planet := &model.Planet{
		Buildings: []string{"Food Replicators"},
		Food: model.PlanetFoodOutput{
			Farmers:           0,
			ProducedPerFarmer: 2,
		},
		Industry: model.PlanetIndustryOutput{
			Workers:             2,
			ProducedPerWorker:   2,
			ProducedPerBuilding: 0,
		},
	}

	// Need 10 food, have 4 industry -> can only convert 4 production into 2 food
	result := CalculateFoodWithReplicators(g, planet, 10, 4)

	assert.Equal(t, int64(2), result.FoodProduced)
	assert.Equal(t, int64(4), result.IndustryConsumed)
	assert.Equal(t, int64(2), result.FoodFromReplicators)
}

func TestFoodReplicatorsDoNothingWhenFoodSurplus(t *testing.T) {
	g := &Game{
		Resources: &Resources{
			BuildingCollection: &BuildingCollection{
				Buildings: []model.Building{
					{Name: "Food Replicators"},
				},
			},
		},
	}
	planet := &model.Planet{
		Buildings: []string{"Food Replicators"},
		Food: model.PlanetFoodOutput{
			Farmers:           5,
			ProducedPerFarmer: 2,
		},
		Industry: model.PlanetIndustryOutput{
			Workers:             10,
			ProducedPerWorker:   2,
			ProducedPerBuilding: 0,
		},
	}

	// 10 food produced > 5 food needed, no conversion needed
	result := CalculateFoodWithReplicators(g, planet, 5, 20)

	assert.Equal(t, int64(10), result.FoodProduced)
	assert.Equal(t, int64(0), result.IndustryConsumed)
	assert.Equal(t, int64(0), result.FoodFromReplicators)
}

func TestFoodReplicatorsDoNothingWhenNotBuilt(t *testing.T) {
	g := &Game{
		Resources: &Resources{
			BuildingCollection: &BuildingCollection{
				Buildings: []model.Building{
					{Name: "Food Replicators"},
				},
			},
		},
	}
	planet := &model.Planet{
		Buildings: []string{}, // No Food Replicators
		Food: model.PlanetFoodOutput{
			Farmers:           0,
			ProducedPerFarmer: 2,
		},
		Industry: model.PlanetIndustryOutput{
			Workers:             10,
			ProducedPerWorker:   2,
			ProducedPerBuilding: 0,
		},
	}

	result := CalculateFoodWithReplicators(g, planet, 10, 20)

	assert.Equal(t, int64(0), result.FoodProduced)
	assert.Equal(t, int64(0), result.IndustryConsumed)
	assert.Equal(t, int64(0), result.FoodFromReplicators)
}

func TestFoodReplicatorsWorkInHostileEnvironment(t *testing.T) {
	g := &Game{
		Resources: &Resources{
			BuildingCollection: &BuildingCollection{
				Buildings: []model.Building{
					{Name: "Food Replicators"},
				},
			},
			EnvironmentCollection: &EnvironmentCollection{
				Environments: []PlanetEnvironmentDef{
					{Name: "Toxic", BaseFoodPerFarmer: 0},
				},
			},
		},
	}
	planet := &model.Planet{
		Buildings:   []string{"Food Replicators"},
		Environment: model.PlanetEnvironmentToxic, // Hostile - no farming possible
		Food: model.PlanetFoodOutput{
			Farmers:           0, // Can't farm on toxic
			ProducedPerFarmer: 0,
		},
		Industry: model.PlanetIndustryOutput{
			Workers:             10,
			ProducedPerWorker:   3,
			ProducedPerBuilding: 0,
		},
	}

	// Food Replicators should still work even when no farming is possible
	result := CalculateFoodWithReplicators(g, planet, 10, 30)

	assert.Equal(t, int64(10), result.FoodProduced)
	assert.Equal(t, int64(20), result.IndustryConsumed)
	assert.Equal(t, int64(10), result.FoodFromReplicators)
}
