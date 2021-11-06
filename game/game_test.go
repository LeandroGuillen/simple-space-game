package game_test

import (
	"ssg/game"
	"ssg/game/builder"
	. "ssg/model"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	playerName := "test_player"
	var numStars uint = 1
	g := game.New(playerName, numStars)

	assert.Equal(t, len(g.Players), 1, "there should be only 1 player")
	assert.Equal(t, g.Players[0].Name, playerName, "player name is set up")
}

func TestGame_Turn(t *testing.T) {
	testPlanet := builder.NewAverageTerranPlanet()
	testStar := builder.NewStar(1, "star_name", StarColorYellow, []*Planet{testPlanet})
	g := game.New("test_player", 0)
	g.Galaxy.Stars = append(g.Galaxy.Stars, testStar)

	assert.Equal(t, g.CurrentTurn, uint64(0))
	_ = g.Turn(1)
	assert.Equal(t, g.CurrentTurn, uint64(1))
	_ = g.Turn(1)
	_ = g.Turn(1)
	_ = g.Turn(1)
	assert.Equal(t, g.CurrentTurn, uint64(4))
}

func TestGame_Turn_PopGrowth(t *testing.T) {
	testPlanet := builder.NewAverageTerranPlanet()
	testStar := builder.NewStar(1, "star_name", StarColorYellow, []*Planet{testPlanet})
	g := game.New("test_player", 0)
	g.Galaxy.Stars = append(g.Galaxy.Stars, testStar)

	game.ColonizePlanet2(1, testPlanet)

	assert.Equal(t, int64(1000), testPlanet.CurrentPopMillis)
	assert.Equal(t, int64(1), testPlanet.CurrentPop)
	_ = g.Turn(1)
	assert.Equal(t, int64(1050), testPlanet.CurrentPopMillis)
	assert.Equal(t, int64(1), testPlanet.CurrentPop)
	for i := 0; i < 20; i++ {
		_ = g.Turn(1)
	}
	assert.Equal(t, int64(2050), testPlanet.CurrentPopMillis)
	assert.Equal(t, int64(2), testPlanet.CurrentPop)
}

func TestGame_Turn_PopGrowthStarvation(t *testing.T) {
	testPlanet := builder.NewAverageTerranPlanet()
	testStar := builder.NewStar(1, "star_name", StarColorYellow, []*Planet{testPlanet})
	g := game.New("test_player", 0)
	g.Galaxy.Stars = append(g.Galaxy.Stars, testStar)

	game.ColonizePlanet2(1, testPlanet)
	testPlanet.PopGrowth = -100
	testPlanet.CurrentPop = 3
	testPlanet.Farmers = 1
	testPlanet.Workers = 1
	testPlanet.Scientists = 1
	testPlanet.CurrentPopMillis = 3000

	_ = g.Turn(1)
	assert.Equal(t, int64(2900), testPlanet.CurrentPopMillis)
	assert.Equal(t, int64(2), testPlanet.CurrentPop)
	for i := 0; i < 10; i++ {
		_ = g.Turn(1)
	}
	assert.Equal(t, int64(1900), testPlanet.CurrentPopMillis)
	assert.Equal(t, int64(1), testPlanet.CurrentPop)
}
