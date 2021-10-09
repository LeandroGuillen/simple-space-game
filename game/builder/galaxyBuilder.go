package builder

import (
	"fmt"
	"math/rand"
	"ssg/data"
	. "ssg/model"
)

func NewGalaxy(numStars uint) Galaxy {
	var stars []*Star
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

		star, numPlanets := newRandomStar(i+1, planetIDCounter, randomName)
		star.ID = i + 1
		star.Name = randomName
		planetIDCounter += numPlanets

		// assign random position on map
		star.X = rand.Int63() % galaxyWidth
		star.Y = rand.Int63() % galaxyHeight

		stars = append(stars, star)
	}

	galaxy := Galaxy{
		Stars: stars,
	}

	return galaxy
}

func newRandomStar(parentStarId, planetIDCounter uint, parentStarName string) (*Star, uint) {
	var numPlanets = uint(rand.Intn(5) + 1)
	var planets []*Planet
	for i := uint(0); i < numPlanets; i++ {
		planet := newRandomPlanet()
		planet.ID = planetIDCounter
		planetIDCounter++
		planet.ParentStar = parentStarId
		planet.Name = fmt.Sprintf("%s %s", parentStarName, romanNumeral(i+1))

		planets = append(planets, planet)
	}

	var color = StarColor(rand.Intn(int(StarColorMax)))

	star := &Star{
		Color:   color,
		Planets: planets,
	}

	return star, numPlanets
}

func romanNumeral(n uint) string {
	switch n {
	case 0:
		return ""
	case 1:
		return "I"
	case 2:
		return "II"
	case 3:
		return "III"
	case 4:
		return "IV"
	case 5:
		return "V"
	case 6:
		return "VI"
	case 7:
		return "VII"
	case 8:
		return "VIII"
	case 9:
		return "IX"
	case 10:
		return "X"
	default:
		return "?"
	}
}
