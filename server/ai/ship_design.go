package ai

import (
	"fmt"
	"slices"
	"ssg/model"
)

// planShipDesign creates a new immutable revision when a better available
// component changes. It never edits player or stock designs.
func planShipDesign(s *VisibleGame, p *model.Player, planets []*model.Planet, out *Decision) {
	if s.Resources == nil || s.Resources.ShipParts == nil || s.Resources.BeamCollection == nil {
		return
	}
	parts := s.Resources.ShipParts
	c := model.DesignShipCommand{Hull: "Frigate"}
	for _, h := range parts.HullSizes {
		if h.Name == c.Hull && h.Space > 0 && slices.Contains(p.Available.HullSizes, h.Name) {
			c.Hull = h.Name
			break
		}
	}
	if !slices.Contains(p.Available.HullSizes, c.Hull) {
		return
	}
	var speed int64
	for _, v := range parts.Drives {
		if slices.Contains(p.Available.Drives, v.Name) && v.InterstellarSpeed > speed {
			speed = v.InterstellarSpeed
			c.Drive = v.Name
		}
	}
	var rangeValue int64
	for _, v := range parts.FuelCells {
		if slices.Contains(p.Available.FuelCells, v.Name) && v.Autonomy > rangeValue {
			rangeValue = v.Autonomy
			c.FuelCell = v.Name
		}
	}
	var armorValue int64 = -1
	for _, v := range parts.Armors {
		if slices.Contains(p.Available.Armors, v.Name) && v.Multiplier > armorValue {
			armorValue = v.Multiplier
			c.Armor = v.Name
		}
	}
	var compValue int64 = -1
	for _, v := range parts.Computers {
		if slices.Contains(p.Available.Computers, v.Name) && v.BeamAttackBonus > compValue {
			compValue = v.BeamAttackBonus
			c.Computer = v.Name
		}
	}
	var shieldValue int64 = -1
	for _, v := range parts.Shields {
		if slices.Contains(p.Available.Shields, v.Name) && v.ReduceDamage > shieldValue {
			shieldValue = v.ReduceDamage
			c.Shield = v.Name
		}
	}
	var damage int64
	for _, v := range s.Resources.BeamCollection.Beams {
		if slices.Contains(p.Available.Beams, v.Name) && v.MaxDamage > damage {
			damage = v.MaxDamage
			c.Beam = v.Name
		}
	}
	if c.Drive == "" || c.FuelCell == "" || c.Armor == "" || c.Computer == "" || c.Shield == "" || c.Beam == "" {
		return
	}
	for _, d := range p.ShipDesigns {
		if d.HullSize.Name == c.Hull && d.Drive.Name == c.Drive && d.FuelCell.Name == c.FuelCell && d.Armor.Name == c.Armor && d.Computer.Name == c.Computer && d.Shield.Name == c.Shield && len(d.Beams) == 1 && d.Beams[0].Name == c.Beam {
			count := 0
			for _, f := range s.Fleets {
				if f != nil && f.Owner == p.ID {
					for _, ship := range f.Ships {
						if ship.DesignName == d.Name {
							count++
						}
					}
				}
			}
			if count >= 2 {
				return
			}
			for _, planet := range planets {
				if !planet.IsBuilding && (planet.CurrentProject == "" || planet.CurrentProject == "Trade Goods") && len(planet.BuildQueue) == 0 {
					out.Commands.Build = append(out.Commands.Build, &model.BuildCommand{PlanetID: planet.ID, Building: d.Name})
					return
				}
			}
			return
		}
	}
	used := map[string]bool{}
	for _, d := range p.ShipDesigns {
		used[d.Name] = true
	}
	if s.Resources.ShipDesignCollection != nil {
		for _, d := range s.Resources.ShipDesignCollection.ShipDesigns {
			used[d.Name] = true
		}
	}
	for n := 1; ; n++ {
		c.Name = fmt.Sprintf("AI Frigate %d", n)
		if !used[c.Name] {
			break
		}
	}
	out.Commands.DesignShip = &c
}
