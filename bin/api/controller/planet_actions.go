package controller

import (
	"corion/game"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ColonizePlanetRequest struct {
	OwnerID  uint `uri:"owner" binding:"required" json:"owner_id"`
	PlanetID uint `uri:"planet" binding:"required" json:"planet_id"`
}

func ColonizePlanet(c *gin.Context) {
	var request ColonizePlanetRequest
	if err := c.ShouldBindUri(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	rawGame, gameExists := c.Get("game")
	if !gameExists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Game state error"})
		return
	}

	g := rawGame.(*game.Game)

	if err := g.ColonizePlanet(request.OwnerID, request.PlanetID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}
