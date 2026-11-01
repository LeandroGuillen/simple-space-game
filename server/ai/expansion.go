package ai

import (
	"math"
	"ssg/game"
	"ssg/model"
)

func colonizable(p *model.Planet) bool {
	return p != nil && p.Owner == 0 && p.Environment != model.PlanetEnvironmentInvalid && p.Environment != model.PlanetEnvironmentGasGiant && p.Environment != model.PlanetEnvironmentAsteroidBelt
}

// planExpansion uses only explored planets and the same fuel range calculation
// as turn validation. Pending projects and targets are reserved within a turn.
func planExpansion(s *VisibleGame, me *model.Player, planets []*model.Planet, d *Decision) {
	reservedStars := map[uint]bool{}
	reservedPlanets := map[uint]bool{}
	busy := map[uint]bool{}
	for _, c := range d.Commands.Invade {
		if c != nil {
			busy[c.FleetID] = true
		}
	}
	for _, c := range d.Commands.ColonizeMany {
		if c != nil {
			reservedPlanets[c.PlanetID] = true
			if c.FleetID != 0 {
				busy[c.FleetID] = true
			}
		}
	}
	if d.Commands.ColonyShip != nil {
		busy[d.Commands.ColonyShip.FleetID] = true
	}
	ships := 0
	for _, f := range s.Fleets {
		if f == nil || f.Owner != me.ID {
			continue
		}
		for _, ship := range f.Ships {
			if ship.Type == "colony_ship" {
				ships++
			}
		}
		if f.DestinationStarID != 0 {
			reservedStars[f.DestinationStarID] = true
		}
	}
	for _, f := range s.Fleets {
		if f == nil || f.Owner != me.ID || f.DestinationStarID != 0 || f.TurnsRemaining != 0 || busy[f.ID] {
			continue
		}
		colony := false
		for _, ship := range f.Ships {
			if ship.Type == "colony_ship" {
				colony = true
				break
			}
		}
		if !colony {
			continue
		}
		star := findStarByID(s, f.StarID)
		if star != nil {
			for _, p := range star.Planets {
				if colonizable(p) && !reservedPlanets[p.ID] {
					d.Commands.ColonizeMany = append(d.Commands.ColonizeMany, &model.ColonizeCommand{PlanetID: p.ID, FleetID: f.ID})
					reservedPlanets[p.ID] = true
					busy[f.ID] = true
					break
				}
			}
		}
		if busy[f.ID] {
			continue
		}
		if target := expansionTarget(s, me.ID, f, reservedStars); target != 0 {
			d.Commands.FleetMove = append(d.Commands.FleetMove, &model.FleetMoveCommand{FleetID: f.ID, DestinationStarID: target})
			reservedStars[target] = true
		}
	}
	opportunity := false
	for _, star := range s.Galaxy.Stars {
		if star == nil {
			continue
		}
		if star.Name == "" {
			opportunity = true
			break
		}
		for _, planet := range star.Planets {
			if colonizable(planet) {
				opportunity = true
				break
			}
		}
		if opportunity {
			break
		}
	}
	if !opportunity {
		return
	}
	baseStars := map[uint]bool{}
	for _, p := range planets {
		if p.IsBuilding && p.CurrentProject == "Colony Base" {
			baseStars[p.ParentStar] = true
		}
	}
	for _, sourceID := range s.PendingColonyBases[me.ID] {
		for _, p := range planets {
			if p.ID == sourceID {
				baseStars[p.ParentStar] = true
			}
		}
	}
	for _, p := range planets {
		if p.IsBuilding || len(p.BuildQueue) > 0 || (p.CurrentProject != "" && p.CurrentProject != "Trade Goods" && p.CurrentProject != "Housing") || baseStars[p.ParentStar] || !techUnlocked(me, "Colony Base") {
			continue
		}
		star := findStarByID(s, p.ParentStar)
		if star == nil {
			continue
		}
		for _, target := range star.Planets {
			if colonizable(target) && !reservedPlanets[target.ID] {
				d.Commands.Build = append(d.Commands.Build, &model.BuildCommand{PlanetID: p.ID, Building: "Colony Base"})
				baseStars[p.ParentStar] = true
				break
			}
		}
	}
	for _, p := range planets {
		if p.CurrentProject == "Colony Ship" && p.IsBuilding {
			ships++
		}
		for _, q := range p.BuildQueue {
			if q == "Colony Ship" {
				ships++
			}
		}
	}
	for _, p := range planets {
		planned := false
		for _, b := range d.Commands.Build {
			if b.PlanetID == p.ID {
				planned = true
				break
			}
		}
		if planned {
			continue
		}
		if p.IsBuilding || len(p.BuildQueue) > 0 || (p.CurrentProject != "" && p.CurrentProject != "Trade Goods" && p.CurrentProject != "Housing") {
			continue
		}
		if ships > 0 || !techUnlocked(me, "Colony Ship") {
			continue
		}
		d.Commands.Build = append(d.Commands.Build, &model.BuildCommand{PlanetID: p.ID, Building: "Colony Ship"})
		ships++
	}
}
func expansionTarget(s *VisibleGame, id uint, f *model.Fleet, reserved map[uint]bool) uint {
	rng, unlimited, ok := game.ShipsFuelRange(f.Ships, visibleFuelRangeLookup(s, id))
	if !ok {
		return 0
	}
	best := uint(0)
	distance := math.MaxFloat64
	for _, star := range s.Galaxy.Stars {
		if star == nil || star.Name == "" || star.ID == f.StarID || reserved[star.ID] || !game.FuelDestinationReachable(float64(star.X), float64(star.Y), ownedSupplyStarCoords(s, id), rng, unlimited) {
			continue
		}
		found := false
		for _, p := range star.Planets {
			if colonizable(p) {
				found = true
				break
			}
		}
		if !found {
			continue
		}
		dx, dy := float64(star.X)-f.X, float64(star.Y)-f.Y
		dist := dx*dx + dy*dy
		if dist < distance {
			best, distance = star.ID, dist
		}
	}
	return best
}
