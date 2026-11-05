package game

import (
	"encoding/json"
	"testing"

	. "ssg/model"

	"github.com/stretchr/testify/require"
)

func TestResourcesMarshalJSONUsesPublishedCatalog(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	custom := ShipDesign{
		Name:     "JSON Marshal Scout",
		HullSize: HullSize{Name: "Frigate"},
		Drive:    Drive{Name: "Nuclear"},
		Armor:    Armor{Name: "Titanium"},
		Computer: Computer{Name: "None"},
		FuelCell: FuelCell{Name: "Standard"},
		Shield:   Shield{Name: "None"},
	}
	resources.ApplyMergedShipDesignCatalog([]ShipDesign{custom})

	before, err := json.Marshal(&resources.Resources)
	require.NoError(t, err)

	var beforeMap map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(before, &beforeMap))
	require.Contains(t, beforeMap, "ship_design_collection")
	require.Contains(t, beforeMap, "ship_parts")
	require.Contains(t, beforeMap, "tech_tree")

	var coll struct {
		ShipDesigns []ShipDesign `json:"ship_designs"`
	}
	require.NoError(t, json.Unmarshal(beforeMap["ship_design_collection"], &coll))
	found := false
	for _, d := range coll.ShipDesigns {
		if d.Name == custom.Name {
			found = true
			break
		}
	}
	require.True(t, found, "merged custom design must appear in marshaled catalog")

	// Round-trip decode must republish the catalog for readers.
	var decoded Resources
	require.NoError(t, json.Unmarshal(before, &decoded))
	require.Equal(t, resources.ShipDesignCatalog(), decoded.ShipDesignCatalog())
}

func TestResourcesMarshalJSONMatchesLegacyTopLevelKeys(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	payload, err := json.Marshal(&resources.Resources)
	require.NoError(t, err)

	var keys map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(payload, &keys))
	require.ElementsMatch(t, []string{
		"ship_parts",
		"ship_system_collection",
		"building_collection",
		"modifier_collection",
		"name_collection",
		"ship_design_collection",
		"beam_collection",
		"environment_collection",
		"tech_tree",
	}, keysFromMap(keys))
}

func keysFromMap(m map[string]json.RawMessage) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
