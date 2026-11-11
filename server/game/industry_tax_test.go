package game

import (
	"fmt"
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MoO2 1.50 assumptions: 1 BC per industry, floor per colony, after
// pollution and replicators. Population tax, morale and pollution stay intact.
func TestIndustryTaxTurn(t *testing.T) {
	for _, rate := range []float64{0, .3, .5} {
		t.Run(fmt.Sprint(rate), func(t *testing.T) {
			g, player, p := housingTestGame()
			player.TaxRate = rate
			p.CurrentProject = "Test building"
			p.IsBuilding = true
			p.BuildBucket = 100
			require.NoError(t, planetTurn(p, model.Commands{}, player, g))
			tax := int64(10 * rate)
			assert.Equal(t, int64(90)+tax, p.BuildBucket)
			assert.Equal(t, int64(10)+tax, p.Income.Total)
			assert.Equal(t, tax, p.Income.IndustryTax)
			assert.Equal(t, int64(10), p.Income.Taxes)
			assert.Equal(t, int64(10), GamePlanetIndustryProduced(g, p))
		})
	}
}

func TestIndustryTaxNotMultipliedByIncomeBonuses(t *testing.T) {
	g, player, p := housingTestGame()
	player.TaxRate = .3
	p.IsBuilding = true
	p.Buildings = []string{"Space Port", "Stock Exchange"}
	g.Resources.BuildingCollection.Buildings = []model.Building{
		{Name: "Space Port", Grants: model.Grants{Modifiers: []model.Modifier{{Name: "income_bonus", Value: .5}}}},
		{Name: "Stock Exchange", Grants: model.Grants{Modifiers: []model.Modifier{{Name: "income_bonus", Value: 1}}}},
	}
	assert.Equal(t, int64(28), PlanetIncome(g, p, 10).Total)
	// Fractional industry tax is floored at each colony.
	assert.Equal(t, int64(28), PlanetIncome(g, p, 11).Total)
}

func TestIndustryTaxPreviewMatchesTurnWithReplicators(t *testing.T) {
	for _, replicators := range []bool{false, true} {
		t.Run(fmt.Sprint(replicators), func(t *testing.T) {
			g, player, p := housingTestGame()
			player.TaxRate = .3
			p.CurrentProject = tradeGoodsProject
			p.Food.Farmers = 5
			if replicators {
				p.Buildings = []string{"Food Replicators"}
				p.Food.Farmers = 3 // 6 food; replicators consume 8 of the 18 industry.
				g.Resources.BuildingCollection.Buildings = []model.Building{{Name: "Food Replicators", Grants: model.Grants{Modifiers: []model.Modifier{{Name: "industry_flat", Value: 8}}}}}
			}
			g.Galaxy = model.Galaxy{Stars: []*model.Star{{Planets: []*model.Planet{p}}}}
			copies, err := g.GetColonyList(player.ID)
			require.NoError(t, err)
			live, err := g.colonyListUnlocked(player.ID)
			require.NoError(t, err)
			assert.Equal(t, copies[0].Income, live[0].Income)
			expected := int64(16) // 10 population + 3 trade industry + 3 tax
			if replicators {
				expected = 16
			} // 10 population + 3 trade industry + floor((18-8)*.3)
			assert.Equal(t, expected, copies[0].Income.Total)
			require.NoError(t, planetTurn(p, model.Commands{}, player, g))
			assert.Equal(t, copies[0].Income, p.Income)
		})
	}
}

func TestIndustryTaxReducesHousingGrowth(t *testing.T) {
	g, player, p := housingTestGame()
	player.TaxRate = .3
	p.Food.Farmers = 5
	baseline := PopulationGrowthFromFood(player, p.Population, 10, MaxPop(g, p, player), 0, 0)
	require.NoError(t, planetTurn(p, model.Commands{Build: []*model.BuildCommand{{PlanetID: p.ID, Building: housingProject}}}, player, g))
	// Seven remaining industry gives 28% housing growth at ten population units.
	assert.Equal(t, baseline*128/100, p.LastTurnPopGrowth)
	assert.Equal(t, int64(3), p.Income.IndustryTax)
}
