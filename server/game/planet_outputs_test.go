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
