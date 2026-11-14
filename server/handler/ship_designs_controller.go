package server

import (
	"errors"
	"net/http"
	"ssg/game"
	"ssg/model"

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
	Beam     string `json:"beam"`
	Missile  string `json:"missile,omitempty"`
	Name     string `binding:"required" json:"name"`
	Drive    string `binding:"required" json:"drive"`
	Computer string `binding:"required" json:"computer"`
	Shield   string `binding:"required" json:"shield"`
	Armor    string `binding:"required" json:"armor"`
	HullSize string `binding:"required" json:"hull_size"`
	FuelCell string `binding:"required" json:"fuel_cell"`
}

func (req CreateShipDesignRequest) def() model.ShipDesignsDef {
	return model.ShipDesignsDef{
		Name:     req.Name,
		Beam:     req.Beam,
		Missile:  req.Missile,
		Drive:    req.Drive,
		Computer: req.Computer,
		Shield:   req.Shield,
		Armor:    req.Armor,
		HullSize: req.HullSize,
		FuelCell: req.FuelCell,
	}
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

	err = g.CreateShipDesignFromDef(playerID, req.def())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	} else {
		c.JSON(http.StatusCreated, gin.H{"ok": true})
		s.hub.BroadcastGameUpdate(g)
	}
}

func UpdateShipDesign(c *gin.Context) {
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

	err = g.UpdateShipDesignFromDef(playerID, req.def())
	if err != nil {
		status := http.StatusBadRequest
		var inUse *game.ShipDesignInUseError
		if errors.As(err, &inUse) {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"message": err.Error()})
		return
	} else {
		c.JSON(http.StatusCreated, gin.H{"ok": true})
		s.hub.BroadcastGameUpdate(g)
	}
}

func GetShipPartsList(c *gin.Context) {
	if c.GetHeader(GameIDHeader) == "" {
		s, ok := serverFromContext(c)
		if !ok {
			return
		}
		c.JSON(http.StatusOK, s.ResourceManager.ShipPartsCollection)
		return
	}

	gameID, _, err := readHeaders(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	g, _, err := getGameHandle(c, gameID)
	if err != nil {
		return
	}

	// Resources catalog is immutable after load, but still take the read lock
	// so every handler that reaches a live Game goes through a locked accessor.
	c.JSON(http.StatusOK, g.ShipPartsCollection())
}

// SuggestShipDesign returns an unsaved draft built from the requesting
// player's researched parts. It never creates or changes a design.
func SuggestShipDesign(c *gin.Context) {
	gameID, playerID, err := readHeaders(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	g, _, err := getGameHandle(c, gameID)
	if err != nil {
		return
	}

	def, err := g.SuggestShipDesign(playerID)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, model.ErrNoSuitableShipPart) {
			status = http.StatusUnprocessableEntity
		}
		c.JSON(status, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"suggestion": CreateShipDesignRequest{
		Name:     def.Name,
		Beam:     def.Beam,
		Missile:  def.Missile,
		Drive:    def.Drive,
		Computer: def.Computer,
		Shield:   def.Shield,
		Armor:    def.Armor,
		HullSize: def.HullSize,
		FuelCell: def.FuelCell,
	}})
}
