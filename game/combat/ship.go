package combat

import (
	"fmt"
	"math"
	"math/rand"
)

//https://gamefaqs.gamespot.com/pc/197873-master-of-orion-ii-battle-at-antares/faqs/16743

type ShipDesign struct {
	Name           string
	Weapons        []Beam
	SpecialSystems []SpecialSystem
	BaseSpace      int64
	HullSize
	Drive
	Armor
	Computer
	FuelCell
	Shield
}

type ShipStatus struct {
	// hitpoints
	CurrentArmorHP     int64
	CurrentStructureHP int64
	MaxArmorHP         int64
	MaxStructureHP     int64

	// equipment hitpoints
	CurrentComputerHP int64
	CurrentDriveHP    int64
	CurrentShieldHP   int64
}

type Ship struct {
	Name string
	ShipDesign
	ShipStatus
	SpaceAvailable int64
	BeamAttack     int64
	BeamDefense    int64
	MissileEvasion int64
}

//func (ss ShipStatus) String() string {
//	return fmt.Sprintf("Armor: %d/%d - Hull %d/%d",
//		ss.CurrentArmorHP, ss.MaxArmorHP,
//		ss.CurrentStructureHP, ss.MaxStructureHP,
//	)
//}

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

func NewShipDesign(name string, weapons []Beam, specialSystems []SpecialSystem, hull HullSize, drive Drive, armor Armor, computer Computer, fuelCell FuelCell, shield Shield) ShipDesign {
	return ShipDesign{name, weapons, specialSystems, 10, hull, drive, armor, computer, fuelCell, shield}
}

func NewEmptyBattleShip(name string) Ship {
	return NewEmptyShip(name, HullSize{"BS_1", 250, 20, 5, 50, 7, 15, 8})
}

func NewEmptyShip(name string, hull HullSize) Ship {
	basicDesign := NewShipDesign(
		name,
		[]Beam{},
		[]SpecialSystem{},
		hull,
		Drives[0],
		Armors[0],
		Computers[0],
		FuelCells[0],
		Shields[0],
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
		BeamAttack:     d.Computer.BeamAttackBonus,
		BeamDefense:    d.BaseSpeed * 5,
		MissileEvasion: 0,
	}
}

func FireOn(attacker, defender *Ship, distance int64) int64 {
	// Chance to hit
	// =============
	// https://gamefaqs.gamespot.com/pc/197873-master-of-orion-ii-battle-at-antares/faqs/16743
	// Section 2

	// Percent chance to hit = 100/(1 + 2^E)
	// where E = -(BA - BD)/16
	// BA = Beam Attack
	// BD = Beam Defense
	BA := attacker.BeamAttack
	BD := defender.BeamDefense
	E := -float64(BA-BD) / 16
	chanceToHit := 1 / (1 + math.Pow(2, E))
	//fmt.Printf("--> BA:%d BD:%d (%.2f)\n", BA, BD, 100*chanceToHit)

	totalDamage := int64(0)
	for _, beam := range attacker.Weapons {
		roll := rand.Float64()

		if roll <= chanceToHit {
			//fmt.Printf("[%d %s] HIT, damage: %d (rolled: %.2f)\n", i, beam.Name, beam.MaxDamage, 100*roll)
			totalDamage += beam.MaxDamage
		} else {
			//fmt.Printf("[%d %s] MISS (rolled: %.2f)\n", i, beam.Name, 100*roll)
		}
	}

	// apply damage to receiving ship

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
