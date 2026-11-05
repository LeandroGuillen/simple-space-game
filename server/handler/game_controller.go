package server

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"ssg/game"
	"ssg/model"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type NewGameRequest struct {
	NumStars       uint               `json:"num_stars,omitempty"`
	PlayerName     string             `json:"player_name,omitempty"`
	PlayerColor    string             `json:"player_color,omitempty"`
	Difficulty     string             `json:"difficulty,omitempty"`
	GalaxySize     string             `json:"galaxy_size,omitempty"`
	GalaxyAge      string             `json:"galaxy_age,omitempty"`
	Players        uint               `json:"players,omitempty"`
	TechLevel      string             `json:"tech_level,omitempty"`
	Race           string             `json:"race,omitempty"`
	CustomRaceName string             `json:"custom_race_name,omitempty"`
	CustomRace     *CustomRaceRequest `json:"custom_race,omitempty"`
}

type CustomRaceRequest struct {
	Name       string   `json:"name"`
	Government string   `json:"government"`
	TraitIDs   []string `json:"trait_ids"`
	RenameFrom string   `json:"rename_from,omitempty"`
	SaveAsNew  bool     `json:"save_as_new,omitempty"`
}

type GameListResponse struct {
	GameList []GameListResponseItem
}

type GameListResponseItem struct {
	GameID      string               `json:"game_id"`
	NumStars    int                  `json:"num_stars"`
	CurrentTurn float64              `json:"current_turn"`
	CreatedAt   *time.Time           `json:"created_at,omitempty"`
	LastTurnAt  *time.Time           `json:"last_turn_at,omitempty"`
	Debug       bool                 `json:"debug,omitempty"`
	Players     []GameListPlayerInfo `json:"players"`
}

type GameListPlayerInfo struct {
	ID         uint   `json:"id"`
	Name       string `json:"name"`
	RaceName   string `json:"race_name"`
	Color      string `json:"color"`
	NumStars   int    `json:"num_stars"`
	Controller string `json:"controller,omitempty"`
}

type GetGalaxyRequest struct {
	GameID string `uri:"gameID" binding:"required" json:"game_id"`
}

func NewGame(c *gin.Context) {
	var req NewGameRequest
	// ShouldBindJSON leaves the response to us; c.Bind would abort with its
	// own 400 and let the handler write a second one.
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request body: %s", err.Error())})
		return
	}
	if req.Difficulty != "" || req.GalaxyAge != "" || req.TechLevel != "" {
		log.Printf("new game: difficulty, galaxy_age and tech_level are ignored")
	}

	rawServer, exists := c.Get("server")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "could not get server reference",
		})
		return
	}
	server := rawServer.(*Server)

	galaxySize, err := galaxySizeFromRequest(req.GalaxySize)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	race, err := raceFromRequest(server, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	g, err := game.NewGame(server.ResourceManager, game.NewGameOptions{
		PlayerName:  req.PlayerName,
		PlayerColor: req.PlayerColor,
		NumStars:    req.NumStars,
		GalaxySize:  galaxySize,
		Race:        race,
		NumPlayers:  req.Players,
	})
	if err != nil {
		// Setup values the caller can fix are client errors, not server bugs.
		var invalidInput *game.InvalidInputError
		if errors.As(err, &invalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": fmt.Sprintf("error creating game: %s", err.Error()),
		})
		return
	}
	server.putGame(g.ID, g)
	response := gin.H{"ok": true, "gameID": g.ID}
	if err := server.autosave(g); err != nil {
		log.Printf("initial autosave failed for game %s: %v", g.ID, err)
		response["warning"] = "Game created, but autosave failed. Use Save now to retry."
	}
	c.JSON(http.StatusOK, response)
}

// galaxySizeFromRequest validates the requested galaxy size. An empty size
// means the server default; unknown sizes are rejected instead of silently
// building a default map.
func galaxySizeFromRequest(size string) (string, error) {
	galaxySize, ok := game.NormalizeGalaxySize(size)
	if !ok {
		return "", fmt.Errorf(
			"unknown galaxy size %q (expected one of: %s)",
			size,
			strings.Join(game.BuiltinGalaxySizes, ", "),
		)
	}
	return galaxySize, nil
}

func raceFromRequest(server *Server, req NewGameRequest) (model.Race, error) {
	switch strings.TrimSpace(strings.ToLower(req.Race)) {
	case "custom":
		if req.CustomRace != nil {
			return game.BuildCustomRace(server.ResourceManager, game.RaceDesign{
				Name:       req.CustomRace.Name,
				Government: req.CustomRace.Government,
				TraitIDs:   req.CustomRace.TraitIDs,
			})
		}
		name := strings.TrimSpace(req.CustomRaceName)
		if name == "" {
			return model.Race{}, fmt.Errorf("custom race name or payload is required")
		}
		saved, ok := server.customRace(name)
		if !ok {
			return model.Race{}, fmt.Errorf("unknown custom race %q", name)
		}
		if !saved.Playable {
			return model.Race{}, fmt.Errorf("custom race %q is not playable", saved.Name)
		}
		return game.BuildCustomRace(server.ResourceManager, saved)
	case "psilon":
		return game.DefaultRace("Psilon"), nil
	default:
		key := strings.TrimSpace(strings.ToLower(req.Race))
		if key == "" {
			key = "human"
		}
		design, ok := server.ResourceManager.BuiltInRaceByKey(key)
		if !ok {
			return model.Race{}, fmt.Errorf("unknown race %q", req.Race)
		}
		if !design.Playable {
			return model.Race{}, fmt.Errorf("race %q is not playable", design.Name)
		}
		return game.BuildCustomRace(server.ResourceManager, game.RaceDesign{
			Name:       design.Name,
			Government: design.Government,
			TraitIDs:   design.TraitIDs,
		})
	}
}

func GetPlayerColors(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"max_players": model.MaxPlayers,
		"colors":      model.PlayerColors,
	})
}

func GetGameList(c *gin.Context) {
	rawServer, exists := c.Get("server")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "server object was not created",
		})
		return
	}

	s := rawServer.(*Server)

	var gameList = make([]GameListResponseItem, 0)
	for gameID, g := range s.listGames() {
		// Snapshot under the game's read lock — do not touch live fields here.
		summary := g.ListSummary()
		playerInfos := make([]GameListPlayerInfo, len(summary.Players))
		for i, p := range summary.Players {
			playerInfos[i] = GameListPlayerInfo{
				ID:         p.ID,
				Name:       p.Name,
				RaceName:   p.RaceName,
				Color:      p.Color,
				NumStars:   p.NumStars,
				Controller: p.Controller,
			}
		}
		item := GameListResponseItem{
			GameID:      gameID,
			NumStars:    summary.NumStars,
			CurrentTurn: summary.CurrentTurn,
			LastTurnAt:  summary.LastTurnAt,
			Debug:       summary.Debug,
			Players:     playerInfos,
		}
		if !summary.CreatedAt.IsZero() {
			c := summary.CreatedAt
			item.CreatedAt = &c
		}
		gameList = append(gameList, item)
	}

	c.JSON(http.StatusOK, gin.H{"game_list": gameList})
}

func GetGame(c *gin.Context) {
	var req GetGalaxyRequest
	if err := c.BindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gameID, playerID, err := readHeaders(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if gameID != req.GameID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "game id mismatch between URI and headers"})
		return
	}

	rawServer, exists := c.Get("server")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "server object was not created",
		})
		return
	}

	s := rawServer.(*Server)
	g := s.getGame(req.GameID)
	if g == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": fmt.Sprintf("game with id %s was not found", req.GameID),
		})
		return
	}

	visible, err := g.GetVisibleGameForPlayer(playerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, visible)
}

func GetGameSave(c *gin.Context) {
	var req GetGalaxyRequest
	if err := c.BindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gameID, playerID, err := readHeaders(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if gameID != req.GameID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "game id mismatch between URI and headers"})
		return
	}
	g, _, err := getGameHandle(c, req.GameID)
	if err != nil {
		return
	}
	data, err := g.MarshalSinglePlayerSave(playerID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	c.Data(http.StatusOK, "application/json; charset=utf-8", data)
}
