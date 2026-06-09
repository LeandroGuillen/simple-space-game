package game

import (
	. "ssg/model"
)

// maxGroundBattleRounds bounds ground combat resolution to prevent infinite loops
const maxGroundBattleRounds = 20

// baseHitChance is the baseline chance for ground units to hit each other
const baseHitChance = 0.7

// resolveGroundInvasions processes all ground invasions at planets where
// fleets with marines are present at enemy or neutral planets
func (g *Game) resolveGroundInvasions() []*InvasionReport {
	if g == nil {
		return nil
	}

	var reports []*InvasionReport

	// Group fleets by planet location
	fleetsByPlanet := make(map[uint][]*Fleet)
	for _, fleet := range g.Fleets {
		if fleet == nil || fleet.DestinationStarID != 0 || fleet.TurnsRemaining != 0 {
			continue
		}
		// Find planets at this star  
		star, err := g.getStar(fleet.StarID)
		if err != nil {
			continue
		}
		for _, planet := range star.Planets {
			fleetsByPlanet[planet.ID] = append(fleetsByPlanet[planet.ID], fleet)
		}
	}

	// Check each planet for potential invasions
	for planetID, fleets := range fleetsByPlanet {
		planet, err := g.getPlanet(planetID)
		if err != nil {
			continue
		}

		// Find attacking fleets (different owner than planet, with marine transports)
		var attackingFleets []*Fleet
		for _, fleet := range fleets {
			if fleet.Owner != planet.Owner && g.fleetHasTransportMarines(fleet) {
				attackingFleets = append(attackingFleets, fleet)
			}
		}

		if len(attackingFleets) == 0 {
			continue
		}

		// Process invasion for each attacking player
		processedAttackers := make(map[uint]bool)
		for _, attackerFleet := range attackingFleets {
			if processedAttackers[attackerFleet.Owner] {
				continue // Already processed this player's invasion this turn
			}
			processedAttackers[attackerFleet.Owner] = true

			// Gather all marines from this player's transport ships at this planet
			attackerForce := g.gatherMarinesFromTransports(attackerFleet.Owner, fleets)
			if attackerForce.TotalUnits() == 0 {
				continue
			}

			// Create defender force from planet
			defenderForce := g.createPlanetDefenseForce(planet)

			// Resolve the invasion
			report := g.resolveInvasion(planet, attackerFleet.Owner, attackerForce, defenderForce)
			if report != nil {
				reports = append(reports, report)

				// Apply results
				if report.Success && report.PlanetCaptured {
					g.capturePlanet(planet, attackerFleet.Owner, &report.SurvivingUnits)
				} else {
					// Update planet defenders
					g.updatePlanetDefenders(planet, &report.SurvivingUnits)
				}

		// Remove marines from attacking transport ships
		g.removeMarinesFromTransports(attackerFleet.Owner, fleets, &attackerForce, &report.SurvivingUnits)
			}
		}
	}

	if len(reports) > 0 {
		g.InvasionReports = append(g.InvasionReports, reports...)
	}

	return reports
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

// gatherMarinesFromTransports creates a ground force from all marines in player's transport ships at a planet
func (g *Game) gatherMarinesFromTransports(ownerID uint, fleets []*Fleet) GroundForce {
	force := GroundForce{OwnerID: ownerID}
	
	for _, fleet := range fleets {
		if fleet.Owner != ownerID {
			continue
		}
		for _, ship := range fleet.Ships {
			marineCount := g.getTransportMarineCount(ship)
			for i := int64(0); i < marineCount; i++ {
				force.Marines = append(force.Marines, NewGroundUnit(MarineUnit))
			}
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

func (g *Game) removeMarinesFromTransports(ownerID uint, fleets []*Fleet, originalForce, survivingUnits *GroundForce) {
	// Remove transport ships that deployed their marines
	// Transport ships are consumed in the invasion (they land and don't return to fleet)
	for _, fleet := range fleets {
		if fleet.Owner == ownerID {
			// Remove transport ships from the fleet
			remainingShips := []FleetShip{}
			for _, ship := range fleet.Ships {
				if ship.Type != "transport" {
					remainingShips = append(remainingShips, ship)
				}
			}
			fleet.Ships = remainingShips
		}
	}
}

