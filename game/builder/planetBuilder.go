package builder

import (
	. "corion/model"
	"math/rand"
)

func newRandomPlanet() *Planet {
	var mineralClass = PlanetMineral(rand.Intn(int(PlanetMineralMax)-1) + 1)
	var size = PlanetSize(rand.Intn(int(PlanetSizeMax)-1) + 1)
	var gravity = PlanetGravity(rand.Intn(int(PlanetGravityMax)-1) + 1)

	return newPlanet(mineralClass, size, gravity)
}

func newPlanet(mineralClass PlanetMineral, size PlanetSize, gravity PlanetGravity) *Planet {
	planet := &Planet{
		MineralClass: mineralClass,
		Size:         size,
		Gravity:      gravity,
		MaxPop:       int64(size * 2),
	}

	return planet
}
