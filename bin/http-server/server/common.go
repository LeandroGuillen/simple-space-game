package server

import (
	"errors"
	"fmt"
	"net/http"
	"ssg/game"
	"strconv"

	"github.com/gin-gonic/gin"
)

func readHeaders(header http.Header) (gameID string, playerID uint, err error) {
	if len(header[GameIDHeader]) == 0 {
		return "", 0, errors.New(fmt.Sprintf("%s header missing", GameIDHeader))
	}
	gameID = header[GameIDHeader][0]

	playerIDstring := header[PlayerIDHeader][0]
	playerID64, err := strconv.ParseUint(playerIDstring, 10, 64)
	if len(header[PlayerIDHeader]) == 0 {
		return "", 0, errors.New(fmt.Sprintf("%s header missing", PlayerIDHeader))
	}
	playerID = uint(playerID64)

	return
}

func getGameHandle(c *gin.Context, gameID string) (g *game.Game, err error) {
	rawServer, serverExists := c.Get("server")
	if !serverExists {
		const msg = "server object was not created"
		c.JSON(http.StatusInternalServerError, gin.H{"message": msg})
		return nil, errors.New(msg)
	}

	s := rawServer.(*Server)
	g, gameFound := s.GameList[gameID]
	if !gameFound {
		msg := fmt.Sprintf("game with id %s was not found", gameID)
		c.JSON(http.StatusNotFound, gin.H{"message": msg})
		return nil, errors.New(msg)
	}

	return g, nil
}
