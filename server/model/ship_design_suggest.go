package model

import (
	"errors"
	"fmt"
	"slices"
)

// ErrNoSuitableShipPart marks a suggestion that cannot be built because a
// required part category has nothing suitable among the player's unlocks.
var ErrNoSuitableShipPart = errors.New("no suitable ship part")

// ShipDesignPartCatalog is the part catalog a design suggestion chooses from.
// Only entries that are also listed in the player's AvailableTech are used.
type ShipDesignPartCatalog struct {
	HullSizes []HullSize
	Drives    []Drive
	FuelCells []FuelCell
	Armors    []Armor
	Computers []Computer
	Shields   []Shield
	Beams     []Beam
	Missiles  []Missile
}

// ShipDesignPartChoice names the parts picked for a suggested design. At most
// one of Beam and Missile is set (one weapon slot).
type ShipDesignPartChoice struct {
	Hull     string
	Drive    string
	FuelCell string
	Armor    string
	Computer string
	Shield   string
	Beam     string
	Missile  string
}

// SuggestShipDesignParts picks the best available part in every category,
// using only parts listed in avail:
//
//   - hull: preferredHull when it is available with weapon space, otherwise the
//     smallest available hull with space;
//   - drive: highest interstellar speed (must be > 0);
//   - fuel cell: highest autonomy (must be > 0);
//   - armor: highest multiplier; computer: highest beam attack bonus;
//     shield: highest damage reduction (a "None" part is acceptable);
//   - weapon: the researched beam or missile with the highest per-hit damage
//     (beam max damage vs missile warhead damage). A tie goes to the beam,
//     which has unlimited shots. With no researched weapon the slot is empty.
//
// Ties keep the first entry in catalog order. A required category without any
// suitable part returns an error wrapping ErrNoSuitableShipPart.
func SuggestShipDesignParts(avail AvailableTech, cat ShipDesignPartCatalog, preferredHull string) (ShipDesignPartChoice, error) {
	var c ShipDesignPartChoice
	var hullSpace int64
	for _, h := range cat.HullSizes {
		if h.Space <= 0 || !slices.Contains(avail.HullSizes, h.Name) {
			continue
		}
		if h.Name == preferredHull {
			c.Hull = h.Name
			break
		}
		if c.Hull == "" || h.Space < hullSpace {
			c.Hull, hullSpace = h.Name, h.Space
		}
	}
	if c.Hull == "" {
		return c, missingPart("hull")
	}

	var speed int64
	for _, v := range cat.Drives {
		if slices.Contains(avail.Drives, v.Name) && v.InterstellarSpeed > speed {
			speed, c.Drive = v.InterstellarSpeed, v.Name
		}
	}
	if c.Drive == "" {
		return c, missingPart("interstellar drive")
	}

	var autonomy int64
	for _, v := range cat.FuelCells {
		if slices.Contains(avail.FuelCells, v.Name) && v.Autonomy > autonomy {
			autonomy, c.FuelCell = v.Autonomy, v.Name
		}
	}
	if c.FuelCell == "" {
		return c, missingPart("fuel cell with range")
	}

	var armor int64 = -1
	for _, v := range cat.Armors {
		if slices.Contains(avail.Armors, v.Name) && v.Multiplier > armor {
			armor, c.Armor = v.Multiplier, v.Name
		}
	}
	if c.Armor == "" {
		return c, missingPart("armor")
	}

	var computer int64 = -1
	for _, v := range cat.Computers {
		if slices.Contains(avail.Computers, v.Name) && v.BeamAttackBonus > computer {
			computer, c.Computer = v.BeamAttackBonus, v.Name
		}
	}
	if c.Computer == "" {
		return c, missingPart("computer")
	}

	var shield int64 = -1
	for _, v := range cat.Shields {
		if slices.Contains(avail.Shields, v.Name) && v.ReduceDamage > shield {
			shield, c.Shield = v.ReduceDamage, v.Name
		}
	}
	if c.Shield == "" {
		return c, missingPart("shield")
	}

	var damage int64
	for _, v := range cat.Beams {
		if slices.Contains(avail.Beams, v.Name) && v.MaxDamage > damage {
			damage, c.Beam = v.MaxDamage, v.Name
		}
	}
	for _, v := range cat.Missiles {
		if slices.Contains(avail.Missiles, v.Name) && v.Damage > damage {
			damage, c.Beam, c.Missile = v.Damage, "", v.Name
		}
	}
	return c, nil
}

func missingPart(category string) error {
	return fmt.Errorf("%w: no researched %s is available", ErrNoSuitableShipPart, category)
}
