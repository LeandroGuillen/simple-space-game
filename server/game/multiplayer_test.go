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

func TestNewGameAssignsDistinctPlayerColors(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	// Player 1 picks a color that, under the old slot-based mapping, would
	// have collided with slot 2's default — verify slot 2 (and the rest)
	// still get a distinct color.
	chosen := model.PlayerColors[1]
	g, err := NewGame(resources, NewGameOptions{
		PlayerName:  "Ada",
		PlayerColor: chosen,
		NumStars:    model.MaxPlayers,
		Race:        DefaultRace("Human"),
		NumPlayers:  uint(model.MaxPlayers),
	})
	require.NoError(t, err)
	require.Len(t, g.Players, model.MaxPlayers)

	seen := map[string]uint{}
	for _, p := range g.Players {
		if other, dup := seen[p.Color]; dup {
			t.Fatalf("player %d (%s) shares color %q with player %d", p.ID, p.Name, p.Color, other)
		}
		seen[p.Color] = p.ID
		assert.True(t, model.IsValidPlayerColor(p.Color), "color %q is not in the valid palette", p.Color)
	}
	assert.Equal(t, chosen, g.Players[0].Color)
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

func TestInvalidOrderDoesNotMutateGameState(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, NewGameOptions{
		PlayerName:  "Ada",
		PlayerColor: model.DefaultPlayerColor(),
		NumStars:    8,
		Race:        DefaultRace("Human"),
		NumPlayers:  1,
	})
	require.NoError(t, err)

	homeworld := getPlayerHomeworld(g, 1)
	require.NotNil(t, homeworld)
	initialPopulation := homeworld.Population
	initialMarines := homeworld.Marines
	initialTurn := g.CurrentTurn

	// Submit an invalid colonize order (no pending colony base)
	invalidCommands := model.Commands{
		Colonize: &model.ColonizeCommand{PlanetID: 2},
	}
	_, advanced, err := g.SubmitTurn(1, invalidCommands)

	// Should fail with an error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no pending colony base")

	// Game state should be unchanged
	assert.False(t, advanced)
	assert.Equal(t, initialTurn, g.CurrentTurn)
	assert.Equal(t, initialPopulation, homeworld.Population, "population should not change on rejected turn")
	assert.Equal(t, initialMarines, homeworld.Marines, "marines should not change on rejected turn")

	// Player should not be marked ready
	assert.Empty(t, g.TurnSubmissions)
	assert.Empty(t, g.TurnStatus.ReadyPlayerIDs)
}

func TestInvalidOrderInMultiplayerDoesNotBlockOtherPlayer(t *testing.T) {
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

	// Player 1 submits invalid orders
	invalidCommands := model.Commands{
		Colonize: &model.ColonizeCommand{PlanetID: 999},
	}
	_, advanced, err := g.SubmitTurn(1, invalidCommands)
	require.Error(t, err)
	assert.False(t, advanced)

	// Player 1 should NOT be marked ready
	assert.Empty(t, g.TurnSubmissions)

	// Player 2 can still submit valid orders
	validCommands := model.Commands{}
	_, advanced, err = g.SubmitTurn(2, validCommands)
	require.NoError(t, err)
	assert.False(t, advanced, "should not advance with only 1 of 2 players ready")
	assert.Equal(t, []uint{2}, g.TurnStatus.ReadyPlayerIDs)
	assert.Equal(t, []uint{1}, g.TurnStatus.WaitingPlayerIDs)
}

func getPlayerHomeworld(g *Game, playerID uint) *model.Planet {
	for _, star := range g.Galaxy.Stars {
		for _, planet := range star.Planets {
			if planet.Owner == playerID && hasBuilding(planet, "Capitol") {
				return planet
			}
		}
	}
	return nil
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
