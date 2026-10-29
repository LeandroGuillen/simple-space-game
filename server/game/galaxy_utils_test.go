package game_test

import (
	"math"
	"ssg/game"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func assertValidStarLayout(t *testing.T, galaxySize string, numStars uint, seed int64) game.GalaxyPlacementStats {
	t.Helper()
	game.SetRandomSeed(seed)
	g, stats := game.NewGalaxyForSizeWithStats(numStars, galaxySize)
	require.Len(t, g.Stars, int(numStars))
	require.Greater(t, stats.MinSeparation, 0.0)

	seen := make(map[[2]int64]bool, len(g.Stars))
	for i, star := range g.Stars {
		require.NotNil(t, star)
		assert.GreaterOrEqual(t, star.X, int64(0), "star %d X", i)
		assert.Less(t, star.X, g.Width, "star %d X", i)
		assert.GreaterOrEqual(t, star.Y, int64(0), "star %d Y", i)
		assert.Less(t, star.Y, g.Height, "star %d Y", i)
		key := [2]int64{star.X, star.Y}
		assert.False(t, seen[key], "duplicate position (%d,%d) for star %d", star.X, star.Y, star.ID)
		seen[key] = true

		for j := i + 1; j < len(g.Stars); j++ {
			other := g.Stars[j]
			dx := float64(star.X - other.X)
			dy := float64(star.Y - other.Y)
			dist := math.Sqrt(dx*dx + dy*dy)
			assert.GreaterOrEqual(t, dist, stats.MinSeparation-1e-9,
				"size=%s seed=%d pair %d,%d dist %.3f < sep %.3f", galaxySize, seed, star.ID, other.ID, dist, stats.MinSeparation)
		}
	}

	// (0,0) is only allowed if a star was legitimately placed there (unique + sep ok).
	atOrigin := 0
	for _, star := range g.Stars {
		if star.X == 0 && star.Y == 0 {
			atOrigin++
		}
	}
	assert.LessOrEqual(t, atOrigin, 1, "at most one star may sit at the legitimate (0,0) cell")
	return stats
}

func TestTinyGalaxyPlacesFifteenDistinctStars(t *testing.T) {
	// Compiles against e145d6d APIs; fails there by assertion (late stars stack at 0,0).
	game.SetRandomSeed(1)
	g := game.NewGalaxyForSize(15, "tiny")
	require.Len(t, g.Stars, 15)
	seen := make(map[[2]int64]int)
	for _, star := range g.Stars {
		require.NotNil(t, star)
		assert.GreaterOrEqual(t, star.X, int64(0))
		assert.Less(t, star.X, g.Width)
		assert.GreaterOrEqual(t, star.Y, int64(0))
		assert.Less(t, star.Y, g.Height)
		key := [2]int64{star.X, star.Y}
		seen[key]++
	}
	for pos, count := range seen {
		assert.Equal(t, 1, count, "stacked stars at (%d,%d)", pos[0], pos[1])
	}
}

func TestClientGalaxySizesPlaceAllStarsDistinctly(t *testing.T) {
	seeds := []int64{1, 7, 42, 4242, 99991}
	cases := []struct {
		size  string
		stars uint
	}{
		{"tiny", game.ClientGalaxyStarCounts["tiny"]},
		{"medium", game.ClientGalaxyStarCounts["medium"]},
		{"huge", game.ClientGalaxyStarCounts["huge"]},
		{"", game.ClientGalaxyStarCounts["medium"]}, // default fallback → medium dims
	}
	for _, tc := range cases {
		for _, seed := range seeds {
			t.Run(tc.size+"/"+itoa(seed), func(t *testing.T) {
				stats := assertValidStarLayout(t, tc.size, tc.stars, seed)
				assert.False(t, stats.ExhaustedRandom,
					"built-in size %q with %d stars must not hit random-try fallback (seed %d)", tc.size, tc.stars, seed)
				assert.Equal(t, 0, stats.GridFallbacks)
				if tc.size == "tiny" {
					assert.True(t, stats.UsedLattice, "tiny must use lattice packing")
				} else {
					assert.False(t, stats.UsedLattice, "medium/huge keep random placement at full parsec sep")
				}
			})
		}
	}
}

func TestTinyGalaxyKeepsMediumHugeFullParsecSeparation(t *testing.T) {
	// Medium/Huge at client counts keep the full 60px look; Tiny must scale.
	_, tinyStats := game.NewGalaxyForSizeWithStats(game.ClientGalaxyStarCounts["tiny"], "tiny")
	_, medStats := game.NewGalaxyForSizeWithStats(game.ClientGalaxyStarCounts["medium"], "medium")
	_, hugeStats := game.NewGalaxyForSizeWithStats(game.ClientGalaxyStarCounts["huge"], "huge")
	assert.InDelta(t, 60.0, medStats.MinSeparation, 1e-9)
	assert.InDelta(t, 60.0, hugeStats.MinSeparation, 1e-9)
	assert.Less(t, tinyStats.MinSeparation, 60.0, "tiny must scale spacing to fit 15 stars")
	assert.GreaterOrEqual(t, tinyStats.MinSeparation, 30.0, "tiny spacing should stay meaningful")
	assert.True(t, tinyStats.UsedLattice)
	assert.False(t, medStats.UsedLattice)
	assert.False(t, hugeStats.UsedLattice)
}

func TestTinyGalaxyGenerationIsFast(t *testing.T) {
	game.SetRandomSeed(12345)
	start := time.Now()
	g, stats := game.NewGalaxyForSizeWithStats(game.ClientGalaxyStarCounts["tiny"], "tiny")
	elapsed := time.Since(start)
	require.Len(t, g.Stars, 15)
	assert.False(t, stats.ExhaustedRandom)
	assert.True(t, stats.UsedLattice)
	assert.Less(t, elapsed, time.Second, "tiny galaxy generation took %s", elapsed)
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [32]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
