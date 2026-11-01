package game

import (
	"reflect"
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
				FleetID:           p2Fleet.ID,
				DestinationStarID: p2Star + 1,
				ShipIndexes:       &badIndexes,
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

func TestFleetCannotMoveAndInvadeInSameTurn(t *testing.T) {
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

	// Find player 1's home star and create a fleet
	var p1Star uint
	for _, star := range g.Galaxy.Stars {
		for _, planet := range star.Planets {
			if planet.Owner == 1 {
				p1Star = star.ID
				break
			}
		}
		if p1Star != 0 {
			break
		}
	}
	require.NotZero(t, p1Star)

	// Create a fleet for player 1 with transport ships (unlimited fuel range)
	p1Fleet := &model.Fleet{
		ID:     g.nextFleetID(),
		Owner:  1,
		StarID: p1Star,
		Ships: []model.FleetShip{
			{Type: "transport", Name: "Transport 1", FuelCell: "Thorium", UnlimitedFuelRange: true},
		},
	}
	g.Fleets = append(g.Fleets, p1Fleet)
	if player, err := g.getPlayer(1); err == nil {
		player.Fleets = append(player.Fleets, p1Fleet.ID)
	}

	// Find player 2's home planet as invasion target
	var p2Planet uint
	for _, star := range g.Galaxy.Stars {
		for _, planet := range star.Planets {
			if planet.Owner == 2 {
				p2Planet = planet.ID
				break
			}
		}
		if p2Planet != 0 {
			break
		}
	}
	require.NotZero(t, p2Planet)

	// Find another star to move to
	var destStar uint
	for _, star := range g.Galaxy.Stars {
		if star.ID != p1Star {
			destStar = star.ID
			break
		}
	}
	require.NotZero(t, destStar)

	// Try to submit both a move and an invade for the same fleet
	_, _, err = g.SubmitTurn(1, model.Commands{
		FleetMove: []*model.FleetMoveCommand{
			{FleetID: p1Fleet.ID, DestinationStarID: destStar},
		},
		Invade: []*model.InvadeCommand{
			{FleetID: p1Fleet.ID, TargetPlanetID: p2Planet},
		},
	})

	// Should be rejected at submit time
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot invade and move in the same turn")
	assert.NotContains(t, g.TurnStatus.ReadyPlayerIDs, uint(1))
}

// Tests for submit-time validation of research, mutation, and colonize orders.

func TestBadSelectResearchUnknownTechRejectedAtSubmit(t *testing.T) {
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

	home := getPlayerHomeworld(g, 1)
	require.NotNil(t, home)
	initialPop := home.Population
	initialTreasury := g.Players[0].Treasury

	_, _, err = g.SubmitTurn(1, model.Commands{
		SelectResearch: &model.SelectResearchCommand{TechName: "No Such Tech"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
	assert.NotContains(t, g.TurnStatus.ReadyPlayerIDs, uint(1))
	assert.Empty(t, g.TurnSubmissions)
	assert.Equal(t, initialPop, home.Population)
	assert.Equal(t, initialTreasury, g.Players[0].Treasury)

	_, _, err = g.SubmitTurn(2, model.Commands{})
	require.NoError(t, err)
	assert.Contains(t, g.TurnStatus.ReadyPlayerIDs, uint(2))
}

func TestBadSelectResearchAlreadyCompleteRejectedAtSubmit(t *testing.T) {
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

	// Completing one tech in a non-force-all level marks the level complete for Humans.
	g.research(g.Players[0], "Anti-Missile Rockets")

	_, _, err = g.SubmitTurn(1, model.Commands{
		SelectResearch: &model.SelectResearchCommand{TechName: "Fighter Bays"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already complete")
	assert.NotContains(t, g.TurnStatus.ReadyPlayerIDs, uint(1))
}

func TestBadSelectResearchOutOfOrderRejectedAtSubmit(t *testing.T) {
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

	// Automated Factory is in Advanced Construction; Advanced Engineering must come first.
	_, _, err = g.SubmitTurn(1, model.Commands{
		SelectResearch: &model.SelectResearchCommand{TechName: "Automated Factory"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires")
	assert.Contains(t, err.Error(), "completed first")
	assert.NotContains(t, g.TurnStatus.ReadyPlayerIDs, uint(1))
}

func TestBadMutateRaceNoPointsRejectedAtSubmit(t *testing.T) {
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

	require.Equal(t, 0, g.Players[0].EvolutionaryMutationPoints)

	_, _, err = g.SubmitTurn(1, model.Commands{
		MutateRace: &model.MutateRaceCommand{TraitIDs: []string{}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no evolutionary mutation points")
	assert.NotContains(t, g.TurnStatus.ReadyPlayerIDs, uint(1))
}

func TestBadMutateRaceAlreadyUsedRejectedAtSubmit(t *testing.T) {
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

	g.Players[0].EvolutionaryMutationPoints = 4
	g.Players[0].HasUsedEvolutionaryMutation = true

	_, _, err = g.SubmitTurn(1, model.Commands{
		MutateRace: &model.MutateRaceCommand{TraitIDs: []string{}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already been used")
	assert.NotContains(t, g.TurnStatus.ReadyPlayerIDs, uint(1))
}

func TestBadMutateRaceInvalidTraitsRejectedAtSubmit(t *testing.T) {
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

	g.Players[0].EvolutionaryMutationPoints = 4

	_, _, err = g.SubmitTurn(1, model.Commands{
		MutateRace: &model.MutateRaceCommand{TraitIDs: []string{"not-a-real-trait"}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid race mutation")
	assert.NotContains(t, g.TurnStatus.ReadyPlayerIDs, uint(1))
}

func nextTestPlanetID(g *Game) uint {
	maxID := uint(0)
	for _, star := range g.Galaxy.Stars {
		for _, planet := range star.Planets {
			if planet != nil && planet.ID > maxID {
				maxID = planet.ID
			}
		}
	}
	return maxID + 1
}

func multiplayerColonyShipAtHome(t *testing.T, g *Game, playerID uint) (*model.Fleet, *model.Planet) {
	t.Helper()
	home := getPlayerHomeworld(g, playerID)
	require.NotNil(t, home)
	fleet := &model.Fleet{
		ID:     g.nextFleetID(),
		Owner:  playerID,
		StarID: home.ParentStar,
		Ships: []model.FleetShip{
			{Type: "colony_ship", Name: "Colony Ship"},
		},
	}
	g.Fleets = append(g.Fleets, fleet)
	if player, err := g.getPlayer(playerID); err == nil {
		player.Fleets = append(player.Fleets, fleet.ID)
	}
	return fleet, home
}

func findUnownedColonizableInSystem(g *Game, starID, excludePlanetID uint) *model.Planet {
	star, err := g.getStar(starID)
	if err != nil {
		return nil
	}
	for _, planet := range star.Planets {
		if planet == nil || planet.ID == excludePlanetID {
			continue
		}
		if planet.Owner == 0 &&
			planet.Environment != model.PlanetEnvironmentGasGiant &&
			planet.Environment != model.PlanetEnvironmentAsteroidBelt {
			return planet
		}
	}
	return nil
}

func TestBadColonizeNonexistentPlanetRejectedAtSubmit(t *testing.T) {
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

	fleet, _ := multiplayerColonyShipAtHome(t, g, 1)

	_, _, err = g.SubmitTurn(1, model.Commands{
		Colonize: &model.ColonizeCommand{FleetID: fleet.ID, PlanetID: 99999},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.NotContains(t, g.TurnStatus.ReadyPlayerIDs, uint(1))
}

func TestBadColonizeOtherSystemRejectedAtSubmit(t *testing.T) {
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

	fleet, home := multiplayerColonyShipAtHome(t, g, 1)
	p2Home := getPlayerHomeworld(g, 2)
	require.NotNil(t, p2Home)
	require.NotEqual(t, home.ParentStar, p2Home.ParentStar)

	_, _, err = g.SubmitTurn(1, model.Commands{
		Colonize: &model.ColonizeCommand{FleetID: fleet.ID, PlanetID: p2Home.ID},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "same system")
	assert.NotContains(t, g.TurnStatus.ReadyPlayerIDs, uint(1))
}

func TestBadColonizeOwnedPlanetRejectedAtSubmit(t *testing.T) {
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

	fleet, home := multiplayerColonyShipAtHome(t, g, 1)

	_, _, err = g.SubmitTurn(1, model.Commands{
		Colonize: &model.ColonizeCommand{FleetID: fleet.ID, PlanetID: home.ID},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already colonized")
	assert.NotContains(t, g.TurnStatus.ReadyPlayerIDs, uint(1))
}

func TestBadColonizeGasGiantRejectedAtSubmit(t *testing.T) {
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

	fleet, home := multiplayerColonyShipAtHome(t, g, 1)
	star, err := g.getStar(home.ParentStar)
	require.NoError(t, err)

	gas := &model.Planet{
		ID:          nextTestPlanetID(g),
		ParentStar:  home.ParentStar,
		Environment: model.PlanetEnvironmentGasGiant,
		Size:        model.PlanetSizeHuge,
	}
	star.Planets = append(star.Planets, gas)

	_, _, err = g.SubmitTurn(1, model.Commands{
		Colonize: &model.ColonizeCommand{FleetID: fleet.ID, PlanetID: gas.ID},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot colonize this planet type")
	assert.NotContains(t, g.TurnStatus.ReadyPlayerIDs, uint(1))
}

func TestBadColonizeAsteroidRejectedAtSubmit(t *testing.T) {
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

	fleet, home := multiplayerColonyShipAtHome(t, g, 1)
	star, err := g.getStar(home.ParentStar)
	require.NoError(t, err)

	belt := &model.Planet{
		ID:          nextTestPlanetID(g),
		ParentStar:  home.ParentStar,
		Environment: model.PlanetEnvironmentAsteroidBelt,
		Size:        model.PlanetSizeLarge,
	}
	star.Planets = append(star.Planets, belt)

	_, _, err = g.SubmitTurn(1, model.Commands{
		Colonize: &model.ColonizeCommand{FleetID: fleet.ID, PlanetID: belt.ID},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot colonize this planet type")
	assert.NotContains(t, g.TurnStatus.ReadyPlayerIDs, uint(1))
}

func TestBadColonizeFleetNotInOrbitRejectedAtSubmit(t *testing.T) {
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

	fleet, home := multiplayerColonyShipAtHome(t, g, 1)
	fleet.DestinationStarID = home.ParentStar + 1
	fleet.TurnsRemaining = 3
	fleet.TotalTurns = 3

	target := findUnownedColonizableInSystem(g, home.ParentStar, home.ID)
	if target == nil {
		star, err := g.getStar(home.ParentStar)
		require.NoError(t, err)
		target = &model.Planet{
			ID:          nextTestPlanetID(g),
			ParentStar:  home.ParentStar,
			Environment: model.PlanetEnvironmentTerran,
			Size:        model.PlanetSizeMedium,
		}
		star.Planets = append(star.Planets, target)
	}

	_, _, err = g.SubmitTurn(1, model.Commands{
		Colonize: &model.ColonizeCommand{FleetID: fleet.ID, PlanetID: target.ID},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be in orbit")
	assert.NotContains(t, g.TurnStatus.ReadyPlayerIDs, uint(1))
}

func TestOtherPlayerValidSubmitAfterBadResearchRejection(t *testing.T) {
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

	_, _, err = g.SubmitTurn(1, model.Commands{
		SelectResearch: &model.SelectResearchCommand{TechName: "No Such Tech"},
	})
	require.Error(t, err)
	assert.NotContains(t, g.TurnStatus.ReadyPlayerIDs, uint(1))

	_, advanced, err := g.SubmitTurn(2, model.Commands{})
	require.NoError(t, err)
	assert.False(t, advanced)

	_, advanced, err = g.SubmitTurn(1, model.Commands{})
	require.NoError(t, err)
	assert.True(t, advanced)
	assert.Greater(t, g.CurrentTurn, initialTurn)
}

func TestFleetCannotMoveAndColonizeInSameTurn(t *testing.T) {
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

	fleet, home := multiplayerColonyShipAtHome(t, g, 1)
	fleet.Ships[0].FuelCell = "Thorium"
	fleet.Ships[0].UnlimitedFuelRange = true

	target := findUnownedColonizableInSystem(g, home.ParentStar, home.ID)
	if target == nil {
		star, err := g.getStar(home.ParentStar)
		require.NoError(t, err)
		target = &model.Planet{
			ID:          nextTestPlanetID(g),
			ParentStar:  home.ParentStar,
			Environment: model.PlanetEnvironmentTerran,
			Size:        model.PlanetSizeMedium,
		}
		star.Planets = append(star.Planets, target)
	}

	var destStar uint
	for _, star := range g.Galaxy.Stars {
		if star.ID != home.ParentStar {
			destStar = star.ID
			break
		}
	}
	require.NotZero(t, destStar)

	_, _, err = g.SubmitTurn(1, model.Commands{
		FleetMove: []*model.FleetMoveCommand{
			{FleetID: fleet.ID, DestinationStarID: destStar},
		},
		Colonize: &model.ColonizeCommand{FleetID: fleet.ID, PlanetID: target.ID},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot colonize and move in the same turn")
	assert.NotContains(t, g.TurnStatus.ReadyPlayerIDs, uint(1))
}

func TestCommandsFieldsHaveSubmitTimeValidators(t *testing.T) {
	// Guard: every Commands field must be covered by validatePlayerCommands.
	// When adding a field to model.Commands, add submit-time validation and list it here.
	covered := map[string]bool{
		"Build":                true, // validated via buy-production projection / build apply path
		"MovePopulation":       true,
		"TransferPopulation":   true,
		"SelectResearch":       true,
		"Colonize":             true,
		"ColonizeMany":         true,
		"ArtificialPlanet":     true,
		"ArtificialPlanetMany": true,
		"ColonyShip":           true,
		"FleetMove":            true,
		"Invade":               true,
		"BuyProduction":        true,
		"MutateRace":           true,
		"DesignShip":           true,
	}
	rt := reflect.TypeOf(model.Commands{})
	require.Equal(t, reflect.Struct, rt.Kind())
	for i := 0; i < rt.NumField(); i++ {
		name := rt.Field(i).Name
		assert.True(t, covered[name], "model.Commands.%s has no submit-time validator entry; add validation in validatePlayerCommands and list it here", name)
	}
}

func TestSubmitFleetMoveRejectsShipWithoutDrive(t *testing.T) {
	for _, grouped := range []bool{false, true} {
		g := multiPlayerGame(t, 2, 30)
		origin := g.Galaxy.Stars[0]
		dest := g.Galaxy.Stars[1]
		fleet := createTestFleet(g, 1, origin.ID)
		fleet.Ships = []model.FleetShip{{Type: "ship_design", Name: "Immobile", Drive: "None", UnlimitedFuelRange: true}, {Type: "ship_design", Name: "Mobile", InterstellarSpeed: 2, UnlimitedFuelRange: true}}
		moves := []*model.FleetMoveCommand{{FleetID: fleet.ID, DestinationStarID: dest.ID}}
		if grouped {
			indexes1, indexes2 := []uint{0}, []uint{1}
			moves[0].ShipIndexes = &indexes1
			moves = append(moves, &model.FleetMoveCommand{FleetID: fleet.ID, DestinationStarID: dest.ID, ShipIndexes: &indexes2})
		}
		_, _, err := g.SubmitTurn(1, model.Commands{FleetMove: moves})
		require.ErrorContains(t, err, "no interstellar drive")
		assert.NotContains(t, g.TurnSubmissions, uint(1))
		_, _, err = g.SubmitTurn(2, model.Commands{})
		require.NoError(t, err)
	}
}

func TestSubmitRejectsUnbuildableOrdersWithoutLosingProgress(t *testing.T) {
	g := multiPlayerGame(t, 2, 30)
	p := g.Players[0]
	p.Available.Buildings = append(p.Available.Buildings, "Automated Factory")
	colonies, err := g.GetColonyList(p.ID)
	require.NoError(t, err)
	require.NotEmpty(t, colonies)
	planet := colonies[0]
	require.True(t, Build(planet, "Automated Factory", p.Available.Buildings, g))
	planet.BuildBucket -= 10
	progress := currentBuildProgress(planet, g)
	var other *model.Planet
	for _, star := range g.Galaxy.Stars {
		for _, candidate := range star.Planets {
			if candidate.Owner == 2 {
				other = candidate
			}
		}
	}
	require.NotNil(t, other)
	for _, order := range []*model.BuildCommand{
		{PlanetID: planet.ID, Building: "Nonexistent Building"},
		{PlanetID: planet.ID, Building: "AUTOMATED FACTORY"},
		{PlanetID: planet.ID, BuildQueue: []string{"Nonexistent Building"}},
		{PlanetID: other.ID, Building: "Trade Goods"},
	} {
		_, _, err := g.SubmitTurn(p.ID, model.Commands{Build: []*model.BuildCommand{order}})
		require.Error(t, err)
		assert.NotContains(t, g.TurnSubmissions, p.ID)
		assert.Equal(t, "Automated Factory", planet.CurrentProject)
		assert.Equal(t, progress, currentBuildProgress(planet, g))
	}
	// A stale order reaching the turn processor must also preserve the investment.
	require.NoError(t, planetTurn(planet, model.Commands{Build: []*model.BuildCommand{{PlanetID: planet.ID, Building: "Nonexistent Building"}}}, p, g))
	assert.Equal(t, "Automated Factory", planet.CurrentProject)
	assert.GreaterOrEqual(t, currentBuildProgress(planet, g), progress)
}
