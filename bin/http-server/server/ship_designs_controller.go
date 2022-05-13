package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetShipDesignList(c *gin.Context) {
	gameID, playerID, err := readHeaders(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	g, _, err := getGameHandle(c, gameID)
	if err != nil {
		return
	}

	if shipDesignsList, err := g.GetShipDesignsList(playerID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	} else {
		c.JSON(http.StatusOK, shipDesignsList)
	}
}

type CreateShipDesignRequest struct {
	Name     string `binding:"required" json:"name"`
	Drive    string `binding:"required" json:"drive"`
	Computer string `binding:"required" json:"computer"`
	Shield   string `binding:"required" json:"shield"`
	Armor    string `binding:"required" json:"armor"`
	HullSize string `binding:"required" json:"hull_size"`
	FuelCell string `binding:"required" json:"fuel_cell"`
}

func CreateShipDesign(c *gin.Context) {
	gameID, playerID, err := readHeaders(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	g, s, err := getGameHandle(c, gameID)
	if err != nil {
		return
	}

	var req CreateShipDesignRequest
	if err := c.Bind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = g.CreateShipDesign(
		playerID,
		req.Name,
		req.Drive,
		req.Computer,
		req.Shield,
		req.Armor,
		req.HullSize,
		req.FuelCell,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	} else {
		c.JSON(http.StatusCreated, gin.H{"ok": true})
		s.hub.BroadcastGameUpdate(g)
	}
}

func GetShipPartsList(c *gin.Context) {
	gameID, _, err := readHeaders(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	g, _, err := getGameHandle(c, gameID)
	if err != nil {
		return
	}

	c.JSON(http.StatusOK, g.Resources.ShipPartsCollection)
}
