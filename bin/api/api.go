package api

import (
	"corion/bin/api/controller"
	"corion/game"
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type API struct {
	router *gin.Engine
	game   *game.Game
}

func AddGameState(g *game.Game) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("game", g)
		c.Next()
	}
}

func NewAPI() *API {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:4200"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "content-type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	g := &game.Game{}
	r.Use(AddGameState(g))

	r.GET("/status", controller.Status)
	r.POST("/galaxy", controller.NewGalaxy)
	r.GET("/galaxy", controller.GetGalaxy)
	r.POST("/randomGalaxy", controller.NewRandomGalaxy)

	r.GET("/action/:owner/colonize/:planet", controller.ColonizePlanet)

	return &API{
		router: r,
		game:   g,
	}
}

func (a *API) Start() {
	err := a.router.Run() // listen and serve on 0.0.0.0:8080
	if err != nil {
		log.Printf("API error: %s", err)
	}
}
