package server

import (
	"errors"
	"log"
	"net/http"

	"ssg/game"
	"ssg/model"

	"github.com/gin-gonic/gin"
)

type TurnRequest struct {
	Commands model.Commands `json:"commands"`
}

func Turn(c *gin.Context) {
	gameID, playerID, err := readHeaders(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req TurnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	g, s, err := getGameHandle(c, gameID)
	if err != nil {
		return
	}

	turnStatus, advanced, err := g.SubmitTurn(playerID, req.Commands)
	if err != nil {
		// The dropped player's AI watches readiness updates to resubmit,
		// including when another player triggered the failed advance.
		var submissionErr *game.TurnSubmissionError
		if errors.As(err, &submissionErr) {
			s.hub.BroadcastGameUpdate(g)
			if submissionErr.PlayerID != playerID {
				c.JSON(http.StatusOK, gin.H{
					"ok":          true,
					"advanced":    false,
					"turn_status": turnStatus,
					"message":     err.Error(),
				})
				return
			}
		}
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	response := gin.H{"ok": true, "advanced": advanced, "turn_status": turnStatus}
	if advanced {
		if err := s.autosave(g); err != nil {
			log.Printf("autosave failed for game %s: %v", gameID, err)
			response["warning"] = "Turn completed, but autosave failed. Use Save now to retry."
		}
	}
	c.JSON(http.StatusOK, response)
	s.hub.BroadcastGameUpdate(g)
}
