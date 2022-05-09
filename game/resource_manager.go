package game

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"ssg/game/combat"
	"ssg/game/combat/computers"
	"ssg/game/combat/weapons"
	"ssg/game/model"
)

type Resources struct {
	*ShipPartsCollection  `json:"ship_parts"`
	*BuildingCollection   `json:"building_collection"`
	*NameCollection       `json:"name_collection"`
	*ShipDesignCollection `json:"ship_design_collection"`
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

type NameCollection struct {
	Stars []string `json:"stars"`
}

type ShipDesignCollection struct {
	ShipDesignsDefs []combat.ShipDesignsDef `json:"ship_design_defs"`
	ShipDesigns     []combat.ShipDesign     `json:"ship_designs"`
}

type ResourceManager struct {
	Resources
}

//go:embed resources/equipment.json
var equipmentFile []byte

//go:embed resources/buildings.json
var buildingsFile []byte

//go:embed resources/names.json
var namesFile []byte

//go:embed resources/ship_designs.json
var shipDesignsFile []byte

func (r *ResourceManager) loadDataFromFiles() error {
	var shipPartsCollection ShipPartsCollection
	if err := json.Unmarshal(equipmentFile, &shipPartsCollection); err != nil {
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
	if err := json.Unmarshal(buildingsFile, &buildingCollection); err != nil {
		return err
	}
	r.BuildingCollection = &buildingCollection

	var nameCollection NameCollection
	if err := json.Unmarshal(namesFile, &nameCollection); err != nil {
		return err
	}
	r.NameCollection = &nameCollection
	rand.Shuffle(len(nameCollection.Stars), func(i int, j int) {
		nameCollection.Stars[i], nameCollection.Stars[j] = nameCollection.Stars[j], nameCollection.Stars[i]
	})

	var shipDesignCollection ShipDesignCollection
	if err := json.Unmarshal(shipDesignsFile, &shipDesignCollection); err != nil {
		return err
	}
	r.ShipDesignCollection = &shipDesignCollection
	for _, def := range shipDesignCollection.ShipDesignsDefs {
		armor, err := ArmorFromName(def.Armor, shipPartsCollection.Armors)
		if err != nil {
			return errors.New(fmt.Sprintf("cannot read ship design data: %s", armor))
		}
		drive, err := DriveFromName(def.Drive, shipPartsCollection.Drives)
		if err != nil {
			return errors.New(fmt.Sprintf("cannot read ship design data: %s", drive))
		}
		hullSize, err := HullSizeFromName(def.HullSize, shipPartsCollection.HullSizes)
		if err != nil {
			return errors.New(fmt.Sprintf("cannot read ship design data: %s", hullSize))
		}
		computer, err := ComputerFromName(def.Computer, shipPartsCollection.Computers)
		if err != nil {
			return errors.New(fmt.Sprintf("cannot read ship design data: %s", computer))
		}
		fuelCell, err := FuelCellFromName(def.FuelCell, shipPartsCollection.FuelCells)
		if err != nil {
			return errors.New(fmt.Sprintf("cannot read ship design data: %s", fuelCell))
		}
		shield, err := ShieldFromName(def.Shield, shipPartsCollection.Shields)
		if err != nil {
			return errors.New(fmt.Sprintf("cannot read ship design data: %s", shield))
		}

		design := combat.NewShipDesign(
			def.Name,
			[]weapons.Weapon{},
			[]combat.SpecialSystem{},
			hullSize,
			drive,
			armor,
			computer,
			fuelCell,
			shield,
		)

		// these designs are common to all players
		r.ShipDesigns = append(r.ShipDesigns, design)
	}

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
