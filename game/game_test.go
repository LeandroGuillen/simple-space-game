package game_test

import (
	"ssg/game"
	"ssg/game/builder"
	"ssg/game/model"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	playerName := "test_player"
	var numStars uint = 1
	g := game.NewGame(playerName, numStars)

	assert.Equal(t, len(g.Players), 1, "there should be only 1 player")
	assert.Equal(t, g.Players[0].Name, playerName, "player name is set up")
}

func TestGame_Turn(t *testing.T) {
	testPlanet := builder.NewAverageTerranPlanet()
	testStar := builder.NewStar(1, "star_name", model.StarColorYellow, []*model.Planet{testPlanet})
	g := game.NewGame("test_player", 0)
	g.Galaxy.Stars = append(g.Galaxy.Stars, testStar)

	assert.Equal(t, g.CurrentTurn, uint64(0))
	_ = g.Turn(1)
	assert.Equal(t, g.CurrentTurn, uint64(1))
	_ = g.Turn(1)
	_ = g.Turn(1)
	_ = g.Turn(1)
	assert.Equal(t, g.CurrentTurn, uint64(4))
}
