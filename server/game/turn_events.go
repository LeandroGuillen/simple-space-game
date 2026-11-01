package game

import (
	"fmt"
	. "ssg/model"
)

func (g *Game) recordTurnEvent(event TurnEvent) {
	if g != nil && event.Owner != 0 {
		g.TurnEvents = append(g.TurnEvents, event)
	}
}
func (g *Game) recordBattleEvent(report *BattleReport) {
	if report == nil {
		return
	}
	for _, side := range report.Sides {
		outcome := "draw"
		if report.WinnerOwnerID != 0 {
			outcome = "defeat"
			if report.WinnerOwnerID == side.OwnerID {
				outcome = "victory"
			}
		}
		location := fmt.Sprintf("star %d", report.StarID)
		if star, err := g.getStar(report.StarID); err == nil && star != nil && star.Name != "" {
			location = star.Name
		}
		g.recordTurnEvent(TurnEvent{Owner: side.OwnerID, Kind: "battle", Text: fmt.Sprintf("Battle at %s: %s", location, outcome), StarID: report.StarID})
	}
}
func (g *Game) recordInvasionEvent(report *InvasionReport) {
	if report == nil {
		return
	}
	result := "repelled"
	if report.Cancelled {
		result = "cancelled"
	} else if report.PlanetCaptured {
		result = "captured"
	}
	text := fmt.Sprintf("Invasion of %s: %s", report.PlanetName, result)
	for _, id := range []uint{report.AttackerOwnerID, report.DefenderOwnerID} {
		g.recordTurnEvent(TurnEvent{Owner: id, Kind: "invasion", Text: text, PlanetID: report.PlanetID})
	}
}
