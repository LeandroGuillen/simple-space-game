package server

import (
	"fmt"
	"net/http"
	"ssg/game"
	"ssg/model"
	"time"

	"github.com/gin-gonic/gin"
)

type NewGameRequest struct {
	NumStars       uint   `json:"num_stars,omitempty"`
	PlayerName     string `json:"player_name,omitempty"`
	PlayerColor    string `json:"player_color,omitempty"`
	Difficulty     string `json:"difficulty,omitempty"`
	GalaxySize     string `json:"galaxy_size,omitempty"`
	GalaxyAge      string `json:"galaxy_age,omitempty"`
	Players        uint   `json:"players,omitempty"`
	TechLevel      string `json:"tech_level,omitempty"`
	Race           string `json:"race,omitempty"`
	CustomRaceName string `json:"custom_race_name,omitempty"`
}

type GameListResponse struct {
	GameList []GameListResponseItem
}

type GameListResponseItem struct {
	GameID      string               `json:"game_id"`
	NumStars    int                  `json:"num_stars"`
	CurrentTurn int64                `json:"current_turn"`
	CreatedAt   *time.Time           `json:"created_at,omitempty"`
	LastTurnAt  *time.Time           `json:"last_turn_at,omitempty"`
	Players     []GameListPlayerInfo `json:"players"`
}

type GameListPlayerInfo struct {
	Name     string `json:"name"`
	Color    string `json:"color"`
	NumStars int    `json:"num_stars"`
}

type GetGalaxyRequest struct {
	GameID string `uri:"gameID" binding:"required" json:"game_id"`
}

func NewGame(c *gin.Context) {
	var req NewGameRequest
	_ = c.Bind(&req)

	rawServer, exists := c.Get("server")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "could not get server reference",
		})
		return
	}
	server := rawServer.(*Server)

	g, err := game.NewGame(server.ResourceManager, req.PlayerName, req.NumStars, req.PlayerColor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": fmt.Sprintf("error creating game: %s", err.Error()),
		})
		return
	}
	server.GameList[g.ID] = g

	c.JSON(http.StatusOK, gin.H{"ok": true, "gameID": g.ID})
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
	for gameID, g := range s.GameList {
		playerInfos := make([]GameListPlayerInfo, len(g.Players))
		for i, p := range g.Players {
			ownedStars := 0
			for _, star := range g.Galaxy.Stars {
				for _, planet := range star.Planets {
					if planet.Owner == p.ID {
						ownedStars++
						break // count each star once if it has any owned planet
					}
				}
			}
			playerInfos[i] = GameListPlayerInfo{
				Name:     p.Name,
				Color:    p.Color,
				NumStars: ownedStars,
			}
		}
		item := GameListResponseItem{
			GameID:      gameID,
			NumStars:    len(g.Galaxy.Stars),
			CurrentTurn: g.CurrentTurn,
			LastTurnAt:  g.LastTurnAt,
			Players:     playerInfos,
		}
		if !g.CreatedAt.IsZero() {
			c := g.CreatedAt
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
	g, gameFound := s.GameList[req.GameID]
	if !gameFound {
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
