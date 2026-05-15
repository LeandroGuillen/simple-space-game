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
	assert.Equal(t, int64(600), g.Width)
	assert.Equal(t, int64(400), g.Height)
}

func TestNewGalaxyForSizeSetsDimensions(t *testing.T) {
	tests := []struct {
		name       string
		galaxySize string
		wantWidth  int64
		wantHeight int64
	}{
		{name: "tiny", galaxySize: "tiny", wantWidth: 240, wantHeight: 180},
		{name: "medium", galaxySize: "medium", wantWidth: 600, wantHeight: 400},
		{name: "huge", galaxySize: "huge", wantWidth: 1000, wantHeight: 800},
		{name: "unknown", galaxySize: "massive", wantWidth: 600, wantHeight: 400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := game.NewGalaxyForSize(1, tt.galaxySize)
			assert.Equal(t, tt.wantWidth, g.Width)
			assert.Equal(t, tt.wantHeight, g.Height)
		})
	}
}

func TestNewGalaxyMinSeparationOneParsec(t *testing.T) {
	game.SetRandomSeed(4242)
	g := game.NewGalaxy(50)
	assert.Len(t, g.Stars, 50)
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

func TestNewHugeGalaxyPlacesStarsWithinBoundsAndMinSeparation(t *testing.T) {
	game.SetRandomSeed(4242)
	g := game.NewGalaxyForSize(50, "huge")
	assert.Len(t, g.Stars, 50)
	assert.Equal(t, int64(1000), g.Width)
	assert.Equal(t, int64(800), g.Height)

	minDist := float64(60)
	for i := 0; i < len(g.Stars); i++ {
		assert.GreaterOrEqual(t, g.Stars[i].X, int64(0))
		assert.Less(t, g.Stars[i].X, g.Width)
		assert.GreaterOrEqual(t, g.Stars[i].Y, int64(0))
		assert.Less(t, g.Stars[i].Y, g.Height)

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

	// Alcor (610,235) and Kitalpha (579,278): actual distance is ~53 pixels.
	parsecDx := float64(610 - 579)
	parsecDy := float64(235 - 278)
	parsecDist := math.Sqrt(parsecDx*parsecDx + parsecDy*parsecDy)
	assert.GreaterOrEqual(t, parsecDist, 53.0)
}

func TestNewEmptyGalaxy(t *testing.T) {
	galaxy := game.NewEmptyGalaxy()
	assert.Equal(t, len(galaxy.Stars), 0, "galaxy should be empty")
	assert.Equal(t, int64(600), galaxy.Width)
	assert.Equal(t, int64(400), galaxy.Height)
}
