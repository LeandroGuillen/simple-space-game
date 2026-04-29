package game

import (
	. "ssg/model"
)

func NewShipDesign(
	name string,
	beams []Beam,
	specialSystems []SpecialSystem,
	hull HullSize,
	drive Drive,
	armor Armor,
	computer Computer,
	fuelCell FuelCell,
	shield Shield,
) ShipDesign {
	return ShipDesign{
		Name:           name,
		Beams:          beams,
		SpecialSystems: specialSystems,
		BaseSpace:      10,
		Drive:          drive,
		Computer:       computer,
		Shield:         shield,
		Armor:          armor,
		HullSize:       hull,
		FuelCell:       fuelCell,
	}
}
