package game

import (
	"fmt"
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
	for _, d := range g.Resources.ShipDesignCatalog() {
		if d.Name == c.Name {
			return empty, fmt.Errorf("ship design name already exists")
		}
	}
	for _, d := range p.ShipDesigns {
		if d.Name == c.Name {
			return empty, fmt.Errorf("ship design name already exists")
		}
	}
	if c.Beam == "" {
		return empty, fmt.Errorf("combat ship design needs a beam")
	}
	design, err := g.Resources.resolveShipDesign(ShipDesignsDef{Name: c.Name, Drive: c.Drive, Armor: c.Armor, Computer: c.Computer, FuelCell: c.FuelCell, Shield: c.Shield, HullSize: c.Hull, Beam: c.Beam})
	if err != nil {
		return empty, err
	}
	if design.HullSize.Name != "Frigate" {
		return empty, fmt.Errorf("ship design requires a frigate with weapon space")
	}
	if err := validatePlayerCanUseShipDesignParts(p, design); err != nil {
		return empty, err
	}
	return design, nil
}
