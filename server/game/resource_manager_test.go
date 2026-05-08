package game

import (
	"encoding/json"
	"testing"

	. "ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceManagerLoadsShipSystems(t *testing.T) {
	var shipSystems ShipSystemCollection
	require.NoError(t, json.Unmarshal(shipSystemsFile, &shipSystems))

	assert.True(t, shipSystemExists(shipSystems.ShipSystems, "Reinforced Hull"))
	assert.True(t, shipSystemExists(shipSystems.ShipSystems, "Anti-Missile Rockets"))
}

func TestValidateTechDefinitionsRequiresShipSystemUnlocksToExist(t *testing.T) {
	resources := &ResourceManager{
		Resources: Resources{
			ModifierCollection:   &ModifierCollection{},
			BuildingCollection:   &BuildingCollection{},
			ShipPartsCollection:  &ShipPartsCollection{},
			ShipSystemCollection: &ShipSystemCollection{},
			BeamCollection:       &BeamCollection{},
			TechTree: &TechTree{
				AllTechs: []Tech{
					{
						Name: "Incomplete Tech",
						Grants: Grants{
							Unlocks: Unlocks{ShipSystems: []string{"Missing Ship System"}},
						},
					},
				},
			},
		},
	}
	err := resources.validateTechDefinitions()

	require.Error(t, err)
	assert.Contains(t, err.Error(), `tech "Incomplete Tech" unlocks missing ship system "Missing Ship System"`)
}

func TestValidateTechDefinitionsReportsAllMissingDefinitions(t *testing.T) {
	resources := &ResourceManager{
		Resources: Resources{
			ModifierCollection:   &ModifierCollection{},
			BuildingCollection:   &BuildingCollection{},
			ShipPartsCollection:  &ShipPartsCollection{},
			ShipSystemCollection: &ShipSystemCollection{},
			BeamCollection:       &BeamCollection{},
			TechTree: &TechTree{
				AllTechs: []Tech{
					{
						Name: "Incomplete Tech",
						Grants: Grants{
							Unlocks: Unlocks{
								Buildings:   []string{"Missing Building"},
								ShipSystems: []string{"Missing Ship System"},
							},
							Modifiers: []Modifier{{Name: "missing_modifier"}},
						},
					},
				},
				Subjects: []TechSubject{
					{
						Name: "Test Subject",
						Levels: []TechLevel{
							{
								Name:           "Test Level",
								AvailableTechs: []string{"Missing Tech A", "Missing Tech B"},
							},
						},
					},
				},
			},
		},
	}

	err := resources.validateTechDefinitions()

	require.Error(t, err)
	assert.Contains(t, err.Error(), `tech "Incomplete Tech" unlocks missing building "Missing Building"`)
	assert.Contains(t, err.Error(), `tech "Incomplete Tech" unlocks missing ship system "Missing Ship System"`)
	assert.Contains(t, err.Error(), `tech "Incomplete Tech" grants undefined modifier "missing_modifier"`)
	assert.Contains(t, err.Error(), `tech tree references undefined tech "Missing Tech A"`)
	assert.Contains(t, err.Error(), `tech tree references undefined tech "Missing Tech B"`)
}
