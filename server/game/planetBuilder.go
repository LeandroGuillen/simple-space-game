package game

import (
	"ssg/model"
)

func NewPlanet(
	mineralClass model.PlanetMineral,
	size model.PlanetSize,
	gravity model.PlanetGravity,
	env model.PlanetEnvironment,
) *model.Planet {
	planet := &model.Planet{
		MineralClass: mineralClass,
		Size:         size,
		Gravity:      gravity,
		Environment:  env,
	}

	return planet
}

func NewRandomPlanet() *model.Planet {
	var mineralClass = model.PlanetMineral(randomIntn(int(model.PlanetMineralMax)-1) + 1)
	var size = model.PlanetSize(randomIntn(int(model.PlanetSizeMax)-1) + 1)
	var gravity = model.PlanetGravity(randomIntn(int(model.PlanetGravityMax)-1) + 1)
	var environment = model.PlanetEnvironment(randomIntn(int(model.PlanetEnvironmentMax)-1) + 1)

	return NewPlanet(mineralClass, size, gravity, environment)
}

func NewAverageTerranPlanet() *model.Planet {
	return NewPlanet(
		model.PlanetMineralAbundant,
		model.PlanetSizeMedium,
		model.PlanetGravityNormal,
		model.PlanetEnvironmentTerran,
	)
}
