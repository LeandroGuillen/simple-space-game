package game

import (
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartingCapitalCommandPoints(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: DefaultRace("Human")})
	require.NoError(t, err)

	player := g.Players[0]
	g.HydrateCommandPoints()

	assert.Equal(t, int64(7), player.CommandPointsCapacity, "5 base + 1 colony + Star Base (+1)")
	assert.Equal(t, int64(0), player.CommandPointsUsed)
	assert.Equal(t, int64(7), player.CommandPointsRemaining)
	assert.Equal(t, int64(0), player.CommandOverdrawBC)
}

func TestCommandPointOverdrawChargesTreasury(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: DefaultRace("Human")})
	require.NoError(t, err)

	player := g.Players[0]
	treasuryBefore := player.Treasury

	for range 8 {
		g.addShipToOrbitFleet(player.ID, 1, model.FleetShip{
			Type: "colony_ship",
			Name: "Colony Ship",
		})
	}

	g.HydrateCommandPoints()
	assert.Equal(t, int64(7), player.CommandPointsCapacity)
	assert.Equal(t, int64(8), player.CommandPointsUsed)
	assert.Equal(t, int64(-1), player.CommandPointsRemaining)
	assert.Equal(t, int64(10), player.CommandOverdrawBC)

	require.NoError(t, g.Turn(player.ID, model.Commands{}))

	colonies, err := g.GetColonyList(player.ID)
	require.NoError(t, err)
	var colonyIncome int64
	for _, colony := range colonies {
		colonyIncome += colony.Income.Total
	}

	assert.Equal(t, treasuryBefore+colonyIncome-10, player.Treasury)
}

func TestShipCommandPointCostColonyShip(t *testing.T) {
	assert.Equal(t, int64(1), shipCommandPointCost(model.FleetShip{Type: "colony_ship"}))
	assert.Equal(t, int64(0), shipCommandPointCost(model.FleetShip{Type: "frigate", Name: "Scout"}))
}
