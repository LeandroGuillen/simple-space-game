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

// Guards the Beam JSON tags: beams.json uses camelCase keys, so a tag rename
// would silently zero every beam's damage.
func TestResourceManagerLoadsBeamDamage(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)
	var laser *Beam
	for i := range resources.Beams {
		if resources.Beams[i].Name == "Laser" {
			laser = &resources.Beams[i]
		}
	}
	require.NotNil(t, laser, "Laser beam missing from beams.json")
	assert.Equal(t, int64(1), laser.MinDamage)
	assert.Equal(t, int64(4), laser.MaxDamage)
}

func TestEveryResearchedBeamHasDamage(t *testing.T) {
	resources := testResources(t)
	unlocked := make(map[string]bool)
	for _, tech := range resources.TechTree.AllTechs {
		for _, name := range tech.Grants.Unlocks.Beams {
			unlocked[name] = true
			t.Run(tech.Name+"/"+name, func(t *testing.T) {
				var beam *Beam
				for i := range resources.Beams {
					if resources.Beams[i].Name == name {
						beam = &resources.Beams[i]
						break
					}
				}
				require.NotNil(t, beam, "tech unlock must exist in beams.json")
				assert.Positive(t, beam.MinDamage)
				assert.Positive(t, beam.MaxDamage)
				assert.GreaterOrEqual(t, beam.MaxDamage, beam.MinDamage)
			})
		}
	}
	require.NotEmpty(t, unlocked)
	for _, beam := range resources.Beams {
		assert.True(t, unlocked[beam.Name], "catalog beam %q must be researchable", beam.Name)
		if beam.Name == "Fusion Beam" {
			assert.Equal(t, int64(2), beam.MinDamage)
			assert.Equal(t, int64(8), beam.MaxDamage)
		}
	}
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

	require.NotEmpty(t, resources.ShipDesignCatalog())
	design := resources.ShipDesignCatalog()[0]
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

func TestResourceManagerLoadsPlanetEnvironmentBaseFood(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	cases := map[PlanetEnvironment]int64{
		PlanetEnvironmentBarren: 0,
		PlanetEnvironmentToxic:  0,
		PlanetEnvironmentDesert: 1,
		PlanetEnvironmentArid:   1,
		PlanetEnvironmentTerran: 2,
		PlanetEnvironmentOcean:  2,
		PlanetEnvironmentGaia:   3,
	}
	for env, want := range cases {
		value, ok := resources.EnvironmentCollection.baseFoodPerFarmer(env)
		require.Truef(t, ok, "environment %s should be data-driven", env)
		assert.Equalf(t, want, value, "base food per farmer for %s", env)
	}

	// Uncolonizable environments are absent from the data.
	_, ok := resources.EnvironmentCollection.baseFoodPerFarmer(PlanetEnvironmentGasGiant)
	assert.False(t, ok)
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
