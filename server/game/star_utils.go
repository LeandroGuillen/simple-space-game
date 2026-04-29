package game

import (
	"fmt"
	"ssg/model"
)

func NewRandomStar(starId, planetIDCounter uint, starName string) (*model.Star, uint) {
	var numPlanets = uint(randomIntn(5) + 1)
	var planets []*model.Planet
	for i := uint(0); i < numPlanets; i++ {
		planet := NewRandomPlanet()
		planet.ID = planetIDCounter
		planetIDCounter++
		planet.ParentStar = starId
		planet.Name = fmt.Sprintf("%s %s", starName, romanNumeral(i+1))

		planets = append(planets, planet)
	}

	var color = model.StarColor(randomIntn(int(model.StarColorMax)))

	star := NewStar(starId, starName, color, planets)
	return star, numPlanets
}

func NewStar(starId uint, starName string, color model.StarColor, planets []*model.Planet) *model.Star {
	star := &model.Star{
		ID:      starId,
		Name:    starName,
		Color:   color,
		Planets: planets,
	}

	return star
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
