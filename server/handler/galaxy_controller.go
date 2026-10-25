package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type NewGalaxyRequest struct {
	NumStars uint
}

type NewRandomGalaxyRequest struct{}

type GetColonyListRequest struct {
	GameID   string `binding:"required" json:"game_id"`
	PlayerID uint   `binding:"required" json:"player_id"`
}

func GetColonyList(c *gin.Context) {
	var req GetColonyListRequest
	if err := c.Bind(&req); err != nil {
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
	g := s.getGame(req.GameID)
	if g == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": fmt.Sprintf("game with id %s was not found", req.GameID),
		})
		return
	}

	colonyList, err := g.GetColonyList(req.PlayerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, colonyList)
}
