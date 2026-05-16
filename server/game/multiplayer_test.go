package game

import (
	"sync"
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGameCreatesMultiplePlayerSlotsWithUniqueHomeworlds(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, NewGameOptions{
		PlayerName:  "Ada",
		PlayerColor: model.DefaultPlayerColor(),
		NumStars:    8,
		Race:        DefaultRace("Human"),
		NumPlayers:  3,
	})
	require.NoError(t, err)

	require.Len(t, g.Players, 3)
	assert.Equal(t, uint(1), g.Players[0].ID)
	assert.Equal(t, "Ada", g.Players[0].Name)
	assert.Equal(t, uint(2), g.Players[1].ID)
	assert.Equal(t, "Player 2", g.Players[1].Name)
	assert.Equal(t, uint(3), g.Players[2].ID)
	assert.Equal(t, "Player 3", g.Players[2].Name)

	homeworldStars := map[uint]uint{}
	for _, star := range g.Galaxy.Stars {
		for _, planet := range star.Planets {
			if planet.Owner != 0 && hasBuilding(planet, "Capitol") {
				homeworldStars[planet.Owner] = star.ID
			}
		}
	}
	require.Len(t, homeworldStars, 3)
	assert.Equal(t, uint(1), homeworldStars[1])
	assert.NotEqual(t, homeworldStars[1], homeworldStars[2])
	assert.NotEqual(t, homeworldStars[1], homeworldStars[3])
	assert.NotEqual(t, homeworldStars[2], homeworldStars[3])
}

func TestMultiplayerPlayersOnlySeeTheirStartingSystems(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, NewGameOptions{
		PlayerName:  "Ada",
		PlayerColor: model.DefaultPlayerColor(),
		NumStars:    8,
		Race:        DefaultRace("Human"),
		NumPlayers:  2,
	})
	require.NoError(t, err)

	playerOneVisible, err := g.GetVisibleGameForPlayer(1)
	require.NoError(t, err)
	playerTwoVisible, err := g.GetVisibleGameForPlayer(2)
	require.NoError(t, err)

	playerOneExplored := exploredStarIDs(playerOneVisible)
	playerTwoExplored := exploredStarIDs(playerTwoVisible)
	assert.Len(t, playerOneExplored, 1)
	assert.Len(t, playerTwoExplored, 1)
	assert.NotEqual(t, playerOneExplored[0], playerTwoExplored[0])
}

func TestSubmitTurnWaitsForAllPlayersAndAdvancesOnce(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, NewGameOptions{
		PlayerName:  "Ada",
		PlayerColor: model.DefaultPlayerColor(),
		NumStars:    8,
		Race:        DefaultRace("Human"),
		NumPlayers:  2,
	})
	require.NoError(t, err)

	initialTurn := g.CurrentTurn
	status, advanced, err := g.SubmitTurn(1, model.Commands{})
	require.NoError(t, err)
	assert.False(t, advanced)
	assert.Equal(t, initialTurn, g.CurrentTurn)
	assert.Equal(t, []uint{1}, status.ReadyPlayerIDs)
	assert.Equal(t, []uint{2}, status.WaitingPlayerIDs)

	status, advanced, err = g.SubmitTurn(2, model.Commands{})
	require.NoError(t, err)
	assert.True(t, advanced)
	assert.Equal(t, advanceStardate(initialTurn), g.CurrentTurn)
	assert.Empty(t, status.ReadyPlayerIDs)
	assert.ElementsMatch(t, []uint{1, 2}, status.WaitingPlayerIDs)
}

func TestSubmitTurnReplacesExistingSubmission(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, NewGameOptions{
		PlayerName:  "Ada",
		PlayerColor: model.DefaultPlayerColor(),
		NumStars:    8,
		Race:        DefaultRace("Human"),
		NumPlayers:  2,
	})
	require.NoError(t, err)

	_, advanced, err := g.SubmitTurn(1, model.Commands{Build: []*model.BuildCommand{{PlanetID: 1, Building: "Trade Goods"}}})
	require.NoError(t, err)
	require.False(t, advanced)
	_, advanced, err = g.SubmitTurn(1, model.Commands{})
	require.NoError(t, err)
	require.False(t, advanced)
	assert.Len(t, g.TurnSubmissions, 1)
	assert.Empty(t, g.TurnSubmissions[1].Build)
}

func TestConcurrentSubmitTurnDoesNotDoubleAdvance(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, NewGameOptions{
		PlayerName:  "Ada",
		PlayerColor: model.DefaultPlayerColor(),
		NumStars:    8,
		Race:        DefaultRace("Human"),
		NumPlayers:  2,
	})
	require.NoError(t, err)

	initialTurn := g.CurrentTurn
	var wg sync.WaitGroup
	for _, playerID := range []uint{1, 2} {
		wg.Add(1)
		go func(playerID uint) {
			defer wg.Done()
			_, _, err := g.SubmitTurn(playerID, model.Commands{})
			require.NoError(t, err)
		}(playerID)
	}
	wg.Wait()

	assert.Equal(t, advanceStardate(initialTurn), g.CurrentTurn)
}

func exploredStarIDs(g *Game) []uint {
	ids := []uint{}
	for _, star := range g.Galaxy.Stars {
		if star.Name != "" {
			ids = append(ids, star.ID)
		}
	}
	return ids
}
