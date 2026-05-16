package game

import (
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/assert"
)

func gravityTestGame(gravity model.PlanetGravity, traitID string) (*Game, *model.Player, *model.Planet) {
	player := &model.Player{ID: 1}
	if traitID != "" {
		player.Race.TraitIDs = []string{traitID}
	}
	planet := &model.Planet{
		ID:           1,
		Owner:        1,
		Size:         model.PlanetSizeHuge,
		MineralClass: model.PlanetMineralAbundant,
		Gravity:      gravity,
		Population:   10000,
		Food: model.PlanetFoodOutput{
			Farmers:           4,
			ProducedPerFarmer: 2,
		},
		Industry: model.PlanetIndustryOutput{
			Workers:           4,
			ProducedPerWorker: 3,
		},
		Science: model.PlanetScienceOutput{
			Scientists:           4,
			ProducedPerScientist: 3,
		},
	}
	game := &Game{
		Players:   []*model.Player{player},
		Resources: &Resources{BuildingCollection: &BuildingCollection{}},
	}
	return game, player, planet
}

func TestGravityOutputMultiplier(t *testing.T) {
	cases := []struct {
		name     string
		gravity  model.PlanetGravity
		trait    string
		expected float64
	}{
		{"normal/no trait", model.PlanetGravityNormal, "", 1.0},
		{"normal/low-g race", model.PlanetGravityNormal, "low_g_world", 0.75},
		{"normal/high-g race", model.PlanetGravityNormal, "high_g_world", 1.0},
		{"low/no trait", model.PlanetGravityLow, "", 0.75},
		{"low/low-g race", model.PlanetGravityLow, "low_g_world", 1.0},
		{"low/high-g race", model.PlanetGravityLow, "high_g_world", 0.75},
		{"high/no trait", model.PlanetGravityHigh, "", 0.5},
		{"high/low-g race", model.PlanetGravityHigh, "low_g_world", 0.5},
		{"high/high-g race", model.PlanetGravityHigh, "high_g_world", 1.0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, player, planet := gravityTestGame(tc.gravity, tc.trait)
			assert.InDelta(t, tc.expected, gravityOutputMultiplier(planet, player), 1e-9)
		})
	}
}

func TestGravityAffectsLaborOutputs(t *testing.T) {
	cases := []struct {
		name        string
		gravity     model.PlanetGravity
		trait       string
		wantFood    int64
		wantIndustry int64
		wantScience int64
	}{
		// Base: 4 farmers * 2 = 8; 4 workers * 3 = 12; 4 scientists * 3 = 12
		{"normal/no trait", model.PlanetGravityNormal, "", 8, 12, 12},
		{"low/no trait", model.PlanetGravityLow, "", 6, 9, 9},
		{"high/no trait", model.PlanetGravityHigh, "", 4, 6, 6},
		{"low/low-g race", model.PlanetGravityLow, "low_g_world", 8, 12, 12},
		{"normal/low-g race", model.PlanetGravityNormal, "low_g_world", 6, 9, 9},
		{"high/low-g race", model.PlanetGravityHigh, "low_g_world", 4, 6, 6},
		{"high/high-g race", model.PlanetGravityHigh, "high_g_world", 8, 12, 12},
		{"normal/high-g race", model.PlanetGravityNormal, "high_g_world", 8, 12, 12},
		{"low/high-g race", model.PlanetGravityLow, "high_g_world", 6, 9, 9},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			game, _, planet := gravityTestGame(tc.gravity, tc.trait)
			assert.Equal(t, tc.wantFood, GamePlanetFoodProduced(game, planet), "food")
			assert.Equal(t, tc.wantIndustry, GamePlanetGrossIndustry(game, planet), "industry")
			assert.Equal(t, tc.wantScience, GamePlanetScienceProduced(game, planet), "science")
		})
	}
}
