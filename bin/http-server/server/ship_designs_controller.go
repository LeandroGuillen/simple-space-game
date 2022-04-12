package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type GetShipDesignListRequest struct {
	GameID   string `binding:"required" json:"game_id"`
	PlayerID uint   `binding:"required" json:"player_id"`
}

func GetShipDesignList(c *gin.Context) {
	var req GetShipDesignListRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	if shipDesignsList, err := g.GetShipDesignsList(req.PlayerID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	} else {
		c.JSON(http.StatusOK, gin.H{"ship_designs": shipDesignsList})
	}
}
