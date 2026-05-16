package game

import (
	"fmt"
	"strings"

	. "ssg/model"
)

// ShipDesignPartsDef flattens a resolved ship design into serializable part names.
func ShipDesignPartsDef(d ShipDesign) ShipDesignsDef {
	return ShipDesignsDef{
		Name:     d.Name,
		Drive:    d.Drive.Name,
		Computer: d.Computer.Name,
		Shield:   d.Shield.Name,
		Armor:    d.Armor.Name,
		HullSize: d.HullSize.Name,
		FuelCell: d.FuelCell.Name,
	}
}

// SaveableCustomShipDesign resolves part names against equipment data and returns a normalized design.
func SaveableCustomShipDesign(resources *ResourceManager, def ShipDesignsDef) (ShipDesign, error) {
	if resources == nil || resources.ShipPartsCollection == nil {
		return ShipDesign{}, fmt.Errorf("ship parts unavailable")
	}
	name := strings.TrimSpace(def.Name)
	if name == "" {
		return ShipDesign{}, fmt.Errorf("ship design name is required")
	}
	parts := resources.ShipPartsCollection

	armor, err := ArmorFromName(def.Armor, parts.Armors)
	if err != nil {
		return ShipDesign{}, err
	}
	drive, err := DriveFromName(def.Drive, parts.Drives)
	if err != nil {
		return ShipDesign{}, err
	}
	hullSize, err := HullSizeFromName(def.HullSize, parts.HullSizes)
	if err != nil {
		return ShipDesign{}, err
	}
	computer, err := ComputerFromName(def.Computer, parts.Computers)
	if err != nil {
		return ShipDesign{}, err
	}
	fuelCell, err := FuelCellFromName(def.FuelCell, parts.FuelCells)
	if err != nil {
		return ShipDesign{}, err
	}
	shield, err := ShieldFromName(def.Shield, parts.Shields)
	if err != nil {
		return ShipDesign{}, err
	}

	design := NewShipDesign(
		name,
		[]Beam{},
		[]SpecialSystem{},
		hullSize,
		drive,
		armor,
		computer,
		fuelCell,
		shield,
	)
	return design, nil
}
