package game

import "ssg/model"

// evaluateConquest runs only after turn outcomes settle. A tactical battle may
// still destroy colony ships or release a postponed invasion, so wait for all
// pending battles before deciding. One-slot games remain a sandbox until the
// player is eliminated; otherwise they would be won after their first turn.
func (g *Game) evaluateConquest() {
	if g.Status != "active" {
		return
	}
	for _, b := range g.PendingBattles {
		if b != nil && b.Status == model.TacticalBattleActive {
			return
		}
	}
	colonies := make(map[uint]int)
	for _, star := range g.Galaxy.Stars {
		if star == nil {
			continue
		}
		for _, planet := range star.Planets {
			if planet != nil && planet.Owner != 0 {
				colonies[planet.Owner]++
			}
		}
	}
	colonyShips := make(map[uint]bool)
	for _, fleet := range g.Fleets {
		if fleet == nil {
			continue
		}
		for _, ship := range fleet.Ships {
			if ship.Type == "colony_ship" {
				colonyShips[fleet.Owner] = true
				break
			}
		}
	}
	var remaining uint
	count := 0
	for _, player := range g.Players {
		if player == nil || player.Eliminated {
			continue
		}
		if colonies[player.ID] == 0 && !colonyShips[player.ID] {
			player.Eliminated = true
			g.endWarsWith(player.ID)
			delete(g.TurnSubmissions, player.ID)
			continue
		}
		remaining = player.ID
		count++
	}
	if count == 0 || (count == 1 && len(g.Players) > 1 && colonies[remaining] > 0) {
		g.Status = "finished"
		g.WinnerID = remaining
		g.TurnSubmissions = make(map[uint]model.Commands)
	}
}
