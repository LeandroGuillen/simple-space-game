package ai

import (
	"fmt"
	"math"
	"ssg/game"
	"ssg/model"
)

// shipCombatDesign resolves only the player's designs and public stock designs.
func shipCombatDesign(s *VisibleGame, p *model.Player, ship model.FleetShip) *model.ShipDesign {
	if ship.Type != "ship_design" {
		return nil
	}
	for i := range p.ShipDesigns {
		if p.ShipDesigns[i].Name == ship.DesignName || (ship.DesignName == "" && p.ShipDesigns[i].Name == ship.Name) {
			return &p.ShipDesigns[i]
		}
	}
	if s.Resources != nil && s.Resources.ShipDesignCollection != nil {
		for i := range s.Resources.ShipDesignCollection.ShipDesigns {
			d := &s.Resources.ShipDesignCollection.ShipDesigns[i]
			if d.Name == ship.DesignName || (ship.DesignName == "" && d.Name == ship.Name) {
				return d
			}
		}
	}
	return nil
}
func armedShip(s *VisibleGame, p *model.Player, ship model.FleetShip) bool {
	d := shipCombatDesign(s, p, ship)
	return d != nil && len(d.Beams) > 0
}
func mobileCombatShip(s *VisibleGame, p *model.Player, ship model.FleetShip) bool {
	if !armedShip(s, p, ship) {
		return false
	}
	// Follow the server's precedence: a stamped catalog drive overrides the design.
	if ship.Drive != "" && s.Resources != nil && s.Resources.ShipParts != nil {
		for _, d := range s.Resources.ShipParts.Drives {
			if d.Name == ship.Drive {
				return d.InterstellarSpeed > 0
			}
		}
	}
	if ship.InterstellarSpeed > 0 {
		return true
	}
	d := shipCombatDesign(s, p, ship)
	return d != nil && d.Drive.InterstellarSpeed > 0
}

// hostileOwner reports whether owner is a rival empire the AI may attack or
// invade: anyone but itself and empires it has a peace treaty with. Without a
// treaty (at war or not) attacks are allowed; the fight auto-declares war.
func hostileOwner(p *model.Player, owner uint) bool {
	return owner != 0 && owner != p.ID && !p.HasPeaceTreatyWith(owner)
}

func planFleetAttacks(s *VisibleGame, p *model.Player, orders *Decision) {
	busy := map[uint]bool{}
	reserved := map[uint]bool{}
	for _, m := range orders.Commands.FleetMove {
		if m != nil {
			busy[m.FleetID] = true
			reserved[m.DestinationStarID] = true
		}
	}
	for _, c := range orders.Commands.ColonizeMany {
		if c != nil && c.FleetID != 0 {
			busy[c.FleetID] = true
		}
	}
	for _, c := range orders.Commands.Invade {
		if c != nil {
			busy[c.FleetID] = true
		}
	}
	if orders.Commands.ColonyShip != nil {
		busy[orders.Commands.ColonyShip.FleetID] = true
	}
	for _, f := range s.Fleets {
		if f != nil && f.Owner == p.ID && f.DestinationStarID != 0 {
			reserved[f.DestinationStarID] = true
		}
	}
	supplies := ownedSupplyStarCoords(s, p.ID)
	for _, f := range s.Fleets {
		if f == nil || f.Owner != p.ID || f.DestinationStarID != 0 || f.TurnsRemaining != 0 || busy[f.ID] {
			continue
		}
		ownStar := findStarByID(s, f.StarID)
		localColony := false
		if ownStar != nil {
			for _, planet := range ownStar.Planets {
				if planet != nil && planet.Owner == p.ID {
					localColony = true
					break
				}
			}
		}
		var indexes []uint
		var ships []model.FleetShip
		for i, ship := range f.Ships {
			if mobileCombatShip(s, p, ship) {
				indexes = append(indexes, uint(i))
				ships = append(ships, ship)
			}
		}
		// Leave one armed ship behind on every colony the attacking fleet departs.
		if localColony && len(indexes) > 0 {
			indexes = indexes[1:]
			ships = ships[1:]
		}
		if len(ships) == 0 {
			continue
		}
		rng, unlimited, ok := game.ShipsFuelRange(ships, visibleFuelRangeLookup(s, p.ID))
		if !ok {
			continue
		}
		var target uint
		distance := math.MaxFloat64
		for _, star := range s.Galaxy.Stars {
			if star == nil || star.Name == "" || star.ID == f.StarID || reserved[star.ID] || !game.FuelDestinationReachable(float64(star.X), float64(star.Y), supplies, rng, unlimited) {
				continue
			}
			enemy := false
			for _, planet := range star.Planets {
				if planet != nil && hostileOwner(p, planet.Owner) {
					enemy = true
					break
				}
			}
			if !enemy {
				continue
			}
			// Rival ship details are limited to the visible fleet snapshot. Count every
			// visible ship conservatively; hidden fleets are not available to the AI.
			// Treaty partners' ships do not fight, so they are not defenders.
			defenders := 0
			for _, other := range s.Fleets {
				if other != nil && hostileOwner(p, other.Owner) && other.StarID == star.ID {
					defenders += len(other.Ships)
				}
			}
			if len(ships) <= defenders {
				continue
			}
			dx, dy := float64(star.X)-f.X, float64(star.Y)-f.Y
			dist := dx*dx + dy*dy
			if dist < distance {
				distance = dist
				target = star.ID
			}
		}
		if target == 0 {
			continue
		}
		selected := append([]uint(nil), indexes...)
		orders.Commands.FleetMove = append(orders.Commands.FleetMove, &model.FleetMoveCommand{FleetID: f.ID, DestinationStarID: target, ShipIndexes: &selected})
		reserved[target] = true
		busy[f.ID] = true
		orders.Notes = append(orders.Notes, fmt.Sprintf("fleet %d: sending %d warship(s) to attack star %d", f.ID, len(selected), target))
	}
}

// Build one invasion transport when there is a visible rival and a combat
// force available. Existing construction and colony plans take precedence.
func planMilitaryProduction(s *VisibleGame, p *model.Player, planets []*model.Planet, orders *Decision) {
	if !techUnlocked(p, "Transport") {
		return
	}
	enemy := false
	for _, star := range s.Galaxy.Stars {
		if star == nil || star.Name == "" {
			continue
		}
		for _, planet := range star.Planets {
			if planet != nil && hostileOwner(p, planet.Owner) {
				enemy = true
				break
			}
		}
	}
	if !enemy {
		return
	}
	ships := 0
	for _, fleet := range s.Fleets {
		if fleet == nil || fleet.Owner != p.ID {
			continue
		}
		for _, ship := range fleet.Ships {
			if ship.Type == "transport" {
				return
			}
			if armedShip(s, p, ship) {
				ships++
			}
		}
	}
	if ships < 2 {
		return
	}
	for _, planet := range planets {
		if planet.CurrentProject == "Transport" && planet.IsBuilding {
			return
		}
		for _, q := range planet.BuildQueue {
			if q == "Transport" {
				return
			}
		}
	}
	for _, planet := range planets {
		if planet.IsBuilding || len(planet.BuildQueue) > 0 || (planet.CurrentProject != "" && planet.CurrentProject != "Trade Goods") {
			continue
		}
		planned := false
		for _, cmd := range orders.Commands.Build {
			if cmd != nil && cmd.PlanetID == planet.ID {
				planned = true
				break
			}
		}
		if planned {
			continue
		}
		orders.Commands.Build = append(orders.Commands.Build, &model.BuildCommand{PlanetID: planet.ID, Building: "Transport"})
		return
	}
}

// Reinforce an already-secured enemy orbit with transports, without moving a
// colony ship or the defensive ships in their original fleet.
func planTransportMoves(s *VisibleGame, p *model.Player, d *Decision) {
	busy := map[uint]bool{}
	for _, c := range d.Commands.Invade {
		if c != nil {
			busy[c.FleetID] = true
		}
	}
	for _, c := range d.Commands.FleetMove {
		if c != nil {
			busy[c.FleetID] = true
		}
	}
	for _, c := range d.Commands.ColonizeMany {
		if c != nil && c.FleetID != 0 {
			busy[c.FleetID] = true
		}
	}
	if d.Commands.ColonyShip != nil {
		busy[d.Commands.ColonyShip.FleetID] = true
	}
	supplies := ownedSupplyStarCoords(s, p.ID)
	for _, fleet := range s.Fleets {
		if fleet == nil || fleet.Owner != p.ID || fleet.DestinationStarID != 0 || fleet.TurnsRemaining != 0 || busy[fleet.ID] {
			continue
		}
		var indexes []uint
		var ships []model.FleetShip
		for i, ship := range fleet.Ships {
			if ship.Type == "transport" {
				indexes = append(indexes, uint(i))
				ships = append(ships, ship)
			}
		}
		if len(ships) == 0 {
			continue
		}
		// Transports without explicit designs use the best researched drive.
		mobile := false
		if s.Resources != nil && s.Resources.ShipParts != nil {
			for _, drive := range s.Resources.ShipParts.Drives {
				if drive.InterstellarSpeed > 0 {
					for _, name := range p.Available.Drives {
						if name == drive.Name {
							mobile = true
						}
					}
				}
			}
		}
		if !mobile {
			continue
		}
		rng, unlimited, ok := game.ShipsFuelRange(ships, visibleFuelRangeLookup(s, p.ID))
		if !ok {
			continue
		}
		for _, star := range s.Galaxy.Stars {
			if star == nil || star.Name == "" || star.ID == fleet.StarID || !game.FuelDestinationReachable(float64(star.X), float64(star.Y), supplies, rng, unlimited) {
				continue
			}
			enemyPlanet := false
			for _, planet := range star.Planets {
				if planet != nil && hostileOwner(p, planet.Owner) {
					enemyPlanet = true
					break
				}
			}
			if !enemyPlanet {
				continue
			}
			escort, hostile := false, false
			for _, other := range s.Fleets {
				if other == nil || other.StarID != star.ID || other.DestinationStarID != 0 || other.TurnsRemaining != 0 {
					continue
				}
				if hostileOwner(p, other.Owner) && len(other.Ships) > 0 {
					hostile = true
				}
				if other.Owner == p.ID {
					for _, ship := range other.Ships {
						if armedShip(s, p, ship) {
							escort = true
						}
					}
				}
			}
			if !escort || hostile {
				continue
			}
			selected := append([]uint(nil), indexes...)
			d.Commands.FleetMove = append(d.Commands.FleetMove, &model.FleetMoveCommand{FleetID: fleet.ID, DestinationStarID: star.ID, ShipIndexes: &selected})
			break
		}
	}
}
