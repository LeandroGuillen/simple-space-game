package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type GovernmentResponse struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	PickCost    int    `json:"pick_cost"`
	Default     bool   `json:"default,omitempty"`
}

func GetGovernments(c *gin.Context) {
	server, ok := serverFromContext(c)
	if !ok {
		return
	}

	governments := make([]GovernmentResponse, 0)
	for _, tech := range server.ResourceManager.Resources.TechTree.AllTechs {
		if tech.Type != "government" || len(tech.Upgrades) > 0 {
			continue
		}
		governments = append(governments, GovernmentResponse{
			Name:        tech.Name,
			Description: tech.Description,
			PickCost:    tech.PickCost,
			Default:     tech.Default,
		})
	}
	c.JSON(http.StatusOK, gin.H{"governments": governments})
}
