package game

import (
	"fmt"
	"math"
	"ssg/model"
)

const (
	galaxyWidth  int64 = 700
	galaxyHeight int64 = 400
	// 1 parsec in map pixels, calibrated from visual spacing examples.
	parsecInPixels = 60
	// Reject a candidate if Euclidean distance to an existing star is below 1 parsec.
	minStarSeparationParsecSq   = 1.0 * parsecInPixels * parsecInPixels
	maxRandomStarPlacementTries = 500_000
)

func starDistanceSq(x1, y1, x2, y2 int64) float64 {
	dx := float64(x1 - x2)
	dy := float64(y1 - y2)
	return dx*dx + dy*dy
}

func tooCloseToExisting(x, y int64, stars []*model.Star) bool {
	for _, s := range stars {
		if starDistanceSq(x, y, s.X, s.Y) < minStarSeparationParsecSq {
			return true
		}
	}
	return false
}

// assignGalaxyStarPosition sets X,Y so every star is at least 1 parsec from all previously placed stars.
func assignGalaxyStarPosition(star *model.Star, existing []*model.Star) {
	w := int(galaxyWidth)
	h := int(galaxyHeight)
	for try := 0; try < maxRandomStarPlacementTries; try++ {
		x := int64(randomIntn(w))
		y := int64(randomIntn(h))
		if !tooCloseToExisting(x, y, existing) {
			star.X, star.Y = x, y
			return
		}
	}
	for y := int64(0); y < galaxyHeight; y++ {
		for x := int64(0); x < galaxyWidth; x++ {
			if !tooCloseToExisting(x, y, existing) {
				star.X, star.Y = x, y
				return
			}
		}
	}
	panic(fmt.Sprintf("galaxy: could not place star (already have %d stars)", len(existing)))
}

func NewGalaxy(numStars uint) model.Galaxy {
	var stars []*model.Star
	var starNames = make(map[string]int)
	var planetIDCounter uint = 1

	for i := uint(0); i < numStars; i++ {
		// assign random name, adding numbers if necessary
		randomName := GetRandomStarName()
		repetitions, nameExists := starNames[randomName]
		if nameExists {
			starNames[randomName] = repetitions + 1
			randomName = fmt.Sprintf("%s %d", randomName, repetitions+1)
		} else {
			starNames[randomName] = 1
		}

		star, numPlanets := NewRandomStar(i+1, planetIDCounter, randomName)
		planetIDCounter += numPlanets

		assignGalaxyStarPosition(star, stars)
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
	n := uint(math.Abs(math.Floor(randomNormFloat64()*stdDev + mean)))
	galaxy := NewGalaxy(n)
	return galaxy
}
