package ai_test

import (
	"github.com/stretchr/testify/require"
	"ssg/ai"
	"ssg/game"
	"ssg/model"
	"testing"
)

// A two-star Medium scenario with an explored, undefended rival one parsec
// away. The AI starts without ships or researched weapons.
func TestAIConstructsFleetAndAttacksVisibleWeakerRival(t *testing.T) {
	game.SetRandomSeed(41)
	resources, err := game.NewResourceManager()
	require.NoError(t, err)
	g, err := game.NewGame(resources, game.NewGameOptions{PlayerName: "Human", PlayerColor: model.DefaultPlayerColor(), NumStars: 40, GalaxySize: "medium", Race: game.DefaultRace("Human"), NumPlayers: 2})
	require.NoError(t, err)
	aiPlayer := g.Players[1]
	home := firstOwnedPlanet(t, g, aiPlayer.ID)
	rival := firstOwnedPlanet(t, g, 1)
	homeStar := starByID(t, g, home.ParentStar)
	rivalStar := starByID(t, g, rival.ParentStar)
	homeStar.X = 100
	homeStar.Y = 100
	rivalStar.X = 160
	rivalStar.Y = 100
	homeStar.Planets = []*model.Planet{home}
	rivalStar.Planets = []*model.Planet{rival}
	g.Galaxy.Stars = []*model.Star{homeStar, rivalStar}
	require.NoError(t, g.ExploreStar(aiPlayer.ID, rivalStar.ID))
	built := false
	launched := false
	for turn := 0; turn < 250; turn++ {
		visible, err := g.GetVisibleGameForPlayer(aiPlayer.ID)
		require.NoError(t, err)
		d := ai.SimplePolicy{}.Decide(visibleGameFromGame(t, visible), aiPlayer.ID)
		for _, f := range g.Fleets {
			if f != nil && f.Owner == aiPlayer.ID && len(f.Ships) >= 2 {
				built = true
			}
		}
		attacking := false
		for _, move := range d.Commands.FleetMove {
			if move.DestinationStarID == rivalStar.ID {
				attacking = true
				require.NotNil(t, move.ShipIndexes)
				require.NotEmpty(t, *move.ShipIndexes)
			}
		}
		_, _, err = g.SubmitTurn(aiPlayer.ID, d.Commands)
		require.NoError(t, err, "AI turn %d: %v", turn, d.Notes)
		_, _, err = g.SubmitTurn(1, model.Commands{})
		require.NoError(t, err, "human turn %d", turn)
		if attacking {
			require.True(t, built)
			launched = true
		}
		if len(d.Commands.Invade) > 0 {
			require.True(t, launched)
			return
		}
	}
	t.Fatalf("AI did not launch an attack and invasion; launched=%v", launched)
}
