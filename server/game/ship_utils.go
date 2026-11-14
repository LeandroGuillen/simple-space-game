package game

import (
	. "ssg/model"
)

func EquipBeam(s *Ship, b Beam) bool {
	s.Beams = append(s.Beams, b)
	return true
}

func EquipBeams(s *Ship, beams []Beam) bool {
	for _, weapon := range beams {
		EquipBeam(s, weapon)
	}
	return true
}

func FireOn(attacker *Ship, defender *Ship, distance int64, attackerPlayer, defenderPlayer *Player) (totalDamage int64) {
	attackBonus := shipRaceAttackBonus(attackerPlayer, attacker.Computer.BeamAttackBonus)
	defenderDefense := defender.BeamDefense + shipRaceDefenseBonus(defenderPlayer, defender.BeamDefense)
	chanceToHit := GetChanceToHit(attacker.Computer, defenderDefense, attackBonus, distance)

	for _, beam := range attacker.Beams {
		roll := randomFloat64()
		if roll <= chanceToHit {
			totalDamage += shieldedHitDamage(GetDamage(beam, distance), defender.Shield.ReduceDamage)
		}
	}

	applyShipDamage(defender, totalDamage)
	return totalDamage
}

// fireMissilesOn launches one missile per loaded rack at the defender during
// auto-resolve and applies the damage. racks is the attacker's per-battle
// magazine; the fixed auto-resolve range does not affect missiles.
func fireMissilesOn(racks []MissileRack, defender *Ship) int64 {
	if !missilesLoaded(racks) {
		return 0
	}
	volley := launchMissiles(racks, defender.MissileEvasion, defender.Shield.ReduceDamage)
	applyShipDamage(defender, volley.damage)
	return volley.damage
}

// applyShipDamage takes damage off armor first, then structure.
func applyShipDamage(defender *Ship, totalDamage int64) {
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
}

// shieldedHitDamage applies a shield's per-hit damage reduction to one beam
// or missile hit. Shields reduce each hit separately, never the whole volley, and can
// absorb a hit entirely but never heal. A missing shield (reduce 0) or a
// negative value leaves the hit unchanged.
func shieldedHitDamage(damage, reduceDamage int64) int64 {
	if reduceDamage > 0 {
		damage -= reduceDamage
	}
	if damage < 0 {
		return 0
	}
	return damage
}

func NewEmptyBattleShip(name string) Ship {
	return NewEmptyShip(
		name,
		HullSize{Name: "BS_1", SizeRating: 5, Space: 250, Marines: 20, BaseSpeed: 5, BaseHP: 50, ComputerHP: 7, DriveHP: 15, ShieldHP: 8},
		Drive{}, Armor{}, Computer{}, FuelCell{}, Shield{},
	)
}

func NewEmptyShip(name string, hull HullSize, drive Drive, armor Armor,
	computer Computer, fuelCell FuelCell, shield Shield) Ship {
	basicDesign := NewShipDesign(
		name,
		[]Beam{},
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
