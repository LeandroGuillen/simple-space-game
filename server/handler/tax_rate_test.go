package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ssg/game"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestTurnRejectsInvalidTaxRateWithBadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, rate := range []string{"-0.1", "0.50001", "NaN", "1e999"} {
		t.Run(rate, func(t *testing.T) {
			resources, err := game.NewResourceManager()
			require.NoError(t, err)
			g, err := game.NewGame(resources, game.NewGameOptions{
				PlayerName: "Ada", NumStars: 8, Race: game.DefaultRace("Human"), NumPlayers: 2,
			})
			require.NoError(t, err)
			s := &Server{gameList: map[string]*game.Game{g.ID: g}, ResourceManager: resources}
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/turn", strings.NewReader(
				fmt.Sprintf(`{"commands":{"set_tax_rate":{"rate":%s}}}`, rate)))
			ctx.Request.Header.Set("Content-Type", "application/json")
			ctx.Request.Header.Set(GameIDHeader, g.ID)
			ctx.Request.Header.Set(PlayerIDHeader, "1")
			ctx.Set("server", s)
			Turn(ctx)
			require.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
			require.NotContains(t, g.TurnSubmissions, uint(1))
			require.Zero(t, g.Players[0].TaxRate)
		})
	}
}
