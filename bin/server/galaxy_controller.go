package server

import (
	"corion/game"
	"corion/game/builder"
	"fmt"
	"math"
	"math/rand"
	"net/http"

	"github.com/gin-gonic/gin"
)

type NewGalaxyRequest struct {
	NumStars uint
}

type NewRandomGalaxyRequest struct{}

type GetGalaxyRequest struct {
	GameID string `uri:"gameID" binding:"required" json:"game_id"`
}

func NewGalaxy(c *gin.Context) {
	var req NewGalaxyRequest
	_ = c.Bind(&req)

	if rawGame, exists := c.Get("game"); exists {
		g := rawGame.(*game.Game)
		galaxy := g.NewGalaxy(req.NumStars)
		g.Galaxy = galaxy
		g.Players = []game.Player{
			{ID: 1},
		}
		c.JSON(http.StatusOK, galaxy)
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "ay ay ay!",
		})
	}
}

func NewRandomGalaxy(c *gin.Context) {
	var req NewRandomGalaxyRequest
	_ = c.Bind(&req)

	// new galaxy parameters
	var stdDev float64 = 2
	var mean float64 = 6
	n := uint(math.Abs(math.Floor(rand.NormFloat64()*stdDev + mean)))
	galaxy := builder.NewGalaxy(n)

	c.JSON(http.StatusOK, galaxy)
}

func GetGalaxy(c *gin.Context) {
	var request GetGalaxyRequest
	if err := c.ShouldBindUri(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	rawServer, exists := c.Get("server")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "server object was not created",
		})
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

	c.JSON(http.StatusOK, g.Galaxy)
}
