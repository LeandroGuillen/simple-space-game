package game_test

import (
	"math"
	"ssg/game"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewGalaxy(t *testing.T) {
	g := game.NewGalaxy(1)
	assert.Equal(t, len(g.Stars), 1)
}

func TestNewGalaxyMinSeparationOneParsec(t *testing.T) {
	game.SetRandomSeed(4242)
	g := game.NewGalaxy(60)
	assert.Len(t, g.Stars, 60)
	minDist := float64(60)
	for i := 0; i < len(g.Stars); i++ {
		for j := i + 1; j < len(g.Stars); j++ {
			dx := float64(g.Stars[i].X - g.Stars[j].X)
			dy := float64(g.Stars[i].Y - g.Stars[j].Y)
			dist := math.Sqrt(dx*dx + dy*dy)
			assert.GreaterOrEqual(t, dist, minDist-1e-9, "pair %d,%d centers closer than 1 parsec", i, j)
		}
	}
}

func TestCalibratedParsecDistanceExamples(t *testing.T) {
	minDist := float64(60)

	// Alcor (610,235) and Alkalurops 2 (616,217): should be too close.
	tooCloseDx := float64(610 - 616)
	tooCloseDy := float64(235 - 217)
	tooCloseDist := math.Sqrt(tooCloseDx*tooCloseDx + tooCloseDy*tooCloseDy)
	assert.Less(t, tooCloseDist, minDist)

	// Alcor (610,235) and Kitalpha (579,278): target baseline for 1 parsec.
	parsecDx := float64(610 - 579)
	parsecDy := float64(235 - 278)
	parsecDist := math.Sqrt(parsecDx*parsecDx + parsecDy*parsecDy)
	assert.GreaterOrEqual(t, parsecDist, minDist-1e-9)
}

func TestNewEmptyGalaxy(t *testing.T) {
	galaxy := game.NewEmptyGalaxy()
	assert.Equal(t, len(galaxy.Stars), 0, "galaxy should be empty")
}
