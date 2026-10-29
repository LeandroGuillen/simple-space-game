package game

import (
	"math"

	. "ssg/model"
)

const (
	commandPointOverdrawBcPerCp int64 = 10
	// startingCommandPointsBonus is added to every empire's CP capacity (colonies + building grants).
	startingCommandPointsBonus int64 = 5
)

func shipCommandPointCost(ship FleetShip) int64 {
	switch {
	case ship.Type == "colony_ship":
		return 1
	case ship.HullSize == "Frigate":
		return 1
	default:
		return 0
	}
}

func (g *Game) playerCommandPointsCapacity(player *Player) int64 {
	// Unlocked: called from HydrateCommandPoints while SubmitTurn holds the
	// write lock, or while hydrating a private visible-game copy (mu == nil).
	colonies, err := g.colonyListUnlocked(player.ID)
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
	baseCapacity := startingCommandPointsBonus + int64(len(colonies)) + fromBuildings
	multiplier := governmentCommandPointsMultiplier(g, player)
	if multiplier < 0 {
		multiplier = 0
	}
	return int64(math.Floor(float64(baseCapacity) * multiplier))
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

// hydratePlayerCommandPoints fills command point fields on one player from the
// full game state. It does not take g.mu.
func (g *Game) hydratePlayerCommandPoints(player *Player) {
	if g == nil || g.Resources == nil || player == nil {
		return
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

// HydrateCommandPoints fills command point fields on every player from the full
// game state. It does not take g.mu: callers must hold the write lock on the
// live game, or invoke it only on a private deep copy (never mutate live state
// from a read path such as GetVisibleGameForPlayer).
func (g *Game) HydrateCommandPoints() {
	if g == nil {
		return
	}
	for _, player := range g.Players {
		g.hydratePlayerCommandPoints(player)
	}
}
