package game_test

import (
	"ssg/game"
	"ssg/game/model"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewGame(t *testing.T) {
	playerName := "test_player"
	var numStars uint = 1
	g, _ := game.NewGame(playerName, numStars, "#FFFFFF")

	assert.Equal(t, len(g.Players), 1, "there should be only 1 player")
	assert.Equal(t, g.Players[0].Name, playerName, "player name is set up")
}

func TestGame_TurnPassing(t *testing.T) {
	testPlanet := game.NewAverageTerranPlanet()
	testStar := game.NewStar(1, "star_name", model.StarColorYellow, []*model.Planet{testPlanet})
	g, _ := game.NewGame("test_player", 0, "#FFFFFF")
	g.Galaxy.Stars = append(g.Galaxy.Stars, testStar)

	assert.Equal(t, g.CurrentTurn, int64(3500))
	_ = g.Turn(1)
	assert.Equal(t, g.CurrentTurn, int64(3501))
	_ = g.Turn(1)
	_ = g.Turn(1)
	_ = g.Turn(1)
	assert.Equal(t, g.CurrentTurn, int64(3504))
}

func TestGame_TurnPassing_GrowsCapitalPopulationByBaseline(t *testing.T) {
	g, _ := game.NewGame("test_player", 1, "#FFFFFF")

	// CreateCapitalPlanet sets Population=8000 on a Normal-gravity Terran
	// planet, which is the best-case +40 per turn.
	capital := g.Galaxy.Stars[0].Planets[0]
	assert.Equal(t, int64(8000), capital.Population)

	err := g.Turn(1)
	assert.NoError(t, err)
	assert.Equal(t, int64(8040), capital.Population)
}

func TestGame_VisibleSystems_AreScopedPerPlayerExploration(t *testing.T) {
	g, _ := game.NewGame("test_player", 2, "#FFFFFF")

	visible, err := g.GetVisibleGameForPlayer(1)
	assert.NoError(t, err)
	assert.Len(t, visible.Galaxy.Stars, 2)
	assert.NotEmpty(t, visible.Galaxy.Stars[0].Name)
	assert.NotEmpty(t, visible.Galaxy.Stars[0].Planets)
	assert.Empty(t, visible.Galaxy.Stars[1].Name)
	assert.Nil(t, visible.Galaxy.Stars[1].Planets)
}

func TestGame_ExploreStar_RevealsNameAndPlanets(t *testing.T) {
	g, _ := game.NewGame("test_player", 2, "#FFFFFF")

	err := g.ExploreStar(1, 2)
	assert.NoError(t, err)

	visible, err := g.GetVisibleGameForPlayer(1)
	assert.NoError(t, err)
	assert.NotEmpty(t, visible.Galaxy.Stars[1].Name)
	assert.NotEmpty(t, visible.Galaxy.Stars[1].Planets)
}
