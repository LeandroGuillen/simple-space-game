package server

import (
	"corion/game"
	"log"
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

	r.POST("/galaxy", NewGalaxy)
	r.GET("/galaxy/:gameID", GetGalaxy)
	r.POST("/randomGalaxy", NewRandomGalaxy)

	r.POST("/colonies", GetColonyList)

	r.POST("/action/:owner/colonize/:planet", ColonizePlanet)

	server.router = r
	return server
}

func (s *Server) Start() {
	err := s.router.Run() // listen and serve on 0.0.0.0:8080
	if err != nil {
		log.Printf("API error: %s", err)
	}
}
