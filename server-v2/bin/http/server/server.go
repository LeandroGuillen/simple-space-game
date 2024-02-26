package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Player struct {
	ID   string
	Name string
}

type Game struct {
	ID      string
	Players map[string]*Player
}

type Server struct {
	GameList map[string]*Game
	mux      *http.ServeMux
}

func (s *Server) Start() {
	log.Fatal(http.ListenAndServe(":8080", s.mux))
}

func NewServer() *Server {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", logger(homeHandler))
	mux.HandleFunc("GET /status", logger(statusHandler))

	server := &Server{
		GameList: make(map[string]*Game),
		mux:      mux,
	}

	return server

	// r.GET("/status", Status)
	// r.POST("/games", NewGame)
	// r.GET("/games", GetGameList)
	// r.GET("/games/:gameID", GetGame)

	// r.POST("/colonies", GetColonyList)
	// r.POST("/colonies/build", BuildOnPlanet)

	// r.POST("/action/:owner/colonize/:planet", ColonizePlanet)

	// resources := r.Group("/resources")
	// {
	// 	resources.GET("/shipdesigns", GetShipDesignList)
	// 	resources.POST("/shipdesigns", CreateShipDesign)
	// 	resources.PUT("/shipdesigns", UpdateShipDesign)

	// 	resources.GET("/shipparts", GetShipPartsList)
	// }

	// r.GET("/gameupdates", func(c *gin.Context) {
	// 	ws.Handler(server.hub, c)
	// })
	// r.GET("/turn", Turn)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "This is not a website")
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	payload := map[string]interface{}{
		"ok": true,
	}
	json.NewEncoder(w).Encode(payload)
}
