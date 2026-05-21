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

func TestNewResourceManagerLoadsHumanBuiltinRace(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)
	require.Len(t, resources.BuiltInRaces, 13)
	assert.Equal(t, "Human", resources.BuiltInRaces[0].Name)
	assert.True(t, resources.BuiltInRaces[0].BuiltIn)
	assert.True(t, resources.BuiltInRaces[0].Playable)
}

func TestResourceManagerLoadsBuildableStarterFrigate(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	require.NotEmpty(t, resources.ShipDesigns)
	design := resources.ShipDesigns[0]
	assert.Equal(t, "Flunker I", design.Name)
	assert.Equal(t, "Frigate", design.HullSize.Name)
	assert.Equal(t, "Nuclear", design.Drive.Name)
	assert.Equal(t, "Titanium", design.Armor.Name)
	assert.Equal(t, "None", design.Computer.Name)
	assert.Equal(t, "None", design.Shield.Name)
	assert.Equal(t, "Standard", design.FuelCell.Name)

	g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: DefaultPlayerColor(), Race: DefaultRace("Ada")})
	require.NoError(t, err)
	player := g.Players[0]

	assert.Contains(t, player.Available.HullSizes, design.HullSize.Name)
	assert.Contains(t, player.Available.Drives, design.Drive.Name)
	assert.Contains(t, player.Available.Armors, design.Armor.Name)
	assert.Contains(t, player.Available.Computers, design.Computer.Name)
	assert.Contains(t, player.Available.Shields, design.Shield.Name)
	assert.Contains(t, player.Available.FuelCells, design.FuelCell.Name)
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
