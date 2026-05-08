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

func TestPlanetIncomeBonusesStackAdditively(t *testing.T) {
	g := &Game{
		Resources: &Resources{
			BuildingCollection: &BuildingCollection{
				Buildings: []model.Building{
					{
						Name:        "Space Port",
						Maintenance: 1,
						Grants:      model.Grants{Modifiers: []model.Modifier{{Name: "income_bonus", Value: 0.5}}},
					},
					{
						Name:        "Stock Exchange",
						Maintenance: 2,
						Grants:      model.Grants{Modifiers: []model.Modifier{{Name: "income_bonus", Value: 1}}},
					},
				},
			},
		},
	}
	planet := &model.Planet{
		Population: 10000,
		Buildings:  []string{"Space Port", "Stock Exchange"},
		IsBuilding: true,
	}

	income := PlanetIncome(g, planet, 0)

	assert.Equal(t, int64(10), income.Taxes)
	assert.Equal(t, int64(3), income.Maintenance)
	assert.Equal(t, int64(22), income.Total)
	assert.Equal(t, 1.5, income.Bonus)
}

func TestPlanetIncomeIncludesResearchedTechBonuses(t *testing.T) {
	g := &Game{
		Players: []*model.Player{
			{
				ID:              1,
				ResearchedTechs: []string{"Galactic Currency Exchange"},
			},
		},
		Resources: &Resources{
			TechTree: &model.TechTree{
				AllTechs: []model.Tech{
					{
						Name:   "Galactic Currency Exchange",
						Grants: model.Grants{Modifiers: []model.Modifier{{Name: "income_bonus", Value: 0.5}}},
					},
				},
			},
		},
	}
	planet := &model.Planet{
		Owner:      1,
		Population: 10000,
		IsBuilding: true,
	}

	income := PlanetIncome(g, planet, 0)

	assert.Equal(t, int64(15), income.Total)
	assert.Equal(t, 0.5, income.Bonus)
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
