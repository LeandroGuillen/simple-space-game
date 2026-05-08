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
	assert.Equal(t, int64(14), PlanetIndustryProduced(planet))
}

func TestIndustryPerColonistDoesNotGeneratePollution(t *testing.T) {
	planet := &model.Planet{
		Population: 20000,
		Size:       model.PlanetSizeTiny,
		Industry: model.PlanetIndustryOutput{
			ProducedPerColonist: 1,
		},
	}

	assert.Equal(t, int64(20), PlanetGrossIndustry(planet))
	assert.Equal(t, int64(0), Pollution(planet))
	assert.Equal(t, int64(20), PlanetIndustryProduced(planet))
}
