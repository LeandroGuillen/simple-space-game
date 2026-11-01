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

func TestGravityGeneratorRemovesGravityPenalty(t *testing.T) {
	cases := []struct {
		name    string
		gravity model.PlanetGravity
		trait   string
	}{
		{"low gravity, no trait", model.PlanetGravityLow, ""},
		{"high gravity, no trait", model.PlanetGravityHigh, ""},
		{"high gravity, low-g race", model.PlanetGravityHigh, "low_g_world"},
		{"normal gravity, low-g race", model.PlanetGravityNormal, "low_g_world"},
		{"low gravity, high-g race", model.PlanetGravityLow, "high_g_world"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, player, planet := gravityTestGame(tc.gravity, tc.trait)
			assert.Less(t, gravityOutputMultiplier(planet, player), 1.0, "precondition: expected gravity penalty")
			planet.Buildings = append(planet.Buildings, gravityGeneratorBuildingName)
			assert.Equal(t, 1.0, gravityOutputMultiplier(planet, player))
		})
	}
}

func TestBuildGravityGeneratorRejectedWhenGravityMatchesRace(t *testing.T) {
	game, _, planet := gravityTestGame(model.PlanetGravityNormal, "")
	game.Resources.Buildings = []model.Building{{Name: gravityGeneratorBuildingName, Cost: 120}}

	assert.False(t, canBuild(planet, gravityGeneratorBuildingName, []string{gravityGeneratorBuildingName}, game))
	assert.False(t, Build(planet, gravityGeneratorBuildingName, []string{gravityGeneratorBuildingName}, game))
	assert.False(t, planet.IsBuilding)
}

func TestBuildGravityGeneratorAllowedWhenGravityMismatches(t *testing.T) {
	game, _, planet := gravityTestGame(model.PlanetGravityHigh, "")
	game.Resources.Buildings = []model.Building{{Name: gravityGeneratorBuildingName, Cost: 120}}

	assert.True(t, canBuild(planet, gravityGeneratorBuildingName, []string{gravityGeneratorBuildingName}, game))
	assert.True(t, Build(planet, gravityGeneratorBuildingName, []string{gravityGeneratorBuildingName}, game))
	assert.True(t, planet.IsBuilding)
	assert.Equal(t, gravityGeneratorBuildingName, planet.CurrentProject)
}

func TestGravityGeneratorApplicableOnlyWhenGravityDiffers(t *testing.T) {
	cases := []struct {
		name     string
		gravity  model.PlanetGravity
		trait    string
		expected bool
	}{
		{"normal/no trait", model.PlanetGravityNormal, "", false},
		{"low/no trait", model.PlanetGravityLow, "", true},
		{"high/no trait", model.PlanetGravityHigh, "", true},
		{"low/low-g race", model.PlanetGravityLow, "low_g_world", false},
		{"normal/low-g race", model.PlanetGravityNormal, "low_g_world", true},
		{"high/low-g race", model.PlanetGravityHigh, "low_g_world", true},
		{"high/high-g race", model.PlanetGravityHigh, "high_g_world", false},
		{"normal/high-g race", model.PlanetGravityNormal, "high_g_world", true},
		{"low/high-g race", model.PlanetGravityLow, "high_g_world", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, player, planet := gravityTestGame(tc.gravity, tc.trait)
			assert.Equal(t, tc.expected, gravityGeneratorApplicable(planet, player))
		})
	}
}

func TestGravityAffectsLaborOutputs(t *testing.T) {
	cases := []struct {
		name         string
		gravity      model.PlanetGravity
		trait        string
		wantFood     int64
		wantIndustry int64
		wantScience  int64
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

func TestCapitalGravityMatchesRace(t *testing.T) {
	for _, tc := range []struct {
		name, trait string
		gravity     model.PlanetGravity
	}{
		{"normal", "", model.PlanetGravityNormal},
		{"low", "low_g_world", model.PlanetGravityLow},
		{"high", "high_g_world", model.PlanetGravityHigh},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resources, err := NewResourceManager()
			if err != nil {
				t.Fatal(err)
			}
			race := DefaultRace("Human")
			if tc.trait != "" {
				race.TraitIDs = append(race.TraitIDs, tc.trait)
			}
			g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: race})
			if err != nil {
				t.Fatal(err)
			}
			player := g.Players[0]
			var capital *model.Planet
			for _, star := range g.Galaxy.Stars {
				for _, p := range star.Planets {
					if p.Owner == player.ID {
						capital = p
					}
				}
			}
			if capital == nil {
				t.Fatal("capital missing")
			}
			assert.Equal(t, tc.gravity, capital.Gravity)
			assert.Equal(t, float64(1), gravityOutputMultiplier(capital, player))
		})
	}
}
