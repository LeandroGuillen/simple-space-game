package server

import (
	"fmt"
	"net/http"
	"ssg/game"

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
	GameID      string  `json:"game_id"`
	NumStars    int     `json:"num_stars"`
	CurrentTurn float64 `json:"current_turn"`
}

type GetGalaxyRequest struct {
	GameID string `uri:"gameID" binding:"required" json:"game_id"`
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

	g, err := game.NewGame(req.PlayerName, req.NumStars)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": fmt.Sprintf("error creating game: %s", err.Error()),
		})
		return
	}
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
			GameID:      gameID,
			NumStars:    len(g.Galaxy.Stars),
			CurrentTurn: g.CurrentTurn,
		})
	}

	c.JSON(http.StatusOK, gin.H{"game_list": gameList})
}

func GetGame(c *gin.Context) {
	var req GetGalaxyRequest
	if err := c.BindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rawServer, exists := c.Get("server")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "server object was not created",
		})
		return
	}

	s := rawServer.(*Server)
	g, gameFound := s.GameList[req.GameID]
	if !gameFound {
		c.JSON(http.StatusNotFound, gin.H{
			"message": fmt.Sprintf("game with id %s was not found", req.GameID),
		})
		return
	}

	c.JSON(http.StatusOK, g)
}
