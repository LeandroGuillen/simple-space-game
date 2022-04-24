package server

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	GameIDHeader   = "Game-Id"
	PlayerIDHeader = "Player-Id"
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

func GetShipDesignList(c *gin.Context) {
	gameID, playerID, err := readHeaders(c.Request.Header)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rawServer, serverExists := c.Get("server")
	if !serverExists {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "server object was not created"})
		return
	}

	s := rawServer.(*Server)
	g, gameFound := s.GameList[gameID]
	if !gameFound {
		c.JSON(http.StatusNotFound, gin.H{
			"message": fmt.Sprintf("game with id %s was not found", gameID),
		})
		return
	}

	if shipDesignsList, err := g.GetShipDesignsList(playerID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	} else {
		c.JSON(http.StatusOK, gin.H{"ship_designs": shipDesignsList})
	}
}
