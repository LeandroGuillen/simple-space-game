package game

import (
	"fmt"
	"math"
	"sort"
	. "ssg/model"
	"strings"
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

// fleetDepartureStar returns the orbit a moving fleet departed from.
func fleetDepartureStar(fleet *Fleet) uint {
	if fleet.DestinationStarID != 0 && fleet.OriginStarID != 0 {
		return fleet.OriginStarID
	}
	return fleet.StarID
}

func (g *Game) beginFleetMovement(fleet *Fleet, destStarID uint) error {
	destination, err := g.getStar(destStarID)
	if err != nil {
		return err
	}
	originStarID := fleetDepartureStar(fleet)
	turns := uint(1)
	if !g.isStarGateTravel(fleet.Owner, originStarID, destStarID) {
		turns, err = g.calculateFleetTravelTurnsForShips(fleet.Owner, fleet.Ships, fleet.X, fleet.Y, float64(destination.X), float64(destination.Y))
		if err != nil {
			return err
		}
	}
	fleet.DestinationStarID = destStarID
	fleet.OriginStarID = originStarID
	fleet.OriginX, fleet.OriginY = fleet.X, fleet.Y
	fleet.StarID = 0
	fleet.TotalTurns, fleet.TurnsRemaining = turns, turns
	return nil
}

// isStarGateTravel returns true if both origin and destination are player-owned
// colonies and the player has researched Star Gate technology
func (g *Game) isStarGateTravel(playerID uint, originStarID, destStarID uint) bool {
	player, err := g.getPlayer(playerID)
	if err != nil {
		return false
	}
	if !playerHasTech(player, "Star Gate") {
		return false
	}
	// Both stars must have player-owned planets (colonies)
	originOwned := g.starHasPlayerColony(originStarID, playerID)
	destOwned := g.starHasPlayerColony(destStarID, playerID)
	return originOwned && destOwned
}

// starHasPlayerColony checks if a star has at least one planet owned by the player
func (g *Game) starHasPlayerColony(starID uint, playerID uint) bool {
	star, err := g.getStar(starID)
	if err != nil {
		return false
	}
	for _, planet := range star.Planets {
		if planet != nil && planet.Owner == playerID {
			return true
		}
	}
	return false
}

func distanceParsecs(x1, y1, x2, y2 float64) float64 {
	return DistanceParsecs(x1, y1, x2, y2)
}

// DistanceParsecs is the shared map-distance helper used by fuel-range checks
// (submit validation and AI scouting). 1 parsec = parsecInPixels map units.
func DistanceParsecs(x1, y1, x2, y2 float64) float64 {
	dx := x2 - x1
	dy := y2 - y1
	return math.Sqrt(dx*dx+dy*dy) / float64(parsecInPixels)
}

func fuelCellUnlimited(cell FuelCell) bool {
	return strings.EqualFold(cell.Name, "Thorium") || cell.Autonomy >= 100
}

func (g *Game) playerOwnedSupplyStars(playerID uint) []*Star {
	if g == nil {
		return nil
	}
	stars := make([]*Star, 0)
	for _, star := range g.Galaxy.Stars {
		if star == nil {
			continue
		}
		for _, planet := range star.Planets {
			if planet != nil && planet.Owner == playerID {
				stars = append(stars, star)
				break
			}
		}
	}
	return stars
}

func bestAvailableFuelCell(player *Player, resources *Resources) (FuelCell, bool) {
	if player == nil || resources == nil || resources.ShipPartsCollection == nil {
		return FuelCell{}, false
	}
	return bestFuelCellByNames(player.Available.FuelCells, resources.FuelCells)
}

func bestAvailableDrive(player *Player, resources *Resources) (Drive, bool) {
	if player == nil || resources == nil || resources.ShipPartsCollection == nil {
		return Drive{}, false
	}
	var best Drive
	found := false
	for _, driveName := range player.Available.Drives {
		drive, err := DriveFromName(driveName, resources.Drives)
		if err != nil {
			continue
		}
		if !found || drive.InterstellarSpeed > best.InterstellarSpeed {
			best = drive
			found = true
		}
	}
	return best, found
}

func (g *Game) findShipDesignForFleetShip(player *Player, ship FleetShip) (ShipDesign, bool) {
	if ship.DesignName == "" {
		return ShipDesign{}, false
	}
	var playerDesigns, stock []ShipDesign
	if player != nil {
		playerDesigns = player.ShipDesigns
	}
	if g != nil && g.Resources != nil && g.Resources.ShipDesignCollection != nil {
		stock = g.Resources.ShipDesigns
	}
	return findDesignForFuelRange(ship.DesignName, playerDesigns, stock)
}

func (g *Game) fleetShipFuelRange(player *Player, ship FleetShip) (rangeParsecs int64, unlimited bool, ok bool) {
	return ResolveShipFuelRange(ship, g.fuelRangeLookup(player))
}

func (g *Game) fuelRangeLookup(player *Player) FuelRangeLookup {
	lookup := FuelRangeLookup{}
	if g != nil && g.Resources != nil {
		if g.Resources.ShipPartsCollection != nil {
			lookup.FuelCells = g.Resources.FuelCells
		}
		if g.Resources.ShipDesignCollection != nil {
			lookup.StockDesigns = g.Resources.ShipDesigns
		}
	}
	if player != nil {
		lookup.PlayerDesigns = player.ShipDesigns
		lookup.AvailableCellNames = player.Available.FuelCells
	}
	return lookup
}

// FleetShipCombatSpeed returns combat squares per turn for a fleet ship (drive + race bonus).
func (g *Game) FleetShipCombatSpeed(player *Player, ship FleetShip) (int64, bool) {
	bonus := RaceCombatSpeedBonus(player)
	if g != nil && g.Resources != nil && g.Resources.ShipPartsCollection != nil && ship.Drive != "" {
		if drive, err := DriveFromName(ship.Drive, g.Resources.Drives); err == nil {
			return drive.CombatSpeed + bonus, true
		}
	}
	if ship.DesignName != "" {
		if design, found := g.findShipDesignForFleetShip(player, ship); found {
			return design.Drive.CombatSpeed + bonus, true
		}
		return bonus, bonus > 0
	}
	if drive, found := bestAvailableDrive(player, g.Resources); found {
		return drive.CombatSpeed + bonus, true
	}
	if bonus > 0 {
		return bonus, true
	}
	return 0, false
}

func (g *Game) fleetShipInterstellarSpeed(player *Player, ship FleetShip) (int64, bool) {
	if g != nil && g.Resources != nil && g.Resources.ShipPartsCollection != nil && ship.Drive != "" {
		if drive, err := DriveFromName(ship.Drive, g.Resources.Drives); err == nil {
			return drive.InterstellarSpeed, true
		}
	}
	if ship.InterstellarSpeed > 0 {
		return ship.InterstellarSpeed, true
	}
	if ship.DesignName != "" {
		if design, found := g.findShipDesignForFleetShip(player, ship); found {
			return design.Drive.InterstellarSpeed, true
		}
		return 0, false
	}
	if drive, found := bestAvailableDrive(player, g.Resources); found {
		return drive.InterstellarSpeed, true
	}
	return 0, false
}

func (g *Game) selectedShipsInterstellarSpeed(player *Player, ships []FleetShip) (int64, error) {
	if len(ships) == 0 {
		return 0, fmt.Errorf("no ships selected for fleet move")
	}
	var slowest int64
	for _, ship := range ships {
		speed, ok := g.fleetShipInterstellarSpeed(player, ship)
		if !ok {
			return 0, fmt.Errorf("cannot determine drive speed for ship %q", ship.Name)
		}
		if speed <= 0 {
			return 0, fmt.Errorf("ship %q has no interstellar drive", ship.Name)
		}
		if slowest == 0 || speed < slowest {
			slowest = speed
		}
	}
	return slowest, nil
}

func (g *Game) calculateFleetTravelTurnsForShips(ownerID uint, ships []FleetShip, x1, y1, x2, y2 float64) (uint, error) {
	if g == nil || g.Resources == nil || g.Resources.ShipPartsCollection == nil {
		return calculateFleetTravelTurnsFromFloat(x1, y1, x2, y2), nil
	}
	player, err := g.getPlayer(ownerID)
	if err != nil {
		return 0, err
	}
	speed, err := g.selectedShipsInterstellarSpeed(player, ships)
	if err != nil {
		return 0, err
	}
	return calculateFleetTravelTurnsFromFloatWithSpeed(x1, y1, x2, y2, speed), nil
}

func (g *Game) selectedShipsFuelRange(player *Player, ships []FleetShip) (rangeParsecs int64, unlimited bool, err error) {
	if len(ships) == 0 {
		return 0, false, fmt.Errorf("no ships selected for fleet move")
	}
	allUnlimited := true
	limitedFound := false
	var minRange int64
	for _, ship := range ships {
		shipRange, shipUnlimited, ok := g.fleetShipFuelRange(player, ship)
		if !ok {
			return 0, false, fmt.Errorf("cannot determine fuel range for ship %q", ship.Name)
		}
		if shipUnlimited {
			continue
		}
		allUnlimited = false
		if !limitedFound || shipRange < minRange {
			minRange = shipRange
			limitedFound = true
		}
	}
	if allUnlimited {
		return 0, true, nil
	}
	return minRange, false, nil
}

func (g *Game) destinationInFuelRange(playerID uint, destStarID uint, rangeParsecs int64, unlimited bool) bool {
	if unlimited {
		return true
	}
	destination, err := g.getStar(destStarID)
	if err != nil {
		return false
	}
	return FuelDestinationReachable(
		float64(destination.X), float64(destination.Y),
		g.playerOwnedSupplyStarCoords(playerID),
		rangeParsecs, unlimited,
	)
}

// FuelDestinationReachable is the single source of truth for "is this star
// inside fuel range of any owned supply colony?". Used by submit validation
// and by the AI scout policy against a visible snapshot.
func FuelDestinationReachable(destX, destY float64, supplyCoords [][2]float64, rangeParsecs int64, unlimited bool) bool {
	if unlimited {
		return true
	}
	for _, c := range supplyCoords {
		if DistanceParsecs(c[0], c[1], destX, destY) <= float64(rangeParsecs) {
			return true
		}
	}
	return false
}

// FuelRangeLookup carries catalogs needed to resolve a ship's fuel range the
// same way fleetShipFuelRange / validateFleetMoveFuelRange do: stamps, then
// player/stock designs by design_name, then the player's best available cell.
type FuelRangeLookup struct {
	FuelCells          []FuelCell
	PlayerDesigns      []ShipDesign
	StockDesigns       []ShipDesign
	AvailableCellNames []string
}

// ResolveShipFuelRange is the single source of truth for one ship's fuel
// range. Order: UnlimitedFuelRange / FuelCell stamp / FuelRangeParsecs stamp,
// then design_name → player or stock design fuel cell, then best available
// fuel cell among AvailableCellNames. Matches Game.fleetShipFuelRange.
func ResolveShipFuelRange(ship FleetShip, lookup FuelRangeLookup) (rangeParsecs int64, unlimited bool, ok bool) {
	if rangeParsecs, unlimited, ok = listedShipFuelRange(ship, lookup.FuelCells); ok {
		return rangeParsecs, unlimited, true
	}
	if ship.DesignName != "" {
		if design, found := findDesignForFuelRange(ship.DesignName, lookup.PlayerDesigns, lookup.StockDesigns); found {
			return design.FuelCell.Autonomy, fuelCellUnlimited(design.FuelCell), true
		}
		return 0, false, false
	}
	if cell, found := bestFuelCellByNames(lookup.AvailableCellNames, lookup.FuelCells); found {
		return cell.Autonomy, fuelCellUnlimited(cell), true
	}
	return 0, false, false
}

// ShipsFuelRange returns the fleet's limiting fuel range (shortest limited
// ship) using ResolveShipFuelRange for each ship. Used by submit validation
// (via fleetShipFuelRange) and the AI scout policy.
func ShipsFuelRange(ships []FleetShip, lookup FuelRangeLookup) (rangeParsecs int64, unlimited bool, ok bool) {
	if len(ships) == 0 {
		return 0, false, false
	}
	allUnlimited := true
	limitedFound := false
	var minRange int64
	for _, ship := range ships {
		shipRange, shipUnlimited, shipOK := ResolveShipFuelRange(ship, lookup)
		if !shipOK {
			return 0, false, false
		}
		if shipUnlimited {
			continue
		}
		allUnlimited = false
		if !limitedFound || shipRange < minRange {
			minRange = shipRange
			limitedFound = true
		}
	}
	if allUnlimited {
		return 0, true, true
	}
	return minRange, false, true
}

// ShipsListedFuelRange is the stamp/catalog-only path (no design / available
// fallback). Prefer ShipsFuelRange when player designs and available cells are
// known — that matches submit validation fully.
func ShipsListedFuelRange(ships []FleetShip, cells []FuelCell) (rangeParsecs int64, unlimited bool, ok bool) {
	return ShipsFuelRange(ships, FuelRangeLookup{FuelCells: cells})
}

// listedShipFuelRange is the stamp branch shared by ResolveShipFuelRange.
func listedShipFuelRange(ship FleetShip, cells []FuelCell) (rangeParsecs int64, unlimited bool, ok bool) {
	if ship.UnlimitedFuelRange {
		return 0, true, true
	}
	if ship.FuelCell != "" && len(cells) > 0 {
		if cell, err := FuelCellFromName(ship.FuelCell, cells); err == nil {
			return cell.Autonomy, fuelCellUnlimited(cell), true
		}
	}
	if ship.FuelRangeParsecs > 0 {
		return ship.FuelRangeParsecs, ship.FuelRangeParsecs >= 100, true
	}
	return 0, false, false
}

func findDesignForFuelRange(name string, playerDesigns, stockDesigns []ShipDesign) (ShipDesign, bool) {
	for _, design := range playerDesigns {
		if design.Name == name {
			return design, true
		}
	}
	for _, design := range stockDesigns {
		if design.Name == name {
			return design, true
		}
	}
	return ShipDesign{}, false
}

func bestFuelCellByNames(names []string, cells []FuelCell) (FuelCell, bool) {
	if len(names) == 0 || len(cells) == 0 {
		return FuelCell{}, false
	}
	var best FuelCell
	found := false
	for _, fuelCellName := range names {
		cell, err := FuelCellFromName(fuelCellName, cells)
		if err != nil {
			continue
		}
		if !found || fuelCellUnlimited(cell) || (!fuelCellUnlimited(best) && cell.Autonomy > best.Autonomy) {
			best = cell
			found = true
		}
	}
	return best, found
}

func (g *Game) playerOwnedSupplyStarCoords(playerID uint) [][2]float64 {
	stars := g.playerOwnedSupplyStars(playerID)
	out := make([][2]float64, 0, len(stars))
	for _, star := range stars {
		if star == nil {
			continue
		}
		out = append(out, [2]float64{float64(star.X), float64(star.Y)})
	}
	return out
}

func (g *Game) validateFleetMoveFuelRange(player *Player, ships []FleetShip, destStarID uint) error {
	if g == nil || player == nil || g.Resources == nil || g.Resources.ShipPartsCollection == nil {
		return nil
	}
	rangeParsecs, unlimited, err := g.selectedShipsFuelRange(player, ships)
	if err != nil {
		return err
	}
	if g.destinationInFuelRange(player.ID, destStarID, rangeParsecs, unlimited) {
		return nil
	}
	if unlimited {
		return fmt.Errorf("destination is outside fleet fuel range")
	}
	return fmt.Errorf("destination is outside fleet fuel range (%d parsecs)", rangeParsecs)
}

func shipsByIndexes(ships []FleetShip, indexes []uint) []FleetShip {
	selected := make([]FleetShip, 0, len(indexes))
	for _, index := range indexes {
		selected = append(selected, ships[index])
	}
	return selected
}

func (g *Game) issueFleetMove(playerID uint, command *FleetMoveCommand) error {
	// Use shared validator
	fleet, _, shipIdxs, err := g.validateSingleFleetMoveOrder(playerID, command)
	if err != nil {
		return err
	}

	// Recall selected ships to the origin star (merge into the orbiting stack).
	inTransit := fleet.TurnsRemaining > 0 && fleet.DestinationStarID != 0
	if command.DestinationStarID == fleetDepartureStar(fleet) {
		if !inTransit {
			return nil
		}
		toLand := removeShipsByIndexes(fleet, shipIdxs)
		if err := g.mergeShipsIntoOrbitFleet(playerID, fleetDepartureStar(fleet), toLand); err != nil {
			return err
		}
		if len(fleet.Ships) == 0 {
			g.removeFleet(fleet)
		}
		return nil
	}

	// Handle in-transit fleet movements
	if inTransit {
		if len(shipIdxs) == len(fleet.Ships) {
			return g.beginFleetMovement(fleet, command.DestinationStarID)
		}
		toMove := removeShipsByIndexes(fleet, shipIdxs)
		if len(toMove) == 0 {
			return nil
		}
		child, err := g.newFleetAtStar(playerID, fleetDepartureStar(fleet), toMove)
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

	// Handle normal fleet movements
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
	// Use shared validator
	fleet, _, batches, err := g.validateFleetMoveGroupOrders(playerID, commands)
	if err != nil {
		return err
	}

	// Check if in-transit (validator returns nil batches for in-transit case)
	inTransit := fleet.TurnsRemaining > 0 && fleet.DestinationStarID != 0
	if inTransit {
		// Handle each command individually
		for _, command := range commands {
			if err := g.issueFleetMove(playerID, command); err != nil {
				return err
			}
		}
		return nil
	}

	// No batches means all moves were recalls or there's nothing to do
	if len(batches) == 0 {
		return nil
	}

	// Perform the grouped movements (mutation logic)
	originStarID := fleet.StarID
	originalShips := append([]FleetShip(nil), fleet.Ships...)

	// Build selected map from batches
	selected := make(map[uint]struct{})
	for _, batch := range batches {
		for _, index := range batch.indexes {
			selected[index] = struct{}{}
		}
	}

	// Update remaining ships in the original fleet
	remaining := make([]FleetShip, 0, len(originalShips)-len(selected))
	for i, ship := range originalShips {
		if _, moving := selected[uint(i)]; !moving {
			remaining = append(remaining, ship)
		}
	}
	fleet.Ships = remaining

	// Create child fleets for each batch
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
			g.recordTurnEvent(TurnEvent{Owner: fleet.Owner, Kind: "arrival", Text: fmt.Sprintf("Fleet %d arrived at %s", fleet.ID, destination.Name), StarID: destStarID, FleetID: fleet.ID})
			fleet.StarID = destStarID
			// Write lock already held by SubmitTurn → AdvanceTurn.
			_ = g.exploreStarUnlocked(fleet.Owner, fleet.StarID)
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

func calculateFleetTravelTurnsFromFloatWithSpeed(x1, y1, x2, y2 float64, speedParsecsPerTurn int64) uint {
	if speedParsecsPerTurn <= 0 {
		return 0
	}
	dx := x2 - x1
	dy := y2 - y1
	distance := math.Sqrt(dx*dx + dy*dy)
	pixelsPerTurn := float64(speedParsecsPerTurn * parsecInPixels)
	return uint(max(1, int(math.Ceil(distance/pixelsPerTurn))))
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

func (g *Game) issueInvadeCommands(playerID uint, commands []*InvadeCommand) error {
	for _, command := range commands {
		if command == nil {
			continue
		}
		if err := g.issueInvadeCommand(playerID, command); err != nil {
			return err
		}
	}
	return nil
}

func (g *Game) issueInvadeCommand(playerID uint, command *InvadeCommand) error {
	// Use shared validator
	fleet, planet, err := g.validateInvadeOrder(playerID, command)
	if err != nil {
		return err
	}
	if fleet == nil || planet == nil {
		return nil // command was nil
	}

	for _, pending := range g.PendingInvasions {
		if pending.FleetID == fleet.ID {
			return fmt.Errorf("fleet %d already has an invade order this turn", fleet.ID)
		}
	}

	g.PendingInvasions = append(g.PendingInvasions, PendingInvasion{
		OwnerID:        playerID,
		FleetID:        fleet.ID,
		TargetPlanetID: planet.ID,
		ShipIndexes:    copyUintIndexList(command.ShipIndexes),
	})
	return nil
}

func copyUintIndexList(indexes *[]uint) *[]uint {
	if indexes == nil {
		return nil
	}
	cp := cloneSlice(*indexes)
	return &cp
}
