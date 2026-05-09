package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ColonizePlanetRequest struct {
	GameID string `binding:"required" json:"game_id"`
}

type ExploreStarRequest struct {
	GameID string `binding:"required" json:"game_id"`
}

type BuildOnPlanetRequest struct {
	GameID     string `binding:"required" json:"game_id"`
	OwnerID    uint   `binding:"required" json:"owner_id"`
	PlanetID   uint   `binding:"required" json:"planet_id"`
	BuildingID *uint  `binding:"required" json:"building_id"`
}

func ColonizePlanet(c *gin.Context) {
	var req ColonizePlanetRequest
	if err := c.Bind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
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
	g, gameFound := s.GameList[req.GameID]
	if !gameFound {
		c.JSON(http.StatusNotFound, gin.H{
			"message": fmt.Sprintf("game with id %s was not found", req.GameID),
		})
		return
	}

	if err := g.ColonizePlanet(uint(ownerID), uint(planetID), uint(planetID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	visible, err := g.GetVisibleGameForPlayer(uint(ownerID))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"game": visible})
}

func ExploreStar(c *gin.Context) {
	var req ExploreStarRequest
	if err := c.Bind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	ownerID, err := strconv.ParseUint(c.Param("owner"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "owner ID is invalid or not present"})
		return
	}

	starID, err := strconv.ParseUint(c.Param("star"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "star ID is invalid or not present"})
		return
	}

	g, _, err := getGameHandle(c, req.GameID)
	if err != nil {
		return
	}

	if err := g.ExploreStar(uint(ownerID), uint(starID)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	visible, err := g.GetVisibleGameForPlayer(uint(ownerID))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"game": visible})
}
