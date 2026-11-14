package game

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	. "ssg/model"
	"strings"
	"sync/atomic"
)

type Resources struct {
	*ShipPartsCollection   `json:"ship_parts"`
	*ShipSystemCollection  `json:"ship_system_collection"`
	*BuildingCollection    `json:"building_collection"`
	*ModifierCollection    `json:"modifier_collection"`
	*NameCollection        `json:"name_collection"`
	*ShipDesignCollection  `json:"ship_design_collection"`
	*BeamCollection        `json:"beam_collection"`
	*MissileCollection     `json:"missile_collection"`
	*EnvironmentCollection `json:"environment_collection"`
	*TechTree              `json:"tech_tree"`
	shipDesignCatalog      atomic.Pointer[ShipDesignCollection]
}

type ShipPartsCollection struct {
	BeamOptions []Beam `json:"beams"`
	// MissileOptions lists missile racks for the editor's weapon slot.
	MissileOptions []Missile  `json:"missiles"`
	HullSizes      []HullSize `json:"hull_sizes"`
	Drives         []Drive    `json:"drives"`
	Armors         []Armor    `json:"armors"`
	Computers      []Computer `json:"computers"`
	FuelCells      []FuelCell `json:"fuel_cells"`
	Shields        []Shield   `json:"shields"`
}

type ShipSystemCollection struct {
	ShipSystems []ShipSystem `json:"ship_systems"`
}

type BuildingCollection struct {
	Buildings []Building `json:"buildings"`
}

type ModifierCollection struct {
	Modifiers []ModifierDefinition `json:"modifiers"`
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

type MissileCollection struct {
	Missiles []Missile `json:"missiles"`
}

// PlanetEnvironmentDef holds the data-driven gameplay attributes of a planet
// environment, keyed by the short enum name (e.g. "Terran", "Barren").
type PlanetEnvironmentDef struct {
	Name              string `json:"name"`
	BaseFoodPerFarmer int64  `json:"base_food_per_farmer"`
	Description       string `json:"description"`
}

type EnvironmentCollection struct {
	Environments []PlanetEnvironmentDef `json:"environments"`
}

// baseFoodPerFarmer returns the configured base food per farmer for the given
// environment, or (0, false) when the environment is not described in the data.
func (c *EnvironmentCollection) baseFoodPerFarmer(env PlanetEnvironment) (int64, bool) {
	if c == nil {
		return 0, false
	}
	name := strings.TrimPrefix(env.String(), "PlanetEnvironment")
	for _, def := range c.Environments {
		if def.Name == name {
			return def.BaseFoodPerFarmer, true
		}
	}
	return 0, false
}

type ResourceManager struct {
	Resources
	BuiltInRaces       []RaceDesign
	BuiltinShipDesigns []ShipDesign
}

//go:embed resources/equipment.json
var equipmentFile []byte

//go:embed resources/ship-systems.json
var shipSystemsFile []byte

//go:embed resources/buildings.json
var buildingsFile []byte

//go:embed resources/modifiers.json
var modifiersFile []byte

//go:embed resources/names.json
var namesFile []byte

//go:embed resources/ship_designs.json
var shipDesignsFile []byte

//go:embed resources/beams.json
var beamsFile []byte

//go:embed resources/missiles.json
var missilesFile []byte

//go:embed resources/tech-tree.json
var techTreeFile []byte

//go:embed resources/techs.starting.json
var startingTechsFile []byte

//go:embed resources/techs.buildings.json
var buildingTechsFile []byte

//go:embed resources/techs.ship-parts.json
var shipPartTechsFile []byte

//go:embed resources/techs.ship-weapons.json
var shipWeaponTechsFile []byte

//go:embed resources/techs.ship-systems.json
var shipSystemTechsFile []byte

//go:embed resources/techs.empire-effects.json
var empireEffectTechsFile []byte

//go:embed resources/races.json
var racesFile []byte

//go:embed resources/planet-environments.json
var planetEnvironmentsFile []byte

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

	var shipSystemCollection ShipSystemCollection
	if err := json.Unmarshal(shipSystemsFile, &shipSystemCollection); err != nil {
		return err
	}
	r.ShipSystemCollection = &shipSystemCollection

	var buildingCollection BuildingCollection
	if err := json.Unmarshal(buildingsFile, &buildingCollection); err != nil {
		return err
	}
	r.BuildingCollection = &buildingCollection

	var environmentCollection EnvironmentCollection
	if err := json.Unmarshal(planetEnvironmentsFile, &environmentCollection); err != nil {
		return err
	}
	r.EnvironmentCollection = &environmentCollection

	var modifierCollection ModifierCollection
	if err := json.Unmarshal(modifiersFile, &modifierCollection); err != nil {
		return err
	}
	r.ModifierCollection = &modifierCollection

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
	r.ShipPartsCollection.BeamOptions = slices.Clone(beamCollection.Beams)

	var missileCollection MissileCollection
	if err := json.Unmarshal(missilesFile, &missileCollection); err != nil {
		return fmt.Errorf("cannot read missiles.json: %w", err)
	}
	r.MissileCollection = &missileCollection
	r.ShipPartsCollection.MissileOptions = slices.Clone(missileCollection.Missiles)

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
	r.BuiltinShipDesigns = slices.Clone(r.ShipDesigns)
	r.Resources.publishShipDesignCatalog(r.ShipDesignCollection)

	var techTree TechTree
	if err := json.Unmarshal(techTreeFile, &techTree); err != nil {
		return err
	}
	r.TechTree = &techTree

	techs, err := loadTechDefinitions(
		startingTechsFile,
		buildingTechsFile,
		shipPartTechsFile,
		shipWeaponTechsFile,
		shipSystemTechsFile,
		empireEffectTechsFile,
	)
	if err != nil {
		return err
	}
	r.TechTree.AllTechs = techs
	if err := r.validateTechDefinitions(); err != nil {
		return err
	}
	annotateWeaponRequiredTechs(r.ShipPartsCollection, r.TechTree)

	var raceCollection RaceCollection
	if err := json.Unmarshal(racesFile, &raceCollection); err != nil {
		return fmt.Errorf("cannot read races.json: %w", err)
	}
	r.BuiltInRaces = raceCollection.Races
	if err := r.validateBuiltInRaces(); err != nil {
		return err
	}

	return nil
}

func (r *ResourceManager) validateBuiltInRaces() error {
	if r == nil {
		return nil
	}
	seen := map[string]struct{}{}
	norm := make([]RaceDesign, 0, len(r.BuiltInRaces))
	for _, d := range r.BuiltInRaces {
		trimmed := strings.TrimSpace(d.Name)
		if trimmed == "" {
			return fmt.Errorf("built-in race name is required")
		}
		if len(trimmed) > 32 {
			return fmt.Errorf("built-in race name %q exceeds 32 characters", trimmed)
		}
		lower := strings.ToLower(trimmed)
		if _, dup := seen[lower]; dup {
			return fmt.Errorf("duplicate built-in race name %q", trimmed)
		}
		seen[lower] = struct{}{}

		mr, err := BuildCustomRace(r, RaceDesign{Name: trimmed, Government: d.Government, TraitIDs: d.TraitIDs})
		if err != nil {
			return fmt.Errorf("built-in race %q: %w", trimmed, err)
		}
		computedPlayable := racePlayable(r, mr.Government, mr.TraitIDs)
		if d.Playable != computedPlayable {
			return fmt.Errorf("built-in race %q: playable field is %v but computed playability is %v", trimmed, d.Playable, computedPlayable)
		}
		norm = append(norm, RaceDesign{
			Name:       mr.Name,
			Government: mr.Government,
			TraitIDs:   append([]string{}, mr.TraitIDs...),
			Playable:   computedPlayable,
			BuiltIn:    true,
		})
	}
	r.BuiltInRaces = norm
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
			if _, exists := techsByName[tech.Name]; exists {
				return nil, fmt.Errorf("duplicate tech definition %q", tech.Name)
			}
			techsByName[tech.Name] = tech
			orderedNames = append(orderedNames, tech.Name)
		}
	}

	techs := make([]Tech, 0, len(orderedNames))
	for _, name := range orderedNames {
		techs = append(techs, techsByName[name])
	}
	return techs, nil
}

func (r *ResourceManager) validateTechDefinitions() error {
	var validationErrors []error
	techNames := make(map[string]bool)
	for _, tech := range r.TechTree.AllTechs {
		techNames[tech.Name] = true
		if err := r.validateUnlocks(tech.Name, tech.Grants.Unlocks); err != nil {
			validationErrors = append(validationErrors, err)
		}
		if err := r.validateModifiers("tech", tech.Name, tech.Grants.Modifiers); err != nil {
			validationErrors = append(validationErrors, err)
		}
	}

	for _, building := range r.Buildings {
		if err := r.validateModifiers("building", building.Name, building.Grants.Modifiers); err != nil {
			validationErrors = append(validationErrors, err)
		}
	}

	for _, subject := range r.TechTree.Subjects {
		for _, level := range subject.Levels {
			for _, techName := range level.AvailableTechs {
				if !techNames[techName] {
					validationErrors = append(validationErrors, fmt.Errorf("tech tree references undefined tech %q", techName))
				}
			}
		}
	}
	if err := r.validateGovernments(); err != nil {
		validationErrors = append(validationErrors, err)
	}
	return errors.Join(validationErrors...)
}

func (r *ResourceManager) validateGovernments() error {
	var validationErrors []error
	defaults := 0
	for _, government := range governmentTechs(&r.Resources) {
		if government.Default {
			if !isBaseGovernment(government) {
				validationErrors = append(validationErrors, fmt.Errorf("advanced government %q cannot be default", government.Name))
			}
			defaults++
		}
		for _, upgraded := range government.Upgrades {
			tech, ok := findGovernmentTech(&r.Resources, upgraded)
			if !ok {
				validationErrors = append(validationErrors, fmt.Errorf("government %q upgrades missing government %q", government.Name, upgraded))
				continue
			}
			if !isGovernmentTech(tech) {
				validationErrors = append(validationErrors, fmt.Errorf("government %q upgrades non-government %q", government.Name, upgraded))
			}
		}
	}
	if defaults != 1 {
		validationErrors = append(validationErrors, fmt.Errorf("expected exactly one default base government, found %d", defaults))
	}
	if defaultGovernmentName(&r.Resources) != "Dictatorship" {
		validationErrors = append(validationErrors, fmt.Errorf("default government must be Dictatorship"))
	}
	return errors.Join(validationErrors...)
}

func (r *ResourceManager) validateModifiers(sourceType, sourceName string, modifiers []Modifier) error {
	var validationErrors []error
	for _, modifier := range modifiers {
		if !modifierDefinitionExists(r.Modifiers, modifier.Name) {
			validationErrors = append(validationErrors, fmt.Errorf("%s %q grants undefined modifier %q", sourceType, sourceName, modifier.Name))
		}
	}
	return errors.Join(validationErrors...)
}

func modifierDefinitionExists(modifiers []ModifierDefinition, name string) bool {
	for _, modifier := range modifiers {
		if modifier.Name == name {
			return true
		}
	}
	return false
}

func (r *ResourceManager) validateUnlocks(techName string, unlocks Unlocks) error {
	var validationErrors []error
	for _, name := range unlocks.Buildings {
		if !buildingExists(r.Buildings, name) {
			validationErrors = append(validationErrors, fmt.Errorf("tech %q unlocks missing building %q", techName, name))
		}
	}
	for _, name := range unlocks.HullSizes {
		if !hullSizeExists(r.HullSizes, name) {
			validationErrors = append(validationErrors, fmt.Errorf("tech %q unlocks missing hull size %q", techName, name))
		}
	}
	for _, name := range unlocks.Drives {
		if !driveExists(r.Drives, name) {
			validationErrors = append(validationErrors, fmt.Errorf("tech %q unlocks missing drive %q", techName, name))
		}
	}
	for _, name := range unlocks.Armors {
		if !armorExists(r.Armors, name) {
			validationErrors = append(validationErrors, fmt.Errorf("tech %q unlocks missing armor %q", techName, name))
		}
	}
	for _, name := range unlocks.Computers {
		if !computerExists(r.Computers, name) {
			validationErrors = append(validationErrors, fmt.Errorf("tech %q unlocks missing computer %q", techName, name))
		}
	}
	for _, name := range unlocks.FuelCells {
		if !fuelCellExists(r.FuelCells, name) {
			validationErrors = append(validationErrors, fmt.Errorf("tech %q unlocks missing fuel cell %q", techName, name))
		}
	}
	for _, name := range unlocks.Shields {
		if !shieldExists(r.Shields, name) {
			validationErrors = append(validationErrors, fmt.Errorf("tech %q unlocks missing shield %q", techName, name))
		}
	}
	for _, name := range unlocks.Beams {
		if !beamExists(r.Beams, name) {
			validationErrors = append(validationErrors, fmt.Errorf("tech %q unlocks missing beam %q", techName, name))
		}
	}
	for _, name := range unlocks.Missiles {
		if _, ok := r.missileByName(name); !ok {
			validationErrors = append(validationErrors, fmt.Errorf("tech %q unlocks missing missile %q", techName, name))
		}
	}
	for _, name := range unlocks.ShipSystems {
		if !shipSystemExists(r.ShipSystems, name) {
			validationErrors = append(validationErrors, fmt.Errorf("tech %q unlocks missing ship system %q", techName, name))
		}
	}
	return errors.Join(validationErrors...)
}

func buildingExists(buildings []Building, name string) bool {
	for _, building := range buildings {
		if building.Name == name {
			return true
		}
	}
	return false
}

func hullSizeExists(hullSizes []HullSize, name string) bool {
	for _, hullSize := range hullSizes {
		if hullSize.Name == name {
			return true
		}
	}
	return false
}

func driveExists(drives []Drive, name string) bool {
	for _, drive := range drives {
		if drive.Name == name {
			return true
		}
	}
	return false
}

func armorExists(armors []Armor, name string) bool {
	for _, armor := range armors {
		if armor.Name == name {
			return true
		}
	}
	return false
}

func computerExists(computers []Computer, name string) bool {
	for _, computer := range computers {
		if computer.Name == name {
			return true
		}
	}
	return false
}

func fuelCellExists(fuelCells []FuelCell, name string) bool {
	for _, fuelCell := range fuelCells {
		if fuelCell.Name == name {
			return true
		}
	}
	return false
}

func shieldExists(shields []Shield, name string) bool {
	for _, shield := range shields {
		if shield.Name == name {
			return true
		}
	}
	return false
}

func beamExists(beams []Beam, name string) bool {
	for _, beam := range beams {
		if beam.Name == name {
			return true
		}
	}
	return false
}

func shipSystemExists(shipSystems []ShipSystem, name string) bool {
	for _, shipSystem := range shipSystems {
		if shipSystem.Name == name {
			return true
		}
	}
	return false
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
