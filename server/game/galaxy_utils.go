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
	// Kept high so counts that reliably succeed at random@60px (medium 43–56,
	// huge 143–170, tiny 8–12) stay on random; only near-capacity / over-capacity
	// go straight to lattice. Web client counts (medium 40, huge 100) stay random.
	randomSafeCapacityFraction = 0.85
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

type latticeCell struct{ x, y int64 }

func latticeCells(cols, rows int, stepX, stepY, width, height int64, n int) []latticeCell {
	cells := make([]latticeCell, 0, n)
	for r := 0; r < rows && len(cells) < n; r++ {
		for c := 0; c < cols && len(cells) < n; c++ {
			x := int64(c) * stepX
			y := int64(r) * stepY
			if x >= width {
				x = width - 1
			}
			if y >= height {
				y = height - 1
			}
			cells = append(cells, latticeCell{x: x, y: y})
		}
	}
	return cells
}

func preferredGridCells(width, height int64, sep float64) []latticeCell {
	step := int64(sep)
	if step < 1 {
		step = 1
	}
	cols := int(math.Floor(float64(width-1)/sep)) + 1
	rows := int(math.Floor(float64(height-1)/sep)) + 1
	return latticeCells(cols, rows, step, step, width, height, cols*rows)
}

// selectSpreadCells picks n cells from all, spaced through the slice so a
// partial fill still covers the map instead of only the first rows.
func selectSpreadCells(all []latticeCell, n int) []latticeCell {
	if n <= 0 {
		return nil
	}
	if n >= len(all) {
		out := make([]latticeCell, len(all))
		copy(out, all)
		return out
	}
	out := make([]latticeCell, n)
	for i := 0; i < n; i++ {
		// Evenly spaced indices across the full grid list.
		idx := int(uint64(i) * uint64(len(all)) / uint64(n))
		if idx >= len(all) {
			idx = len(all) - 1
		}
		out[i] = all[idx]
	}
	return out
}

// placeStarsOnLattice assigns stars across the WHOLE map. Prefers
// packingLattice whenever its spacing is >= 60px (even steps over width/height).
// When packingLattice would squeeze below 60px but a 60px grid still fits n,
// takes a spatially spread subset of that grid. Otherwise uses packingLattice
// at the densest fitting spacing. Cells stay in spread order so startingStarIDs
// (stride by ID) lands homes across the map instead of down one edge.
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

	plCols, plRows, plStepX, plStepY, plSep := packingLattice(width, height, n)
	var cells []latticeCell
	var sep float64

	switch {
	case plSep+1e-9 >= desiredMinStarSeparation:
		// Evenly stepped lattice covering the full map at >= 1 parsec.
		sep = plSep
		cells = latticeCells(plCols, plRows, plStepX, plStepY, width, height, int(n))
	case gridCapacity(width, height, desiredMinStarSeparation) >= n:
		// packingLattice squeezed below 60px, but a 60px grid still fits —
		// spread a subset of that grid over the map (not row-fill from 0,0).
		sep = desiredMinStarSeparation
		all := preferredGridCells(width, height, desiredMinStarSeparation)
		cells = selectSpreadCells(all, int(n))
	default:
		sep = plSep
		cells = latticeCells(plCols, plRows, plStepX, plStepY, width, height, int(n))
	}

	for i := 0; i < int(n) && i < len(cells); i++ {
		stars[i].X = cells[i].x
		stars[i].Y = cells[i].y
	}
	return sep
}

// jitterTinyLattice varies the 15-star Tiny layout while preserving at least
// 47px between stars: the 59px lattice spacing loses at most 6px per side.
// Shuffling the cells also varies the first player's starting location.
func jitterTinyLattice(stars []*model.Star, width, height int64) {
	cells := make([]latticeCell, len(stars))
	for i, star := range stars {
		cells[i] = latticeCell{
			x: max(0, min(width-1, star.X+int64(randomIntn(13)-6))),
			y: max(0, min(height-1, star.Y+int64(randomIntn(13)-6))),
		}
	}
	randomShuffle(len(cells), func(i, j int) { cells[i], cells[j] = cells[j], cells[i] })
	for i, cell := range cells {
		stars[i].X, stars[i].Y = cell.x, cell.y
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
		if galaxySize == "tiny" && numStars == 15 {
			jitterTinyLattice(stars, width, height)
			stats.MinSeparation = 47
		}
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
