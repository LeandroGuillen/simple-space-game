package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"ssg/game"
	"ssg/model"

	"github.com/stretchr/testify/require"
)

func TestBattleDemoWritesFusionRaider(t *testing.T) {
	t.Chdir(t.TempDir())
	require.NoError(t, os.MkdirAll(filepath.Join("data", "scenarios"), 0o755))
	require.NotPanics(t, main)
	data, err := os.ReadFile(filepath.Join("data", "scenarios", "battle-demo.json"))
	require.NoError(t, err)
	var loaded game.Game
	require.NoError(t, json.Unmarshal(data, &loaded))
	require.Len(t, loaded.Players, 2)
	require.Len(t, loaded.Players[1].ShipDesigns, 1)
	raider := loaded.Players[1].ShipDesigns[0]
	require.Equal(t, "Fusion Raider", raider.Name)
	require.Equal(t, []model.Beam{
		{Name: "Fusion Beam", MinDamage: 2, MaxDamage: 8},
		{Name: "Fusion Beam", MinDamage: 2, MaxDamage: 8},
	}, raider.Beams)
}
