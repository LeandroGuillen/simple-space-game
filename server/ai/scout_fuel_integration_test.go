package ai_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"ssg/ai"
	"ssg/game"
	"ssg/model"

	"github.com/stretchr/testify/require"
)

// TestAIHumanLargeGalaxyFiftyTurnsNoFuelErrors drives a 2-player game on a
// huge galaxy for 50 turns: player 1 (human) submits empty turns; player 2
// uses SimplePolicy. Any submit error fails the test — especially the
// pre-fix "outside fleet fuel range" stall.
func TestAIHumanLargeGalaxyFiftyTurnsNoFuelErrors(t *testing.T) {
	start := time.Now()
	resources, err := game.NewResourceManager()
	require.NoError(t, err)
	g, err := game.NewGame(resources, game.NewGameOptions{
		PlayerName:  "Human",
		PlayerColor: model.DefaultPlayerColor(),
		NumStars:    100,
		GalaxySize:  "huge",
		Race:        game.DefaultRace("Human"),
		NumPlayers:  2,
	})
	require.NoError(t, err)
	require.Len(t, g.Players, 2)

	aiPlayer := g.Players[1]
	aiHome := firstOwnedPlanet(t, g, aiPlayer.ID)
	var aiStar *model.Star
	for _, star := range g.Galaxy.Stars {
		if star != nil && star.ID == aiHome.ParentStar {
			aiStar = star
			break
		}
	}
	require.NotNil(t, aiStar)

	g.Fleets = append(g.Fleets, &model.Fleet{
		ID: 9001, Owner: aiPlayer.ID, StarID: aiStar.ID,
		X: float64(aiStar.X), Y: float64(aiStar.Y),
		Ships: []model.FleetShip{{
			Type: "ship_design", Name: "Scout",
			// Standard autonomy is 2; stamp must match (server prefers FuelCell).
			FuelCell: "Standard", FuelRangeParsecs: 2,
		}},
	})

	policy := ai.SimplePolicy{}
	scoutMoves := 0
	for turn := 0; turn < 50; turn++ {
		visible, err := g.GetVisibleGameForPlayer(aiPlayer.ID)
		require.NoError(t, err)
		vg := visibleGameFromGame(t, visible)
		decision := policy.Decide(vg, aiPlayer.ID)
		scoutMoves += len(decision.Commands.FleetMove)

		_, _, err = g.SubmitTurn(aiPlayer.ID, decision.Commands)
		if err != nil {
			t.Fatalf("AI submit turn %d failed: %v (notes=%v moves=%v)",
				turn, err, decision.Notes, decision.Commands.FleetMove)
		}
		_, _, err = g.SubmitTurn(g.Players[0].ID, model.Commands{})
		if err != nil {
			t.Fatalf("human submit turn %d failed: %v", turn, err)
		}
	}
	if scoutMoves == 0 {
		t.Fatalf("expected at least one AI scout move over 50 turns")
	}
	elapsed := time.Since(start)
	if elapsed > 5*time.Second {
		t.Fatalf("50-turn AI/human huge-galaxy test took %s (want < 5s)", elapsed)
	}
	t.Logf("50 turns completed in %s (%d scout moves)", elapsed, scoutMoves)
}

func TestAIScoutDecisionNeverPicksOutOfRangeOnLiveVisible(t *testing.T) {
	// Smoke: Decide on a real visible snapshot never emits a move the server
	// would reject for fuel range.
	resources, err := game.NewResourceManager()
	require.NoError(t, err)
	g, err := game.NewGame(resources, game.NewGameOptions{
		PlayerName:  "Human",
		PlayerColor: model.DefaultPlayerColor(),
		NumStars:    40,
		GalaxySize:  "medium",
		Race:        game.DefaultRace("Human"),
		NumPlayers:  2,
	})
	require.NoError(t, err)
	aiID := g.Players[1].ID
	home := firstOwnedPlanet(t, g, aiID)
	var star *model.Star
	for _, s := range g.Galaxy.Stars {
		if s != nil && s.ID == home.ParentStar {
			star = s
			break
		}
	}
	require.NotNil(t, star)
	g.Fleets = append(g.Fleets, &model.Fleet{
		ID: 42, Owner: aiID, StarID: star.ID,
		X: float64(star.X), Y: float64(star.Y),
		Ships: []model.FleetShip{{Name: "Scout", Type: "ship_design", FuelRangeParsecs: 3}},
	})

	visible, err := g.GetVisibleGameForPlayer(aiID)
	require.NoError(t, err)
	vg := visibleGameFromGame(t, visible)
	d := ai.SimplePolicy{}.Decide(vg, aiID)
	for _, move := range d.Commands.FleetMove {
		require.NotNil(t, move)
		_, _, err := g.SubmitTurn(aiID, model.Commands{FleetMove: []*model.FleetMoveCommand{move}})
		if err != nil && strings.Contains(err.Error(), "fuel range") {
			t.Fatalf("policy issued out-of-range move to star %d: %v", move.DestinationStarID, err)
		}
		// Undo readiness if submit succeeded without the other player — reset submissions.
		g.TurnSubmissions = map[uint]model.Commands{}
		g.TurnStatus = visible.TurnStatus
	}
}

func firstOwnedPlanet(t *testing.T, g *game.Game, owner uint) *model.Planet {
	t.Helper()
	for _, star := range g.Galaxy.Stars {
		for _, p := range star.Planets {
			if p != nil && p.Owner == owner {
				return p
			}
		}
	}
	t.Fatalf("no owned planet for player %d", owner)
	return nil
}

func visibleGameFromGame(t *testing.T, g *game.Game) *ai.VisibleGame {
	t.Helper()
	raw, err := json.Marshal(g)
	require.NoError(t, err)
	var vg ai.VisibleGame
	require.NoError(t, json.Unmarshal(raw, &vg))
	vg.Raw = raw
	return &vg
}
