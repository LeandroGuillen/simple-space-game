package server

import (
	"fmt"
	"net/http"
	"path/filepath"
	"ssg/game"
	"ssg/model"
	"ssg/ws"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Server struct {
	gameList          map[string]*game.Game
	gameListMu        sync.RWMutex
	autosaves         *autosaveStore
	CustomRaces       map[string]game.RaceDesign
	CustomShipDesigns map[string]model.ShipDesign
	ResourceManager   *game.ResourceManager
	RaceStore         *game.RaceStore
	ShipStore         *game.ShipStore
	router            *gin.Engine
	hub               *ws.Hub
	hubOnce           sync.Once
	turnOrchestration sync.Mutex
	// debug enables /debug/* routes for this server instance only.
	debug bool
}

// ServerOptions configures NewServerWithOptions. Zero value matches NewServer().
type ServerOptions struct {
	Debug bool
}

// DebugEnabled reports whether this server registered debug endpoints.
func (s *Server) DebugEnabled() bool {
	return s != nil && s.debug
}

func (s *Server) getGame(id string) *game.Game {
	s.gameListMu.RLock()
	defer s.gameListMu.RUnlock()
	return s.gameList[id]
}

func (s *Server) putGame(id string, g *game.Game) {
	s.gameListMu.Lock()
	defer s.gameListMu.Unlock()
	s.gameList[id] = g
}

func (s *Server) listGames() map[string]*game.Game {
	s.gameListMu.RLock()
	defer s.gameListMu.RUnlock()
	result := make(map[string]*game.Game, len(s.gameList))
	for k, v := range s.gameList {
		result[k] = v
	}
	return result
}

func (s *Server) GetGame(id string) *game.Game {
	return s.getGame(id)
}

func AddGameState(s *Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("server", s)
		c.Next()
	}
}

func NewServer() (*Server, error) {
	return NewServerWithOptions(ServerOptions{})
}

func NewServerWithOptions(opts ServerOptions) (*Server, error) {
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
		gameList:          make(map[string]*game.Game),
		autosaves:         &autosaveStore{dir: filepath.Join(game.SavesDir(), ".autosaves")},
		CustomRaces:       customMap,
		CustomShipDesigns: customShipMap,
		ResourceManager:   resourceManager,
		RaceStore:         raceStore,
		ShipStore:         shipStore,
		debug:             opts.Debug,
	}

	server.rebuildShipDesignCatalog()
	restored, err := server.autosaves.Load(resourceManager)
	if err != nil {
		return nil, fmt.Errorf("cannot load autosaves: %w", err)
	}
	for _, restoredGame := range restored {
		server.putGame(restoredGame.ID, restoredGame)
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
	r.GET("/games/:gameID/save", GetGameSave)
	r.POST("/games/:gameID/save", SaveGameNow)
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

	r.PUT("/planets/:planet/autobuild", SetPlanetAutobuild)
	if s.debug {
		r.POST("/debug/research-all", DebugResearchAllTechs)
	}

	resources := r.Group("/resources")
	{
		resources.GET("/shipdesigns", GetShipDesignList)
		resources.POST("/shipdesigns", CreateShipDesign)
		resources.PUT("/shipdesigns", UpdateShipDesign)

		resources.GET("/shipparts", GetShipPartsList)
		resources.GET("/governments", GetGovernments)
	}

	r.GET("/battles/:battleID", GetBattle)
	r.POST("/battles/:battleID/action", PostBattleAction)

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
