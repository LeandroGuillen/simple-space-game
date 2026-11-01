package server

import (
	"log"
	"net/http"
	"sort"
	"strings"

	"ssg/game"

	"github.com/gin-gonic/gin"
)

func GetRaces(c *gin.Context) {
	server, ok := serverFromContext(c)
	if !ok {
		return
	}
	out := make([]game.RaceDesign, 0, len(server.ResourceManager.BuiltInRaces)+len(server.CustomRaces))
	out = append(out, server.ResourceManager.BuiltInRaces...)

	custom := make([]game.RaceDesign, 0, len(server.CustomRaces))
	for _, race := range server.CustomRaces {
		custom = append(custom, race)
	}
	sort.Slice(custom, func(i, j int) bool {
		return custom[i].Name < custom[j].Name
	})
	out = append(out, custom...)

	c.JSON(http.StatusOK, gin.H{"races": out})
}

func GetCustomRaces(c *gin.Context) {
	server, ok := serverFromContext(c)
	if !ok {
		return
	}
	races := make([]game.RaceDesign, 0, len(server.CustomRaces))
	for _, race := range server.CustomRaces {
		races = append(races, race)
	}
	sort.Slice(races, func(i, j int) bool {
		return races[i].Name < races[j].Name
	})
	c.JSON(http.StatusOK, gin.H{"custom_races": races})
}

func SaveCustomRace(c *gin.Context) {
	server, ok := serverFromContext(c)
	if !ok {
		return
	}
	var req CustomRaceRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	renameFrom := strings.TrimSpace(req.RenameFrom)
	if req.SaveAsNew && renameFrom != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot combine save_as_new with rename_from"})
		return
	}
	if server.ResourceManager.IsBuiltInRaceName(req.Name) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot save over a built-in race name"})
		return
	}
	design, err := game.SaveableCustomRace(server.ResourceManager, game.RaceDesign{
		Name:       req.Name,
		Government: req.Government,
		TraitIDs:   req.TraitIDs,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.SaveAsNew {
		if _, exists := server.CustomRaces[design.Name]; exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "a custom race with this name already exists"})
			return
		}
		server.CustomRaces[design.Name] = design
		if err := server.persistCustomRaces(); err != nil {
			delete(server.CustomRaces, design.Name)
			log.Printf("cannot save custom races: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot save custom races"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"custom_race": design})
		return
	}

	if renameFrom != "" {
		if server.ResourceManager.IsBuiltInRaceName(renameFrom) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot rename from a built-in race"})
			return
		}
		backup, srcOk := server.CustomRaces[renameFrom]
		if !srcOk {
			c.JSON(http.StatusBadRequest, gin.H{"error": "source custom race not found"})
			return
		}
		if design.Name != renameFrom {
			if _, exists := server.CustomRaces[design.Name]; exists {
				c.JSON(http.StatusBadRequest, gin.H{"error": "a custom race with this name already exists"})
				return
			}
		}
		delete(server.CustomRaces, renameFrom)
		server.CustomRaces[design.Name] = design
		if err := server.persistCustomRaces(); err != nil {
			delete(server.CustomRaces, design.Name)
			server.CustomRaces[renameFrom] = backup
			log.Printf("cannot save custom races: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot save custom races"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"custom_race": design})
		return
	}

	server.CustomRaces[design.Name] = design
	if err := server.persistCustomRaces(); err != nil {
		delete(server.CustomRaces, design.Name)
		log.Printf("cannot save custom races: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot save custom races"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"custom_race": design})
}

func DeleteCustomRace(c *gin.Context) {
	server, ok := serverFromContext(c)
	if !ok {
		return
	}
	name := strings.TrimSpace(c.Param("name"))
	if server.ResourceManager.IsBuiltInRaceName(name) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete built-in race"})
		return
	}
	if _, exists := server.CustomRaces[name]; !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "custom race not found"})
		return
	}
	backup := server.CustomRaces[name]
	delete(server.CustomRaces, name)
	if err := server.persistCustomRaces(); err != nil {
		server.CustomRaces[name] = backup
		log.Printf("cannot save custom races: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot save custom races"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func serverFromContext(c *gin.Context) (*Server, bool) {
	rawServer, exists := c.Get("server")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "could not get server reference"})
		return nil, false
	}
	return rawServer.(*Server), true
}
