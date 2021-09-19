package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ColonizePlanetRequest struct {
	//OwnerID  uint   `uri:"owner" binding:"required" json:"owner_id"`
	//PlanetID uint   `uri:"planet" binding:"required" json:"planet_id"`
	GameID string `uri:"gameID" binding:"required" json:"game_id"`
}

//r.POST("/action/:owner/colonize/:planet", ColonizePlanet)
func ColonizePlanet(c *gin.Context) {
	var request ColonizePlanetRequest
	if err := c.Bind(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err})
		return
	}

	ownerID, err := strconv.ParseUint(c.Param("owner"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "owner ID is invalid or not present"})
		return
	}

	planetID, err := strconv.ParseUint(c.Param("planet"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "planet ID is invalid or not present"})
		return
	}

	rawServer, serverExists := c.Get("server")
	if !serverExists {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "server object was not created"})
		return
	}

	s := rawServer.(*Server)
	g, gameFound := s.GameList[request.GameID]
	if !gameFound {
		c.JSON(http.StatusNotFound, gin.H{
			"message": fmt.Sprintf("game with id %s was not found", request.GameID),
		})
		return
	}

	if err := g.ColonizePlanet(uint(ownerID), uint(planetID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}
