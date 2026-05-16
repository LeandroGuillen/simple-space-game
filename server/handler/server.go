package server

import (
	"fmt"
	"net/http"
	"ssg/game"
	"ssg/ws"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Server struct {
	GameList        map[string]*game.Game
	CustomRaces     map[string]game.RaceDesign
	ResourceManager *game.ResourceManager
	RaceStore       *game.RaceStore
	router          *gin.Engine
	hub             *ws.Hub
	hubOnce         sync.Once
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

func NewServer() (*Server, error) {
	var resourceManager *game.ResourceManager
	var err error
	if resourceManager, err = game.NewResourceManager(); err != nil {
		return nil, fmt.Errorf("cannot load resources: %v", err)
	}

	raceStore := game.NewRaceStore(game.CustomRacesPath())
	customList, err := raceStore.Load(resourceManager)
	if err != nil {
		return nil, fmt.Errorf("cannot load custom races: %w", err)
	}
	customMap := make(map[string]game.RaceDesign)
	for _, cr := range customList {
		customMap[cr.Name] = cr
	}

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOriginFunc: func(string) bool { return true },
		AllowWebSockets: true,
		AllowMethods:    []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:    []string{"Origin", "content-type", "game-id", "player-id"},
		ExposeHeaders:   []string{"Content-Length"},
		MaxAge:          12 * time.Hour,
	}))

	server := &Server{
		GameList:        make(map[string]*game.Game),
		CustomRaces:     customMap,
		ResourceManager: resourceManager,
		RaceStore:       raceStore,
	}

	server.hub = ws.NewHub(server)

	r.Use(AddGameState(server))
	server.registerRoutes(r)

	server.router = r
	server.startHub()
	return server, nil
}

func (s *Server) persistCustomRaces() error {
	if s == nil || s.RaceStore == nil {
		return fmt.Errorf("race persistence unavailable")
	}
	list := make([]game.RaceDesign, 0, len(s.CustomRaces))
	for _, r := range s.CustomRaces {
		list = append(list, r)
	}
	return s.RaceStore.SaveAll(s.ResourceManager, list)
}

func (s *Server) registerRoutes(r *gin.Engine) {
	r.GET("/status", Status)
	r.POST("/games", NewGame)
	r.GET("/games", GetGameList)
	r.GET("/games/player-colors", GetPlayerColors)
	r.GET("/games/:gameID", GetGame)
	r.GET("/races", GetRaces)
	r.GET("/races/custom", GetCustomRaces)
	r.POST("/races/custom", SaveCustomRace)
	r.DELETE("/races/custom/:name", DeleteCustomRace)

	r.POST("/colonies", GetColonyList)

	r.POST("/action/:owner/colonize/:planet", ColonizePlanet)
	r.POST("/action/:owner/explore/:star", ExploreStar)
	r.PUT("/planets/:planet/autobuild", SetPlanetAutobuild)
	r.POST("/debug/research-all", DebugResearchAllTechs)

	resources := r.Group("/resources")
	{
		resources.GET("/shipdesigns", GetShipDesignList)
		resources.POST("/shipdesigns", CreateShipDesign)
		resources.PUT("/shipdesigns", UpdateShipDesign)

		resources.GET("/shipparts", GetShipPartsList)
		resources.GET("/governments", GetGovernments)
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

func (s *Server) Start(port string) error {
	s.startHub()
	return s.router.Run(":" + port)
}
