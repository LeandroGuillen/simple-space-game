package game

import (
	"math"

	. "ssg/model"
)

// missileAttack is MoO2's fixed Missile Attack rating. Missiles roll against
// the target's Missile Evasion with the beam sigmoid, but neither distance,
// the attacker's computer nor race beam bonuses enter the roll.
const missileAttack int64 = 50

// defaultMissileAmmo is the MoO2 classic x5 magazine, used when a missile
// definition does not set its own ammo.
const defaultMissileAmmo = 5

// missileRoll draws the hit roll for one missile. Tests replace it to force
// hits or misses.
var missileRoll = randomFloat64

// missileHitChance is P(hit) = 1 / (1 + 2^(-(MA - ME)/16)) with MA = 50.
// It is 0.897 against an ME 0 target.
func missileHitChance(missileEvasion int64) float64 {
	return 1 / (1 + math.Pow(2, -float64(missileAttack-missileEvasion)/16))
}

// newMissileRacks loads a full magazine into each of the design's racks for
// one battle.
func newMissileRacks(missiles []Missile) []MissileRack {
	if len(missiles) == 0 {
		return nil
	}
	racks := make([]MissileRack, 0, len(missiles))
	for _, m := range missiles {
		ammo := m.Ammo
		if ammo <= 0 {
			ammo = defaultMissileAmmo
		}
		racks = append(racks, MissileRack{Name: m.Name, Damage: m.Damage, AmmoLeft: ammo, AmmoMax: ammo})
	}
	return racks
}

// missilesLoaded reports whether any rack still has ammo.
func missilesLoaded(racks []MissileRack) bool {
	for _, r := range racks {
		if r.AmmoLeft > 0 {
			return true
		}
	}
	return false
}

// missileVolley is the outcome of launchMissiles.
type missileVolley struct {
	damage   int64
	hits     int
	misses   int
	launched int
	name     string
}

// launchMissiles fires one missile from every rack with ammo at a target
// with the given missile evasion and shield reduction. Missiles hit on
// launch: each spends one ammo, rolls the evasion hit chance, and a hit
// deals the warhead damage minus the shield's per-hit reduction. Range does
// not matter. Both tactical fire and auto-resolve use this helper.
func launchMissiles(racks []MissileRack, missileEvasion, shieldReduceDamage int64) missileVolley {
	volley := missileVolley{}
	chance := missileHitChance(missileEvasion)
	for i := range racks {
		rack := &racks[i]
		if rack.AmmoLeft <= 0 {
			continue
		}
		rack.AmmoLeft--
		volley.launched++
		if volley.name == "" {
			volley.name = rack.Name
		}
		if missileRoll() <= chance {
			volley.hits++
			volley.damage += shieldedHitDamage(rack.Damage, shieldReduceDamage)
		} else {
			volley.misses++
		}
	}
	return volley
}
