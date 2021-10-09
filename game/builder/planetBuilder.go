package builder

import (
	"math/rand"
	. "ssg/model"
)

func newRandomPlanet() *Planet {
	var mineralClass = PlanetMineral(rand.Intn(int(PlanetMineralMax)-1) + 1)
	var size = PlanetSize(rand.Intn(int(PlanetSizeMax)-1) + 1)
	var gravity = PlanetGravity(rand.Intn(int(PlanetGravityMax)-1) + 1)
	var environment = PlanetEnvironment(rand.Intn(int(PlanetEnvironmentMax)-1) + 1)

	return newPlanet(mineralClass, size, gravity, environment)
}

func newPlanet(mineralClass PlanetMineral, size PlanetSize, gravity PlanetGravity, env PlanetEnvironment) *Planet {
	planet := &Planet{
		MineralClass: mineralClass,
		Size:         size,
		Gravity:      gravity,
		Environment:  env,
		MaxPop:       int64(size * 5),
	}

	return planet
}
