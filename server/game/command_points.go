package game

import . "ssg/model"

const (
	commandPointOverdrawBcPerCp int64 = 10
	// startingCommandPointsBonus is added to every empire's CP capacity (colonies + building grants).
	startingCommandPointsBonus int64 = 5
)

// shipCommandPointCost returns how many command points one ship consumes (Phase 1: colony ships only).
func shipCommandPointCost(ship FleetShip) int64 {
	if ship.Type == "colony_ship" {
		return 1
	}
	return 0
}

func (g *Game) playerCommandPointsCapacity(player *Player) int64 {
	colonies, err := g.GetColonyList(player.ID)
	if err != nil {
		return 0
	}
	var fromBuildings int64
	for _, planet := range colonies {
		for _, buildingName := range planet.Buildings {
			building, err := getBuildingFromName(g, buildingName)
			if err != nil {
				continue
			}
			for _, modifier := range building.Grants.Modifiers {
				if modifier.Name == "command_points" {
					fromBuildings += int64(modifier.Value)
				}
			}
		}
	}
	return startingCommandPointsBonus + int64(len(colonies)) + fromBuildings
}

func (g *Game) playerCommandPointsUsed(playerID uint) int64 {
	var used int64
	for _, fleet := range g.Fleets {
		if fleet.Owner != playerID {
			continue
		}
		for _, ship := range fleet.Ships {
			used += shipCommandPointCost(ship)
		}
	}
	return used
}

// HydrateCommandPoints fills command point fields on every player from the full game state.
func (g *Game) HydrateCommandPoints() {
	if g == nil || g.Resources == nil {
		return
	}
	for _, player := range g.Players {
		if player == nil {
			continue
		}
		capacity := g.playerCommandPointsCapacity(player)
		used := g.playerCommandPointsUsed(player.ID)
		player.CommandPointsStartingBonus = startingCommandPointsBonus
		player.CommandPointsCapacity = capacity
		player.CommandPointsUsed = used
		player.CommandPointsRemaining = capacity - used
		overdrawCP := used - capacity
		if overdrawCP < 0 {
			overdrawCP = 0
		}
		player.CommandOverdrawBC = overdrawCP * commandPointOverdrawBcPerCp
	}
}
