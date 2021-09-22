package server

import (
	"corion/game"
	"net/http"

	"github.com/gin-gonic/gin"
)

type NewGameRequest struct {
	NumStars   uint   `json:"num_stars,omitempty"`
	PlayerName string `json:"player_name,omitempty"`
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

	g := game.New(req.PlayerName, req.NumStars)
	server := rawServer.(*Server)
	server.GameList[g.ID] = g

	c.JSON(http.StatusOK, gin.H{"ok": true, "gameID": g.ID})
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
	for gameID, g := range s.GameList {
		gameList = append(gameList, GameListResponseItem{
			GameID:   gameID,
			NumStars: len(g.Stars),
		})
	}

	c.JSON(http.StatusOK, gin.H{"game_list": gameList})
}
