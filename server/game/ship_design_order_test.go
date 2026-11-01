package game

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	. "ssg/model"
	"testing"
)

func TestShipDesignOrderValidatesUnlocksAndKeepsExistingDesigns(t *testing.T) {
	g := newTestGame(t)
	p := g.Players[0]
	p.ResearchedTechs = append(p.ResearchedTechs, "Laser Cannon")
	g.hydratePlayerAvailability(p)
	c := &DesignShipCommand{Name: "AI Frigate 1", Hull: "Frigate", Drive: "Nuclear", Armor: "Titanium", Computer: "None", FuelCell: "Standard", Shield: "None", Beam: "Laser"}
	invalid := *c
	invalid.Beam = "Fusion"
	_, _, err := g.SubmitTurn(p.ID, Commands{DesignShip: &invalid})
	require.Error(t, err)
	require.Empty(t, p.ShipDesigns)
	_, advanced, err := g.SubmitTurn(p.ID, Commands{DesignShip: c})
	require.NoError(t, err)
	require.True(t, advanced)
	require.Len(t, p.ShipDesigns, 1)
	assert.Equal(t, "Laser", p.ShipDesigns[0].Beams[0].Name)
	assert.True(t, g.playerCanBuildShipDesign(p, p.ShipDesigns[0]))
	_, _, err = g.SubmitTurn(p.ID, Commands{DesignShip: c})
	require.Error(t, err)
	require.Len(t, p.ShipDesigns, 1)
}
