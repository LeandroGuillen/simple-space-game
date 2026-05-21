package game

import (
	"math"

	"ssg/model"
)

func hydratePlanetPollution(game *Game, p *model.Planet) {
	p.Pollution = Pollution(game, p)
}

func Pollution(game *Game, p *model.Planet) int64 {
	player := gamePlayerForPlanet(game, p)
	if playerHasRaceTrait(player, "tolerant") {
		return 0
	}

	dirtyProduction := p.Industry.Workers * p.Industry.ProducedPerWorker
	if dirtyProduction <= 0 {
		return 0
	}

	divisor := pollutionDivisor(p)
	size := pollutionPlanetSize(p)

	raw := (float64(dirtyProduction)/float64(divisor) - float64(size)) / 2
	if raw <= 0 {
		return 0
	}
	return int64(math.Ceil(raw))
}

func pollutionDivisor(p *model.Planet) int64 {
	hasProcessor := hasBuilding(p, "Pollution Processor")
	hasRenewer := hasBuilding(p, "Atmospheric Renewer")
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
	if hasBuilding(p, "Nano Disassemblers") {
		size *= 2
	}
	return size
}
