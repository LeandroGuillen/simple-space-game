package combat

import (
	"fmt"
	"math/rand"
	"ssg/game/combat/computers"
	"ssg/game/combat/weapons"
)

//https://gamefaqs.gamespot.com/pc/197873-master-of-orion-ii-battle-at-antares/faqs/16743

type Ship struct {
	Name string
	ShipDesign
	ShipStatus
	SpaceAvailable int64
	BeamDefense    int64
	MissileEvasion int64
}

func (s Ship) String() string {
	return fmt.Sprintf("[%s %s] Armor: %d/%d - Hull %d/%d",
		s.HullSize.Name,
		s.Name,
		s.ShipStatus.CurrentArmorHP,
		s.ShipStatus.MaxArmorHP,
		s.ShipStatus.CurrentStructureHP,
		s.ShipStatus.MaxStructureHP,
	)
}

func NewEmptyBattleShip(name string) Ship {
	return NewEmptyShip(
		name,
		HullSize{"BS_1", 250, 20, 5, 50, 7, 15, 8},
		Drive{}, Armor{}, computers.Computer{}, FuelCell{}, Shield{},
	)
}

func NewEmptyShip(name string, hull HullSize, drive Drive, armor Armor,
	computer computers.Computer, fuelCell FuelCell, shield Shield) Ship {
	basicDesign := NewShipDesign(
		name,
		[]weapons.Weapon{},
		[]SpecialSystem{},
		hull,
		drive,
		armor,
		computer,
		fuelCell,
		shield,
	)

	return NewShip(name, basicDesign)
}

func NewShip(name string, d ShipDesign) Ship {
	return Ship{
		Name:       name,
		ShipDesign: d,
		ShipStatus: ShipStatus{
			CurrentArmorHP:     d.BaseHP * d.Armor.Multiplier,
			MaxArmorHP:         d.BaseHP * d.Armor.Multiplier,
			CurrentStructureHP: d.BaseHP * d.Armor.Multiplier,
			MaxStructureHP:     d.BaseHP * d.Armor.Multiplier,
			CurrentComputerHP:  d.ComputerHP,
			CurrentDriveHP:     d.DriveHP,
			CurrentShieldHP:    d.ShieldHP,
		},
		SpaceAvailable: d.BaseSpace,
		BeamDefense:    d.BaseSpeed * 5,
		MissileEvasion: 0,
	}
}

func (s *Ship) EquipWeapon(w weapons.Weapon) bool {
	s.Weapons = append(s.Weapons, w)
	return true
}

func (s *Ship) EquipWeapons(weapons []weapons.Weapon) bool {
	for _, weapon := range weapons {
		s.EquipWeapon(weapon)
	}
	return true
}

func (attacker *Ship) FireOn(defender *Ship, distance int64) (totalDamage int64) {
	// TODO attackBonus could be >0
	chanceToHit := attacker.Computer.GetChanceToHit(defender.BeamDefense, 0, distance)

	for _, beam := range attacker.Weapons {
		roll := rand.Float64()
		if roll <= chanceToHit {
			totalDamage += beam.GetDamage(distance)
		}
	}

	// ------------------------------
	// apply damage to receiving ship
	// ------------------------------
	if defender.CurrentArmorHP > 0 && defender.CurrentArmorHP >= totalDamage {
		// armor absorbed the damage
		defender.CurrentArmorHP -= totalDamage
	} else if defender.CurrentArmorHP > 0 && defender.CurrentArmorHP < totalDamage {
		// armor gets destroyed, damage spills into hull
		difference := totalDamage - defender.CurrentArmorHP
		defender.CurrentArmorHP = 0

		// can the hull absorb the hit?
		if difference < defender.CurrentStructureHP {
			defender.CurrentStructureHP -= difference
		} else {
			// defender destroyed
			defender.CurrentStructureHP = 0
		}
	} else if defender.CurrentArmorHP == 0 && defender.CurrentStructureHP >= totalDamage {
		// no armor, apply to hull
		defender.CurrentStructureHP -= totalDamage
	} else if defender.CurrentArmorHP == 0 && defender.CurrentStructureHP < totalDamage {
		// no armor, no hull, defender destroyed
		defender.CurrentStructureHP = 0
	}

	return totalDamage
}
