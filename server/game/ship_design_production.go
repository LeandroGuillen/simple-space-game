package game

import (
	"fmt"
	"slices"

	. "ssg/model"
)

const frigateProductionCost int64 = 100

func (g *Game) findBuildableShipDesign(player *Player, name string) (*ShipDesign, bool) {
	if g == nil || g.Resources == nil || g.Resources.ShipDesignCollection == nil || player == nil || name == "" {
		return nil, false
	}
	for i := range g.Resources.ShipDesigns {
		design := &g.Resources.ShipDesigns[i]
		if design.Name == name && g.playerCanBuildShipDesign(player, *design) {
			return design, true
		}
	}
	for i := range player.ShipDesigns {
		design := &player.ShipDesigns[i]
		if design.Name == name && g.playerCanBuildShipDesign(player, *design) {
			return design, true
		}
	}
	return nil, false
}

func (g *Game) playerCanBuildShipDesign(player *Player, design ShipDesign) bool {
	if player == nil {
		return false
	}
	if shipDesignProductionCost(design) <= 0 {
		return false
	}
	return design.HullSize.Name != "" &&
		design.Drive.Name != "" &&
		design.Armor.Name != "" &&
		design.Computer.Name != "" &&
		design.FuelCell.Name != "" &&
		design.Shield.Name != "" &&
		slices.Contains(player.Available.HullSizes, design.HullSize.Name) &&
		slices.Contains(player.Available.Drives, design.Drive.Name) &&
		slices.Contains(player.Available.Armors, design.Armor.Name) &&
		slices.Contains(player.Available.Computers, design.Computer.Name) &&
		slices.Contains(player.Available.FuelCells, design.FuelCell.Name) &&
		slices.Contains(player.Available.Shields, design.Shield.Name)
}

func shipDesignProductionCost(design ShipDesign) int64 {
	if design.HullSize.Name == "Frigate" {
		return frigateProductionCost
	}
	return 0
}

func fleetShipFromDesign(design ShipDesign) FleetShip {
	return FleetShip{
		Type:               "ship_design",
		Name:               design.Name,
		DesignName:         design.Name,
		HullSize:           design.HullSize.Name,
		Drive:              design.Drive.Name,
		InterstellarSpeed:  design.Drive.InterstellarSpeed,
		FuelCell:           design.FuelCell.Name,
		FuelRangeParsecs:   design.FuelCell.Autonomy,
		UnlimitedFuelRange: fuelCellUnlimited(design.FuelCell),
	}
}

func playerHasShipPartUnlocks(player *Player, design ShipDesign) bool {
	if player == nil {
		return false
	}
	return slices.Contains(player.Available.HullSizes, design.HullSize.Name) &&
		slices.Contains(player.Available.Drives, design.Drive.Name) &&
		slices.Contains(player.Available.Armors, design.Armor.Name) &&
		slices.Contains(player.Available.Computers, design.Computer.Name) &&
		slices.Contains(player.Available.FuelCells, design.FuelCell.Name) &&
		slices.Contains(player.Available.Shields, design.Shield.Name)
}

func validatePlayerCanUseShipDesignParts(player *Player, design ShipDesign) error {
	if player == nil {
		return fmt.Errorf("player is required")
	}
	if !slices.Contains(player.Available.HullSizes, design.HullSize.Name) {
		return fmt.Errorf("hull size %q has not been researched", design.HullSize.Name)
	}
	if !slices.Contains(player.Available.Drives, design.Drive.Name) {
		return fmt.Errorf("drive %q has not been researched", design.Drive.Name)
	}
	if !slices.Contains(player.Available.Armors, design.Armor.Name) {
		return fmt.Errorf("armor %q has not been researched", design.Armor.Name)
	}
	if !slices.Contains(player.Available.Computers, design.Computer.Name) {
		return fmt.Errorf("computer %q has not been researched", design.Computer.Name)
	}
	if !slices.Contains(player.Available.FuelCells, design.FuelCell.Name) {
		return fmt.Errorf("fuel cell %q has not been researched", design.FuelCell.Name)
	}
	if !slices.Contains(player.Available.Shields, design.Shield.Name) {
		return fmt.Errorf("shield %q has not been researched", design.Shield.Name)
	}
	return nil
}
