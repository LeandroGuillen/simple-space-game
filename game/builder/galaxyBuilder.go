package builder

import (
	"corion/data"
	. "corion/model"
	"fmt"
	"math/rand"
)

func NewGalaxy(numStars uint) Galaxy {
	var stars []*Star
	var size int64 = 100
	var starNames = make(map[string]int)

	for i := uint(0); i < numStars; i++ {
		star := newRandomStar(i + 1)
		star.ID = i + 1

		// assign random name, adding numbers if necessary
		randomName := data.GetRandomStarName()
		repetitions, nameExists := starNames[randomName]
		if nameExists {
			starNames[randomName] = repetitions + 1
			randomName = fmt.Sprintf("%s %d", randomName, repetitions+1)
		} else {
			starNames[randomName] = 1
		}
		star.Name = randomName

		// assign random position on map
		star.X = rand.Int63() % size
		star.Y = rand.Int63() % size

		stars = append(stars, star)
	}

	galaxy := Galaxy{
		Stars: stars,
	}

	return galaxy
}

func newRandomStar(parentStarId uint) *Star {
	var numPlanets = uint(rand.Intn(5) + 1)
	var planets []*Planet
	for i := uint(0); i < numPlanets; i++ {
		planet := newRandomPlanet()
		planet.ID = i + 1
		planet.ParentStar = parentStarId

		planets = append(planets, planet)
	}

	var color = StarColor(rand.Intn(int(StarColorMax)))

	star := &Star{
		Color:   color,
		Planets: planets,
	}

	return star
}
