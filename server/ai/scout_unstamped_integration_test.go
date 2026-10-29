package ai_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ssg/ai"
	"ssg/game"
	"ssg/model"

	"github.com/stretchr/testify/require"
)

func TestAIUnstampedFleetsScoutAndServerAccepts(t *testing.T) {
	// Design-only, Transport, and Colony Ship fleets must get a reachable
	// scout move that SubmitTurn accepts. On 2e1d8aa these stay idle.
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
	aiPlayer := g.Players[1]
	ensurePlayerHasStandardFuel(t, aiPlayer)
	home := firstOwnedPlanet(t, g, aiPlayer.ID)
	star := starByID(t, g, home.ParentStar)

	cases := []struct {
		name  string
		ships []model.FleetShip
	}{
		{
			name: "design-only",
			ships: []model.FleetShip{{
				Type: "ship_design", Name: "Dark Blade",
				DesignName: aiPlayer.ShipDesigns[0].Name,
			}},
		},
		{
			name:  "transport",
			ships: []model.FleetShip{{Type: "transport", Name: "Transport"}},
		},
		{
			name:  "colony_ship",
			ships: []model.FleetShip{{Type: "colony_ship", Name: "Colony Ship"}},
		},
	}

	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fleetID := uint(8000 + i)
			g.Fleets = append(g.Fleets, &model.Fleet{
				ID: fleetID, Owner: aiPlayer.ID, StarID: star.ID,
				X: float64(star.X), Y: float64(star.Y),
				Ships: append([]model.FleetShip(nil), tc.ships...),
			})
			visible, err := g.GetVisibleGameForPlayer(aiPlayer.ID)
			require.NoError(t, err)
			vg := visibleGameFromGame(t, visible)
			d := ai.SimplePolicy{}.Decide(vg, aiPlayer.ID)

			var move *model.FleetMoveCommand
			for _, m := range d.Commands.FleetMove {
				if m != nil && m.FleetID == fleetID {
					move = m
					break
				}
			}
			require.NotNil(t, move, "expected scout move for %s fleet (notes=%v)", tc.name, d.Notes)

			_, _, err = g.SubmitTurn(aiPlayer.ID, model.Commands{
				FleetMove: []*model.FleetMoveCommand{move},
			})
			require.NoError(t, err, "server rejected %s scout move: %v", tc.name, err)
			g.TurnSubmissions = map[uint]model.Commands{}
			g.TurnStatus = visible.TurnStatus
		})
	}
}

func TestAIMergedScoutWithTransportStillScouts(t *testing.T) {
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
	aiPlayer := g.Players[1]
	ensurePlayerHasStandardFuel(t, aiPlayer)
	home := firstOwnedPlanet(t, g, aiPlayer.ID)
	star := starByID(t, g, home.ParentStar)

	g.Fleets = append(g.Fleets, &model.Fleet{
		ID: 8100, Owner: aiPlayer.ID, StarID: star.ID,
		X: float64(star.X), Y: float64(star.Y),
		Ships: []model.FleetShip{
			{Type: "ship_design", Name: "Scout", FuelCell: "Standard", FuelRangeParsecs: 4},
			{Type: "transport", Name: "Transport"},
		},
	})

	visible, err := g.GetVisibleGameForPlayer(aiPlayer.ID)
	require.NoError(t, err)
	vg := visibleGameFromGame(t, visible)
	d := ai.SimplePolicy{}.Decide(vg, aiPlayer.ID)
	require.NotEmpty(t, d.Commands.FleetMove, "merged scout+transport must still scout (notes=%v)", d.Notes)

	_, _, err = g.SubmitTurn(aiPlayer.ID, model.Commands{FleetMove: d.Commands.FleetMove})
	require.NoError(t, err)
}

func TestBattleDemoFiftyTurnsIssuesScoutMoves(t *testing.T) {
	// battle-demo fleets are design-only (no fuel stamps). On 2e1d8aa the AI
	// never scouts them; after the fix we must see at least one FleetMove and
	// zero fuel-range rejections over 50 turns.
	start := time.Now()
	resources, err := game.NewResourceManager()
	require.NoError(t, err)

	path := filepath.Join("data", "scenarios", "battle-demo.json")
	data, err := os.ReadFile(path)
	if err != nil {
		path = filepath.Join("..", "data", "scenarios", "battle-demo.json")
		data, err = os.ReadFile(path)
	}
	require.NoError(t, err)

	var g game.Game
	require.NoError(t, json.Unmarshal(data, &g))
	game.HydrateLoadedGame(&g, resources)
	require.GreaterOrEqual(t, len(g.Players), 2)
	require.NotEmpty(t, g.Fleets)

	aiPlayer := g.Players[1]
	human := g.Players[0]
	// Separate the co-located battle fleets so 50 turns can advance without a
	// pending tactical battle blocking End Turn; design-only ships stay as-is.
	g.PendingBattles = nil
	humanHome := firstOwnedPlanet(t, &g, human.ID)
	humanStar := starByID(t, &g, humanHome.ParentStar)
	aiHome := firstOwnedPlanet(t, &g, aiPlayer.ID)
	aiStar := starByID(t, &g, aiHome.ParentStar)
	for _, f := range g.Fleets {
		if f == nil {
			continue
		}
		f.DestinationStarID = 0
		f.TurnsRemaining = 0
		if f.Owner == human.ID {
			f.StarID = humanStar.ID
			f.X, f.Y = float64(humanStar.X), float64(humanStar.Y)
		}
		if f.Owner == aiPlayer.ID {
			// Park design-only fleets at the AI home so supply-range scouting applies.
			f.StarID = aiStar.ID
			f.X, f.Y = float64(aiStar.X), float64(aiStar.Y)
		}
	}
	// battle-demo's sparse map has no unexplored star within Standard (2pc) of
	// the AI home; add one so a design-only scout move is possible.
	g.Galaxy.Stars = append(g.Galaxy.Stars, &model.Star{
		ID: 99, X: aiStar.X + 60, Y: aiStar.Y, // 1 parsec east, unnamed → unexplored
	})
	g.TurnSubmissions = map[uint]model.Commands{}

	policy := ai.SimplePolicy{}
	scoutMoves := 0
	for turn := 0; turn < 50; turn++ {
		visible, err := g.GetVisibleGameForPlayer(aiPlayer.ID)
		require.NoError(t, err)
		vg := visibleGameFromGame(t, visible)
		decision := policy.Decide(vg, aiPlayer.ID)
		if len(decision.Commands.FleetMove) > 0 {
			scoutMoves += len(decision.Commands.FleetMove)
		}
		_, _, err = g.SubmitTurn(aiPlayer.ID, decision.Commands)
		if err != nil {
			if strings.Contains(err.Error(), "fuel range") {
				t.Fatalf("fuel-range rejection on turn %d: %v", turn, err)
			}
			t.Fatalf("AI submit turn %d failed: %v (scoutMovesSoFar=%d)", turn, err, scoutMoves)
		}
		_, _, err = g.SubmitTurn(human.ID, model.Commands{})
		if err != nil {
			t.Fatalf("human submit turn %d failed: %v (scoutMovesSoFar=%d)", turn, err, scoutMoves)
		}
	}
	if scoutMoves == 0 {
		t.Fatalf("expected at least one AI scout move over 50 turns on battle-demo; got 0")
	}
	elapsed := time.Since(start)
	if elapsed > 5*time.Second {
		t.Fatalf("battle-demo 50-turn test took %s (want < 5s)", elapsed)
	}
	t.Logf("battle-demo: %d scout moves in %s", scoutMoves, elapsed)
}

func ensurePlayerHasStandardFuel(t *testing.T, p *model.Player) {
	t.Helper()
	require.NotNil(t, p)
	has := false
	for _, name := range p.Available.FuelCells {
		if name == "Standard" {
			has = true
			break
		}
	}
	if !has {
		p.Available.FuelCells = append(p.Available.FuelCells, "Standard")
	}
	if len(p.ShipDesigns) == 0 {
		p.ShipDesigns = []model.ShipDesign{{
			Name:     "Laser Gunship",
			FuelCell: model.FuelCell{Name: "Standard", Autonomy: 2},
		}}
	}
}

func starByID(t *testing.T, g *game.Game, id uint) *model.Star {
	t.Helper()
	for _, s := range g.Galaxy.Stars {
		if s != nil && s.ID == id {
			return s
		}
	}
	t.Fatalf("star %d not found", id)
	return nil
}
