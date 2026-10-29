package server

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type BuildOnPlanetRequest struct {
	GameID     string `binding:"required" json:"game_id"`
	OwnerID    uint   `binding:"required" json:"owner_id"`
	PlanetID   uint   `binding:"required" json:"planet_id"`
	BuildingID *uint  `binding:"required" json:"building_id"`
}

type SetPlanetAutobuildRequest struct {
	Enabled bool `json:"enabled"`
}

func SetPlanetAutobuild(c *gin.Context) {
	gameID, playerID, err := readHeaders(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	planetID, err := strconv.ParseUint(c.Param("planet"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "planet ID is invalid or not present"})
		return
	}

	var req SetPlanetAutobuildRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	g, s, err := getGameHandle(c, gameID)
	if err != nil {
		return
	}

	if err := g.SetPlanetAutobuild(playerID, uint(planetID), req.Enabled); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	visible, err := g.GetVisibleGameForPlayer(playerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, visible)
	s.hub.BroadcastGameUpdate(g)
}
