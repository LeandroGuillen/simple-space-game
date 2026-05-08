package game

import (
	_ "embed"
	"encoding/json"
	"fmt"
	. "ssg/model"
)

type Resources struct {
	*ShipPartsCollection  `json:"ship_parts"`
	*BuildingCollection   `json:"building_collection"`
	*NameCollection       `json:"name_collection"`
	*ShipDesignCollection `json:"ship_design_collection"`
	*BeamCollection       `json:"beam_collection"`
	*TechTree             `json:"tech_tree"`
}

type ShipPartsCollection struct {
	HullSizes []HullSize `json:"hull_sizes"`
	Drives    []Drive    `json:"drives"`
	Armors    []Armor    `json:"armors"`
	Computers []Computer `json:"computers"`
	FuelCells []FuelCell `json:"fuel_cells"`
	Shields   []Shield   `json:"shields"`
}

type BuildingCollection struct {
	Buildings []Building `json:"buildings"`
}

type NameCollection struct {
	Stars []string `json:"stars"`
}

type ShipDesignCollection struct {
	ShipDesignsDefs []ShipDesignsDef `json:"ship_design_defs"`
	ShipDesigns     []ShipDesign     `json:"ship_designs"`
}

type BeamCollection struct {
	Beams []Beam `json:"beams"`
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

//go:embed resources/beams.json
var beamsFile []byte

//go:embed resources/tech-tree.json
var techTreeFile []byte

//go:embed resources/techs.json
var techsFile []byte

//go:embed resources/techs.construction.json
var constructionTechsFile []byte

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
	randomShuffle(len(nameCollection.Stars), func(i int, j int) {
		nameCollection.Stars[i], nameCollection.Stars[j] = nameCollection.Stars[j], nameCollection.Stars[i]
	})

	var beamCollection BeamCollection
	if err := json.Unmarshal(beamsFile, &beamCollection); err != nil {
		return err
	}
	r.BeamCollection = &beamCollection

	var shipDesignCollection ShipDesignCollection
	if err := json.Unmarshal(shipDesignsFile, &shipDesignCollection); err != nil {
		return err
	}
	r.ShipDesignCollection = &shipDesignCollection
	for _, def := range shipDesignCollection.ShipDesignsDefs {
		armor, err := ArmorFromName(def.Armor, shipPartsCollection.Armors)
		if err != nil {
			return fmt.Errorf("cannot read ship design data: %w", err)
		}
		drive, err := DriveFromName(def.Drive, shipPartsCollection.Drives)
		if err != nil {
			return fmt.Errorf("cannot read ship design data: %w", err)
		}
		hullSize, err := HullSizeFromName(def.HullSize, shipPartsCollection.HullSizes)
		if err != nil {
			return fmt.Errorf("cannot read ship design data: %w", err)
		}
		computer, err := ComputerFromName(def.Computer, shipPartsCollection.Computers)
		if err != nil {
			return fmt.Errorf("cannot read ship design data: %w", err)
		}
		fuelCell, err := FuelCellFromName(def.FuelCell, shipPartsCollection.FuelCells)
		if err != nil {
			return fmt.Errorf("cannot read ship design data: %w", err)
		}
		shield, err := ShieldFromName(def.Shield, shipPartsCollection.Shields)
		if err != nil {
			return fmt.Errorf("cannot read ship design data: %w", err)
		}

		design := NewShipDesign(
			def.Name,
			[]Beam{},
			[]SpecialSystem{},
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

	var techTree TechTree
	if err := json.Unmarshal(techTreeFile, &techTree); err != nil {
		return err
	}
	r.TechTree = &techTree

	techs, err := loadTechDefinitions(constructionTechsFile)
	if err != nil {
		return err
	}
	r.TechTree.AllTechs = techs

	return nil
}

func loadTechDefinitions(files ...[]byte) ([]Tech, error) {
	techsByName := make(map[string]Tech)
	orderedNames := make([]string, 0)
	for _, file := range files {
		var techs []Tech
		if err := json.Unmarshal(file, &techs); err != nil {
			return nil, err
		}
		for _, tech := range techs {
			if tech.Name == "" {
				continue
			}
			if _, exists := techsByName[tech.Name]; !exists {
				orderedNames = append(orderedNames, tech.Name)
			}
			techsByName[tech.Name] = tech
		}
	}

	techs := make([]Tech, 0, len(orderedNames))
	for _, name := range orderedNames {
		techs = append(techs, techsByName[name])
	}
	return techs, nil
}

func NewResourceManager() (*ResourceManager, error) {
	rm := &ResourceManager{}
	if err := rm.loadDataFromFiles(); err != nil {
		return nil, err
	}
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
		return fmt.Errorf("drive with name %q does not exist", drive)
	}

	found = false
	for _, p := range parts.Computers {
		if computer == p.Name {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("computer with name %q does not exist", computer)
	}

	found = false
	for _, p := range parts.Shields {
		if shield == p.Name {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("shield with name %q does not exist", shield)
	}

	found = false
	for _, p := range parts.Armors {
		if armor == p.Name {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("armor with name %q does not exist", armor)
	}

	found = false
	for _, p := range parts.HullSizes {
		if hullSize == p.Name {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("hull size with name %q does not exist", hullSize)
	}

	found = false
	for _, p := range parts.FuelCells {
		if fuelCell == p.Name {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("fuel cell with name %q does not exist", fuelCell)
	}

	return nil
}

func ArmorFromName(name string, list []Armor) (Armor, error) {
	for _, v := range list {
		if v.Name == name {
			return v, nil
		}
	}
	return Armor{}, fmt.Errorf("%s not found", name)
}

func DriveFromName(name string, list []Drive) (Drive, error) {
	for _, v := range list {
		if v.Name == name {
			return v, nil
		}
	}
	return Drive{}, fmt.Errorf("%s not found", name)
}

func HullSizeFromName(name string, list []HullSize) (HullSize, error) {
	for _, v := range list {
		if v.Name == name {
			return v, nil
		}
	}
	return HullSize{}, fmt.Errorf("%s not found", name)
}

func ComputerFromName(name string, list []Computer) (Computer, error) {
	for _, v := range list {
		if v.Name == name {
			return v, nil
		}
	}
	return Computer{}, fmt.Errorf("%s not found", name)
}

func FuelCellFromName(name string, list []FuelCell) (FuelCell, error) {
	for _, v := range list {
		if v.Name == name {
			return v, nil
		}
	}
	return FuelCell{}, fmt.Errorf("%s not found", name)
}

func ShieldFromName(name string, list []Shield) (Shield, error) {
	for _, v := range list {
		if v.Name == name {
			return v, nil
		}
	}
	return Shield{}, fmt.Errorf("%s not found", name)
}
