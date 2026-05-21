package game

import (
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOmniscientPlayerSeesAllStars(t *testing.T) {
	g := newGameWithRaceTraits(t, []string{"omniscient"})
	require.GreaterOrEqual(t, len(g.Galaxy.Stars), 2)

	player := g.Players[0]
	assert.Len(t, player.ExploredStars, 1)

	visible, err := g.GetVisibleGameForPlayer(player.ID)
	require.NoError(t, err)

	named := exploredStarIDs(visible)
	assert.Len(t, named, len(g.Galaxy.Stars))
}

func TestNonOmniscientPlayerOnlySeesExploredStars(t *testing.T) {
	g := newGameWithRaceTraits(t, []string{"creative"})
	require.GreaterOrEqual(t, len(g.Galaxy.Stars), 2)

	player := g.Players[0]
	visible, err := g.GetVisibleGameForPlayer(player.ID)
	require.NoError(t, err)

	assert.Len(t, exploredStarIDs(visible), 1)
}

func TestOmniscientSeesEnemyColoniesWithoutViewerHydration(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	omniscient, err := BuildCustomRace(resources, CustomRaceDesign{Name: "Seers", TraitIDs: []string{"omniscient"}})
	require.NoError(t, err)
	g, err := NewGame(resources, NewGameOptions{
		PlayerName:  "Ada",
		PlayerColor: model.DefaultPlayerColor(),
		NumStars:    4,
		Race:        omniscient,
		NumPlayers:  2,
	})
	require.NoError(t, err)

	var enemyPlanet *model.Planet
	for _, star := range g.Galaxy.Stars {
		for _, planet := range star.Planets {
			if planet.Owner == 2 {
				enemyPlanet = planet
				break
			}
		}
	}
	require.NotNil(t, enemyPlanet)

	visible, err := g.GetVisibleGameForPlayer(1)
	require.NoError(t, err)

	var visibleEnemy *model.Planet
	for _, star := range visible.Galaxy.Stars {
		require.NotEmpty(t, star.Name)
		for _, planet := range star.Planets {
			if planet.ID == enemyPlanet.ID {
				visibleEnemy = planet
				break
			}
		}
	}
	require.NotNil(t, visibleEnemy)
	assert.Equal(t, uint(2), visibleEnemy.Owner)
	assert.Equal(t, enemyPlanet.Name, visibleEnemy.Name)
	assert.Equal(t, enemyPlanet.Government.Name, visibleEnemy.Government.Name)

	var ownPlanet *model.Planet
	for _, star := range visible.Galaxy.Stars {
		for _, planet := range star.Planets {
			if planet.Owner == 1 {
				ownPlanet = planet
				break
			}
		}
	}
	require.NotNil(t, ownPlanet)
	assert.NotZero(t, ownPlanet.Income.Total)
	assert.Equal(t, g.Players[0].Race.Government, ownPlanet.Government.Name)
}
