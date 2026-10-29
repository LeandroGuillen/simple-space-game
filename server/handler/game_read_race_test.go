package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"ssg/game"
	"ssg/model"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestConcurrentSubmitTurnAndHTTPGetGameNoRace(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resources, err := game.NewResourceManager()
	require.NoError(t, err)

	g, err := game.NewGame(resources, game.NewGameOptions{
		PlayerName:  "Ada",
		PlayerColor: model.DefaultPlayerColor(),
		NumStars:    4,
		Race:        game.DefaultRace("Human"),
		NumPlayers:  2,
	})
	require.NoError(t, err)

	server := &Server{
		gameList:        map[string]*game.Game{g.ID: g},
		ResourceManager: resources,
	}

	const readers = 6
	const iterations = 30
	var wg sync.WaitGroup

	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func(playerID uint) {
			defer wg.Done()
			for n := 0; n < iterations*4; n++ {
				recorder := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(recorder)
				ctx.Params = gin.Params{{Key: "gameID", Value: g.ID}}
				ctx.Request = httptest.NewRequest(http.MethodGet, "/games/"+g.ID, nil)
				ctx.Request.Header.Set(GameIDHeader, g.ID)
				ctx.Request.Header.Set(PlayerIDHeader, fmt.Sprintf("%d", playerID))
				ctx.Set("server", server)
				GetGame(ctx)
			}
		}(uint(i%2 + 1))
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			_, _, err := g.SubmitTurn(1, model.Commands{})
			require.NoError(t, err)
			_, advanced, err := g.SubmitTurn(2, model.Commands{})
			require.NoError(t, err)
			require.True(t, advanced)
		}
	}()

	wg.Wait()
}
