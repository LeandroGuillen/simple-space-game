package game

import (
	"math"
	. "ssg/model"
)

func MaxPop(p *Planet) int64 {
	// TODO should take into account other factors
	// Planetary Env.:  Gaia Terran Arid Swamp Other
	// Tiny              5     4     3    2     1     <-- The table shows how
	// Small            10     8     6    4     3         the base max. pop. of
	// Medium           15    12     9    6     4         a planet is determined
	// Large            20    16    12    8     5     <-- by size and climate.
	// Huge             25    20    15   10     6
	return int64(p.Size*4) * 1000
}

// basePopGrowth is the per-turn population growth baseline, before gravity
// and habitability scaling and any future modifiers (race traits, buildings,
// food shortages, etc.).
const basePopGrowth int64 = 40

// gravityPopFactor returns the multiplier applied to base population growth
// for a given planetary gravity.
func gravityPopFactor(g PlanetGravity) float64 {
	switch g {
	case PlanetGravityLow:
		return 0.75
	case PlanetGravityNormal:
		return 1.0
	case PlanetGravityHigh:
		return 0.5
	default:
		return 0.0
	}
}

// habitabilityPopFactor returns the multiplier applied to base population
// growth for a given planetary environment, based on how habitable it is.
func habitabilityPopFactor(e PlanetEnvironment) float64 {
	switch e {
	case PlanetEnvironmentGaia:
		return 1.0
	case PlanetEnvironmentTerran:
		return 1.0
	case PlanetEnvironmentOcean:
		return 0.85
	case PlanetEnvironmentSwamp:
		return 0.65
	case PlanetEnvironmentArid:
		return 0.55
	case PlanetEnvironmentTundra:
		return 0.50
	case PlanetEnvironmentDesert:
		return 0.40
	case PlanetEnvironmentBarren:
		return 0.30
	case PlanetEnvironmentRadiated:
		return 0.20
	case PlanetEnvironmentToxic:
		return 0.15
	case PlanetEnvironmentGasGiant, PlanetEnvironmentAsteroidBelt:
		return 0.0
	default:
		return 0.0
	}
}

// PopulationGrowth returns the per-turn population delta for a planet.
// The result is the baseline (40) scaled by gravity and habitability
// factors. Once future modifiers are wired in (race traits, cloning
// centers, starvation, plagues, ...) the result may exceed the baseline
// or go negative ("people are dying"). Callers are responsible for
// clamping the resulting Population to [0, MaxPop(p)].
//
// Returns 0 for unowned planets and for empty colonies (Population == 0)
// so it is safe to call unconditionally.
//
// TODO: apply race PopulationGrowth bonus from model.Race.Stats once races
// are wired up.
// TODO: apply building/tech modifiers (cloning center, etc.).
// TODO: subtract starvation losses once a food/farming system exists.
func PopulationGrowth(p *Planet) int64 {
	if p.Owner == 0 {
		return 0
	}
	if p.Population <= 0 {
		return 0
	}

	scaled := float64(basePopGrowth) * gravityPopFactor(p.Gravity) * habitabilityPopFactor(p.Environment)
	return int64(math.Round(scaled))
}
