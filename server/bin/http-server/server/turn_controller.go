package server

import (
	"net/http"

	"ssg/game/model"

	"github.com/gin-gonic/gin"
)

type TurnRequest struct {
	Commands model.Commands `json:"commands"`
}

func Turn(c *gin.Context) {
	gameID, playerID, err := readHeaders(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req TurnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	g, s, err := getGameHandle(c, gameID)
	if err != nil {
		return
	}

	if err := g.Turn(playerID, req.Commands); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
	s.hub.BroadcastGameUpdate(g)
}
