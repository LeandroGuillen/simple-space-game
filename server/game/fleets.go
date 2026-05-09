package game

import (
	"fmt"
	"math"
	. "ssg/model"
)

func (g *Game) addShipToOrbitFleet(ownerID, starID uint, ship FleetShip) {
	if fleet := g.findOrbitFleet(ownerID, starID); fleet != nil {
		fleet.Ships = append(fleet.Ships, ship)
		return
	}
	star, err := g.getStar(starID)
	if err != nil {
		return
	}

	fleet := &Fleet{
		ID:     g.nextFleetID(),
		Owner:  ownerID,
		StarID: starID,
		X:      float64(star.X),
		Y:      float64(star.Y),
		OriginX: float64(star.X),
		OriginY: float64(star.Y),
		Ships:  []FleetShip{ship},
	}
	g.Fleets = append(g.Fleets, fleet)
	if player, err := g.getPlayer(ownerID); err == nil {
		player.Fleets = append(player.Fleets, fleet.ID)
	}
}

func (g *Game) findOrbitFleet(ownerID, starID uint) *Fleet {
	for _, fleet := range g.Fleets {
		if fleet.Owner == ownerID && fleet.StarID == starID {
			return fleet
		}
	}
	return nil
}

func (g *Game) nextFleetID() uint {
	maxID := uint(0)
	for _, fleet := range g.Fleets {
		if fleet.ID > maxID {
			maxID = fleet.ID
		}
	}
	return maxID + 1
}

func (g *Game) getFleet(id uint) (*Fleet, error) {
	for _, fleet := range g.Fleets {
		if fleet.ID == id {
			return fleet, nil
		}
	}
	return nil, fmt.Errorf("fleet with ID %d not found", id)
}

func (g *Game) issueFleetMove(playerID uint, command *FleetMoveCommand) error {
	fleet, err := g.getFleet(command.FleetID)
	if err != nil {
		return err
	}
	player, err := g.getPlayer(playerID)
	if err != nil {
		return err
	}
	if fleet.Owner != playerID {
		return fmt.Errorf("fleet %d does not belong to player %d", fleet.ID, playerID)
	}
	if (fleet.TurnsRemaining > 0 || fleet.DestinationStarID != 0) && !playerHasTech(player, "Hyperspace Communications") {
		return fmt.Errorf("fleet %d is already moving", fleet.ID)
	}
	if command.DestinationStarID == 0 || command.DestinationStarID == fleet.StarID {
		return fmt.Errorf("invalid fleet destination")
	}

	destination, err := g.getStar(command.DestinationStarID)
	if err != nil {
		return err
	}

	fleet.DestinationStarID = command.DestinationStarID
	fleet.OriginStarID = fleet.StarID
	fleet.OriginX = fleet.X
	fleet.OriginY = fleet.Y
	fleet.TotalTurns = calculateFleetTravelTurnsFromFloat(fleet.OriginX, fleet.OriginY, float64(destination.X), float64(destination.Y))
	fleet.TurnsRemaining = fleet.TotalTurns
	return nil
}

func (g *Game) advanceFleetMovements() {
	for _, fleet := range g.Fleets {
		if fleet.DestinationStarID == 0 || fleet.TurnsRemaining == 0 || fleet.TotalTurns == 0 {
			continue
		}
		destination, err := g.getStar(fleet.DestinationStarID)
		if err != nil {
			continue
		}

		fleet.TurnsRemaining--
		progress := float64(fleet.TotalTurns-fleet.TurnsRemaining) / float64(fleet.TotalTurns)
		fleet.X = lerp(fleet.OriginX, float64(destination.X), progress)
		fleet.Y = lerp(fleet.OriginY, float64(destination.Y), progress)
		if fleet.TurnsRemaining == 0 {
			fleet.StarID = fleet.DestinationStarID
			_ = g.ExploreStar(fleet.Owner, fleet.StarID)
			fleet.OriginStarID = 0
			fleet.OriginX = fleet.X
			fleet.OriginY = fleet.Y
			fleet.DestinationStarID = 0
			fleet.TotalTurns = 0
		}
	}
}

func calculateFleetTravelTurns(x1, y1, x2, y2 int64) uint {
	return calculateFleetTravelTurnsFromFloat(float64(x1), float64(y1), float64(x2), float64(y2))
}

func calculateFleetTravelTurnsFromFloat(x1, y1, x2, y2 float64) uint {
	const fleetSpeedParsecsPerTurn = 90.0
	dx := x2 - x1
	dy := y2 - y1
	distance := math.Sqrt(dx*dx + dy*dy)
	return uint(max(1, int(math.Ceil(distance/fleetSpeedParsecsPerTurn))))
}

func lerp(from, to, t float64) float64 {
	return from + (to-from)*t
}

func playerHasTech(player *Player, techName string) bool {
	for _, researched := range player.ResearchedTechs {
		if researched == techName {
			return true
		}
	}
	return false
}
