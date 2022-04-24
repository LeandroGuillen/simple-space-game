package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type TurnRequest struct {
	GameID   string `binding:"required" json:"game_id"`
	PlayerID uint   `binding:"required" json:"player_id"`
}

func Turn(c *gin.Context) {
	var req TurnRequest
	if err := c.Bind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	rawServer, serverExists := c.Get("server")
	if !serverExists {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "server object was not created"})
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

	if err := g.Turn(req.PlayerID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "game": g})
}

func Turn2(c *gin.Context) {
	gameID, playerID, err := readHeaders(c.Request.Header)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	g, s, err := getGameHandle(c, gameID)
	if err != nil {
		return
	}

	if err := g.Turn(playerID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
	s.hub.BroadcastGameUpdate(g)
}
