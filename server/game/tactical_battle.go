package game

import (
	"fmt"
	"sort"
	. "ssg/model"
)

// maxTacticalRounds bounds interactive battles the same way maxBattleRounds
// bounds auto-resolve: hitting the cap is a draw and ships disperse intact.
const maxTacticalRounds = 50

// aiPreferredRange is the distance AI ships try to close to before firing;
// inside it beam dissipation no longer reduces damage.
const aiPreferredRange = 2

// starHasHumanConflict reports whether any of the fleets at a contested star
// belongs to a player marked human, which upgrades the fight from instant
// auto-resolve to an interactive tactical battle.
func (g *Game) starHasHumanConflict(fleets []*Fleet) bool {
	for _, fleet := range fleets {
		player, err := g.getPlayer(fleet.Owner)
		if err == nil && player.IsHuman {
			return true
		}
	}
	return false
}

// createTacticalBattle materializes the fleets at starID onto the battle grid
// and stores the battle as pending. Ships act in initiative order; AI sides
// are played immediately until a human ship is up.
func (g *Game) createTacticalBattle(starID uint, fleets []*Fleet) *TacticalBattle {
	battle := &TacticalBattle{
		ID:         g.nextTacticalBattleID(),
		GameTurn:   g.CurrentTurn,
		StarID:     starID,
		Status:     TacticalBattleActive,
		GridWidth:  TacticalGridWidth,
		GridHeight: TacticalGridHeight,
		Round:      1,
	}

	ownerOrder := []uint{}
	shipsByOwner := map[uint][]*TacticalShip{}
	nextShipID := uint(1)
	for _, fleet := range fleets {
		player, _ := g.getPlayer(fleet.Owner)
		if _, seen := shipsByOwner[fleet.Owner]; !seen {
			ownerOrder = append(ownerOrder, fleet.Owner)
			if player != nil && player.IsHuman {
				battle.HumanOwnerIDs = append(battle.HumanOwnerIDs, fleet.Owner)
			}
		}
		for i, fs := range fleet.Ships {
			ship := materializeBattleShip(g, player, fs)
			ts := newTacticalShip(nextShipID, fleet, i, fs, ship)
			nextShipID++
			shipsByOwner[fleet.Owner] = append(shipsByOwner[fleet.Owner], ts)
		}
	}
	sort.SliceStable(ownerOrder, func(i, j int) bool { return ownerOrder[i] < ownerOrder[j] })
	sort.Slice(battle.HumanOwnerIDs, func(i, j int) bool { return battle.HumanOwnerIDs[i] < battle.HumanOwnerIDs[j] })

	for sideIndex, ownerID := range ownerOrder {
		placeSide(battle, shipsByOwner[ownerID], sideIndex)
		battle.Ships = append(battle.Ships, shipsByOwner[ownerID]...)
	}

	battle.TurnOrder = initiativeOrder(battle.Ships)
	for _, ts := range battle.Ships {
		ts.MovesLeft = ts.CombatSpeed
	}
	battle.AppendEvent(BattleEvent{Type: "round_start", Round: 1})
	battle.AnnounceActivation()

	g.PendingBattles = append(g.PendingBattles, battle)
	g.runAIUntilHumanOrEnd(battle)
	return battle
}

func newTacticalShip(id uint, fleet *Fleet, shipIndex int, fs FleetShip, ship *Ship) *TacticalShip {
	speed := int(ship.Drive.CombatSpeed)
	if speed <= 0 {
		speed = int(ship.HullSize.BaseSpeed)
	}
	if speed <= 0 {
		speed = 1
	}
	return &TacticalShip{
		ID:              id,
		OwnerID:         fleet.Owner,
		FleetID:         fleet.ID,
		ShipIndex:       shipIndex,
		Name:            fs.Name,
		DesignName:      fs.DesignName,
		HullSize:        ship.HullSize.Name,
		SizeRating:      ship.HullSize.SizeRating,
		CombatSpeed:     speed,
		Initiative:      int64(speed),
		Beams:           ship.Beams,
		BeamAttackBonus: ship.Computer.BeamAttackBonus,
		BeamDefense:     ship.BeamDefense,
		ShipStatus:      ship.ShipStatus,
	}
}

// placeSide lines a side's ships up along one edge of the grid: side 0 left,
// side 1 right, side 2 top, side 3 bottom. Overflow wraps to the next column
// (or row) inward.
func placeSide(b *TacticalBattle, ships []*TacticalShip, sideIndex int) {
	for i, ts := range ships {
		switch sideIndex {
		case 0:
			ts.X = i / b.GridHeight
			ts.Y = i % b.GridHeight
		case 1:
			ts.X = b.GridWidth - 1 - i/b.GridHeight
			ts.Y = i % b.GridHeight
		case 2:
			ts.X = i % b.GridWidth
			ts.Y = i / b.GridWidth
		default:
			ts.X = i % b.GridWidth
			ts.Y = b.GridHeight - 1 - i/b.GridWidth
		}
	}
}

// initiativeOrder sorts ship IDs by initiative (desc), then size (desc), then
// ID for stability.
func initiativeOrder(ships []*TacticalShip) []uint {
	sorted := make([]*TacticalShip, len(ships))
	copy(sorted, ships)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Initiative != sorted[j].Initiative {
			return sorted[i].Initiative > sorted[j].Initiative
		}
		if sorted[i].SizeRating != sorted[j].SizeRating {
			return sorted[i].SizeRating > sorted[j].SizeRating
		}
		return sorted[i].ID < sorted[j].ID
	})
	order := make([]uint, 0, len(sorted))
	for _, ts := range sorted {
		order = append(order, ts.ID)
	}
	return order
}

func chebyshev(ax, ay, bx, by int) int64 {
	dx := ax - bx
	if dx < 0 {
		dx = -dx
	}
	dy := ay - by
	if dy < 0 {
		dy = -dy
	}
	if dx > dy {
		return int64(dx)
	}
	return int64(dy)
}

// advanceActivation moves to the next living ship in turn order, starting a
// new round (and re-checking the round cap) when the order wraps.
func (g *Game) advanceActivation(b *TacticalBattle) {
	b.TurnIndex++
	g.seekLiveShip(b)
}

// seekLiveShip leaves TurnIndex on the next living ship at or after its
// current position, wrapping into new rounds as needed. The round cap turns
// the fight into a draw, so this always terminates.
func (g *Game) seekLiveShip(b *TacticalBattle) {
	for b.Status == TacticalBattleActive {
		if b.TurnIndex >= len(b.TurnOrder) {
			b.TurnIndex = 0
			b.Round++
			if b.Round > maxTacticalRounds {
				g.finishTacticalBattle(b, 0, true)
				return
			}
			for _, ts := range b.Ships {
				ts.MovesLeft = ts.CombatSpeed
				ts.HasFired = false
			}
			b.AppendEvent(BattleEvent{Type: "round_start", Round: b.Round})
		}
		ts := b.ShipByID(b.TurnOrder[b.TurnIndex])
		if ts != nil && !ts.Destroyed {
			b.AnnounceActivation()
			return
		}
		b.TurnIndex++
	}
}

// GetTacticalBattle finds a pending battle by ID.
func (g *Game) GetTacticalBattle(battleID uint) (*TacticalBattle, error) {
	if g.mu != nil {
		g.mu.Lock()
		defer g.mu.Unlock()
	}
	for _, b := range g.PendingBattles {
		if b != nil && b.ID == battleID {
			return b, nil
		}
	}
	return nil, fmt.Errorf("battle %d not found", battleID)
}

// ApplyBattleAction validates and applies one order from playerID for the
// currently acting ship, then plays AI sides until a human ship is up again
// or the battle ends. It returns the battle so callers can ship state +
// events back to the client.
func (g *Game) ApplyBattleAction(battleID, playerID uint, action BattleAction) (*TacticalBattle, error) {
	if g.mu != nil {
		g.mu.Lock()
		defer g.mu.Unlock()
	}
	var battle *TacticalBattle
	for _, b := range g.PendingBattles {
		if b != nil && b.ID == battleID {
			battle = b
			break
		}
	}
	if battle == nil {
		return nil, fmt.Errorf("battle %d not found", battleID)
	}
	if battle.Status != TacticalBattleActive {
		return battle, fmt.Errorf("battle %d is finished", battleID)
	}
	ship := battle.CurrentShip()
	if ship == nil {
		return battle, fmt.Errorf("no ship is currently acting")
	}
	if ship.OwnerID != playerID {
		return battle, fmt.Errorf("it is not player %d's turn to act", playerID)
	}
	if action.ShipID != 0 && action.ShipID != ship.ID {
		return battle, fmt.Errorf("ship %d is not the acting ship (%d is)", action.ShipID, ship.ID)
	}

	switch action.Type {
	case "move":
		if err := g.moveTacticalShip(battle, ship, action.X, action.Y); err != nil {
			return battle, err
		}
	case "fire":
		if err := g.fireTacticalShip(battle, ship, action.TargetID); err != nil {
			return battle, err
		}
	case "done":
		g.advanceActivation(battle)
	case "wait":
		g.battleWait(battle)
	case "auto":
		g.aiActivate(battle, ship)
	default:
		return battle, fmt.Errorf("unknown battle action %q", action.Type)
	}

	g.runAIUntilHumanOrEnd(battle)
	return battle, nil
}

func (g *Game) moveTacticalShip(b *TacticalBattle, ship *TacticalShip, x, y int) error {
	if x < 0 || x >= b.GridWidth || y < 0 || y >= b.GridHeight {
		return fmt.Errorf("destination (%d,%d) is off the grid", x, y)
	}
	if other := b.ShipAt(x, y); other != nil && other != ship {
		return fmt.Errorf("square (%d,%d) is occupied", x, y)
	}
	cost := int(chebyshev(ship.X, ship.Y, x, y))
	if cost == 0 {
		return nil
	}
	if cost > ship.MovesLeft {
		return fmt.Errorf("not enough movement: need %d, have %d", cost, ship.MovesLeft)
	}
	fromX, fromY := ship.X, ship.Y
	ship.MovesLeft -= cost
	ship.X = x
	ship.Y = y
	b.AppendEvent(BattleEvent{
		Type: "move", ShipID: ship.ID,
		FromX: fromX, FromY: fromY, X: x, Y: y,
	})
	return nil
}

// fireTacticalShip fires every beam on the ship at the target, one volley per
// activation. Hit chance and dissipation reuse the auto-resolve combat math.
func (g *Game) fireTacticalShip(b *TacticalBattle, ship *TacticalShip, targetID uint) error {
	if ship.HasFired {
		return fmt.Errorf("ship %d already fired this round", ship.ID)
	}
	if len(ship.Beams) == 0 {
		return fmt.Errorf("ship %d has no weapons", ship.ID)
	}
	target := b.ShipByID(targetID)
	if target == nil || target.Destroyed {
		return fmt.Errorf("target %d is not a living ship", targetID)
	}
	if target.OwnerID == ship.OwnerID {
		return fmt.Errorf("cannot fire on a friendly ship")
	}

	attacker, _ := g.getPlayer(ship.OwnerID)
	defender, _ := g.getPlayer(target.OwnerID)
	distance := chebyshev(ship.X, ship.Y, target.X, target.Y)
	attackBonus := ship.BeamAttackBonus + shipRaceAttackBonus(attacker, ship.BeamAttackBonus)
	defense := target.BeamDefense + shipRaceDefenseBonus(defender, target.BeamDefense)
	chance := GetChanceToHit(Computer{BeamAttackBonus: attackBonus}, defense, 0, distance)

	totalDamage := int64(0)
	hits, misses := 0, 0
	for _, beam := range ship.Beams {
		if randomFloat64() <= chance {
			hits++
			totalDamage += GetDamage(beam, distance)
		} else {
			misses++
		}
	}
	ship.HasFired = true
	applyTacticalDamage(target, totalDamage)
	b.AppendEvent(BattleEvent{
		Type: "fire", ShipID: ship.ID, TargetID: target.ID,
		Damage: totalDamage, Hits: hits, Misses: misses,
	})
	if target.CurrentStructureHP <= 0 {
		target.Destroyed = true
		b.AppendEvent(BattleEvent{Type: "destroyed", ShipID: target.ID})
		g.checkTacticalBattleEnd(b)
	}
	return nil
}

// applyTacticalDamage mirrors FireOn's armor-then-structure damage model.
func applyTacticalDamage(target *TacticalShip, damage int64) {
	if damage <= 0 {
		return
	}
	if target.CurrentArmorHP >= damage {
		target.CurrentArmorHP -= damage
		return
	}
	spill := damage - target.CurrentArmorHP
	target.CurrentArmorHP = 0
	if spill < target.CurrentStructureHP {
		target.CurrentStructureHP -= spill
	} else {
		target.CurrentStructureHP = 0
	}
}

// battleWait defers the acting ship to the end of the current round. After
// the rotation TurnIndex already points at the next entry, so seek from there.
func (g *Game) battleWait(b *TacticalBattle) {
	id := b.TurnOrder[b.TurnIndex]
	b.TurnOrder = append(b.TurnOrder[:b.TurnIndex], b.TurnOrder[b.TurnIndex+1:]...)
	b.TurnOrder = append(b.TurnOrder, id)
	g.seekLiveShip(b)
}

func (g *Game) checkTacticalBattleEnd(b *TacticalBattle) bool {
	if b.Status != TacticalBattleActive {
		return true
	}
	owners := b.OwnersWithLiveShips()
	if len(owners) > 1 {
		return false
	}
	winner := uint(0)
	if len(owners) == 1 {
		winner = owners[0]
	}
	g.finishTacticalBattle(b, winner, winner == 0)
	return true
}

// finishTacticalBattle reconciles survivors back into their fleets, records a
// BattleReport (same shape the auto-resolve path produces), and removes the
// battle from the pending list.
func (g *Game) finishTacticalBattle(b *TacticalBattle, winnerOwnerID uint, draw bool) {
	if b.Status == TacticalBattleFinished {
		return
	}
	b.Status = TacticalBattleFinished
	b.WinnerOwnerID = winnerOwnerID
	b.Draw = draw
	b.AppendEvent(BattleEvent{Type: "finished", WinnerOwnerID: winnerOwnerID, Draw: draw, Round: b.Round})

	report := &BattleReport{
		ID:            g.nextBattleReportID(),
		Turn:          g.CurrentTurn,
		StarID:        b.StarID,
		WinnerOwnerID: winnerOwnerID,
		Rounds:        b.Round,
		Draw:          draw,
	}
	if star, err := g.getStar(b.StarID); err == nil {
		report.X = float64(star.X)
		report.Y = float64(star.Y)
	}

	// Group battle ships by owner for the report, and by fleet for pruning.
	ownerOrder := []uint{}
	byOwner := map[uint][]*TacticalShip{}
	survivorsByFleet := map[uint]map[int]bool{}
	fleetIDs := []uint{}
	seenFleet := map[uint]bool{}
	for _, ts := range b.Ships {
		if _, ok := byOwner[ts.OwnerID]; !ok {
			ownerOrder = append(ownerOrder, ts.OwnerID)
		}
		byOwner[ts.OwnerID] = append(byOwner[ts.OwnerID], ts)
		if !seenFleet[ts.FleetID] {
			seenFleet[ts.FleetID] = true
			fleetIDs = append(fleetIDs, ts.FleetID)
		}
		if !ts.Destroyed {
			if survivorsByFleet[ts.FleetID] == nil {
				survivorsByFleet[ts.FleetID] = map[int]bool{}
			}
			survivorsByFleet[ts.FleetID][ts.ShipIndex] = true
		}
	}
	sort.Slice(ownerOrder, func(i, j int) bool { return ownerOrder[i] < ownerOrder[j] })

	for _, ownerID := range ownerOrder {
		side := BattleSide{OwnerID: ownerID}
		for _, ts := range byOwner[ownerID] {
			original := g.fleetShipFor(ts)
			side.OriginalShips = append(side.OriginalShips, original)
			if ts.Destroyed {
				side.DestroyedCount++
			} else {
				side.SurvivingShips = append(side.SurvivingShips, original)
			}
		}
		report.Sides = append(report.Sides, side)
	}
	g.BattleReports = append(g.BattleReports, report)

	for _, fleetID := range fleetIDs {
		fleet, err := g.getFleet(fleetID)
		if err != nil {
			continue
		}
		surviving := survivorsByFleet[fleetID]
		if len(surviving) == 0 {
			g.removeFleet(fleet)
			continue
		}
		kept := make([]FleetShip, 0, len(surviving))
		for i := range fleet.Ships {
			if surviving[i] {
				kept = append(kept, fleet.Ships[i])
			}
		}
		fleet.Ships = kept
	}

	g.removePendingBattle(b.ID)
}

// fleetShipFor recovers the original FleetShip a tactical ship came from. The
// fleet is untouched while the battle is pending, so index lookup is safe;
// fall back to a name-only shell if anything shifted.
func (g *Game) fleetShipFor(ts *TacticalShip) FleetShip {
	fleet, err := g.getFleet(ts.FleetID)
	if err == nil && ts.ShipIndex >= 0 && ts.ShipIndex < len(fleet.Ships) {
		return fleet.Ships[ts.ShipIndex]
	}
	return FleetShip{Name: ts.Name, DesignName: ts.DesignName}
}

func (g *Game) removePendingBattle(battleID uint) {
	kept := make([]*TacticalBattle, 0, len(g.PendingBattles))
	for _, b := range g.PendingBattles {
		if b != nil && b.ID != battleID {
			kept = append(kept, b)
		}
	}
	g.PendingBattles = kept
}

func (g *Game) nextTacticalBattleID() uint {
	max := uint(0)
	for _, b := range g.PendingBattles {
		if b != nil && b.ID > max {
			max = b.ID
		}
	}
	// Battle report IDs share the counter so battle IDs stay unique across a
	// game even after battles resolve into reports.
	for _, r := range g.BattleReports {
		if r != nil && r.ID > max {
			max = r.ID
		}
	}
	return max + 1
}

// runAIUntilHumanOrEnd plays AI-owned ship activations until a human-owned
// ship is up or the battle is over.
func (g *Game) runAIUntilHumanOrEnd(b *TacticalBattle) {
	for b.Status == TacticalBattleActive {
		ship := b.CurrentShip()
		if ship == nil || b.IsHumanOwner(ship.OwnerID) {
			return
		}
		g.aiActivate(b, ship)
	}
}

// aiActivate plays one ship's activation: close toward the nearest enemy to
// point-blank dissipation range, fire everything, end activation.
func (g *Game) aiActivate(b *TacticalBattle, ship *TacticalShip) {
	target := nearestEnemy(b, ship)
	if target != nil {
		aiMoveToward(b, ship, target, g)
		if len(ship.Beams) > 0 && !ship.HasFired {
			_ = g.fireTacticalShip(b, ship, target.ID)
		}
	}
	if b.Status == TacticalBattleActive {
		g.advanceActivation(b)
	}
}

func nearestEnemy(b *TacticalBattle, ship *TacticalShip) *TacticalShip {
	var best *TacticalShip
	bestDist := int64(1 << 30)
	for _, ts := range b.Ships {
		if ts.Destroyed || ts.OwnerID == ship.OwnerID {
			continue
		}
		d := chebyshev(ship.X, ship.Y, ts.X, ts.Y)
		if d < bestDist || (d == bestDist && best != nil && ts.ID < best.ID) {
			best = ts
			bestDist = d
		}
	}
	return best
}

// aiMoveToward steps the ship one square at a time toward the target until it
// reaches aiPreferredRange or runs out of moves; occupied squares stop it.
func aiMoveToward(b *TacticalBattle, ship *TacticalShip, target *TacticalShip, g *Game) {
	fromX, fromY := ship.X, ship.Y
	for ship.MovesLeft > 0 && chebyshev(ship.X, ship.Y, target.X, target.Y) > aiPreferredRange {
		stepX := ship.X + sign(target.X-ship.X)
		stepY := ship.Y + sign(target.Y-ship.Y)
		if b.ShipAt(stepX, stepY) != nil {
			break
		}
		ship.X = stepX
		ship.Y = stepY
		ship.MovesLeft--
	}
	if ship.X != fromX || ship.Y != fromY {
		b.AppendEvent(BattleEvent{
			Type: "move", ShipID: ship.ID,
			FromX: fromX, FromY: fromY, X: ship.X, Y: ship.Y,
		})
	}
}

func sign(v int) int {
	if v > 0 {
		return 1
	}
	if v < 0 {
		return -1
	}
	return 0
}

// autoResolveLeftoverBattles force-finishes any still-pending battles by
// letting the AI play every side. Called at the start of turn processing so a
// stale pending battle can never wedge the game.
func (g *Game) autoResolveLeftoverBattles() {
	for len(g.PendingBattles) > 0 {
		b := g.PendingBattles[0]
		humans := b.HumanOwnerIDs
		b.HumanOwnerIDs = nil
		g.runAIUntilHumanOrEnd(b)
		if b.Status == TacticalBattleActive {
			// Should be unreachable; finish as a draw rather than loop.
			b.HumanOwnerIDs = humans
			g.finishTacticalBattle(b, 0, true)
		}
	}
}
