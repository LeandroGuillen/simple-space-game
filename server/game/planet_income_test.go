package game

import (
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/assert"
)

func TestPlanetIncomeIncludesBuildingMaintenance(t *testing.T) {
	g := &Game{
		Resources: &Resources{
			BuildingCollection: &BuildingCollection{
				Buildings: []model.Building{
					{Name: "Capitol", Maintenance: 0},
					{
						Name:        "Star Base",
						Maintenance: 2,
						Grants:      model.Grants{Modifiers: []model.Modifier{{Name: "income_flat", Value: 1}}},
					},
				},
			},
		},
	}
	planet := &model.Planet{
		Population: 8000,
		Buildings:  []string{"Capitol", "Star Base"},
		IsBuilding: true,
	}

	income := PlanetIncome(g, planet, 0)

	assert.Equal(t, int64(1), income.Buildings)
	assert.Equal(t, int64(2), income.Maintenance)
	assert.Equal(t, int64(7), income.Total)
}

func TestPlanetIncomeTradeGoodsRounding(t *testing.T) {
	tests := []struct {
		name             string
		industryProduced int64
		foodProduced     int64
		population       int64
		wantTradeGoods   int64
	}{
		{
			name:             "rounds down industry and food halves separately",
			industryProduced: 5,
			foodProduced:     5,
			population:       3000,
			wantTradeGoods:   3,
		},
		{
			name:             "does not subtract starving food deficit",
			industryProduced: 5,
			foodProduced:     2,
			population:       5000,
			wantTradeGoods:   2,
		},
		{
			name:             "does not subtract negative industry",
			industryProduced: -3,
			foodProduced:     9,
			population:       5000,
			wantTradeGoods:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			planet := &model.Planet{
				Population: tt.population,
				Food: model.PlanetFoodOutput{
					Farmers:             1,
					ProducedPerFarmer:   tt.foodProduced,
					ProducedPerBuilding: 0,
				},
			}

			income := PlanetIncome(&Game{}, planet, tt.industryProduced)

			assert.Equal(t, tt.wantTradeGoods, income.TradeGoods)
		})
	}
}
