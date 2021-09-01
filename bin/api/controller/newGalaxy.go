package controller

import (
	"corion/game"
	"corion/game/builder"
	"math"
	"math/rand"
	"net/http"

	"github.com/gin-gonic/gin"
)

type NewGalaxyRequest struct {
	NumStars int
}

type NewRandomGalaxyRequest struct{}

func NewGalaxy(c *gin.Context) {
	var req NewGalaxyRequest
	_ = c.Bind(&req)

	if rawGame, exists := c.Get("game"); exists {
		g := rawGame.(*game.Game)
		galaxy := g.NewGalaxy(req.NumStars)
		c.JSON(http.StatusOK, galaxy)
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "ay ay ay!",
		})
	}

}

func NewRandomGalaxy(c *gin.Context) {
	var req NewRandomGalaxyRequest
	_ = c.Bind(&req)

	// new galaxy parameters
	var stdDev float64 = 2
	var mean float64 = 6
	n := int(math.Abs(math.Floor(rand.NormFloat64()*stdDev + mean)))
	galaxy := builder.NewGalaxy(n)

	c.JSON(http.StatusOK, galaxy)
}
