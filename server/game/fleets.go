package game

import (
	"fmt"
	"math"
	"sort"
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
		ID:      g.nextFleetID(),
		Owner:   ownerID,
		StarID:  starID,
		X:       float64(star.X),
		Y:       float64(star.Y),
		OriginX: float64(star.X),
		OriginY: float64(star.Y),
		Ships:   []FleetShip{ship},
	}
	g.Fleets = append(g.Fleets, fleet)
	if player, err := g.getPlayer(ownerID); err == nil {
		player.Fleets = append(player.Fleets, fleet.ID)
	}
}

func (g *Game) findOrbitFleet(ownerID, starID uint) *Fleet {
	for _, fleet := range g.Fleets {
		if fleet.Owner == ownerID && fleet.StarID == starID &&
			fleet.DestinationStarID == 0 && fleet.TurnsRemaining == 0 {
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

func (g *Game) removeFleet(fleet *Fleet) {
	id := fleet.ID
	filtered := make([]*Fleet, 0, len(g.Fleets))
	for _, f := range g.Fleets {
		if f.ID != id {
			filtered = append(filtered, f)
		}
	}
	g.Fleets = filtered
	if player, err := g.getPlayer(fleet.Owner); err == nil {
		next := make([]uint, 0, len(player.Fleets))
		for _, fid := range player.Fleets {
			if fid != id {
				next = append(next, fid)
			}
		}
		player.Fleets = next
	}
}

func normalizeFleetMoveShipIndexes(command *FleetMoveCommand, shipCount int) ([]uint, error) {
	if shipCount == 0 {
		return nil, fmt.Errorf("fleet has no ships")
	}
	var raw []uint
	if command.ShipIndexes == nil {
		// JSON omitted ship_indexes: legacy clients move the whole fleet.
		raw = make([]uint, shipCount)
		for i := 0; i < shipCount; i++ {
			raw[i] = uint(i)
		}
	} else if len(*command.ShipIndexes) == 0 {
		return nil, fmt.Errorf("no ships selected for fleet move")
	} else {
		raw = append([]uint(nil), *command.ShipIndexes...)
	}
	seen := make(map[uint]struct{})
	for _, i := range raw {
		if int(i) >= shipCount {
			return nil, fmt.Errorf("ship index %d out of range", i)
		}
		seen[i] = struct{}{}
	}
	out := make([]uint, 0, len(seen))
	for i := range seen {
		out = append(out, i)
	}
	sort.Slice(out, func(a, b int) bool { return out[a] < out[b] })
	return out, nil
}

func removeShipsByIndexes(fleet *Fleet, indexes []uint) []FleetShip {
	remove := make(map[uint]bool, len(indexes))
	for _, i := range indexes {
		remove[i] = true
	}
	toMove := make([]FleetShip, 0, len(indexes))
	kept := make([]FleetShip, 0, len(fleet.Ships)-len(indexes))
	for i, ship := range fleet.Ships {
		if remove[uint(i)] {
			toMove = append(toMove, ship)
		} else {
			kept = append(kept, ship)
		}
	}
	fleet.Ships = kept
	return toMove
}

func (g *Game) newFleetAtStar(ownerID, starID uint, ships []FleetShip) (*Fleet, error) {
	star, err := g.getStar(starID)
	if err != nil {
		return nil, err
	}
	player, err := g.getPlayer(ownerID)
	if err != nil {
		return nil, err
	}
	fleet := &Fleet{
		ID:      g.nextFleetID(),
		Owner:   ownerID,
		StarID:  starID,
		X:       float64(star.X),
		Y:       float64(star.Y),
		OriginX: float64(star.X),
		OriginY: float64(star.Y),
		Ships:   ships,
	}
	g.Fleets = append(g.Fleets, fleet)
	player.Fleets = append(player.Fleets, fleet.ID)
	return fleet, nil
}

func (g *Game) mergeShipsIntoOrbitFleet(ownerID, starID uint, ships []FleetShip) error {
	if len(ships) == 0 {
		return nil
	}
	if orbit := g.findOrbitFleet(ownerID, starID); orbit != nil {
		orbit.Ships = append(orbit.Ships, ships...)
		return nil
	}
	_, err := g.newFleetAtStar(ownerID, starID, ships)
	return err
}

func (g *Game) beginFleetMovement(fleet *Fleet, destStarID uint) error {
	destination, err := g.getStar(destStarID)
	if err != nil {
		return err
	}
	fleet.DestinationStarID = destStarID
	fleet.OriginStarID = fleet.StarID
	fleet.OriginX = fleet.X
	fleet.OriginY = fleet.Y
	fleet.TotalTurns = calculateFleetTravelTurnsFromFloat(fleet.OriginX, fleet.OriginY, float64(destination.X), float64(destination.Y))
	fleet.TurnsRemaining = fleet.TotalTurns
	return nil
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

	shipIdxs, err := normalizeFleetMoveShipIndexes(command, len(fleet.Ships))
	if err != nil {
		return err
	}

	inTransit := fleet.TurnsRemaining > 0 && fleet.DestinationStarID != 0

	// Recall selected ships to the origin star (merge into the orbiting stack).
	if command.DestinationStarID == fleet.StarID {
		if !inTransit {
			return nil
		}
		toLand := removeShipsByIndexes(fleet, shipIdxs)
		if err := g.mergeShipsIntoOrbitFleet(playerID, fleet.StarID, toLand); err != nil {
			return err
		}
		if len(fleet.Ships) == 0 {
			g.removeFleet(fleet)
		}
		return nil
	}

	if inTransit && !playerHasTech(player, "Hyperspace Communications") {
		return fmt.Errorf("fleet %d is already moving", fleet.ID)
	}

	if command.DestinationStarID == 0 {
		return fmt.Errorf("invalid fleet destination")
	}

	if _, err := g.getStar(command.DestinationStarID); err != nil {
		return err
	}

	if inTransit {
		if len(shipIdxs) == len(fleet.Ships) {
			return g.beginFleetMovement(fleet, command.DestinationStarID)
		}
		toMove := removeShipsByIndexes(fleet, shipIdxs)
		if len(toMove) == 0 {
			return nil
		}
		child, err := g.newFleetAtStar(playerID, fleet.StarID, toMove)
		if err != nil {
			return err
		}
		child.X = fleet.X
		child.Y = fleet.Y
		child.OriginX = fleet.X
		child.OriginY = fleet.Y
		if err := g.beginFleetMovement(child, command.DestinationStarID); err != nil {
			return err
		}
		if len(fleet.Ships) == 0 {
			g.removeFleet(fleet)
		}
		return nil
	}

	if len(shipIdxs) == len(fleet.Ships) {
		return g.beginFleetMovement(fleet, command.DestinationStarID)
	}

	toMove := removeShipsByIndexes(fleet, shipIdxs)
	if len(toMove) == 0 {
		return nil
	}
	originStarID := fleet.StarID
	if len(fleet.Ships) == 0 {
		g.removeFleet(fleet)
	}
	child, err := g.newFleetAtStar(playerID, originStarID, toMove)
	if err != nil {
		return err
	}
	return g.beginFleetMovement(child, command.DestinationStarID)
}

func (g *Game) issueFleetMoveCommands(playerID uint, commands []*FleetMoveCommand) error {
	grouped := make(map[uint][]*FleetMoveCommand)
	order := []uint{}
	for _, command := range commands {
		if command == nil {
			continue
		}
		if _, ok := grouped[command.FleetID]; !ok {
			order = append(order, command.FleetID)
		}
		grouped[command.FleetID] = append(grouped[command.FleetID], command)
	}
	for _, fleetID := range order {
		group := grouped[fleetID]
		if len(group) == 1 {
			if err := g.issueFleetMove(playerID, group[0]); err != nil {
				return err
			}
			continue
		}
		if err := g.issueFleetMoveGroup(playerID, group); err != nil {
			return err
		}
	}
	return nil
}

func (g *Game) issueFleetMoveGroup(playerID uint, commands []*FleetMoveCommand) error {
	fleet, err := g.getFleet(commands[0].FleetID)
	if err != nil {
		return err
	}
	if fleet.Owner != playerID {
		return fmt.Errorf("fleet %d does not belong to player %d", fleet.ID, playerID)
	}
	inTransit := fleet.TurnsRemaining > 0 && fleet.DestinationStarID != 0
	if inTransit {
		for _, command := range commands {
			if err := g.issueFleetMove(playerID, command); err != nil {
				return err
			}
		}
		return nil
	}

	type moveBatch struct {
		destinationStarID uint
		indexes           []uint
	}
	batches := make([]moveBatch, 0, len(commands))
	selected := make(map[uint]struct{})
	for _, command := range commands {
		if command.DestinationStarID == 0 {
			return fmt.Errorf("invalid fleet destination")
		}
		if command.DestinationStarID == fleet.StarID {
			continue
		}
		if _, err := g.getStar(command.DestinationStarID); err != nil {
			return err
		}
		indexes, err := normalizeFleetMoveShipIndexes(command, len(fleet.Ships))
		if err != nil {
			return err
		}
		for _, index := range indexes {
			if _, exists := selected[index]; exists {
				return fmt.Errorf("ship index %d assigned to multiple fleet moves", index)
			}
			selected[index] = struct{}{}
		}
		batches = append(batches, moveBatch{destinationStarID: command.DestinationStarID, indexes: indexes})
	}
	if len(batches) == 0 {
		return nil
	}

	originStarID := fleet.StarID
	originalShips := append([]FleetShip(nil), fleet.Ships...)
	remaining := make([]FleetShip, 0, len(originalShips)-len(selected))
	for i, ship := range originalShips {
		if _, moving := selected[uint(i)]; !moving {
			remaining = append(remaining, ship)
		}
	}
	fleet.Ships = remaining

	for _, batch := range batches {
		ships := make([]FleetShip, 0, len(batch.indexes))
		for _, index := range batch.indexes {
			ships = append(ships, originalShips[index])
		}
		child, err := g.newFleetAtStar(playerID, originStarID, ships)
		if err != nil {
			return err
		}
		if err := g.beginFleetMovement(child, batch.destinationStarID); err != nil {
			return err
		}
	}
	if len(fleet.Ships) == 0 {
		g.removeFleet(fleet)
	}
	return nil
}

func (g *Game) advanceFleetMovements() {
	for i := len(g.Fleets) - 1; i >= 0; i-- {
		fleet := g.Fleets[i]
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
			destStarID := fleet.DestinationStarID
			fleet.StarID = destStarID
			_ = g.ExploreStar(fleet.Owner, fleet.StarID)
			if orbit := g.findOrbitFleet(fleet.Owner, destStarID); orbit != nil && orbit.ID != fleet.ID {
				orbit.Ships = append(orbit.Ships, fleet.Ships...)
				g.removeFleet(fleet)
				continue
			}
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
