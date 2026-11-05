package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ssg/game"
	"ssg/model"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestTurnMovePopulationWithOutboundTransfer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s, err := NewServer()
	require.NoError(t, err)
	g, err := game.NewGame(s.ResourceManager, game.NewGameOptions{PlayerName: "Ada", PlayerColor: model.DefaultPlayerColor(), NumStars: 8, Race: game.DefaultRace("Human"), NumPlayers: 1})
	require.NoError(t, err)
	colonies, err := g.GetColonyList(1)
	require.NoError(t, err)
	source := colonies[0]
	var destination *model.Planet
	for _, star := range g.Galaxy.Stars {
		for _, p := range star.Planets {
			if p.ID != source.ID {
				destination = p
				break
			}
		}
		if destination != nil {
			break
		}
	}
	require.NotNil(t, destination)
	destination.Owner = 1
	destination.Environment = model.PlanetEnvironmentTerran
	destination.Size = model.PlanetSizeMedium
	destination.Population = 1000
	destination.Food.Farmers = 1
	g.Players[0].Freighters = 5
	s.putGame(g.ID, g)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	body := fmt.Sprintf(`{"commands":{"movePopulation":[{"planetId":%d,"farmers":3,"workers":3,"scientists":1}],"transfer_population":[{"source_planet_id":%d,"destination_planet_id":%d,"population_units":1}]}}`, source.ID, source.ID, destination.ID)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/turn", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Request.Header.Set(GameIDHeader, g.ID)
	ctx.Request.Header.Set(PlayerIDHeader, "1")
	ctx.Set("server", s)
	Turn(ctx)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Contains(t, recorder.Body.String(), `"advanced":true`)
	require.NotContains(t, recorder.Body.String(), "resubmit")
	require.Len(t, g.PopulationTransfers, 1)
	require.GreaterOrEqual(t, source.Food.Farmers, int64(3))
	require.Equal(t, int64(3), source.Industry.Workers)
	require.Equal(t, int64(1), source.Science.Scientists)
	require.Equal(t, source.Population/1000, source.Food.Farmers+source.Industry.Workers+source.Science.Scientists)
}
