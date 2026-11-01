package ai_test

import (
	"github.com/stretchr/testify/require"
	"ssg/ai"
	"ssg/game"
	"ssg/model"
	"testing"
)

func TestAINewGameResearchesDesignsAndBuildsArmedShip(t *testing.T) {
	game.SetRandomSeed(42)
	resources, err := game.NewResourceManager()
	require.NoError(t, err)
	g, err := game.NewGame(resources, game.NewGameOptions{PlayerName: "Human", PlayerColor: model.DefaultPlayerColor(), NumStars: 40, GalaxySize: "medium", Race: game.DefaultRace("Human"), NumPlayers: 2})
	require.NoError(t, err)
	id := g.Players[1].ID
	for turn := 0; turn < 150; turn++ {
		visible, err := g.GetVisibleGameForPlayer(id)
		require.NoError(t, err)
		d := ai.SimplePolicy{}.Decide(visibleGameFromGame(t, visible), id)
		_, _, err = g.SubmitTurn(id, d.Commands)
		require.NoError(t, err, "turn %d: %v", turn, d.Notes)
		_, _, err = g.SubmitTurn(1, model.Commands{})
		require.NoError(t, err)
		for _, f := range g.Fleets {
			if f.Owner == id {
				for _, ship := range f.Ships {
					if ship.DesignName != "" {
						for _, design := range g.Players[1].ShipDesigns {
							if design.Name == ship.DesignName && len(design.Beams) > 0 {
								return
							}
						}
					}
				}
			}
		}
	}
	t.Fatalf("AI never produced an armed ship; research=%v designs=%+v", g.Players[1].ResearchedTechs, g.Players[1].ShipDesigns)
}
