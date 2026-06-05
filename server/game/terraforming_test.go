package game

import (
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTerraformingTransitions(t *testing.T) {
	tests := []struct {
		name     string
		from     model.PlanetEnvironment
		prereq   string
		expected []model.PlanetEnvironment
	}{
		{"Toxic to Barren", model.PlanetEnvironmentToxic, "Weather Controller", []model.PlanetEnvironment{model.PlanetEnvironmentBarren}},
		{"Radiated to Barren", model.PlanetEnvironmentRadiated, "Planetary Radiation Shield", []model.PlanetEnvironment{model.PlanetEnvironmentBarren}},
		{"Barren to Desert or Tundra", model.PlanetEnvironmentBarren, "", []model.PlanetEnvironment{model.PlanetEnvironmentDesert, model.PlanetEnvironmentTundra}},
		{"Desert to Arid", model.PlanetEnvironmentDesert, "", []model.PlanetEnvironment{model.PlanetEnvironmentArid}},
		{"Tundra to Swamp", model.PlanetEnvironmentTundra, "", []model.PlanetEnvironment{model.PlanetEnvironmentSwamp}},
		{"Arid to Terran", model.PlanetEnvironmentArid, "", []model.PlanetEnvironment{model.PlanetEnvironmentTerran}},
		{"Swamp to Terran", model.PlanetEnvironmentSwamp, "", []model.PlanetEnvironment{model.PlanetEnvironmentTerran}},
		{"Ocean to Terran", model.PlanetEnvironmentOcean, "", []model.PlanetEnvironment{model.PlanetEnvironmentTerran}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, planet := newTerraformingGame(t, tt.from)
			if tt.prereq != "" {
				planet.Buildings = append(planet.Buildings, tt.prereq)
			}

			require.True(t, canTerraformPlanet(planet))
			terraformPlanet(planet)
			assert.Contains(t, tt.expected, planet.Environment)
			assert.Equal(t, int64(1), planet.TerraformingCount)
		})
	}
}

func TestTerraformingPrerequisites(t *testing.T) {
	t.Run("Toxic without Weather Controller", func(t *testing.T) {
		_, planet := newTerraformingGame(t, model.PlanetEnvironmentToxic)
		assert.False(t, canTerraformPlanet(planet))
	})

	t.Run("Toxic with Weather Controller", func(t *testing.T) {
		_, planet := newTerraformingGame(t, model.PlanetEnvironmentToxic)
		planet.Buildings = append(planet.Buildings, "Weather Controller")
		assert.True(t, canTerraformPlanet(planet))
	})

	t.Run("Radiated without Radiation Shield", func(t *testing.T) {
		_, planet := newTerraformingGame(t, model.PlanetEnvironmentRadiated)
		assert.False(t, canTerraformPlanet(planet))
	})

	t.Run("Radiated with Radiation Shield", func(t *testing.T) {
		_, planet := newTerraformingGame(t, model.PlanetEnvironmentRadiated)
		planet.Buildings = append(planet.Buildings, "Planetary Radiation Shield")
		assert.True(t, canTerraformPlanet(planet))
	})
}

func TestTerraformingCostScaling(t *testing.T) {
	_, planet := newTerraformingGame(t, model.PlanetEnvironmentDesert)

	assert.Equal(t, int64(250), effectiveTerraformingCost(planet))

	planet.TerraformingCount = 1
	assert.Equal(t, int64(500), effectiveTerraformingCost(planet))

	planet.TerraformingCount = 2
	assert.Equal(t, int64(750), effectiveTerraformingCost(planet))
}

func TestTerraformingCountIncrements(t *testing.T) {
	_, planet := newTerraformingGame(t, model.PlanetEnvironmentDesert)

	assert.Equal(t, int64(0), planet.TerraformingCount)

	terraformPlanet(planet)
	assert.Equal(t, int64(1), planet.TerraformingCount)
	assert.Equal(t, model.PlanetEnvironmentArid, planet.Environment)

	terraformPlanet(planet)
	assert.Equal(t, int64(2), planet.TerraformingCount)
	assert.Equal(t, model.PlanetEnvironmentTerran, planet.Environment)

	assert.False(t, canTerraformPlanet(planet))
}

func TestTerraformingPreservesBuildings(t *testing.T) {
	_, planet := newTerraformingGame(t, model.PlanetEnvironmentToxic)
	planet.Buildings = append(planet.Buildings, "Weather Controller", "Hydroponic Farm")

	terraformPlanet(planet)

	assert.Equal(t, model.PlanetEnvironmentBarren, planet.Environment)
	assert.Contains(t, planet.Buildings, "Weather Controller")
	assert.Contains(t, planet.Buildings, "Hydroponic Farm")
}

func TestTerraformingInvalidEnvironments(t *testing.T) {
	invalidEnvs := []model.PlanetEnvironment{
		model.PlanetEnvironmentGasGiant,
		model.PlanetEnvironmentAsteroidBelt,
		model.PlanetEnvironmentTerran,
		model.PlanetEnvironmentGaia,
	}

	for _, env := range invalidEnvs {
		t.Run(env.String(), func(t *testing.T) {
			_, planet := newTerraformingGame(t, env)
			assert.False(t, canTerraformPlanet(planet))
		})
	}
}

func TestTerraformingBuildIntegration(t *testing.T) {
	g, planet := newTerraformingGame(t, model.PlanetEnvironmentDesert)
	g.Players[0].Available.Buildings = append(g.Players[0].Available.Buildings, terraformingProject)

	require.True(t, Build(planet, terraformingProject, g.Players[0].Available.Buildings, g))
	assert.True(t, planet.IsBuilding)
	assert.Equal(t, terraformingProject, planet.CurrentProject)
	assert.Equal(t, int64(250), planet.BuildBucket)

	planet.BuildBucket = 1
	require.NoError(t, g.Turn(1, model.Commands{}))

	assert.Equal(t, model.PlanetEnvironmentArid, planet.Environment)
	assert.Equal(t, int64(1), planet.TerraformingCount)
	assert.NotContains(t, planet.Buildings, terraformingProject)
}

func TestTerraformingUnownedPlanet(t *testing.T) {
	_, planet := newTerraformingGame(t, model.PlanetEnvironmentDesert)
	planet.Owner = 0
	assert.False(t, canTerraformPlanet(planet))
}

func newTerraformingGame(t *testing.T, environment model.PlanetEnvironment) (*Game, *model.Planet) {
	t.Helper()
	resources, err := NewResourceManager()
	require.NoError(t, err)
	g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: DefaultRace("Human")})
	require.NoError(t, err)
	planet := g.Galaxy.Stars[0].Planets[0]
	planet.Environment = environment
	planet.Population = 10000
	planet.Industry.Workers = 10
	planet.CurrentProject = "Trade Goods"
	planet.IsBuilding = false
	planet.BuildBucket = 0
	return g, planet
}
