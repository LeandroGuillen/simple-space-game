package game

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"ssg/model"
	"testing"
)

func TestShipWeaponTemplateValidation(t *testing.T) {
	r := testResources(t)
	var def model.ShipDesignsDef
	require.NoError(t, json.Unmarshal([]byte(`{"name":"Armed","drive":"Nuclear","computer":"None","shield":"None","armor":"Titanium","hull_size":"Frigate","fuel_cell":"Standard","beam":"Laser"}`), &def))
	d, err := SaveableCustomShipDesign(r, def)
	require.NoError(t, err)
	require.Len(t, d.Beams, 1)
	require.Equal(t, "Laser", d.Beams[0].Name)
	g := newTestGame(t)
	p := g.Players[0]
	p.Available.Beams = nil
	require.Error(t, validatePlayerCanUseShipDesignParts(p, d))
	require.False(t, g.playerCanBuildShipDesign(p, d))
	p.Available.Beams = []string{"Laser"}
	require.NoError(t, validatePlayerCanUseShipDesignParts(p, d))
	require.True(t, g.playerCanBuildShipDesign(p, d))
}

func TestShipWeaponlessDriveValidation(t *testing.T) {
	g := newTestGame(t)
	require.Error(t, g.CreateShipDesign(1, "Immobile", "None", "None", "None", "Titanium", "Frigate", "Standard"))
}

func TestResearchUnlocksMountableBeams(t *testing.T) {
	resources := testResources(t)
	for _, tech := range resources.TechTree.AllTechs {
		for _, name := range tech.Grants.Unlocks.Beams {
			t.Run(tech.Name+"/"+name, func(t *testing.T) {
				g := newTestGame(t)
				p := g.Players[0]
				g.research(p, tech.Name)
				require.Contains(t, p.Available.Beams, name)
				require.NoError(t, g.CreateShipDesign(p.ID, "Researched weapon", "Nuclear", "None", "None", "Titanium", "Frigate", "Standard", name))
				require.Len(t, p.ShipDesigns, 1)
				design := p.ShipDesigns[0]
				require.Len(t, design.Beams, 1)
				require.Equal(t, name, design.Beams[0].Name)
				require.Positive(t, design.Beams[0].MaxDamage)
				require.Positive(t, GetDamage(design.Beams[0], 0))
				require.True(t, g.playerCanBuildShipDesign(p, design))
			})
		}
	}
}
