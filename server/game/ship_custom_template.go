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
		Beam:     shipDesignBeamName(d),
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

	return resources.Resources.resolveShipDesign(def)
}

func shipDesignBeamName(d ShipDesign) string {
	if len(d.Beams) > 0 {
		return d.Beams[0].Name
	}
	return ""
}

// resolveShipDesign supports explicitly weaponless utility templates.
func (resources *Resources) resolveShipDesign(def ShipDesignsDef) (ShipDesign, error) {
	if resources == nil || resources.ShipPartsCollection == nil {
		return ShipDesign{}, fmt.Errorf("ship parts unavailable")
	}
	name := strings.TrimSpace(def.Name)
	if name == "" || len(name) > 80 {
		return ShipDesign{}, fmt.Errorf("invalid ship design name")
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

	if drive.InterstellarSpeed <= 0 {
		return ShipDesign{}, fmt.Errorf("ship design needs an interstellar drive")
	}
	if fuelCell.Autonomy <= 0 {
		return ShipDesign{}, fmt.Errorf("ship design needs fuel range")
	}
	beams := []Beam{}
	if def.Beam != "" {
		if resources.BeamCollection == nil {
			return ShipDesign{}, fmt.Errorf("beam catalog unavailable")
		}
		for _, b := range resources.Beams {
			if b.Name == canonicalBeamName(def.Beam) && b.MaxDamage > 0 {
				beams = append(beams, b)
				break
			}
		}
		if len(beams) == 0 {
			return ShipDesign{}, fmt.Errorf("unknown or ineffective beam %q", def.Beam)
		}
		if hullSize.Space < 1 {
			return ShipDesign{}, fmt.Errorf("ship design needs weapon space")
		}
	}
	design := NewShipDesign(
		name,
		beams,
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
