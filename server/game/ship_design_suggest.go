package game

import (
	"fmt"

	. "ssg/model"
)

// suggestedShipDesignHull is the preferred hull for suggestions: the only hull
// colonies can currently build (see shipDesignProductionCost).
const suggestedShipDesignHull = "Frigate"

// SuggestShipDesign returns an unsaved design draft built from the best parts
// the player has unlocked (see SuggestShipDesignParts for the rules). It never
// adds, changes or saves designs; the caller decides whether to save the draft.
// The draft name does not collide with the player's or the stock designs.
func (g *Game) SuggestShipDesign(playerID uint) (ShipDesignsDef, error) {
	g.ensureMu()
	g.mu.RLock()
	defer g.mu.RUnlock()
	player, err := g.getPlayer(playerID)
	if err != nil {
		return ShipDesignsDef{}, err
	}
	resources := g.Resources
	if resources == nil || resources.ShipPartsCollection == nil {
		return ShipDesignsDef{}, fmt.Errorf("ship parts unavailable")
	}
	catalog := ShipDesignPartCatalog{
		HullSizes: resources.HullSizes,
		Drives:    resources.Drives,
		FuelCells: resources.FuelCells,
		Armors:    resources.Armors,
		Computers: resources.Computers,
		Shields:   resources.Shields,
	}
	if resources.BeamCollection != nil {
		catalog.Beams = resources.Beams
	}
	if resources.MissileCollection != nil {
		catalog.Missiles = resources.Missiles
	}
	choice, err := SuggestShipDesignParts(player.Available, catalog, suggestedShipDesignHull)
	if err != nil {
		return ShipDesignsDef{}, err
	}
	def := ShipDesignsDef{
		HullSize: choice.Hull,
		Drive:    choice.Drive,
		FuelCell: choice.FuelCell,
		Armor:    choice.Armor,
		Computer: choice.Computer,
		Shield:   choice.Shield,
		Beam:     choice.Beam,
		Missile:  choice.Missile,
	}
	used := map[string]bool{}
	for _, d := range player.ShipDesigns {
		used[d.Name] = true
	}
	for _, d := range resources.ShipDesignCatalog() {
		used[d.Name] = true
	}
	for n := 1; ; n++ {
		def.Name = fmt.Sprintf("Suggested %s %d", choice.Hull, n)
		if !used[def.Name] {
			break
		}
	}
	// Same checks as saving the design, so the draft is always savable.
	if _, err := g.resolvePlayerShipDesign(player, def); err != nil {
		return ShipDesignsDef{}, err
	}
	return def, nil
}
