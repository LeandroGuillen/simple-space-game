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
