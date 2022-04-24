package server

import (
	"log"
	"ssg/game"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Server struct {
	GameList map[string]*game.Game
	router   *gin.Engine
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
		AllowOrigins:     []string{"http://localhost:4200"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "content-type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	server := &Server{}
	server.GameList = make(map[string]*game.Game)
	r.Use(AddGameState(server))

	r.GET("/status", Status)
	r.POST("/games", NewGame)
	r.GET("/games", GetGameList)
	r.GET("/games/:gameID", GetGame)

	r.POST("/colonies", GetColonyList)
	r.POST("/colonies/build", BuildOnPlanet)
	r.POST("/turn", Turn)

	r.POST("/action/:owner/colonize/:planet", ColonizePlanet)

	resources := r.Group("/resources")
	{
		resources.GET("/shipdesigns", GetShipDesignList)
		resources.POST("/shipdesigns", CreateShipDesign)
	}

	server.router = r
	return server
}

func (s *Server) Start() {
	err := s.router.Run() // listen and serve on 0.0.0.0:8080
	if err != nil {
		log.Printf("API error: %s", err)
	}
}
