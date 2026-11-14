package ai

import (
	"fmt"
	"ssg/model"
)

// planShipDesign creates a new immutable revision when a better available
// component changes. It never edits player or stock designs.
func planShipDesign(s *VisibleGame, p *model.Player, planets []*model.Planet, out *Decision) {
	if s.Resources == nil || s.Resources.ShipParts == nil || s.Resources.BeamCollection == nil {
		return
	}
	parts := s.Resources.ShipParts
	// The AI keeps to beam-armed Frigates: missiles are not offered here, and
	// a fallback hull or an unarmed design is skipped.
	choice, err := model.SuggestShipDesignParts(p.Available, model.ShipDesignPartCatalog{
		HullSizes: parts.HullSizes,
		Drives:    parts.Drives,
		FuelCells: parts.FuelCells,
		Armors:    parts.Armors,
		Computers: parts.Computers,
		Shields:   parts.Shields,
		Beams:     s.Resources.BeamCollection.Beams,
	}, "Frigate")
	if err != nil || choice.Hull != "Frigate" || choice.Beam == "" {
		return
	}
	c := model.DesignShipCommand{Hull: choice.Hull, Drive: choice.Drive, FuelCell: choice.FuelCell, Armor: choice.Armor, Computer: choice.Computer, Shield: choice.Shield, Beam: choice.Beam}
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
				planned := false
				for _, b := range out.Commands.Build {
					if b.PlanetID == planet.ID {
						planned = true
						break
					}
				}
				if planned {
					continue
				}
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
