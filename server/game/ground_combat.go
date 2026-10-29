package game

import (
	. "ssg/model"
)

// maxGroundBattleRounds bounds ground combat resolution to prevent infinite loops
const maxGroundBattleRounds = 20

// baseHitChance is the baseline chance for ground units to hit each other
const baseHitChance = 0.7

// resolveGroundInvasions resolves only Invade orders recorded this turn.
// Transports parked at a star without an order do nothing. Stale orders
// (fleet gone / moved / target no longer valid) are dropped safely.
func (g *Game) resolveGroundInvasions() []*InvasionReport {
	if g == nil {
		return nil
	}

	orders := g.PendingInvasions
	g.PendingInvasions = nil

	var reports []*InvasionReport
	for i := range orders {
		order := orders[i]
		report := g.resolvePendingInvasion(order)
		if report != nil {
			reports = append(reports, report)
		}
	}

	if len(reports) > 0 {
		g.InvasionReports = append(g.InvasionReports, reports...)
	}

	return reports
}

func (g *Game) resolvePendingInvasion(order PendingInvasion) *InvasionReport {
	command := &InvadeCommand{
		FleetID:        order.FleetID,
		TargetPlanetID: order.TargetPlanetID,
		ShipIndexes:    order.ShipIndexes,
	}
	fleet, planet, err := g.validateInvadeOrder(order.OwnerID, command)
	if err != nil || fleet == nil || planet == nil {
		return nil
	}

	attackerForce := g.gatherMarinesFromInvasionOrder(fleet, order.ShipIndexes)
	if attackerForce.TotalUnits() == 0 {
		return nil
	}

	defenderForce := g.createPlanetDefenseForce(planet)
	report := g.resolveInvasion(planet, order.OwnerID, attackerForce, defenderForce)
	if report == nil {
		return nil
	}

	if report.Success && report.PlanetCaptured {
		g.capturePlanet(planet, order.OwnerID, &report.SurvivingUnits)
	} else {
		g.updatePlanetDefenders(planet, &report.SurvivingUnits)
	}
	g.removeInvasionTransports(fleet, order.ShipIndexes)
	return report
}

// fleetHasTransportMarines checks if a fleet has Transport ships with marines
func (g *Game) fleetHasTransportMarines(fleet *Fleet) bool {
	for _, ship := range fleet.Ships {
		if ship.Type == "transport" {
			return true
		}
	}
	return false
}

// getTransportMarineCount gets the marine count for a transport ship
func (g *Game) getTransportMarineCount(ship FleetShip) int64 {
	if ship.Type == "transport" {
		// Each transport ship carries 4 marines as specified
		return 4
	}
	return 0
}

// gatherMarinesFromInvasionOrder builds the attacking force from the named
// transport ships (or all transports in the fleet when indexes are omitted).
func (g *Game) gatherMarinesFromInvasionOrder(fleet *Fleet, shipIndexes *[]uint) GroundForce {
	force := GroundForce{OwnerID: fleet.Owner}
	if fleet == nil {
		return force
	}
	if shipIndexes != nil {
		for _, index := range *shipIndexes {
			if int(index) >= len(fleet.Ships) {
				continue
			}
			g.appendTransportMarines(&force, fleet.Ships[index])
		}
		return force
	}
	for _, ship := range fleet.Ships {
		g.appendTransportMarines(&force, ship)
	}
	return force
}

func (g *Game) appendTransportMarines(force *GroundForce, ship FleetShip) {
	marineCount := g.getTransportMarineCount(ship)
	for i := int64(0); i < marineCount; i++ {
		force.Marines = append(force.Marines, NewGroundUnit(MarineUnit))
	}
}

// gatherMarinesFromTransports creates a ground force from all marines in player's transport ships at a planet
func (g *Game) gatherMarinesFromTransports(ownerID uint, fleets []*Fleet) GroundForce {
	force := GroundForce{OwnerID: ownerID}

	for _, fleet := range fleets {
		if fleet.Owner != ownerID {
			continue
		}
		for _, ship := range fleet.Ships {
			g.appendTransportMarines(&force, ship)
		}
	}

	return force
}

// createPlanetDefenseForce creates ground force from planet's marines and armored units
func (g *Game) createPlanetDefenseForce(planet *Planet) GroundForce {
	force := GroundForce{OwnerID: planet.Owner}

	// Add marines
	for i := int64(0); i < planet.Marines; i++ {
		force.Marines = append(force.Marines, NewGroundUnit(MarineUnit))
	}

	// Add armored units
	for i := int64(0); i < planet.Armored; i++ {
		force.Armored = append(force.Armored, NewGroundUnit(ArmoredUnit))
	}

	return force
}

// resolveInvasion runs the ground combat between attacker and defender forces
func (g *Game) resolveInvasion(planet *Planet, attackerOwnerID uint, attackerForce, defenderForce GroundForce) *InvasionReport {
	attackerPlayer, _ := g.getPlayer(attackerOwnerID)
	defenderPlayer, _ := g.getPlayer(planet.Owner)

	star, _ := g.getStar(planet.ParentStar)

	report := &InvasionReport{
		ID:              g.nextInvasionReportID(),
		Turn:            int64(g.CurrentTurn),
		PlanetID:        planet.ID,
		PlanetName:      planet.Name,
		AttackerOwnerID: attackerOwnerID,
		DefenderOwnerID: planet.Owner,
		InitialAttacker: copyGroundForce(attackerForce),
		InitialDefender: copyGroundForce(defenderForce),
	}

	if star != nil {
		report.X = float64(star.X)
		report.Y = float64(star.Y)
	}

	// Run ground combat
	rounds := runGroundBattle(&attackerForce, &defenderForce, attackerPlayer, defenderPlayer)
	report.Rounds = rounds

	// Determine outcome
	attackerSurvived := attackerForce.HasLivingUnits()
	defenderSurvived := defenderForce.HasLivingUnits()

	report.Success = attackerSurvived && !defenderSurvived
	report.PlanetCaptured = report.Success

	// Set surviving units
	if attackerSurvived {
		report.SurvivingUnits = attackerForce
	} else if defenderSurvived {
		report.SurvivingUnits = defenderForce
	}

	return report
}

// runGroundBattle executes rounds of ground combat until one side is eliminated or max rounds reached
func runGroundBattle(attackerForce, defenderForce *GroundForce, attackerPlayer, defenderPlayer *Player) []GroundBattleRound {
	var rounds []GroundBattleRound

	for round := 1; round <= maxGroundBattleRounds; round++ {
		if !attackerForce.HasLivingUnits() || !defenderForce.HasLivingUnits() {
			break
		}

		battleRound := GroundBattleRound{Round: round}

		// Attacker fires first
		attackerDamage := calculateGroundDamage(attackerForce, defenderForce, attackerPlayer, defenderPlayer)
		battleRound.AttackerDamage = attackerDamage
		defenderLosses := applyGroundDamage(defenderForce, attackerDamage)
		battleRound.DefenderLosses = defenderLosses

		// Defender fires back if they have units left
		if defenderForce.HasLivingUnits() {
			defenderDamage := calculateGroundDamage(defenderForce, attackerForce, defenderPlayer, attackerPlayer)
			battleRound.DefenderDamage = defenderDamage
			attackerLosses := applyGroundDamage(attackerForce, defenderDamage)
			battleRound.AttackerLosses = attackerLosses
		}

		battleRound.AttackerSurvive = attackerForce.TotalUnits()
		battleRound.DefenderSurvive = defenderForce.TotalUnits()

		rounds = append(rounds, battleRound)
	}

	return rounds
}

// calculateGroundDamage computes damage dealt by one force against another
func calculateGroundDamage(attackerForce, defenderForce *GroundForce, attackerPlayer, defenderPlayer *Player) int64 {
	if !attackerForce.HasLivingUnits() || !defenderForce.HasLivingUnits() {
		return 0
	}

	totalDamage := int64(0)
	attackerStrength := attackerForce.TotalStrength()

	// Apply racial ground combat multiplier
	multiplier := GroundCombatMultiplier(attackerPlayer)
	effectiveStrength := float64(attackerStrength) * multiplier

	// Each point of strength has a chance to hit
	for i := int64(0); i < int64(effectiveStrength); i++ {
		if randomFloat64() <= baseHitChance {
			totalDamage += 1 // Each hit deals 1 damage
		}
	}

	return totalDamage
}

// applyGroundDamage applies damage to a ground force, returning number of units killed
func applyGroundDamage(force *GroundForce, damage int64) int {
	if damage <= 0 || !force.HasLivingUnits() {
		return 0
	}

	killed := 0
	remainingDamage := damage

	// Damage marines first
	for i := range force.Marines {
		if remainingDamage <= 0 {
			break
		}
		if force.Marines[i].IsAlive() {
			damageToApply := min(remainingDamage, force.Marines[i].Health)
			force.Marines[i].Health -= damageToApply
			remainingDamage -= damageToApply
			if !force.Marines[i].IsAlive() {
				killed++
			}
		}
	}

	// Then damage armored units
	for i := range force.Armored {
		if remainingDamage <= 0 {
			break
		}
		if force.Armored[i].IsAlive() {
			damageToApply := min(remainingDamage, force.Armored[i].Health)
			force.Armored[i].Health -= damageToApply
			remainingDamage -= damageToApply
			if !force.Armored[i].IsAlive() {
				killed++
			}
		}
	}

	return killed
}

// Helper functions

func copyGroundForce(force GroundForce) GroundForce {
	copy := GroundForce{OwnerID: force.OwnerID}
	copy.Marines = make([]GroundUnit, len(force.Marines))
	copy.Armored = make([]GroundUnit, len(force.Armored))

	for i, marine := range force.Marines {
		copy.Marines[i] = marine
	}
	for i, armored := range force.Armored {
		copy.Armored[i] = armored
	}

	return copy
}

func (g *Game) nextInvasionReportID() uint {
	max := uint(0)
	for _, report := range g.InvasionReports {
		if report != nil && report.ID > max {
			max = report.ID
		}
	}
	return max + 1
}

func (g *Game) capturePlanet(planet *Planet, newOwnerID uint, survivingUnits *GroundForce) {
	planet.Owner = newOwnerID
	// Reset planet marines/armored to surviving attacker units
	planet.Marines = int64(len(survivingUnits.Marines))
	planet.Armored = int64(len(survivingUnits.Armored))
	// TODO: Handle population, buildings, etc. based on game rules
}

func (g *Game) updatePlanetDefenders(planet *Planet, survivingUnits *GroundForce) {
	// Update planet with surviving defender units
	livingMarines := int64(0)
	for _, marine := range survivingUnits.Marines {
		if marine.IsAlive() {
			livingMarines++
		}
	}

	livingArmored := int64(0)
	for _, armored := range survivingUnits.Armored {
		if armored.IsAlive() {
			livingArmored++
		}
	}

	planet.Marines = livingMarines
	planet.Armored = livingArmored
}

func (g *Game) removeInvasionTransports(fleet *Fleet, shipIndexes *[]uint) {
	if fleet == nil {
		return
	}
	if shipIndexes != nil {
		remove := make(map[uint]bool, len(*shipIndexes))
		for _, index := range *shipIndexes {
			remove[index] = true
		}
		remaining := make([]FleetShip, 0, len(fleet.Ships))
		for i, ship := range fleet.Ships {
			if remove[uint(i)] {
				continue
			}
			remaining = append(remaining, ship)
		}
		fleet.Ships = remaining
		return
	}
	remaining := make([]FleetShip, 0, len(fleet.Ships))
	for _, ship := range fleet.Ships {
		if ship.Type == "transport" {
			continue
		}
		remaining = append(remaining, ship)
	}
	fleet.Ships = remaining
}

// removeMarinesFromTransports removes transport ships that deployed their marines.
// Kept for callers that still pass a fleet list; prefer removeInvasionTransports.
func (g *Game) removeMarinesFromTransports(ownerID uint, fleets []*Fleet, originalForce, survivingUnits *GroundForce) {
	for _, fleet := range fleets {
		if fleet.Owner == ownerID {
			g.removeInvasionTransports(fleet, nil)
		}
	}
}
