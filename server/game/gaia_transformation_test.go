package game

import (
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGaiaTransformationOnlyOnTerran(t *testing.T) {
	nonTerranEnvs := []model.PlanetEnvironment{
		model.PlanetEnvironmentToxic,
		model.PlanetEnvironmentRadiated,
		model.PlanetEnvironmentBarren,
		model.PlanetEnvironmentDesert,
		model.PlanetEnvironmentArid,
		model.PlanetEnvironmentTundra,
		model.PlanetEnvironmentSwamp,
		model.PlanetEnvironmentOcean,
		model.PlanetEnvironmentGaia,
		model.PlanetEnvironmentGasGiant,
		model.PlanetEnvironmentAsteroidBelt,
	}

	for _, env := range nonTerranEnvs {
		t.Run(env.String(), func(t *testing.T) {
			_, planet := newTerraformingGame(t, env)
			assert.False(t, canGaiaTransformPlanet(planet))
		})
	}

	t.Run("Terran", func(t *testing.T) {
		_, planet := newTerraformingGame(t, model.PlanetEnvironmentTerran)
		assert.True(t, canGaiaTransformPlanet(planet))
	})
}

func TestGaiaTransformationUnownedPlanet(t *testing.T) {
	_, planet := newTerraformingGame(t, model.PlanetEnvironmentTerran)
	planet.Owner = 0
	assert.False(t, canGaiaTransformPlanet(planet))
}

func TestGaiaTransformPlanet(t *testing.T) {
	_, planet := newTerraformingGame(t, model.PlanetEnvironmentTerran)

	gaiaTransformPlanet(planet)
	assert.Equal(t, model.PlanetEnvironmentGaia, planet.Environment)
}

func TestGaiaTransformPlanetIgnoresNonTerran(t *testing.T) {
	_, planet := newTerraformingGame(t, model.PlanetEnvironmentArid)

	gaiaTransformPlanet(planet)
	assert.Equal(t, model.PlanetEnvironmentArid, planet.Environment)
}

func TestGaiaTransformationPreservesBuildings(t *testing.T) {
	_, planet := newTerraformingGame(t, model.PlanetEnvironmentTerran)
	planet.Buildings = append(planet.Buildings, "Hydroponic Farm")

	gaiaTransformPlanet(planet)

	assert.Equal(t, model.PlanetEnvironmentGaia, planet.Environment)
	assert.Contains(t, planet.Buildings, "Hydroponic Farm")
}

func TestGaiaTransformationBuildIntegration(t *testing.T) {
	g, planet := newTerraformingGame(t, model.PlanetEnvironmentTerran)
	g.Players[0].Available.Buildings = append(g.Players[0].Available.Buildings, gaiaTransformationProject)

	require.True(t, Build(planet, gaiaTransformationProject, g.Players[0].Available.Buildings, g))
	assert.True(t, planet.IsBuilding)
	assert.Equal(t, gaiaTransformationProject, planet.CurrentProject)

	planet.BuildBucket = 1
	require.NoError(t, g.Turn(1, model.Commands{}))

	assert.Equal(t, model.PlanetEnvironmentGaia, planet.Environment)
	assert.NotContains(t, planet.Buildings, gaiaTransformationProject)
}

func TestGaiaTransformationBuildRejectedOnNonTerran(t *testing.T) {
	g, planet := newTerraformingGame(t, model.PlanetEnvironmentArid)
	g.Players[0].Available.Buildings = append(g.Players[0].Available.Buildings, gaiaTransformationProject)

	assert.False(t, Build(planet, gaiaTransformationProject, g.Players[0].Available.Buildings, g))
	assert.False(t, planet.IsBuilding)
}

func TestGaiaTransformationNotApplicableAfterTransform(t *testing.T) {
	g, planet := newTerraformingGame(t, model.PlanetEnvironmentTerran)

	require.True(t, buildingApplicableToPlanet(planet, gaiaTransformationProject, g))
	gaiaTransformPlanet(planet)
	assert.False(t, buildingApplicableToPlanet(planet, gaiaTransformationProject, g))
}
