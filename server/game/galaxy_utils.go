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
	// Cap random attempts per star; lattice / grid-jitter handles exhaustion so
	// generation never burns hundreds of thousands of rejects.
	maxRandomStarPlacementTries = 8_000
)

// ClientGalaxyStarCounts are the star counts the web client sends for each
// built-in galaxy size (client-solid SetupPage galaxySizeStars).
var ClientGalaxyStarCounts = map[string]uint{
	"tiny":   15,
	"medium": 40,
	"huge":   100,
}

// GalaxyPlacementStats reports how star positions were chosen. Used by tests
// to assert built-in sizes never exhaust random placement.
type GalaxyPlacementStats struct {
	MinSeparation   float64
	GridFallbacks   int  // stars that fell back after random tries failed
	UsedLattice     bool // entire map placed on a packing lattice (tight pack)
	ExhaustedRandom bool // true if any star needed grid-jitter after random
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
// requested count, uses the lattice spacing that fits. Medium/Huge at client
// star counts keep the full 60px; Tiny (15 into 240×180) is the case that scales.
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

func needsTightLattice(width, height int64, numStars uint) bool {
	if numStars <= 1 {
		return false
	}
	return gridCapacity(width, height, desiredMinStarSeparation) < numStars
}

// assignGalaxyStarPosition sets X,Y at least minSep from all previously placed
// stars via random search. On exhaustion uses deterministic grid-jitter (never
// silently leaves the star at 0,0). Returns whether the fallback was required.
func assignGalaxyStarPosition(star *model.Star, existing []*model.Star, width, height int64, minSep float64) bool {
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
			return false
		}
	}

	if placeOnJitteredGrid(star, existing, width, height, minSep) {
		log.Printf(
			"galaxy: star %d used grid-jitter fallback after %d random tries (%d already placed, sep=%.1f)",
			star.ID, maxRandomStarPlacementTries, len(existing), minSep,
		)
		return true
	}

	for y := int64(0); y < height; y++ {
		for x := int64(0); x < width; x++ {
			occupied := false
			for _, s := range existing {
				if s != nil && s.X == x && s.Y == y {
					occupied = true
					break
				}
			}
			if !occupied {
				star.X, star.Y = x, y
				log.Printf(
					"galaxy: CRITICAL star %d placed without min separation (could not pack sep=%.1f with %d stars)",
					star.ID, minSep, len(existing)+1,
				)
				return true
			}
		}
	}
	log.Printf("galaxy: CRITICAL star %d left unplaced — map fully occupied", star.ID)
	return true
}

// placeOnJitteredGrid walks a lattice stepped by ceil(minSep), jittering within
// each cell from the star id. Used as a fallback after random placement fails.
func placeOnJitteredGrid(star *model.Star, existing []*model.Star, width, height int64, minSep float64) bool {
	step := int64(math.Ceil(minSep))
	if step < 1 {
		step = 1
	}
	minSepSq := minSep * minSep
	id := int64(star.ID)
	for gy := int64(0); gy < height; gy += step {
		for gx := int64(0); gx < width; gx += step {
			jx := (id*37 + gx*17) % step
			jy := (id*91 + gy*13) % step
			if jx < 0 {
				jx = -jx
			}
			if jy < 0 {
				jy = -jy
			}
			x := gx + jx
			y := gy + jy
			if x >= width {
				x = width - 1
			}
			if y >= height {
				y = height - 1
			}
			if !tooCloseToExisting(x, y, existing, minSepSq) {
				star.X, star.Y = x, y
				return true
			}
		}
	}
	for y := int64(0); y < height; y++ {
		for x := int64(0); x < width; x++ {
			if !tooCloseToExisting(x, y, existing, minSepSq) {
				star.X, star.Y = x, y
				return true
			}
		}
	}
	return false
}

// placeStarsOnLattice assigns stars to an even rectangular lattice that
// realizes minSeparationPixels for a tight pack. Deterministic integer steps
// keep pairwise distances >= sep.
func placeStarsOnLattice(stars []*model.Star, width, height int64) {
	n := uint(len(stars))
	cols, rows, stepX, stepY, _ := packingLattice(width, height, n)
	idx := 0
	for r := 0; r < rows && idx < int(n); r++ {
		for c := 0; c < cols && idx < int(n); c++ {
			var x, y int64
			if cols == 1 {
				x = (width - 1) / 2
			} else {
				x = int64(c) * stepX
			}
			if rows == 1 {
				y = (height - 1) / 2
			} else {
				y = int64(r) * stepY
			}
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
	tight := needsTightLattice(width, height, numStars)
	if tight {
		stats.UsedLattice = true
	}

	for i := range numStars {
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

		if !tight {
			// Interleave create+place so seeded NewGame RNG matches prior builds
			// (lucky trait tests and any other seed-sensitive turn logic).
			if assignGalaxyStarPosition(star, stars, width, height, sep) {
				stats.GridFallbacks++
				stats.ExhaustedRandom = true
			}
		}
		stars = append(stars, star)
	}
	if tight {
		placeStarsOnLattice(stars, width, height)
	}

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
