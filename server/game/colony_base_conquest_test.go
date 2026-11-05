package game

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	. "ssg/model"
	"testing"
)

func colonyBaseConquestGame(t *testing.T) (*Game, *Planet, *Planet, *Planet) {
	t.Helper()
	g := multiPlayerGame(t, 2, 40)
	home, err := g.getPlanet(1)
	require.NoError(t, err)
	homeTarget := &Planet{ID: 90001, ParentStar: home.ParentStar}
	var colony, lostTarget *Planet
	for _, star := range g.Galaxy.Stars {
		if star.ID == home.ParentStar {
			star.Planets = append(star.Planets, homeTarget)
		} else if colony == nil {
			colony = &Planet{ID: 90002, ParentStar: star.ID}
			lostTarget = &Planet{ID: 90003, ParentStar: star.ID}
			star.Planets = append(star.Planets, colony, lostTarget)
		}
	}
	require.NotNil(t, colony)
	require.NotNil(t, lostTarget)
	home.Owner, colony.Owner = 1, 1
	homeTarget.Owner, lostTarget.Owner = 0, 0
	homeTarget.Environment, lostTarget.Environment = PlanetEnvironmentDesert, PlanetEnvironmentDesert
	require.Equal(t, home.ParentStar, homeTarget.ParentStar)
	require.Equal(t, colony.ParentStar, lostTarget.ParentStar)
	require.NotEqual(t, home.ParentStar, colony.ParentStar)
	g.PendingColonyBases = map[uint][]uint{1: {home.ID, colony.ID, colony.ID}}
	return g, colony, homeTarget, lostTarget
}

func TestColonyBaseCapturePreservesOtherBases(t *testing.T) {
	g, colony, target, _ := colonyBaseConquestGame(t)
	g.capturePlanet(colony, 2, &GroundForce{OwnerID: 2})
	assert.Equal(t, []uint{1}, g.PendingColonyBases[1])
	require.NoError(t, g.processColonizeCommand(1, &ColonizeCommand{PlanetID: target.ID}))
	assert.Equal(t, uint(1), target.Owner)
	assert.Empty(t, g.PendingColonyBases[1])
}

func TestColonyBaseSubmitRejectsLostSource(t *testing.T) {
	g, colony, homeTarget, lostTarget := colonyBaseConquestGame(t)
	g.capturePlanet(colony, 2, &GroundForce{OwnerID: 2})
	// Simulate stale entries in an older save, even after capture pruning is fixed.
	g.PendingColonyBases[1] = []uint{1, colony.ID}
	_, advanced, err := g.SubmitTurn(1, Commands{Colonize: &ColonizeCommand{PlanetID: lostTarget.ID}})
	require.Error(t, err)
	assert.False(t, advanced)
	_, advanced, err = g.SubmitTurn(1, Commands{Colonize: &ColonizeCommand{PlanetID: homeTarget.ID}})
	require.NoError(t, err)
	assert.False(t, advanced)
	_, advanced, err = g.SubmitTurn(2, Commands{})
	require.NoError(t, err)
	assert.True(t, advanced)
	target, err := g.getPlanet(homeTarget.ID)
	require.NoError(t, err)
	assert.Equal(t, uint(1), target.Owner)
	assert.Empty(t, g.PendingColonyBases[1])
}

func TestColonyBaseTurnPrunesIndividualSources(t *testing.T) {
	g, colony, _, _ := colonyBaseConquestGame(t)
	colony.Owner = 2
	_, _, err := g.SubmitTurn(1, Commands{})
	require.NoError(t, err)
	_, advanced, err := g.SubmitTurn(2, Commands{})
	require.NoError(t, err)
	require.True(t, advanced)
	assert.Equal(t, []uint{1}, g.PendingColonyBases[1])
}

func TestColonyBaseSelectsTargetSystem(t *testing.T) {
	g, colony, target, _ := colonyBaseConquestGame(t)
	require.NoError(t, g.validateColonizeCommand(1, &ColonizeCommand{PlanetID: target.ID}, nil))
	require.NoError(t, g.processColonizeCommand(1, &ColonizeCommand{PlanetID: target.ID}))
	assert.Equal(t, []uint{colony.ID, colony.ID}, g.PendingColonyBases[1])
}
