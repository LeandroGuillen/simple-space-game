package builder

import (
	. "corion/model"
	"math/rand"
)

func newRandomPlanet() Planet {
	var mineralClass = PlanetMineral(rand.Intn(int(PlanetMineralMax)))
	var size = PlanetSize(rand.Intn(int(PlanetSizeMax)))
	var gravity = PlanetGravity(rand.Intn(int(PlanetGravityMax)))

	return newPlanet(mineralClass, size, gravity)
}

func newPlanet(mineralClass PlanetMineral, size PlanetSize, gravity PlanetGravity) Planet {
	planet := Planet{
		MineralClass: mineralClass,
		Size:         size,
		Gravity:      gravity,
		CurrentPop:   0,
		MaxPop:       0,
		Buildings:    []string{},
	}

	return planet
}
