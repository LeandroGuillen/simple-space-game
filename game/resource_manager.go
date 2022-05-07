package game

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"ssg/game/combat"
	"ssg/game/combat/computers"
	"ssg/game/model"
)

type Resources struct {
	*ShipPartsCollection `json:"ship_parts"`
	*BuildingCollection  `json:"building_collection"`
}

type ShipPartsCollection struct {
	HullSizes []combat.HullSize    `json:"hull_sizes"`
	Drives    []combat.Drive       `json:"drives"`
	Armors    []combat.Armor       `json:"armors"`
	Computers []computers.Computer `json:"computers"`
	FuelCells []combat.FuelCell    `json:"fuel_cells"`
	Shields   []combat.Shield      `json:"shields"`
}

type BuildingCollection struct {
	Buildings []model.Building `json:"buildings"`
}

type ResourceManager struct {
	Resources
}

//go:embed resources/equipment.json
var equipmentFile []byte

//go:embed resources/buildings.json
var buildingsFile []byte

func (r *ResourceManager) loadDataFromFiles() error {
	var shipPartsCollection ShipPartsCollection
	err := json.Unmarshal(equipmentFile, &shipPartsCollection)
	if err != nil {
		return err
	}
	r.ShipPartsCollection = &ShipPartsCollection{
		HullSizes: shipPartsCollection.HullSizes,
		Drives:    shipPartsCollection.Drives,
		Armors:    shipPartsCollection.Armors,
		Computers: shipPartsCollection.Computers,
		FuelCells: shipPartsCollection.FuelCells,
		Shields:   shipPartsCollection.Shields,
	}

	var buildingCollection BuildingCollection
	err = json.Unmarshal(buildingsFile, &buildingCollection)
	if err != nil {
		return err
	}
	r.BuildingCollection = &buildingCollection
	return nil
}

func NewResourceManager(g *Game) (*ResourceManager, error) {
	rm := &ResourceManager{}
	if err := rm.loadDataFromFiles(); err != nil {
		return nil, err
	}
	g.Resources = rm.Resources
	return rm, nil
}

func (r *ResourceManager) canICreateThisDesign(
	drive,
	computer,
	shield,
	armor,
	hullSize,
	fuelCell string) error {

	parts := r.ShipPartsCollection

	found := false
	for _, p := range parts.Drives {
		if drive == p.Name {
			found = true
		}
	}
	if !found {
		return errors.New(fmt.Sprintf("drive with name '%s' does not exist", drive))
	}

	found = false
	for _, p := range parts.Computers {
		if computer == p.Name {
			found = true
		}
	}
	if !found {
		return errors.New(fmt.Sprintf("computer with name '%s' does not exist", computer))
	}

	found = false
	for _, p := range parts.Shields {
		if shield == p.Name {
			found = true
		}
	}
	if !found {
		return errors.New(fmt.Sprintf("shield with name '%s' does not exist", shield))
	}

	found = false
	for _, p := range parts.Armors {
		if armor == p.Name {
			found = true
		}
	}
	if !found {
		return errors.New(fmt.Sprintf("armor with name '%s' does not exist", armor))
	}

	found = false
	for _, p := range parts.HullSizes {
		if hullSize == p.Name {
			found = true
		}
	}
	if !found {
		return errors.New(fmt.Sprintf("hull size with name '%s' does not exist", hullSize))
	}

	found = false
	for _, p := range parts.FuelCells {
		if fuelCell == p.Name {
			found = true
		}
	}
	if !found {
		return errors.New(fmt.Sprintf("fuel cell with name '%s' does not exist", fuelCell))
	}

	return nil
}

func ArmorFromName(name string, list []combat.Armor) (combat.Armor, error) {
	for _, v := range list {
		if v.Name == name {
			return v, nil
		}
	}
	return combat.Armor{}, errors.New(fmt.Sprintf("%s not found", name))
}

func DriveFromName(name string, list []combat.Drive) (combat.Drive, error) {
	for _, v := range list {
		if v.Name == name {
			return v, nil
		}
	}
	return combat.Drive{}, errors.New(fmt.Sprintf("%s not found", name))
}

func HullSizeFromName(name string, list []combat.HullSize) (combat.HullSize, error) {
	for _, v := range list {
		if v.Name == name {
			return v, nil
		}
	}
	return combat.HullSize{}, errors.New(fmt.Sprintf("%s not found", name))
}

func ComputerFromName(name string, list []computers.Computer) (computers.Computer, error) {
	for _, v := range list {
		if v.Name == name {
			return v, nil
		}
	}
	return computers.Computer{}, errors.New(fmt.Sprintf("%s not found", name))
}

func FuelCellFromName(name string, list []combat.FuelCell) (combat.FuelCell, error) {
	for _, v := range list {
		if v.Name == name {
			return v, nil
		}
	}
	return combat.FuelCell{}, errors.New(fmt.Sprintf("%s not found", name))
}

func ShieldFromName(name string, list []combat.Shield) (combat.Shield, error) {
	for _, v := range list {
		if v.Name == name {
			return v, nil
		}
	}
	return combat.Shield{}, errors.New(fmt.Sprintf("%s not found", name))
}
