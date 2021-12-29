package builder

import (
	"fmt"
	"math"
	"math/rand"
	"ssg/data"
	"ssg/game/model"
)

func NewGalaxy(numStars uint) model.Galaxy {
	var stars []*model.Star
	var galaxyWidth int64 = 700
	var galaxyHeight int64 = 400
	var starNames = make(map[string]int)
	var planetIDCounter uint = 1

	for i := uint(0); i < numStars; i++ {
		// assign random name, adding numbers if necessary
		randomName := data.GetRandomStarName()
		repetitions, nameExists := starNames[randomName]
		if nameExists {
			starNames[randomName] = repetitions + 1
			randomName = fmt.Sprintf("%s %d", randomName, repetitions+1)
		} else {
			starNames[randomName] = 1
		}

		star, numPlanets := NewRandomStar(i+1, planetIDCounter, randomName)
		planetIDCounter += numPlanets

		// assign random position on map
		star.X = rand.Int63() % galaxyWidth
		star.Y = rand.Int63() % galaxyHeight

		stars = append(stars, star)
	}

	galaxy := model.Galaxy{
		Stars: stars,
	}

	return galaxy
}

func NewEmptyGalaxy() model.Galaxy {
	galaxy := model.Galaxy{
		Stars: []*model.Star{},
	}

	return galaxy
}

func NewRandomGalaxy() model.Galaxy {
	// new galaxy parameters
	var stdDev float64 = 2
	var mean float64 = 6
	n := uint(math.Abs(math.Floor(rand.NormFloat64()*stdDev + mean)))
	galaxy := NewGalaxy(n)
	return galaxy
}
