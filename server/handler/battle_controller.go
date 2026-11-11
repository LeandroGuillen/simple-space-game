package server

import (
	"log"
	"net/http"
	"strconv"

	"ssg/model"

	"github.com/gin-gonic/gin"
)

func parseBattleID(c *gin.Context) (uint, bool) {
	id64, err := strconv.ParseUint(c.Param("battleID"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid battle id"})
		return 0, false
	}
	return uint(id64), true
}

// GetBattle returns the full tactical battle state, including the event log
// clients consume for animation.
func GetBattle(c *gin.Context) {
	gameID, playerID, err := readHeaders(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	g, _, err := getGameHandle(c, gameID)
	if err != nil {
		return
	}
	battleID, ok := parseBattleID(c)
	if !ok {
		return
	}
	battle, err := g.GetTacticalBattle(battleID, playerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"battle": battle})
}

// PostBattleAction applies one order (move/fire/done/wait/auto) for the
// acting ship and plays AI sides forward. When the action ends the battle the
// game update is broadcast so map clients refresh fleets.
func PostBattleAction(c *gin.Context) {
	gameID, playerID, err := readHeaders(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	g, s, err := getGameHandle(c, gameID)
	if err != nil {
		return
	}
	battleID, ok := parseBattleID(c)
	if !ok {
		return
	}
	var action model.BattleAction
	if err := c.ShouldBindJSON(&action); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	battle, err := g.ApplyBattleAction(battleID, playerID, action)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error(), "battle": battle})
		return
	}
	response := gin.H{"ok": true, "battle": battle}
	if battle.Status == model.TacticalBattleFinished {
		if err := s.autosave(g); err != nil {
			log.Printf("autosave after battle failed for game %s: %v", gameID, err)
			response["warning"] = "Battle finished, but autosave failed. Use Save now to retry."
		}
		s.hub.BroadcastGameUpdate(g)
	}
	c.JSON(http.StatusOK, response)
}
