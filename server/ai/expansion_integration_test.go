package ai_test

import (
	"github.com/stretchr/testify/require"
	"ssg/ai"
	"ssg/game"
	"ssg/model"
	"testing"
)

// Seed 42, normal new-game starts, one passive human and three independent AI
// players. All player orders go through submit-time and advance validation.
func TestAIExpansionHundredTurns(t *testing.T) {
	game.SetRandomSeed(42)
	resources, err := game.NewResourceManager()
	require.NoError(t, err)
	g, err := game.NewGame(resources, game.NewGameOptions{PlayerName: "Human", PlayerColor: model.DefaultPlayerColor(), NumStars: 40, GalaxySize: "medium", Race: game.DefaultRace("Human"), NumPlayers: 4})
	require.NoError(t, err)
	for turn := 0; turn < 100; turn++ {
		for _, player := range g.Players[1:] {
			visible, err := g.GetVisibleGameForPlayer(player.ID)
			require.NoError(t, err)
			d := ai.SimplePolicy{}.Decide(visibleGameFromGame(t, visible), player.ID)
			_, _, err = g.SubmitTurn(player.ID, d.Commands)
			require.NoError(t, err, "player %d turn %d: %v", player.ID, turn, d.Notes)
		}
		_, _, err = g.SubmitTurn(1, model.Commands{})
		require.NoError(t, err, "human turn %d", turn)
	}
	for _, player := range g.Players[1:] {
		count := 0
		for _, s := range g.Galaxy.Stars {
			for _, p := range s.Planets {
				if p.Owner == player.ID {
					count++
				}
			}
		}
		require.GreaterOrEqual(t, count, 3, "player %d", player.ID)
	}
}
