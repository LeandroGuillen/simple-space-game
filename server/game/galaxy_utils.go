package game

import (
	"fmt"
	"log"
	"math"
	"ssg/model"
)

const (
	tinyGalaxyWidth    int64 = 240
	tinyGalaxyHeight   int64 = 180
	mediumGalaxyWidth  int64 = 600
	mediumGalaxyHeight int64 = 400
	hugeGalaxyWidth    int64 = 1000
	hugeGalaxyHeight   int64 = 800
	// 1 parsec in map pixels, calibrated from visual spacing examples.
	parsecInPixels = 60
	// Preferred minimum star separation (1 parsec). Scaled down only when a
	// built-in map cannot fit the requested star count at this distance.
	desiredMinStarSeparation = float64(parsecInPixels)
	// Random packing is much less dense than a lattice. Above this fraction of
	// gridCapacity(desiredSep), skip random and place on the lattice up front.
	// Chosen so web client counts (medium 40, huge 100) stay on random@60px,
	// while the previously-broken band (tiny 9–12, medium 53–70, huge 170+)
	// goes straight to lattice.
	randomSafeCapacityFraction = 0.60
	// Cap random attempts per star; on failure the whole galaxy is re-laid on
	// a lattice (never the old 1px CRITICAL path).
	maxRandomStarPlacementTries = 8_000
)

// ClientGalaxyStarCounts are the star counts the web client sends for each
// built-in galaxy size (client-solid SetupPage galaxySizeStars).
var ClientGalaxyStarCounts = map[string]uint{
	"tiny":   15,
	"medium": 40,
	"huge":   100,
}

// GalaxyPlacementStats reports how star positions were chosen.
type GalaxyPlacementStats struct {
	MinSeparation float64
	// UsedLattice is true when every star was placed on the packing lattice
	// (either chosen up front or after random placement failed).
	UsedLattice bool
	// RelayoutFromRandom is true when random placement failed mid-galaxy and
	// all stars were re-placed on the lattice.
	RelayoutFromRandom bool
	// ExhaustedRandom is true when random tries were exhausted for at least
	// one star (same situations as RelayoutFromRandom). Never indicates a
	// collapsed 1px placement — that path is gone.
	ExhaustedRandom bool
	// CollapsedPlacement is always false after generation; retained so tests
	// can assert the old CRITICAL 1px path is unreachable.
	CollapsedPlacement bool
}

func galaxyDimensionsForSize(size string) (int64, int64) {
	switch size {
	case "tiny":
		return tinyGalaxyWidth, tinyGalaxyHeight
	case "medium":
		return mediumGalaxyWidth, mediumGalaxyHeight
	case "huge":
		return hugeGalaxyWidth, hugeGalaxyHeight
	default:
		return mediumGalaxyWidth, mediumGalaxyHeight
	}
}

func starDistanceSq(x1, y1, x2, y2 int64) float64 {
	dx := float64(x1 - x2)
	dy := float64(y1 - y2)
	return dx*dx + dy*dy
}

func tooCloseToExisting(x, y int64, stars []*model.Star, minSepSq float64) bool {
	for _, s := range stars {
		if starDistanceSq(x, y, s.X, s.Y) < minSepSq {
			return true
		}
	}
	return false
}

// gridCapacity estimates how many points fit on a rectangular lattice with
// the given minimum separation inside [0,width) x [0,height).
func gridCapacity(width, height int64, sep float64) uint {
	if width <= 0 || height <= 0 {
		return 0
	}
	if sep < 1 {
		sep = 1
	}
	cols := int64(math.Floor(float64(width-1)/sep)) + 1
	rows := int64(math.Floor(float64(height-1)/sep)) + 1
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	return uint(cols * rows)
}

// packingLattice chooses cols×rows covering n points with aspect close to the
// map, returning integer cell steps and the minimum axis spacing those steps
// realize (so pairwise distances on the lattice are >= sep).
func packingLattice(width, height int64, n uint) (cols, rows int, stepX, stepY int64, sep float64) {
	if n <= 1 {
		return 1, 1, 0, 0, desiredMinStarSeparation
	}
	cols = int(math.Ceil(math.Sqrt(float64(n) * float64(width) / float64(height))))
	if cols < 1 {
		cols = 1
	}
	rows = int(math.Ceil(float64(n) / float64(cols)))
	for cols*rows < int(n) {
		cols++
		rows = int(math.Ceil(float64(n) / float64(cols)))
	}
	stepX, stepY = 1, 1
	if cols > 1 {
		stepX = (width - 1) / int64(cols-1) // floor — keeps last column in bounds
	}
	if rows > 1 {
		stepY = (height - 1) / int64(rows-1)
	}
	if stepX < 1 {
		stepX = 1
	}
	if stepY < 1 {
		stepY = 1
	}
	sep = float64(stepX)
	if float64(stepY) < sep {
		sep = float64(stepY)
	}
	return cols, rows, stepX, stepY, sep
}

// minSeparationPixels returns the placement separation for a galaxy of the
// given size and star count. Prefers 1 parsec (60px); if that cannot pack the
// requested count, uses the largest lattice spacing that fits.
func minSeparationPixels(width, height int64, numStars uint) float64 {
	desired := desiredMinStarSeparation
	if numStars <= 1 {
		return desired
	}
	if gridCapacity(width, height, desired) >= numStars {
		return desired
	}
	_, _, _, _, sep := packingLattice(width, height, numStars)
	return sep
}

// preferLattice reports whether random@60px is unsafe for this count: either
// the desired separation cannot pack n stars, or n is above the safe fraction
// of lattice capacity (random packing fails well before the lattice fills).
func preferLattice(width, height int64, numStars uint) bool {
	if numStars <= 1 {
		return false
	}
	cap := gridCapacity(width, height, desiredMinStarSeparation)
	if cap < numStars {
		return true
	}
	safe := uint(float64(cap) * randomSafeCapacityFraction)
	return numStars > safe
}

// tryRandomStarPosition attempts random placement only. Returns true if placed
// with minSep. On exhaustion returns false without modifying the star and
// without the old CRITICAL 1px fallback.
func tryRandomStarPosition(star *model.Star, existing []*model.Star, width, height int64, minSep float64) bool {
	if star == nil || width <= 0 || height <= 0 {
		return false
	}
	w := int(width)
	h := int(height)
	minSepSq := minSep * minSep

	for range maxRandomStarPlacementTries {
		x := int64(randomIntn(w))
		y := int64(randomIntn(h))
		if !tooCloseToExisting(x, y, existing, minSepSq) {
			star.X, star.Y = x, y
			return true
		}
	}
	return false
}

// placeStarsOnLattice assigns stars to a rectangular lattice. When n fits at
// the preferred 60px separation, uses a 60px grid (largest preferred spacing).
// Otherwise uses packingLattice for the largest spacing that still fits.
// Returns the realized minimum separation.
func placeStarsOnLattice(stars []*model.Star, width, height int64) float64 {
	n := uint(len(stars))
	if n == 0 {
		return desiredMinStarSeparation
	}
	if n == 1 {
		stars[0].X = (width - 1) / 2
		stars[0].Y = (height - 1) / 2
		return desiredMinStarSeparation
	}

	var cols, rows int
	var stepX, stepY int64
	var sep float64

	if gridCapacity(width, height, desiredMinStarSeparation) >= n {
		sep = desiredMinStarSeparation
		stepX = int64(sep)
		stepY = int64(sep)
		cols = int(math.Floor(float64(width-1)/sep)) + 1
		rows = int(math.Floor(float64(height-1)/sep)) + 1
	} else {
		cols, rows, stepX, stepY, sep = packingLattice(width, height, n)
	}

	idx := 0
	for r := 0; r < rows && idx < int(n); r++ {
		for c := 0; c < cols && idx < int(n); c++ {
			x := int64(c) * stepX
			y := int64(r) * stepY
			if x >= width {
				x = width - 1
			}
			if y >= height {
				y = height - 1
			}
			stars[idx].X = x
			stars[idx].Y = y
			idx++
		}
	}
	return sep
}

func NewGalaxy(numStars uint) model.Galaxy {
	galaxy, _ := NewGalaxyForSizeWithStats(numStars, "")
	return galaxy
}

func NewGalaxyForSize(numStars uint, galaxySize string) model.Galaxy {
	galaxy, _ := NewGalaxyForSizeWithStats(numStars, galaxySize)
	return galaxy
}

// NewGalaxyForSizeWithStats builds a galaxy and reports placement stats for tests.
func NewGalaxyForSizeWithStats(numStars uint, galaxySize string) (model.Galaxy, GalaxyPlacementStats) {
	width, height := galaxyDimensionsForSize(galaxySize)
	sep := minSeparationPixels(width, height, numStars)
	stats := GalaxyPlacementStats{MinSeparation: sep}

	var stars []*model.Star
	starNames := make(map[string]int)
	var planetIDCounter uint = 1
	useLattice := preferLattice(width, height, numStars)

	createStar := func(i uint) *model.Star {
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
		return star
	}

	if useLattice {
		stats.UsedLattice = true
		for i := range numStars {
			stars = append(stars, createStar(i))
		}
		stats.MinSeparation = placeStarsOnLattice(stars, width, height)
	} else {
		// Interleave create+place so seeded NewGame RNG matches prior builds
		// for counts that still succeed randomly (medium 40, huge 100, …).
		relayout := false
		for i := range numStars {
			star := createStar(i)
			if !relayout {
				if !tryRandomStarPosition(star, stars, width, height, sep) {
					relayout = true
					stats.ExhaustedRandom = true
					stats.RelayoutFromRandom = true
					log.Printf(
						"galaxy: random placement failed at star %d/%d (sep=%.1f); relayout entire galaxy on lattice",
						i+1, numStars, sep,
					)
				}
			}
			stars = append(stars, star)
		}
		if relayout {
			stats.UsedLattice = true
			stats.MinSeparation = placeStarsOnLattice(stars, width, height)
		}
	}

	stats.CollapsedPlacement = false
	return model.Galaxy{
		Stars:  stars,
		Width:  width,
		Height: height,
	}, stats
}

func NewEmptyGalaxy() model.Galaxy {
	galaxy := model.Galaxy{
		Stars:  []*model.Star{},
		Width:  mediumGalaxyWidth,
		Height: mediumGalaxyHeight,
	}

	return galaxy
}

func NewRandomGalaxy() model.Galaxy {
	var stdDev float64 = 2
	var mean float64 = 6
	n := uint(math.Abs(math.Floor(randomNormFloat64()*stdDev + mean)))
	galaxy := NewGalaxy(n)
	return galaxy
}
