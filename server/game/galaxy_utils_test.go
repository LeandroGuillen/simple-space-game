package game_test

import (
	"fmt"
	"math"
	"ssg/game"
	"ssg/model"
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
	if stats.CollapsedPlacement {
		t.Fatalf("CRITICAL 1px path must be unreachable (size=%s n=%d seed=%d)", galaxySize, numStars, seed)
	}

	seen := make(map[[2]int64]struct{}, len(g.Stars))
	minDist := math.MaxFloat64
	atOrigin := 0
	for i, star := range g.Stars {
		if star == nil {
			t.Fatalf("nil star at %d", i)
		}
		if star.X < 0 || star.X >= g.Width || star.Y < 0 || star.Y >= g.Height {
			t.Fatalf("star %d out of bounds (%d,%d) size=%s", star.ID, star.X, star.Y, galaxySize)
		}
		key := [2]int64{star.X, star.Y}
		if _, ok := seen[key]; ok {
			t.Fatalf("duplicate position (%d,%d) size=%s n=%d seed=%d", star.X, star.Y, galaxySize, numStars, seed)
		}
		seen[key] = struct{}{}
		if star.X == 0 && star.Y == 0 {
			atOrigin++
		}
		for j := i + 1; j < len(g.Stars); j++ {
			other := g.Stars[j]
			dx := float64(star.X - other.X)
			dy := float64(star.Y - other.Y)
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < minDist {
				minDist = dist
			}
		}
	}
	if numStars > 1 && minDist < stats.MinSeparation-1e-9 {
		t.Fatalf("size=%s seed=%d n=%d minDist %.3f < sep %.3f", galaxySize, seed, numStars, minDist, stats.MinSeparation)
	}
	if atOrigin > 1 {
		t.Fatalf("stacked origin stars: %d", atOrigin)
	}
	return stats
}

func TestTinyGalaxyPlacesFifteenDistinctStars(t *testing.T) {
	// Compiles against older APIs; fails on pre-lattice code by assertion (stack at 0,0).
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
					"client count %q/%d must not exhaust random (seed %d)", tc.size, tc.stars, seed)
				assert.False(t, stats.RelayoutFromRandom)
				assert.False(t, stats.CollapsedPlacement)
				if tc.size == "tiny" {
					assert.True(t, stats.UsedLattice, "tiny 15 uses a jittered lattice")
					assert.Less(t, stats.MinSeparation, 60.0)
				} else {
					assert.False(t, stats.UsedLattice, "medium/huge client counts keep random@60")
					assert.InDelta(t, 60.0, stats.MinSeparation, 1e-9)
				}
			})
		}
	}
}

func TestTinyGalaxyKeepsMediumHugeFullParsecSeparation(t *testing.T) {
	_, tinyStats := game.NewGalaxyForSizeWithStats(game.ClientGalaxyStarCounts["tiny"], "tiny")
	_, medStats := game.NewGalaxyForSizeWithStats(game.ClientGalaxyStarCounts["medium"], "medium")
	_, hugeStats := game.NewGalaxyForSizeWithStats(game.ClientGalaxyStarCounts["huge"], "huge")
	assert.InDelta(t, 60.0, medStats.MinSeparation, 1e-9)
	assert.InDelta(t, 60.0, hugeStats.MinSeparation, 1e-9)
	assert.Less(t, tinyStats.MinSeparation, 60.0, "tiny must scale spacing to fit 15 stars")
	assert.Equal(t, 47.0, tinyStats.MinSeparation, "tiny uses scaled 47px spacing")
	assert.True(t, tinyStats.UsedLattice)
	assert.False(t, medStats.UsedLattice)
	assert.False(t, hugeStats.UsedLattice)
}

func TestTinyGalaxyGenerationUsesJitteredLattice(t *testing.T) {
	game.SetRandomSeed(12345)
	g, stats := game.NewGalaxyForSizeWithStats(game.ClientGalaxyStarCounts["tiny"], "tiny")
	require.Len(t, g.Stars, 15)
	assert.False(t, stats.ExhaustedRandom)
	assert.True(t, stats.UsedLattice)
	assert.False(t, stats.CollapsedPlacement)
}

func TestNearCapacityCountsUseLatticeNotCollapsed(t *testing.T) {
	// Previously collapsed to ~1px: tiny 11–12, medium 53–70, huge 170–222.
	// Some of these now succeed at random@60 (early-lattice threshold raised);
	// all must stay distinct, in bounds, and at >= 1 parsec when capacity allows.
	cases := []struct {
		size  string
		count uint
	}{
		{"tiny", 11}, {"tiny", 12},
		{"medium", 53}, {"medium", 60}, {"medium", 70},
		{"huge", 170}, {"huge", 200}, {"huge", 222},
	}
	for _, tc := range cases {
		t.Run(tc.size+"/"+uitoa(uint64(tc.count)), func(t *testing.T) {
			stats := assertValidStarLayout(t, tc.size, tc.count, 1)
			assert.False(t, stats.CollapsedPlacement)
			assert.GreaterOrEqual(t, stats.MinSeparation, 60.0-1e-9,
				"sep must stay >= 1 parsec when 60px grid fits (got %.2f for %s/%d)",
				stats.MinSeparation, tc.size, tc.count)
		})
	}
}

func TestGalaxyPlacementSweepAllSizesAndCounts(t *testing.T) {
	// Three seeds; covers spacing, map coverage, and home spacing for 2–6 players.
	seeds := []int64{1, 42, 99}
	sizes := []struct {
		name     string
		maxStars uint
	}{
		{"tiny", 100},
		{"medium", 100},
		{"huge", 250},
		{"", 100}, // default → medium dims
	}

	start := time.Now()
	for _, sz := range sizes {
		w, h := galaxyDims(sz.name)
		cap60 := latticeCap(w, h, 60)
		for n := uint(5); n <= sz.maxStars; n++ {
			for _, seed := range seeds {
				game.SetRandomSeed(seed)
				g, stats := game.NewGalaxyForSizeWithStats(n, sz.name)
				assertValidGalaxy(t, g, stats, sz.name, n, seed)

				if stats.CollapsedPlacement {
					t.Fatalf("size=%s n=%d seed=%d collapsed", sz.name, n, seed)
				}
				if stats.MinSeparation < 8.0 {
					t.Fatalf("size=%s n=%d seed=%d sep too small: %.2f", sz.name, n, seed, stats.MinSeparation)
				}
				if cap60 >= n && stats.MinSeparation < 60.0-1e-9 {
					t.Fatalf("size=%s n=%d should keep >= 1 parsec sep, got %.2f", sz.name, n, stats.MinSeparation)
				}
				assertMapCoverage(t, g, n, sz.name, seed)
				if n >= 6 && cap60 >= 6 {
					for players := uint(2); players <= 6 && players <= n; players++ {
						assertHomeSpacing(t, g, players, sz.name, n, seed)
					}
				}
			}
		}
	}
	elapsed := time.Since(start)
	t.Logf("placement sweep completed in %s", elapsed)
}

// assertValidGalaxy mirrors assertValidStarLayout without rebuilding the galaxy
// (sweep already constructed it under a fixed seed).
func assertValidGalaxy(t *testing.T, g model.Galaxy, stats game.GalaxyPlacementStats, galaxySize string, numStars uint, seed int64) {
	t.Helper()
	require.Len(t, g.Stars, int(numStars))
	require.Greater(t, stats.MinSeparation, 0.0)
	if stats.CollapsedPlacement {
		t.Fatalf("CRITICAL 1px path must be unreachable (size=%s n=%d seed=%d)", galaxySize, numStars, seed)
	}
	seen := make(map[[2]int64]struct{}, len(g.Stars))
	minDist := math.MaxFloat64
	atOrigin := 0
	for i, star := range g.Stars {
		if star == nil {
			t.Fatalf("nil star at %d", i)
		}
		if star.X < 0 || star.X >= g.Width || star.Y < 0 || star.Y >= g.Height {
			t.Fatalf("star %d out of bounds (%d,%d) size=%s", star.ID, star.X, star.Y, galaxySize)
		}
		key := [2]int64{star.X, star.Y}
		if _, ok := seen[key]; ok {
			t.Fatalf("duplicate position (%d,%d) size=%s n=%d seed=%d", star.X, star.Y, galaxySize, numStars, seed)
		}
		seen[key] = struct{}{}
		if star.X == 0 && star.Y == 0 {
			atOrigin++
		}
		for j := i + 1; j < len(g.Stars); j++ {
			other := g.Stars[j]
			dx := float64(star.X - other.X)
			dy := float64(star.Y - other.Y)
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < minDist {
				minDist = dist
			}
		}
	}
	if numStars > 1 && minDist < stats.MinSeparation-1e-9 {
		t.Fatalf("size=%s seed=%d n=%d minDist %.3f < sep %.3f", galaxySize, seed, numStars, minDist, stats.MinSeparation)
	}
	if atOrigin > 1 {
		t.Fatalf("stacked origin stars: %d", atOrigin)
	}
}

func assertMapCoverage(t *testing.T, g model.Galaxy, n uint, size string, seed int64) {
	t.Helper()
	// Sparse random clouds can legitimately miss a 3x3 cell; enforce once n is
	// large enough that the old top-60% lattice fill is clearly wrong.
	if n < 20 || g.Width <= 0 || g.Height <= 0 {
		return
	}
	var minX, maxX, minY, maxY int64
	minX, minY = g.Width, g.Height
	for _, s := range g.Stars {
		if s.X < minX {
			minX = s.X
		}
		if s.X > maxX {
			maxX = s.X
		}
		if s.Y < minY {
			minY = s.Y
		}
		if s.Y > maxY {
			maxY = s.Y
		}
	}
	spanX := float64(maxX-minX) / float64(g.Width)
	spanY := float64(maxY-minY) / float64(g.Height)
	// Old top-60% lattice fill spans ~0.60 height; require clearly more.
	if spanX < 0.70 || spanY < 0.70 {
		t.Fatalf("size=%s n=%d seed=%d poor coverage spanX=%.2f spanY=%.2f (want >= 0.70)",
			size, n, seed, spanX, spanY)
	}
	const gridN = 3
	occupied := 0
	cellW := float64(g.Width) / gridN
	cellH := float64(g.Height) / gridN
	seen := [gridN][gridN]bool{}
	for _, s := range g.Stars {
		cx := int(float64(s.X) / cellW)
		cy := int(float64(s.Y) / cellH)
		if cx >= gridN {
			cx = gridN - 1
		}
		if cy >= gridN {
			cy = gridN - 1
		}
		if !seen[cx][cy] {
			seen[cx][cy] = true
			occupied++
		}
	}
	// Fail the old top-two-rows lattice (≤6/9); require most of the map.
	want := 7
	if occupied < want {
		t.Fatalf("size=%s n=%d seed=%d 3x3 occupancy %d < %d", size, n, seed, occupied, want)
	}
}

func assertHomeSpacing(t *testing.T, g model.Galaxy, numPlayers uint, size string, n uint, seed int64) {
	t.Helper()
	cap60 := latticeCap(g.Width, g.Height, 60)
	// hasRoom: a 60px grid on this map can hold numPlayers points.
	if cap60 < numPlayers || n < numPlayers {
		return
	}
	ids := game.SelectHomeStarIDs(g.Stars, numPlayers)
	type xy struct{ x, y int64 }
	pos := make([]xy, 0, numPlayers)
	byID := map[uint]*model.Star{}
	for _, s := range g.Stars {
		byID[s.ID] = s
	}
	for _, id := range ids {
		s := byID[id]
		if s == nil {
			t.Fatalf("missing home star id %d size=%s n=%d", id, size, n)
		}
		pos = append(pos, xy{s.X, s.Y})
	}
	for i := 0; i < len(pos); i++ {
		for j := i + 1; j < len(pos); j++ {
			dx := float64(pos[i].x - pos[j].x)
			dy := float64(pos[i].y - pos[j].y)
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist <= 60.0+1e-9 {
				t.Fatalf("size=%s n=%d seed=%d players=%d homes within 1pc: ids=%v dist=%.2f",
					size, n, seed, numPlayers, ids, dist)
			}
		}
	}
}

func galaxyDims(size string) (int64, int64) {
	switch size {
	case "tiny":
		return 240, 180
	case "huge":
		return 1000, 800
	default:
		return 600, 400
	}
}

func latticeCap(width, height int64, sep float64) uint {
	cols := int64(math.Floor(float64(width-1)/sep)) + 1
	rows := int64(math.Floor(float64(height-1)/sep)) + 1
	return uint(cols * rows)
}

func itoa(n int64) string {
	if n < 0 {
		return "-" + uitoa(uint64(-n))
	}
	return uitoa(uint64(n))
}

func uitoa(n uint64) string {
	if n == 0 {
		return "0"
	}
	var buf [32]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func TestLatticeSpreadMedium50CoversFullMap(t *testing.T) {
	// On c140298 (0.60 early lattice + row-fill) spanY≈0.60 and homes 1/11 sit 60px apart.
	game.SetRandomSeed(1)
	g, stats := game.NewGalaxyForSizeWithStats(50, "medium")
	assertMapCoverage(t, g, 50, "medium", 1)
	assertHomeSpacing(t, g, 5, "medium", 50, 1)
	assert.False(t, stats.CollapsedPlacement)
}

func TestSelectHomeStarIDsKeepsHomesApartOnLattice(t *testing.T) {
	game.SetRandomSeed(42)
	g, _ := game.NewGalaxyForSizeWithStats(50, "medium")
	ids := game.SelectHomeStarIDs(g.Stars, 5)
	require.Len(t, ids, 5)
	byID := map[uint]*model.Star{}
	for _, s := range g.Stars {
		byID[s.ID] = s
	}
	for i := 0; i < len(ids); i++ {
		for j := i + 1; j < len(ids); j++ {
			a, b := byID[ids[i]], byID[ids[j]]
			dx := float64(a.X - b.X)
			dy := float64(a.Y - b.Y)
			dist := math.Sqrt(dx*dx + dy*dy)
			assert.Greater(t, dist, 60.0, "homes %d,%d", ids[i], ids[j])
		}
	}
}

func TestSelectHomeStarIDsMatchesStrideAtWebClientCounts(t *testing.T) {
	// Web-client counts on Medium/Huge/default never needed the farthest-point
	// fallback on c140298; SelectHomeStarIDs must equal StartingStarIDs there.
	seeds := []int64{1, 42, 99}
	cases := []struct {
		size  string
		stars uint
	}{
		{"medium", 20}, {"medium", 40},
		{"", 20}, {"", 40},
		{"huge", 20}, {"huge", 40}, {"huge", 60}, {"huge", 100},
	}
	for _, tc := range cases {
		for _, seed := range seeds {
			for players := uint(2); players <= 6; players++ {
				game.SetRandomSeed(seed)
				g := game.NewGalaxyForSize(tc.stars, tc.size)
				got := game.SelectHomeStarIDs(g.Stars, players)
				want := game.StartingStarIDs(players, tc.stars)
				require.Equal(t, want, got,
					"size=%q n=%d seed=%d players=%d: SelectHome must match stride when already fine",
					tc.size, tc.stars, seed, players)
			}
		}
	}
}

func TestSelectHomeStarIDsFallsBackOnlyWhenStrideTooClose(t *testing.T) {
	// Stride for n=3, 2 players is [1,3]; place those 60px apart and star 2 far
	// away so farthest-point can fix it.
	stars := []*model.Star{
		{ID: 1, X: 0, Y: 0},
		{ID: 2, X: 300, Y: 200},
		{ID: 3, X: 60, Y: 0},
	}
	stride := game.StartingStarIDs(2, 3)
	require.Equal(t, []uint{1, 3}, stride)
	got := game.SelectHomeStarIDs(stars, 2)
	require.NotEqual(t, stride, got, "must fall back when stride homes are within 1pc")
	require.Equal(t, []uint{1, 2}, got)
}

func TestTinyGalaxyLayoutAndStartingLocationVaryBySeed(t *testing.T) {
	layouts := map[string]bool{}
	homes := map[[2]int64]bool{}
	for seed := int64(1); seed <= 40; seed++ {
		assertValidStarLayout(t, "tiny", 15, seed)
		game.SetRandomSeed(seed)
		galaxy, stats := game.NewGalaxyForSizeWithStats(15, "tiny")
		require.Equal(t, 47.0, stats.MinSeparation)
		var layout string
		for _, star := range galaxy.Stars {
			layout += fmt.Sprintf("%d,%d;", star.X, star.Y)
		}
		layouts[layout] = true
		homeID := game.SelectHomeStarIDs(galaxy.Stars, 4)[0]
		for _, star := range galaxy.Stars {
			if star.ID == homeID {
				homes[[2]int64{star.X, star.Y}] = true
			}
		}
	}
	require.Greater(t, len(layouts), 1)
	require.Greater(t, len(homes), 1)
}

// Benchmarks track generation speed without enforcing wall-clock deadlines on
// shared CI hosts or race-instrumented builds.
func BenchmarkGalaxyGeneration(b *testing.B) {
	for _, tc := range []struct {
		name  string
		stars uint
	}{
		{"tiny", 15}, {"medium", 40}, {"huge", 100},
	} {
		b.Run(tc.name, func(b *testing.B) {
			game.SetRandomSeed(42)
			for i := 0; i < b.N; i++ {
				game.NewGalaxyForSize(tc.stars, tc.name)
			}
		})
	}
}
