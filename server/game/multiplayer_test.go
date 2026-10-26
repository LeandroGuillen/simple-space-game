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


// Helper to create a fleet for a player in tests
func createTestFleet(g *Game, ownerID uint, starID uint) *model.Fleet {
	fleet := &model.Fleet{
		ID:     g.nextFleetID(),
		Owner:  ownerID,
		StarID: starID,
		Ships: []model.FleetShip{
			{Type: "ship_design", Name: "Test Frigate"},
		},
	}
	g.Fleets = append(g.Fleets, fleet)
	if player, err := g.getPlayer(ownerID); err == nil {
		player.Fleets = append(player.Fleets, fleet.ID)
	}
	return fleet
}

// Tests for P1 bug: Bad fleet moves pass End Turn

func TestBadFleetMoveToNonexistentStarRejectedAtSubmit(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, NewGameOptions{
		PlayerName:  "Player 1",
		PlayerColor: model.DefaultPlayerColor(),
		NumStars:    8,
		Race:        DefaultRace("Human"),
		NumPlayers:  2,
	})
	require.NoError(t, err)

	
	// Find player 2's home star
	var p2Star uint
	for _, star := range g.Galaxy.Stars {
		for _, planet := range star.Planets {
			if planet.Owner == 2 {
				p2Star = star.ID
				break
			}
		}
		if p2Star != 0 {
			break
		}
	}
	require.NotZero(t, p2Star)

	// Create a fleet for player 2
	p2Fleet := createTestFleet(g, 2, p2Star)

	// Record initial state
	initialPop := g.Galaxy.Stars[0].Planets[0].Population
	initialTreasury := g.Players[0].Treasury
	initialResearch := g.Players[0].ResearchPoints

	// Player 2 submits a bad move to nonexistent star 999
	_, _, err = g.SubmitTurn(2, model.Commands{
		FleetMove: []*model.FleetMoveCommand{
			{FleetID: p2Fleet.ID, DestinationStarID: 999},
		},
	})

	// Should reject at submit time
	require.Error(t, err, "Bad move should be rejected at submit time")
	assert.Contains(t, err.Error(), "not found", "Error should mention star not found")

	// Player 2 should NOT be marked ready
	assert.NotContains(t, g.TurnStatus.ReadyPlayerIDs, uint(2), "Player 2 should not be ready after bad submission")

	// Player 1 submits valid empty turn
	_, _, err = g.SubmitTurn(1, model.Commands{})
	require.NoError(t, err, "Player 1's valid submission should succeed")

	// State should not have changed (population, treasury, research)
	assert.Equal(t, initialPop, g.Galaxy.Stars[0].Planets[0].Population, "Population should not have grown")
	assert.Equal(t, initialTreasury, g.Players[0].Treasury, "Treasury should not have grown")
	assert.Equal(t, initialResearch, g.Players[0].ResearchPoints, "Research should not have grown")
}

func TestBadFleetMoveToStar0RejectedAtSubmit(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, NewGameOptions{
		PlayerName:  "Player 1",
		PlayerColor: model.DefaultPlayerColor(),
		NumStars:    8,
		Race:        DefaultRace("Human"),
		NumPlayers:  2,
	})
	require.NoError(t, err)

	// Find player 2's home star
	var p2Star uint
	for _, star := range g.Galaxy.Stars {
		for _, planet := range star.Planets {
			if planet.Owner == 2 {
				p2Star = star.ID
				break
			}
		}
		if p2Star != 0 {
			break
		}
	}
	require.NotZero(t, p2Star)

	// Create a fleet for player 2
	p2Fleet := createTestFleet(g, 2, p2Star)

	// Player 2 submits a move to star 0 (invalid)
	_, _, err = g.SubmitTurn(2, model.Commands{
		FleetMove: []*model.FleetMoveCommand{
			{FleetID: p2Fleet.ID, DestinationStarID: 0},
		},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid fleet destination")
	assert.NotContains(t, g.TurnStatus.ReadyPlayerIDs, uint(2))
}

func TestBadFleetMoveOutOfFuelRangeRejectedAtSubmit(t *testing.T) {
	g := fuelTestGame(t)
	
	// Add a second player
	g.Players = append(g.Players, &model.Player{
		ID: 2,
		Available: model.AvailableTech{
			FuelCells: []string{"Standard"},
			Drives:    []string{"Nuclear"},
		},
	})
	
	// Give player 2 a fleet at star 1
	g.Fleets = append(g.Fleets, &model.Fleet{
		ID:     2,
		Owner:  2,
		StarID: 1,
		X:      0,
		Y:      0,
		Ships: []model.FleetShip{
			{Type: "ship_design", Name: "Standard Frigate", FuelCell: "Standard", FuelRangeParsecs: 2},
		},
	})

	g.TurnSubmissions = make(map[uint]model.Commands)
	g.TurnStatus = TurnStatus{
		ReadyPlayerIDs:    []uint{},
		WaitingPlayerIDs:  []uint{1, 2},
		RequiredPlayerIDs: []uint{1, 2},
	}

	// Player 1 tries to move to star 6 (out of range: 2 parsecs limit)
	_, _, err := g.SubmitTurn(1, model.Commands{
		FleetMove: []*model.FleetMoveCommand{
			{FleetID: 1, DestinationStarID: 6},
		},
	})

	// Should reject at submit time
	require.Error(t, err)
	assert.Contains(t, err.Error(), "outside fleet fuel range")
	assert.NotContains(t, g.TurnStatus.ReadyPlayerIDs, uint(1))

	// Player 2 should still be able to submit successfully
	_, _, err = g.SubmitTurn(2, model.Commands{})
	require.NoError(t, err)
}

func TestBadFleetMoveWithInvalidShipIndexRejectedAtSubmit(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, NewGameOptions{
		PlayerName:  "Player 1",
		PlayerColor: model.DefaultPlayerColor(),
		NumStars:    8,
		Race:        DefaultRace("Human"),
		NumPlayers:  2,
	})
	require.NoError(t, err)

	// Find player 2's home star
	var p2Star uint
	for _, star := range g.Galaxy.Stars {
		for _, planet := range star.Planets {
			if planet.Owner == 2 {
				p2Star = star.ID
				break
			}
		}
		if p2Star != 0 {
			break
		}
	}
	require.NotZero(t, p2Star)

	// Create a fleet for player 2
	p2Fleet := createTestFleet(g, 2, p2Star)

	// Ensure fleet has only 1 ship
	require.Len(t, p2Fleet.Ships, 1)

	// Player 2 tries to move ship at index 5 (but fleet only has 1 ship)
	badIndexes := []uint{5}
	_, _, err = g.SubmitTurn(2, model.Commands{
		FleetMove: []*model.FleetMoveCommand{
			{
				FleetID:         p2Fleet.ID,
				DestinationStarID: p2Star + 1,
				ShipIndexes:     &badIndexes,
			},
		},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
	assert.NotContains(t, g.TurnStatus.ReadyPlayerIDs, uint(2))
}

func TestBadFleetMoveDuplicateShipIndexesRejectedAtSubmit(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, NewGameOptions{
		PlayerName:  "Player 1",
		PlayerColor: model.DefaultPlayerColor(),
		NumStars:    8,
		Race:        DefaultRace("Human"),
		NumPlayers:  2,
	})
	require.NoError(t, err)

	// Find player 2's home star
	var p2Star uint
	for _, star := range g.Galaxy.Stars {
		for _, planet := range star.Planets {
			if planet.Owner == 2 {
				p2Star = star.ID
				break
			}
		}
		if p2Star != 0 {
			break
		}
	}
	require.NotZero(t, p2Star)

	// Create a fleet for player 2 with long fuel range
	p2Fleet := &model.Fleet{
		ID:     g.nextFleetID(),
		Owner:  2,
		StarID: p2Star,
		Ships: []model.FleetShip{
			{Type: "ship_design", Name: "Test Frigate 1", FuelCell: "Thorium", UnlimitedFuelRange: true},
			{Type: "ship_design", Name: "Test Frigate 2", FuelCell: "Thorium", UnlimitedFuelRange: true},
		},
	}
	g.Fleets = append(g.Fleets, p2Fleet)
	if player, err := g.getPlayer(2); err == nil {
		player.Fleets = append(player.Fleets, p2Fleet.ID)
	}

	// Find two other stars for player 2 to move to
	var destStar1, destStar2 uint
	for _, star := range g.Galaxy.Stars {
		if star.ID != p2Fleet.StarID {
			if destStar1 == 0 {
				destStar1 = star.ID
			} else if destStar2 == 0 {
				destStar2 = star.ID
				break
			}
		}
	}
	require.NotZero(t, destStar1)
	require.NotZero(t, destStar2)

	// Try to use ship index 0 in two different moves
	indexes := []uint{0}
	_, _, err = g.SubmitTurn(2, model.Commands{
		FleetMove: []*model.FleetMoveCommand{
			{FleetID: p2Fleet.ID, DestinationStarID: destStar1, ShipIndexes: &indexes},
			{FleetID: p2Fleet.ID, DestinationStarID: destStar2, ShipIndexes: &indexes},
		},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "assigned to multiple fleet moves")
	assert.NotContains(t, g.TurnStatus.ReadyPlayerIDs, uint(2))
}

func TestBadInvadeCommandRejectedAtSubmit(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, NewGameOptions{
		PlayerName:  "Player 1",
		PlayerColor: model.DefaultPlayerColor(),
		NumStars:    8,
		Race:        DefaultRace("Human"),
		NumPlayers:  2,
	})
	require.NoError(t, err)

	
	// Find player 2's home star
	var p2Star uint
	for _, star := range g.Galaxy.Stars {
		for _, planet := range star.Planets {
			if planet.Owner == 2 {
				p2Star = star.ID
				break
			}
		}
		if p2Star != 0 {
			break
		}
	}
	require.NotZero(t, p2Star)

	// Create a fleet for player 2
	p2Fleet := createTestFleet(g, 2, p2Star)

	// Try to invade a nonexistent planet
	_, _, err = g.SubmitTurn(2, model.Commands{
		Invade: []*model.InvadeCommand{
			{FleetID: p2Fleet.ID, TargetPlanetID: 999},
		},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.NotContains(t, g.TurnStatus.ReadyPlayerIDs, uint(2))
}

func TestOtherPlayerValidSubmitAdvancesTurnAfterBadMoveRejection(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, NewGameOptions{
		PlayerName:  "Player 1",
		PlayerColor: model.DefaultPlayerColor(),
		NumStars:    8,
		Race:        DefaultRace("Human"),
		NumPlayers:  2,
	})
	require.NoError(t, err)

	initialTurn := g.CurrentTurn

	// Find player 2's home star
	var p2Star uint
	for _, star := range g.Galaxy.Stars {
		for _, planet := range star.Planets {
			if planet.Owner == 2 {
				p2Star = star.ID
				break
			}
		}
		if p2Star != 0 {
			break
		}
	}
	require.NotZero(t, p2Star)

	// Create a fleet for player 2
	p2Fleet := createTestFleet(g, 2, p2Star)

	// Player 2 submits a bad move (rejected)
	_, _, err = g.SubmitTurn(2, model.Commands{
		FleetMove: []*model.FleetMoveCommand{
			{FleetID: p2Fleet.ID, DestinationStarID: 999},
		},
	})
	require.Error(t, err)

	// Player 2 should not be marked ready
	assert.NotContains(t, g.TurnStatus.ReadyPlayerIDs, uint(2))

	// Player 1 submits valid commands
	_, advanced, err := g.SubmitTurn(1, model.Commands{})
	require.NoError(t, err)
	assert.False(t, advanced, "Turn should not advance with only one player ready")

	// Player 2 resubmits with valid commands
	_, advanced, err = g.SubmitTurn(2, model.Commands{})
	require.NoError(t, err)
	assert.True(t, advanced, "Turn should advance when both players ready")

	// Turn should have advanced
	assert.Greater(t, g.CurrentTurn, initialTurn, "Turn should have advanced")
}
