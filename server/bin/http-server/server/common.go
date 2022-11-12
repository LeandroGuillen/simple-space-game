package server

import (
	"errors"
	"fmt"
	"net/http"
	"ssg/game"
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	GameIDHeader   = "Game-Id"
	PlayerIDHeader = "Player-Id"
)

func readHeaders(c *gin.Context) (gameID string, playerID uint, err error) {
	if c.GetHeader(GameIDHeader) == "" {
		return "", 0, errors.New(fmt.Sprintf("%s header missing", GameIDHeader))
	}
	gameID = c.GetHeader(GameIDHeader)

	playerIDstring := c.GetHeader(PlayerIDHeader)
	if playerIDstring == "" {
		return "", 0, errors.New(fmt.Sprintf("%s header missing", PlayerIDHeader))
	}
	playerID64, err := strconv.ParseUint(playerIDstring, 10, 64)
	playerID = uint(playerID64)

	return
}

func getGameHandle(c *gin.Context, gameID string) (g *game.Game, s *Server, err error) {
	rawServer, serverExists := c.Get("server")
	if !serverExists {
		const msg = "server object was not created"
		c.JSON(http.StatusInternalServerError, gin.H{"message": msg})
		return nil, nil, errors.New(msg)
	}

	s = rawServer.(*Server)
	g, gameFound := s.GameList[gameID]
	if !gameFound {
		msg := fmt.Sprintf("game with id %s was not found", gameID)
		c.JSON(http.StatusNotFound, gin.H{"message": msg})
		return nil, nil, errors.New(msg)
	}

	return g, s, nil
}
