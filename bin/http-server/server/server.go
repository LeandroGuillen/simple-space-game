package server

import (
	"log"
	"ssg/bin/http-server/server/ws"
	"ssg/game"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Server struct {
	GameList map[string]*game.Game
	router   *gin.Engine
	hub      *ws.Hub
}

func AddGameState(s *Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("server", s)
		c.Next()
	}
}

func NewServer() *Server {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowWebSockets:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "content-type", "game-id", "player-id"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	server := &Server{
		GameList: make(map[string]*game.Game),
		hub:      ws.NewHub(),
	}

	r.Use(AddGameState(server))

	r.GET("/status", Status)
	r.POST("/games", NewGame)
	r.GET("/games", GetGameList)
	r.GET("/games/:gameID", GetGame)

	r.POST("/colonies", GetColonyList)
	r.POST("/colonies/build", BuildOnPlanet)

	r.POST("/action/:owner/colonize/:planet", ColonizePlanet)

	resources := r.Group("/resources")
	{
		resources.GET("/shipdesigns", GetShipDesignList)
		resources.POST("/shipdesigns", CreateShipDesign)
		resources.PUT("/shipdesigns", UpdateShipDesign)

		resources.GET("/shipparts", GetShipPartsList)
	}

	r.GET("/gameupdates", func(c *gin.Context) {
		ws.Handler(server.hub, c)
	})
	r.GET("/turn", Turn)

	server.router = r
	return server
}

func (s *Server) Start() {
	go s.hub.Run()
	err := s.router.Run() // listen and serve on 0.0.0.0:8080
	if err != nil {
		log.Printf("API error: %s", err)
	}
}
