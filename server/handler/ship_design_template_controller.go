package server

import (
	"log"
	"net/http"
	"strings"

	"ssg/game"
	"ssg/model"

	"github.com/gin-gonic/gin"
)

type ShipDesignTemplate struct {
	Beam     string `json:"beam"`
	Name     string `json:"name"`
	Drive    string `json:"drive"`
	Computer string `json:"computer"`
	Shield   string `json:"shield"`
	Armor    string `json:"armor"`
	HullSize string `json:"hull_size"`
	FuelCell string `json:"fuel_cell"`
	BuiltIn  bool   `json:"built_in"`
}

type CustomShipDesignRequest struct {
	Beam       string `json:"beam"`
	Name       string `json:"name"`
	Drive      string `json:"drive"`
	Computer   string `json:"computer"`
	Shield     string `json:"shield"`
	Armor      string `json:"armor"`
	HullSize   string `json:"hull_size"`
	FuelCell   string `json:"fuel_cell"`
	RenameFrom string `json:"rename_from,omitempty"`
	SaveAsNew  bool   `json:"save_as_new,omitempty"`
}

func shipDesignTemplateFromModel(d model.ShipDesign, builtIn bool) ShipDesignTemplate {
	return ShipDesignTemplate{
		Name:     d.Name,
		Beam:     game.ShipDesignPartsDef(d).Beam,
		Drive:    d.Drive.Name,
		Computer: d.Computer.Name,
		Shield:   d.Shield.Name,
		Armor:    d.Armor.Name,
		HullSize: d.HullSize.Name,
		FuelCell: d.FuelCell.Name,
		BuiltIn:  builtIn,
	}
}

func GetShipDesignTemplates(c *gin.Context) {
	server, ok := serverFromContext(c)
	if !ok {
		return
	}
	custom := server.customShipDesignList()
	out := make([]ShipDesignTemplate, 0, len(server.ResourceManager.BuiltinShipDesigns)+len(custom))
	for _, d := range server.ResourceManager.BuiltinShipDesigns {
		out = append(out, shipDesignTemplateFromModel(d, true))
	}
	for _, d := range custom {
		out = append(out, shipDesignTemplateFromModel(d, false))
	}
	c.JSON(http.StatusOK, gin.H{"ship_designs": out})
}

func GetCustomShipDesignTemplates(c *gin.Context) {
	server, ok := serverFromContext(c)
	if !ok {
		return
	}
	custom := server.customShipDesignList()
	out := make([]ShipDesignTemplate, 0, len(custom))
	for _, d := range custom {
		out = append(out, shipDesignTemplateFromModel(d, false))
	}
	c.JSON(http.StatusOK, gin.H{"custom_ship_designs": out})
}

func SaveCustomShipDesign(c *gin.Context) {
	server, ok := serverFromContext(c)
	if !ok {
		return
	}
	var req CustomShipDesignRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	renameFrom := strings.TrimSpace(req.RenameFrom)
	if req.SaveAsNew && renameFrom != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot combine save_as_new with rename_from"})
		return
	}
	if server.ResourceManager.IsBuiltInShipDesignName(req.Name) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot save over a built-in ship design name"})
		return
	}

	design, err := game.SaveableCustomShipDesign(server.ResourceManager, model.ShipDesignsDef{
		Name:     req.Name,
		Beam:     req.Beam,
		Drive:    req.Drive,
		Computer: req.Computer,
		Shield:   req.Shield,
		Armor:    req.Armor,
		HullSize: req.HullSize,
		FuelCell: req.FuelCell,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	server.customMu.Lock()
	defer server.customMu.Unlock()

	// Guard both ends of a rename, including names that could shadow a
	// player-local design. Keep turns and game insertion blocked until published.
	release, err := server.guardShipDesignChanges(design.Name, renameFrom)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	defer release()

	if req.SaveAsNew {
		if _, exists := server.CustomShipDesigns[design.Name]; exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "a custom ship design with this name already exists"})
			return
		}
		server.CustomShipDesigns[design.Name] = design
		if err := server.persistCustomShipDesigns(); err != nil {
			delete(server.CustomShipDesigns, design.Name)
			log.Printf("cannot save custom ship designs: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot save custom ship designs"})
			return
		}
		server.rebuildShipDesignCatalog()
		c.JSON(http.StatusOK, gin.H{"custom_ship_design": shipDesignTemplateFromModel(design, false)})
		return
	}

	if renameFrom != "" {
		if server.ResourceManager.IsBuiltInShipDesignName(renameFrom) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot rename from a built-in ship design"})
			return
		}
		backup, srcOk := server.CustomShipDesigns[renameFrom]
		if !srcOk {
			c.JSON(http.StatusBadRequest, gin.H{"error": "source custom ship design not found"})
			return
		}
		if design.Name != renameFrom {
			if _, exists := server.CustomShipDesigns[design.Name]; exists {
				c.JSON(http.StatusBadRequest, gin.H{"error": "a custom ship design with this name already exists"})
				return
			}
		}
		delete(server.CustomShipDesigns, renameFrom)
		server.CustomShipDesigns[design.Name] = design
		if err := server.persistCustomShipDesigns(); err != nil {
			delete(server.CustomShipDesigns, design.Name)
			server.CustomShipDesigns[renameFrom] = backup
			log.Printf("cannot save custom ship designs: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot save custom ship designs"})
			return
		}
		server.rebuildShipDesignCatalog()
		c.JSON(http.StatusOK, gin.H{"custom_ship_design": shipDesignTemplateFromModel(design, false)})
		return
	}

	server.CustomShipDesigns[design.Name] = design
	if err := server.persistCustomShipDesigns(); err != nil {
		delete(server.CustomShipDesigns, design.Name)
		log.Printf("cannot save custom ship designs: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot save custom ship designs"})
		return
	}
	server.rebuildShipDesignCatalog()
	c.JSON(http.StatusOK, gin.H{"custom_ship_design": shipDesignTemplateFromModel(design, false)})
}

func DeleteCustomShipDesign(c *gin.Context) {
	server, ok := serverFromContext(c)
	if !ok {
		return
	}
	name := strings.TrimSpace(c.Param("name"))
	if server.ResourceManager.IsBuiltInShipDesignName(name) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete built-in ship design"})
		return
	}
	server.customMu.Lock()
	defer server.customMu.Unlock()
	if _, exists := server.CustomShipDesigns[name]; !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "custom ship design not found"})
		return
	}
	release, err := server.guardShipDesignChanges(name)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	defer release()
	backup := server.CustomShipDesigns[name]
	delete(server.CustomShipDesigns, name)
	if err := server.persistCustomShipDesigns(); err != nil {
		server.CustomShipDesigns[name] = backup
		log.Printf("cannot save custom ship designs: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot save custom ship designs"})
		return
	}
	server.rebuildShipDesignCatalog()
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Caller holds customMu. Lock order: customMu -> gameListMu -> game locks.
// Save uploads hold customMu for validation through insertion as well.
func (s *Server) guardShipDesignChanges(names ...string) (func(), error) {
	s.gameListMu.RLock()
	releases := []func(){s.gameListMu.RUnlock}
	release := func() {
		for i := len(releases) - 1; i >= 0; i-- {
			releases[i]()
		}
	}
	for _, g := range s.gameList {
		unlock, err := g.GuardShipDesignChanges(names...)
		if err != nil {
			release()
			return nil, err
		}
		releases = append(releases, unlock)
	}
	return release, nil
}
