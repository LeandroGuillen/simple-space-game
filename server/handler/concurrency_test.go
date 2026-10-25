package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"sync"
	"testing"

	"ssg/game"
	"ssg/model"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestConcurrentGameListAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resources, err := game.NewResourceManager()
	require.NoError(t, err)

	server := &Server{
		gameList:        make(map[string]*game.Game),
		ResourceManager: resources,
	}

	const numGames = 10
	const numReaders = 5

	var wg sync.WaitGroup

	for i := 0; i < numGames; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			g, err := game.NewGame(resources, game.NewGameOptions{
				PlayerName:  fmt.Sprintf("Player%d", idx),
				NumStars:    4,
				PlayerColor: model.DefaultPlayerColor(),
				Race:        game.DefaultRace("Human"),
			})
			require.NoError(t, err)

			payload := NewGameRequest{
				PlayerName:  g.Players[0].Name,
				PlayerColor: g.Players[0].Color,
				NumStars:    4,
			}
			body, err := json.Marshal(&payload)
			require.NoError(t, err)

			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest("POST", "/games", bytes.NewReader(body))
			ctx.Request.Header.Set("Content-Type", "application/json")
			ctx.Set("server", server)

			NewGame(ctx)
		}(i)
	}

	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < 20; j++ {
				recorder := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(recorder)
				ctx.Request = httptest.NewRequest("GET", "/games", nil)
				ctx.Set("server", server)

				GetGameList(ctx)
			}
		}()
	}

	wg.Wait()

	allGames := server.listGames()
	require.NotEmpty(t, allGames)
}

func TestConcurrentGameCreationAndRetrieval(t *testing.T) {
	gin.SetMode(gin.TestMode)

	resources, err := game.NewResourceManager()
	require.NoError(t, err)

	server := &Server{
		gameList:        make(map[string]*game.Game),
		ResourceManager: resources,
	}

	const numOps = 15

	var wg sync.WaitGroup
	gameIDs := make(chan string, numOps)

	for i := 0; i < numOps; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			g, err := game.NewGame(resources, game.NewGameOptions{
				PlayerName:  fmt.Sprintf("Concurrent%d", idx),
				NumStars:    3,
				PlayerColor: model.DefaultPlayerColor(),
				Race:        game.DefaultRace("Human"),
			})
			require.NoError(t, err)

			server.putGame(g.ID, g)
			gameIDs <- g.ID
		}(i)
	}

	for i := 0; i < numOps; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			select {
			case gid := <-gameIDs:
				for j := 0; j < 10; j++ {
					retrieved := server.getGame(gid)
					if retrieved != nil {
						require.Equal(t, gid, retrieved.ID)
					}
				}
			default:
			}
		}()
	}

	wg.Wait()
}

func TestConcurrentScenarioLoadAndGameList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	t.Setenv("SSG_SAVES_DIR", dir)

	resources, err := game.NewResourceManager()
	require.NoError(t, err)

	scenario1 := createScenarioFile(t, resources, "scenario-race-1")
	scenario2 := createScenarioFile(t, resources, "scenario-race-2")

	server := &Server{
		gameList:        make(map[string]*game.Game),
		ResourceManager: resources,
	}

	var wg sync.WaitGroup

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			scenarioName := fmt.Sprintf("scenario-race-%d", (idx%2)+1)
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest("POST", "/scenarios/"+scenarioName+"/load", nil)
			ctx.Params = gin.Params{{Key: "name", Value: scenarioName}}
			ctx.Set("server", server)

			LoadScenario(ctx)
		}(i)
	}

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < 10; j++ {
				_ = server.listGames()
			}
		}()
	}

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			for j := 0; j < 10; j++ {
				if idx%2 == 0 {
					_ = server.getGame(scenario1.ID)
				} else {
					_ = server.getGame(scenario2.ID)
				}
			}
		}(i)
	}

	wg.Wait()
}
