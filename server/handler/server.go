package server

import (
	"log"
	"net/http"
	"ssg/ws"
	"ssg/game"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Server struct {
	GameList map[string]*game.Game
	router   *gin.Engine
	hub      *ws.Hub
	hubOnce  sync.Once
}

func (s *Server) GetGame(id string) *game.Game {
	return s.GameList[id]
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
		AllowOriginFunc: func(string) bool { return true },
		AllowWebSockets: true,
		AllowMethods:    []string{"GET", "POST", "PUT", "OPTIONS"},
		AllowHeaders:    []string{"Origin", "content-type", "game-id", "player-id"},
		ExposeHeaders:   []string{"Content-Length"},
		MaxAge:          12 * time.Hour,
	}))

	server := &Server{
		GameList: make(map[string]*game.Game),
	}

	server.hub = ws.NewHub(server)

	r.Use(AddGameState(server))
	server.registerRoutes(r)

	server.router = r
	server.startHub()
	return server
}

func (s *Server) registerRoutes(r *gin.Engine) {
	r.GET("/status", Status)
	r.POST("/games", NewGame)
	r.GET("/games", GetGameList)
	r.GET("/games/player-colors", GetPlayerColors)
	r.GET("/games/:gameID", GetGame)

	r.POST("/colonies", GetColonyList)
	r.POST("/colonies/build", BuildOnPlanet)

	r.POST("/action/:owner/colonize/:planet", ColonizePlanet)
	r.POST("/action/:owner/explore/:star", ExploreStar)

	resources := r.Group("/resources")
	{
		resources.GET("/shipdesigns", GetShipDesignList)
		resources.POST("/shipdesigns", CreateShipDesign)
		resources.PUT("/shipdesigns", UpdateShipDesign)

		resources.GET("/shipparts", GetShipPartsList)
	}

	r.GET("/gameupdates", func(c *gin.Context) {
		ws.Handler(s.hub, c)
	})
	r.POST("/turn", Turn)
}

func (s *Server) startHub() {
	s.hubOnce.Do(func() {
		go s.hub.Run()
	})
}

func (s *Server) Handler() http.Handler {
	s.startHub()
	return s.router
}

func (s *Server) Start() {
	s.startHub()
	err := s.router.Run() // listen and serve on 0.0.0.0:8080
	if err != nil {
		log.Printf("API error: %s", err)
	}
}
