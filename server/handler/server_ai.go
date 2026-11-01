package server

import (
	"encoding/json"
	"fmt"
	"log"
	"ssg/ai"
	"ssg/game"
	"ssg/model"
)

// submitServerAITurns uses each player's filtered view, just as the external
// agent does. Call with turnOrchestration held and only after a submission.
func submitServerAITurns(g *game.Game, status game.TurnStatus) (game.TurnStatus, bool, error) {
	summary := g.ListSummary()
	waiting := make(map[uint]bool, len(status.WaitingPlayerIDs))
	for _, id := range status.WaitingPlayerIDs {
		waiting[id] = true
	}
	// Human and external players must submit before server AI makes decisions.
	for _, p := range summary.Players {
		if waiting[p.ID] && p.Controller != "server_ai" {
			return status, false, nil
		}
	}
	for _, p := range summary.Players {
		if !waiting[p.ID] || p.Controller != "server_ai" {
			continue
		}
		visible, err := g.GetVisibleGameForPlayer(p.ID)
		if err != nil {
			return status, false, err
		}
		data, err := json.Marshal(visible)
		if err != nil {
			return status, false, err
		}
		var snapshot ai.VisibleGame
		if err = json.Unmarshal(data, &snapshot); err != nil {
			return status, false, err
		}
		decision := ai.SimplePolicy{}.Decide(&snapshot, p.ID)
		next, advanced, err := g.SubmitTurn(p.ID, decision.Commands)
		if err != nil {
			// A policy bug must not stall a game: submit an empty turn, while making
			// the failed decision visible to the server operator.
			log.Printf("server AI player %d rejected: %v (notes: %v)", p.ID, err, decision.Notes)
			next, advanced, err = g.SubmitTurn(p.ID, model.Commands{})
			if err != nil {
				return next, false, fmt.Errorf("server AI player %d: %w", p.ID, err)
			}
		}
		status = next
		if advanced {
			return status, true, nil
		}
	}
	return status, false, nil
}
