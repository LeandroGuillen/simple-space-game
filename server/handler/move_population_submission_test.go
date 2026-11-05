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

func TestTurnMovePopulationRequiresAllUnits(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tt := range []struct {
		name                         string
		farmers, workers, scientists int
		status                       int
	}{
		{"zero", 0, 0, 0, http.StatusBadRequest},
		{"partial", 2, 2, 0, http.StatusBadRequest},
		{"excess", 4, 4, 1, http.StatusBadRequest},
		{"exact", 4, 3, 1, http.StatusOK},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s, err := NewServer()
			require.NoError(t, err)
			g, err := game.NewGame(s.ResourceManager, game.NewGameOptions{PlayerName: "Ada", PlayerColor: model.DefaultPlayerColor(), NumStars: 8, Race: game.DefaultRace("Human"), NumPlayers: 2})
			require.NoError(t, err)
			colonies, err := g.GetColonyList(1)
			require.NoError(t, err)
			require.NotEmpty(t, colonies)
			p := colonies[0]
			require.Equal(t, int64(8000), p.Population)
			s.putGame(g.ID, g)
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			body := fmt.Sprintf(`{"commands":{"movePopulation":[{"planetId":%d,"farmers":%d,"workers":%d,"scientists":%d}]}}`, p.ID, tt.farmers, tt.workers, tt.scientists)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/turn", strings.NewReader(body))
			ctx.Request.Header.Set("Content-Type", "application/json")
			ctx.Request.Header.Set(GameIDHeader, g.ID)
			ctx.Request.Header.Set(PlayerIDHeader, "1")
			ctx.Set("server", s)
			Turn(ctx)
			require.Equal(t, tt.status, recorder.Code, recorder.Body.String())
			if tt.status == http.StatusBadRequest {
				require.Contains(t, recorder.Body.String(), "bad pop movement")
				require.NotContains(t, g.TurnSubmissions, uint(1))
				require.Contains(t, g.TurnStatus.WaitingPlayerIDs, uint(1))
			} else {
				require.Contains(t, recorder.Body.String(), `"ok":true`)
			}
		})
	}
}
