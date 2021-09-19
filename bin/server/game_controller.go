package server

import (
	"corion/game"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type NewGameRequest struct {
	NumStars uint
}

type GameListResponse struct {
	GameList []GameListResponseItem
}

type GameListResponseItem struct {
	GameID   string `json:"game_id"`
	NumStars int    `json:"num_stars"`
}

func NewGame(c *gin.Context) {
	var req NewGameRequest
	_ = c.Bind(&req)

	rawServer, exists := c.Get("server")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "server object was not created",
		})
		return
	}

	server := rawServer.(*Server)

	g := &game.Game{
		Players: []game.Player{
			{ID: 1},
		},
	}
	g.Galaxy = g.NewGalaxy(req.NumStars)
	gameID := uuid.New().String()
	server.GameList[gameID] = g

	c.JSON(http.StatusOK, gin.H{"ok": true, "gameID": gameID})
}

func GetGameList(c *gin.Context) {
	rawServer, exists := c.Get("server")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "server object was not created",
		})
		return
	}

	s := rawServer.(*Server)

	var gameList = make([]GameListResponseItem, 0)
	for gameID, game := range s.GameList {
		gameList = append(gameList, GameListResponseItem{
			GameID:   gameID,
			NumStars: len(game.Stars),
		})
	}

	c.JSON(http.StatusOK, gin.H{"game_list": gameList})
}
