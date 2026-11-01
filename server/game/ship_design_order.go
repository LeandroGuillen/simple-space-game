package game

import (
	"fmt"
	"slices"
	. "ssg/model"
	"strings"
)

// shipDesignFromOrder resolves all submitted names against the server catalog;
// callers cannot supply arbitrary component or weapon stats.
func (g *Game) shipDesignFromOrder(p *Player, c *DesignShipCommand) (ShipDesign, error) {
	var empty ShipDesign
	if p == nil || c == nil || g.Resources == nil || g.Resources.ShipPartsCollection == nil || g.Resources.BeamCollection == nil {
		return empty, fmt.Errorf("ship catalog unavailable")
	}
	if strings.TrimSpace(c.Name) != c.Name || len(c.Name) == 0 || len(c.Name) > 80 {
		return empty, fmt.Errorf("invalid ship design name")
	}
	if g.Resources.ShipDesignCollection != nil {
		for _, d := range g.Resources.ShipDesigns {
			if d.Name == c.Name {
				return empty, fmt.Errorf("ship design name already exists")
			}
		}
	}
	for _, d := range p.ShipDesigns {
		if d.Name == c.Name {
			return empty, fmt.Errorf("ship design name already exists")
		}
	}
	parts := g.Resources.ShipPartsCollection
	hull, err := HullSizeFromName(c.Hull, parts.HullSizes)
	if err != nil {
		return empty, err
	}
	if hull.Name != "Frigate" || hull.Space < 1 {
		return empty, fmt.Errorf("ship design requires a frigate with weapon space")
	}
	drive, err := DriveFromName(c.Drive, parts.Drives)
	if err != nil {
		return empty, err
	}
	if drive.InterstellarSpeed <= 0 {
		return empty, fmt.Errorf("ship design needs an interstellar drive")
	}
	armor, err := ArmorFromName(c.Armor, parts.Armors)
	if err != nil {
		return empty, err
	}
	computer, err := ComputerFromName(c.Computer, parts.Computers)
	if err != nil {
		return empty, err
	}
	fuel, err := FuelCellFromName(c.FuelCell, parts.FuelCells)
	if err != nil {
		return empty, err
	}
	if fuel.Autonomy <= 0 {
		return empty, fmt.Errorf("ship design needs fuel range")
	}
	shield, err := ShieldFromName(c.Shield, parts.Shields)
	if err != nil {
		return empty, err
	}
	if !slices.Contains(p.Available.Beams, c.Beam) {
		return empty, fmt.Errorf("beam %q is not researched", c.Beam)
	}
	var weapon Beam
	found := false
	for _, b := range g.Resources.Beams {
		if b.Name == c.Beam && b.MaxDamage > 0 {
			weapon = b
			found = true
			break
		}
	}
	if !found {
		return empty, fmt.Errorf("unknown or ineffective beam %q", c.Beam)
	}
	design := NewShipDesign(c.Name, []Beam{weapon}, nil, hull, drive, armor, computer, fuel, shield)
	if err := validatePlayerCanUseShipDesignParts(p, design); err != nil {
		return empty, err
	}
	return design, nil
}
