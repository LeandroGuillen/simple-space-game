package combat

import (
	"ssg/game/combat/computers"
	"ssg/game/combat/weapons"
)

type ShipDesign struct {
	Name               string           `json:"name"`
	Weapons            []weapons.Weapon `json:"weapons"`
	SpecialSystems     []SpecialSystem  `json:"special_systems"`
	BaseSpace          int64            `json:"base_space"`
	Drive              `json:"drive"`
	computers.Computer `json:"computer"`
	Shield             `json:"shield"`
	Armor              `json:"armor"`
	HullSize           `json:"hull_size"`
	FuelCell           `json:"fuel_cell"`
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
