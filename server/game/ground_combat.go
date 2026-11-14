package game

import (
	. "ssg/model"
)

// maxGroundBattleRounds bounds ground combat resolution to prevent infinite loops
const maxGroundBattleRounds = 20

// baseHitChance is the baseline chance for ground units to hit each other
const baseHitChance = 0.7

// holdsOrbit reports whether ownerID controls orbit at starID for invasion
// purposes (MoO2 rule): the owner has at least one ship parked at the star
// (not in transit), and no other owner still has armed ships there.
//
// "Armed" means the ship materializes with at least one beam weapon from its
// design (colony ships, freighters, and transports are unarmed and do not
// deny orbit). A draw that leaves rival armed ships in orbit means nobody
// holds it. The invader being destroyed or driven off also fails the check.
func (g *Game) holdsOrbit(starID, ownerID uint) bool {
	if g == nil || starID == 0 || ownerID == 0 {
		return false
	}
	ownerPresent := false
	for _, fleet := range g.Fleets {
		if fleet == nil || fleet.StarID != starID {
			continue
		}
		if fleet.DestinationStarID != 0 || fleet.TurnsRemaining != 0 {
			continue
		}
		if len(fleet.Ships) == 0 {
			continue
		}
		if fleet.Owner == ownerID {
			ownerPresent = true
			continue
		}
		if g.fleetHasArmedShips(fleet) {
			return false
		}
	}
	return ownerPresent
}

// fleetHasArmedShips reports whether any ship in the fleet carries beam
// weapons or missile racks when materialized for combat.
func (g *Game) fleetHasArmedShips(fleet *Fleet) bool {
	if fleet == nil {
		return false
	}
	player, _ := g.getPlayer(fleet.Owner)
	for _, fs := range fleet.Ships {
		ship := materializeBattleShip(g, player, fs)
		if ship != nil && (len(ship.Beams) > 0 || len(ship.Missiles) > 0) {
			return true
		}
	}
	return false
}

func (g *Game) hasPendingBattleAtStar(starID uint) bool {
	for _, b := range g.PendingBattles {
		if b != nil && b.StarID == starID && b.Status == TacticalBattleActive {
			return true
		}
	}
	return false
}

// resolveGroundInvasions resolves Invade orders recorded this turn.
// Orders at a star with an active pending tactical battle are moved to
// PostponedInvasions (not fought yet). Otherwise the invader must hold
// orbit (see holdsOrbit) or the order is cancelled with a report.
// Transports parked without an order do nothing. Stale orders are dropped.
func (g *Game) resolveGroundInvasions() []*InvasionReport {
	if g == nil {
		return nil
	}

	orders := g.PendingInvasions
	g.PendingInvasions = nil

	var reports []*InvasionReport
	for i := range orders {
		order := orders[i]
		report := g.tryResolveInvasionOrder(order, true)
		if report != nil {
			// Append immediately so nextInvasionReportID stays unique in-batch.
			g.InvasionReports = append(g.InvasionReports, report)
			g.recordInvasionEvent(report)
			reports = append(reports, report)
		}
	}

	return reports
}

// tryResolveInvasionOrder either postpones (when allowPostpone and a battle is
// pending), cancels (orbit lost), resolves ground combat, or silently drops
// stale orders. Returns a report when combat ran or the order was cancelled.
//
// Stale orders (unknown planet, fleet still exists but never at the target
// star) are dropped with no report. Orbit loss — invader fleet destroyed, or
// present but rival armed ships remain — yields a Cancelled InvasionReport.
func (g *Game) tryResolveInvasionOrder(order PendingInvasion, allowPostpone bool) *InvasionReport {
	planet, err := g.getPlanet(order.TargetPlanetID)
	if err != nil || planet == nil {
		return nil
	}
	starID := planet.ParentStar

	if allowPostpone && g.hasPendingBattleAtStar(starID) {
		// Only keep orders that still validate; garbage does not persist.
		if _, _, vErr := g.validateInvadeOrder(order.OwnerID, &InvadeCommand{
			FleetID: order.FleetID, TargetPlanetID: order.TargetPlanetID, ShipIndexes: order.ShipIndexes,
		}); vErr == nil {
			g.postponeInvasion(order)
		}
		return nil
	}

	if !g.holdsOrbit(starID, order.OwnerID) {
		fleet, ferr := g.getFleet(order.FleetID)
		if ferr != nil {
			// Fleet gone (destroyed / driven off in the orbital fight).
			return g.cancelInvasionOrder(order, planet)
		}
		if fleet.StarID != starID || fleet.DestinationStarID != 0 || fleet.TurnsRemaining != 0 {
			// Order never applied at this star — stale, not an orbit loss.
			return nil
		}
		// Fleet is parked here but does not hold orbit (rival armed ships).
		return g.cancelInvasionOrder(order, planet)
	}

	return g.resolvePendingInvasion(order)
}

// postponeInvasion stores an Invade order until the pending tactical battle at
// its star finishes. Duplicate fleet+planet entries are ignored so a save/
// load or double call cannot replay the same landing.
func (g *Game) postponeInvasion(order PendingInvasion) {
	for _, existing := range g.PostponedInvasions {
		if existing.FleetID == order.FleetID && existing.TargetPlanetID == order.TargetPlanetID {
			return
		}
	}
	g.PostponedInvasions = append(g.PostponedInvasions, order)
}

// releasePostponedInvasionsAtStar runs after a tactical battle at starID ends.
// Invaders that still hold orbit resolve via resolvePendingInvasion;
// otherwise the order is cancelled with a report. Orders for other stars are
// left untouched.
func (g *Game) releasePostponedInvasionsAtStar(starID uint) []*InvasionReport {
	if g == nil || len(g.PostponedInvasions) == 0 {
		return nil
	}

	kept := make([]PendingInvasion, 0, len(g.PostponedInvasions))
	var reports []*InvasionReport
	for _, order := range g.PostponedInvasions {
		planet, err := g.getPlanet(order.TargetPlanetID)
		if err != nil || planet == nil {
			continue
		}
		if planet.ParentStar != starID {
			kept = append(kept, order)
			continue
		}
		// Battle is over; do not re-postpone. Re-validate + holdsOrbit.
		report := g.tryResolveInvasionOrder(order, false)
		if report != nil {
			g.InvasionReports = append(g.InvasionReports, report)
			g.recordInvasionEvent(report)
			reports = append(reports, report)
		}
	}
	g.PostponedInvasions = kept
	return reports
}

func (g *Game) cancelInvasionOrder(order PendingInvasion, planet *Planet) *InvasionReport {
	defenderOwner := uint(0)
	planetName := ""
	var x, y float64
	if planet != nil {
		defenderOwner = planet.Owner
		planetName = planet.Name
		if star, err := g.getStar(planet.ParentStar); err == nil && star != nil {
			x = float64(star.X)
			y = float64(star.Y)
		}
	}
	report := &InvasionReport{
		ID:              g.nextInvasionReportID(),
		Turn:            int64(g.CurrentTurn),
		PlanetID:        order.TargetPlanetID,
		PlanetName:      planetName,
		X:               x,
		Y:               y,
		AttackerOwnerID: order.OwnerID,
		DefenderOwnerID: defenderOwner,
		Success:         false,
		PlanetCaptured:  false,
		Cancelled:       true,
	}
	normalizeInvasionReportForces(report)
	return report
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
		report.CaptureEffects = g.capturePlanet(planet, order.OwnerID, &report.SurvivingUnits)
	} else {
		g.updatePlanetDefenders(planet, &report.SurvivingUnits)
	}
	g.removeInvasionTransports(fleet, order.ShipIndexes)
	if len(fleet.Ships) == 0 {
		g.removeFleet(fleet)
	}
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
// Duplicate indexes are ignored so a buggy order cannot multiply one transport.
func (g *Game) gatherMarinesFromInvasionOrder(fleet *Fleet, shipIndexes *[]uint) GroundForce {
	force := GroundForce{}
	if fleet == nil {
		return force
	}
	force.OwnerID = fleet.Owner
	if shipIndexes != nil {
		seen := make(map[uint]bool, len(*shipIndexes))
		for _, index := range *shipIndexes {
			if seen[index] {
				continue
			}
			seen[index] = true
			if int(index) >= len(fleet.Ships) {
				continue
			}
			ship := fleet.Ships[index]
			if ship.Type != "transport" {
				continue
			}
			g.appendTransportMarines(&force, ship)
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
	normalizeInvasionReportForces(report)

	return report
}

// ensureGroundForceSlices turns nil Marines/Armored into empty non-nil slices so
// invasion reports serialize as [] rather than null. Snapshot cloneSlice still
// preserves nil vs empty for restore; this only normalizes forces at report creation.
func ensureGroundForceSlices(force GroundForce) GroundForce {
	if force.Marines == nil {
		force.Marines = []GroundUnit{}
	}
	if force.Armored == nil {
		force.Armored = []GroundUnit{}
	}
	return force
}

func normalizeInvasionReportForces(report *InvasionReport) {
	if report == nil {
		return
	}
	report.InitialAttacker = ensureGroundForceSlices(report.InitialAttacker)
	report.InitialDefender = ensureGroundForceSlices(report.InitialDefender)
	report.SurvivingUnits = ensureGroundForceSlices(report.SurvivingUnits)
	// Cancelled reports never fight, and runGroundBattle can return a nil
	// slice when no rounds are fought — serialize rounds as [] not null.
	if report.Rounds == nil {
		report.Rounds = []GroundBattleRound{}
	}
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
	return GroundForce{
		OwnerID: force.OwnerID,
		Marines: cloneSlice(force.Marines),
		Armored: cloneSlice(force.Armored),
	}
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

// Conquest destroys one quarter of ordinary buildings (rounded down, in
// construction order). The former empire's unique Capitol is always removed.
// Foreign populations suffer 20% unrest until they are liberated.
const conquestUnrestPenalty = 0.20

func (g *Game) capturePlanet(planet *Planet, newOwnerID uint, survivingUnits *GroundForce) *CaptureEffects {
	if planet.PopulationRace == "" && planet.Population > 0 {
		if oldOwner, err := g.getPlayer(planet.Owner); err == nil && oldOwner != nil {
			planet.PopulationRace = oldOwner.Race.Name
		}
	}
	effects := &CaptureEffects{PopulationRace: planet.PopulationRace, DestroyedBuildings: []string{}, RemovedBuildings: []string{}}
	ordinary := 0
	for _, name := range planet.Buildings {
		if name != "Capitol" {
			ordinary++
		}
	}
	destroy := ordinary / 4
	remaining := make([]string, 0, len(planet.Buildings))
	for _, name := range planet.Buildings {
		if name == "Capitol" {
			effects.RemovedBuildings = append(effects.RemovedBuildings, name)
			continue
		}
		if destroy > 0 {
			effects.DestroyedBuildings = append(effects.DestroyedBuildings, name)
			destroy--
			continue
		}
		remaining = append(remaining, name)
	}
	planet.Buildings = remaining
	formerOwnerID := planet.Owner
	planet.Owner = newOwnerID
	g.prunePendingColonyBases(formerOwnerID)
	g.updatePlanetDefenders(planet, survivingUnits)
	planet.CurrentProject = ""
	planet.BuildQueue = nil
	planet.IndustryStack = 0
	planet.BuildBucket = 0
	planet.IsBuilding = false
	planet.Autobuild = false
	if player, err := g.getPlayer(newOwnerID); err == nil && player != nil {
		_ = g.exploreStarUnlocked(newOwnerID, planet.ParentStar)
		if planet.Population > 0 && planet.PopulationRace != "" && planet.PopulationRace != player.Race.Name && !governmentMoraleNeutral(g, player) {
			effects.UnrestPenalty = conquestUnrestPenalty
		}
		hydratePlanetMorale(g, planet, player)
	}
	return effects
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
