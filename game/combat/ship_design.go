package combat

import (
	"ssg/game/combat/computers"
	"ssg/game/combat/weapons"
)

type ShipDesign struct {
	Name           string
	Weapons        []weapons.Weapon
	SpecialSystems []SpecialSystem
	BaseSpace      int64
	Drive
	computers.Computer
	Shield
	Armor
	HullSize
	FuelCell
}

func NewShipDesign(
	name string,
	weapons []weapons.Weapon,
	specialSystems []SpecialSystem,
	hull HullSize,
	drive Drive,
	armor Armor,
	computer computers.Computer,
	fuelCell FuelCell,
	shield Shield,
) ShipDesign {
	return ShipDesign{
		name,
		weapons,
		specialSystems,
		10,
		drive,
		computer,
		shield,
		armor,
		hull,
		fuelCell,
	}
}
