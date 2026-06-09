package server

import (
	"fmt"
	"net/http"
	"ssg/game"
	"ssg/model"
	"ssg/ws"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Server struct {
	GameList          map[string]*game.Game
	CustomRaces       map[string]game.RaceDesign
	CustomShipDesigns map[string]model.ShipDesign
	ResourceManager   *game.ResourceManager
	RaceStore         *game.RaceStore
	ShipStore         *game.ShipStore
	router            *gin.Engine
	hub               *ws.Hub
	hubOnce           sync.Once
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

	shipStore := game.NewShipStore(game.CustomShipDesignsPath())
	customShipList, err := shipStore.Load(resourceManager)
	if err != nil {
		return nil, fmt.Errorf("cannot load custom ship designs: %w", err)
	}
	customShipMap := make(map[string]model.ShipDesign)
	for _, sd := range customShipList {
		customShipMap[sd.Name] = sd
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
		GameList:          make(map[string]*game.Game),
		CustomRaces:       customMap,
		CustomShipDesigns: customShipMap,
		ResourceManager:   resourceManager,
		RaceStore:         raceStore,
		ShipStore:         shipStore,
	}

	server.rebuildShipDesignCatalog()

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

func (s *Server) persistCustomShipDesigns() error {
	if s == nil || s.ShipStore == nil {
		return fmt.Errorf("ship design persistence unavailable")
	}
	list := make([]model.ShipDesign, 0, len(s.CustomShipDesigns))
	for _, d := range s.CustomShipDesigns {
		list = append(list, d)
	}
	return s.ShipStore.SaveAll(s.ResourceManager, list)
}

func (s *Server) rebuildShipDesignCatalog() {
	if s == nil || s.ResourceManager == nil {
		return
	}
	list := make([]model.ShipDesign, 0, len(s.CustomShipDesigns))
	for _, d := range s.CustomShipDesigns {
		list = append(list, d)
	}
	s.ResourceManager.ApplyMergedShipDesignCatalog(list)
}

func (s *Server) registerRoutes(r *gin.Engine) {
	r.GET("/status", Status)
	r.POST("/games", NewGame)
	r.GET("/games", GetGameList)
	r.GET("/games/player-colors", GetPlayerColors)
	r.GET("/games/:gameID", GetGame)
	r.GET("/scenarios", ListScenarios)
	r.POST("/scenarios/:name/load", LoadScenario)
	r.POST("/saves/upload", LoadUploadedSave)
	r.GET("/races", GetRaces)
	r.GET("/races/custom", GetCustomRaces)
	r.POST("/races/custom", SaveCustomRace)
	r.DELETE("/races/custom/:name", DeleteCustomRace)

	r.GET("/ship-designs", GetShipDesignTemplates)
	r.GET("/ship-designs/custom", GetCustomShipDesignTemplates)
	r.POST("/ship-designs/custom", SaveCustomShipDesign)
	r.DELETE("/ship-designs/custom/:name", DeleteCustomShipDesign)

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
