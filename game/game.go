package game

import (
	"corion/game/builder"
	. "corion/model"
	"math"
	"math/rand"
)

type Game struct{}

func (g *Game) NewGalaxy(numStars int) Galaxy {
	galaxy := builder.NewGalaxy(numStars)
	return galaxy
}

func (g *Game) NewRandomGalaxy() Galaxy {
	// new galaxy parameters
	var stdDev float64 = 2
	var mean float64 = 6
	n := int(math.Abs(math.Floor(rand.NormFloat64()*stdDev + mean)))
	galaxy := builder.NewGalaxy(n)
	return galaxy
}
