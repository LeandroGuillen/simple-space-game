package server

import (
	"net/http"
	"sort"

	"ssg/game"

	"github.com/gin-gonic/gin"
)

func GetCustomRaces(c *gin.Context) {
	server, ok := serverFromContext(c)
	if !ok {
		return
	}
	races := make([]game.CustomRaceDesign, 0, len(server.CustomRaces))
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
	design, err := game.SaveableCustomRace(game.CustomRaceDesign{
		Name:     req.Name,
		TraitIDs: req.TraitIDs,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	server.CustomRaces[design.Name] = design
	c.JSON(http.StatusOK, gin.H{"custom_race": design})
}

func DeleteCustomRace(c *gin.Context) {
	server, ok := serverFromContext(c)
	if !ok {
		return
	}
	name := c.Param("name")
	if _, exists := server.CustomRaces[name]; !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "custom race not found"})
		return
	}
	delete(server.CustomRaces, name)
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
