package game

import (
	"encoding/json"

	. "ssg/model"
)

// resourcesJSON is the on-wire shape of Resources (matches embedded json tags).
type resourcesJSON struct {
	ShipParts             *ShipPartsCollection   `json:"ship_parts"`
	ShipSystemCollection  *ShipSystemCollection  `json:"ship_system_collection"`
	BuildingCollection    *BuildingCollection    `json:"building_collection"`
	ModifierCollection    *ModifierCollection    `json:"modifier_collection"`
	NameCollection        *NameCollection        `json:"name_collection"`
	ShipDesignCollection  *ShipDesignCollection  `json:"ship_design_collection"`
	BeamCollection        *BeamCollection        `json:"beam_collection"`
	EnvironmentCollection *EnvironmentCollection `json:"environment_collection"`
	TechTree              *TechTree              `json:"tech_tree"`
}

// MarshalJSON encodes Resources using the atomically published ship-design catalog so
// concurrent catalog rebuilds never race JSON reflection on the legacy embedded pointer.
func (r *Resources) MarshalJSON() ([]byte, error) {
	if r == nil {
		return []byte("null"), nil
	}
	coll := r.shipDesignCatalog.Load()
	if coll == nil {
		coll = r.ShipDesignCollection
	}
	return json.Marshal(resourcesJSON{
		ShipParts:             r.ShipPartsCollection,
		ShipSystemCollection:  r.ShipSystemCollection,
		BuildingCollection:    r.BuildingCollection,
		ModifierCollection:    r.ModifierCollection,
		NameCollection:        r.NameCollection,
		ShipDesignCollection:  coll,
		BeamCollection:        r.BeamCollection,
		EnvironmentCollection: r.EnvironmentCollection,
		TechTree:              r.TechTree,
	})
}

// UnmarshalJSON loads Resources and publishes the embedded catalog snapshot once.
func (r *Resources) UnmarshalJSON(data []byte) error {
	var wire resourcesJSON
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	r.ShipPartsCollection = wire.ShipParts
	r.ShipSystemCollection = wire.ShipSystemCollection
	r.BuildingCollection = wire.BuildingCollection
	r.ModifierCollection = wire.ModifierCollection
	r.NameCollection = wire.NameCollection
	r.ShipDesignCollection = wire.ShipDesignCollection
	r.BeamCollection = wire.BeamCollection
	r.EnvironmentCollection = wire.EnvironmentCollection
	r.TechTree = wire.TechTree
	if wire.ShipDesignCollection != nil {
		r.publishShipDesignCatalog(wire.ShipDesignCollection)
	}
	return nil
}
