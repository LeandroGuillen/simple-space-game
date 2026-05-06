package game

import (
	"math"
	"strings"

	"ssg/model"
)

func hydratePlanetPollution(p *model.Planet) {
	p.Pollution = Pollution(p)
}

func Pollution(p *model.Planet) int64 {
	dirtyProduction := p.Industry.Workers * p.Industry.ProducedPerWorker
	if dirtyProduction <= 0 {
		return 0
	}

	divisor := pollutionDivisor(p)
	size := pollutionPlanetSize(p)

	// Race tolerance and Environmentalist leaders are not represented in the
	// current model yet, so both coefficients are neutral.
	raw := (float64(dirtyProduction)/float64(divisor) - float64(size)) / 2
	if raw <= 0 {
		return 0
	}
	return int64(math.Ceil(raw))
}

func pollutionDivisor(p *model.Planet) int64 {
	hasProcessor := hasBuildingNamed(p, "Pollution Processor")
	hasRenewer := hasBuildingNamed(p, "Atmospheric Renewer")
	if hasProcessor && hasRenewer {
		return 8
	}
	if hasRenewer {
		return 4
	}
	if hasProcessor {
		return 2
	}
	return 1
}

func pollutionPlanetSize(p *model.Planet) int64 {
	size := int64(p.Size) * 2
	if hasBuildingNamed(p, "Nano Disassemblers") {
		size *= 2
	}
	return size
}

func hasBuildingNamed(p *model.Planet, name string) bool {
	for _, building := range p.Buildings {
		if strings.EqualFold(building.Name, name) {
			return true
		}
	}
	return false
}
