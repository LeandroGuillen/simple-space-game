package game

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/require"
)

func TestLegacyFusionTemplateLoadsAndSavesCanonicalBeam(t *testing.T) {
	resources := testResources(t)
	path := filepath.Join(t.TempDir(), "custom-ship-designs.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"custom_ship_designs":[{
		"name":"Legacy Raider","drive":"Fusion","computer":"None","shield":"None",
		"armor":"Titanium","hull_size":"Frigate","fuel_cell":"Standard","beam":"Fusion"
	}]}`), 0o600))
	store := NewShipStore(path)
	designs, err := store.Load(resources)
	require.NoError(t, err)
	require.Len(t, designs, 1)
	require.Equal(t, "Fusion", designs[0].Drive.Name)
	require.Equal(t, []model.Beam{{Name: "Fusion Beam", MinDamage: 2, MaxDamage: 8}}, designs[0].Beams)
	require.NoError(t, store.SaveAll(resources, designs))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var saved CustomShipDesignCollection
	require.NoError(t, json.Unmarshal(data, &saved))
	require.Equal(t, "Fusion Beam", saved.CustomShipDesigns[0].Beam)
	for _, beam := range resources.BeamOptions {
		require.NotEqual(t, "Fusion", beam.Name, "legacy alias must not be offered as a new weapon")
	}
}

func TestLegacyFusionSaveRemainsBuildable(t *testing.T) {
	for _, unlock := range []string{"Fusion", "Fusion Beam"} {
		t.Run(unlock, func(t *testing.T) {
			g := newTestGame(t)
			p := g.Players[0]
			g.research(p, "Fusion Beam")
			require.NoError(t, g.CreateShipDesign(p.ID, "Legacy Raider", "Nuclear", "None", "None", "Titanium", "Frigate", "Standard", "Fusion Beam"))
			p.ShipDesigns[0].Beams[0].Name = "Fusion"
			p.Available.Beams = []string{unlock}
			data, err := g.MarshalSave()
			require.NoError(t, err)
			var loaded Game
			require.NoError(t, json.Unmarshal(data, &loaded))
			resources := testResources(t)
			require.NoError(t, ValidateLoadedGame(&loaded, resources))
			HydrateLoadedGame(&loaded, resources)
			player := loaded.Players[0]
			design := player.ShipDesigns[0]
			require.Equal(t, "Fusion Beam", design.Beams[0].Name)
			require.Equal(t, []string{"Fusion Beam"}, player.Available.Beams)
			require.True(t, loaded.playerCanBuildShipDesign(player, design))
			require.NoError(t, validatePlayerCanUseShipDesignParts(player, design))
			loaded.hydratePlayerAvailability(player)
			require.True(t, loaded.playerCanBuildShipDesign(player, design), "research refresh must keep the legacy design buildable")
			ship := materializeBattleShip(&loaded, player, model.FleetShip{Name: "Raider", DesignName: design.Name})
			require.Equal(t, []model.Beam{{Name: "Fusion Beam", MinDamage: 2, MaxDamage: 8}}, ship.Beams)
			require.Equal(t, int64(8), GetDamage(ship.Beams[0], 0))
		})
	}
}

func TestLegacyBattleDemoScenarioKeepsFusionDamage(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "data", "scenarios", "battle-demo.json"))
	require.NoError(t, err)
	var loaded Game
	require.NoError(t, json.Unmarshal(data, &loaded))
	resources := testResources(t)
	require.NoError(t, ValidateLoadedGame(&loaded, resources))
	HydrateLoadedGame(&loaded, resources)
	player := loaded.Players[1]
	ship := materializeBattleShip(&loaded, player, model.FleetShip{Name: "Raider", DesignName: "Fusion Raider"})
	require.Len(t, ship.Beams, 2)
	for _, beam := range ship.Beams {
		require.Equal(t, model.Beam{Name: "Fusion Beam", MinDamage: 2, MaxDamage: 8}, beam)
		require.Equal(t, int64(8), GetDamage(beam, 0))
	}
}
