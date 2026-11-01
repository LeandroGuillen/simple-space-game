package ai

import (
	"fmt"
	"math"

	"ssg/game"
	"ssg/model"
)

// SimplePolicy is the minimal-but-functional v1 policy described in
// AI_PLAN.md: pick research when idle, prevent starvation, autobuild basic
// infra, resolve pending colony actions, and send idle scouts to nearest
// unexplored stars.
type SimplePolicy struct{}

func (SimplePolicy) Name() string { return "simple" }

func (SimplePolicy) Decide(snapshot *VisibleGame, playerID uint) Decision {
	d := Decision{}
	if snapshot == nil {
		d.Notes = append(d.Notes, "no snapshot available; submitting empty turn")
		return d
	}
	me := findPlayer(snapshot, playerID)
	if me == nil {
		d.Notes = append(d.Notes, fmt.Sprintf("player %d not present in visible state; submitting empty turn", playerID))
		return d
	}

	planets := myPlanets(snapshot, playerID)
	d.Notes = append(d.Notes, fmt.Sprintf("turn %.0f: %d owned planet(s), %d fleet(s)", snapshot.CurrentTurn, len(planets), len(myFleets(snapshot, playerID))))

	pickResearch(snapshot, me, &d)
	balanceLabor(me, planets, &d)
	resolvePendingColonies(snapshot, playerID, &d)
	issueInvasionsAtEnemyPlanets(snapshot, playerID, &d)
	planExpansion(snapshot, me, planets, &d)
	planShipDesign(snapshot, me, planets, &d)
	keepBuildQueueWarm(planets, me, &d)
	sendIdleScouts(snapshot, playerID, &d)

	return d
}

func pickResearch(snapshot *VisibleGame, me *model.Player, d *Decision) {
	if me.CurrentResearch != nil && (me.CurrentResearch.Subject != "" || me.CurrentResearch.TechName != "") {
		return
	}
	if snapshot.Resources == nil || snapshot.Resources.TechTree == nil {
		d.Notes = append(d.Notes, "no current research; tech tree missing from visible state")
		return
	}
	choice, subject, cost := firstAvailableTech(snapshot.Resources.TechTree, me)
	known := map[string]struct{}{}
	for _, name := range me.ResearchedTechs {
		known[name] = struct{}{}
	}
	prioritized := false
	if len(myPlanets(snapshot, me.ID)) < 3 && !techUnlocked(me, "Colony Ship") {
		for _, candidate := range snapshot.Resources.TechTree.Subjects {
			if candidate.Name != "Power" {
				continue
			}
			for _, level := range candidate.Levels {
				if level.Name != "Nuclear Fission" && level.Name != "Cold Fusion" {
					break
				}
				if !levelComplete(level, known, me) {
					choice, subject, cost = level.Name, candidate.Name, level.Cost
					prioritized = true
					break
				}
			}
		}
	}
	if !prioritized && len(me.Available.Beams) == 0 {
		for _, candidate := range snapshot.Resources.TechTree.Subjects {
			if candidate.Name == "Physics" && len(candidate.Levels) > 0 {
				level := candidate.Levels[0]
				if !levelComplete(level, known, me) {
					choice, subject, cost = level.Name, candidate.Name, level.Cost
				}
				break
			}
		}
	}
	if choice == "" {
		d.Notes = append(d.Notes, "no current research; no incomplete subjects found")
		return
	}
	d.Commands.SelectResearch = &model.SelectResearchCommand{TechName: choice}
	d.Notes = append(d.Notes, fmt.Sprintf("research: selecting %q from %s (cost %d)", choice, subject, cost))
}

// firstAvailableTech walks the tech subjects in order and returns the first
// research choice the player hasn't already completed.
//
// Matches the server's level-completion rule in game/research.go: a
// non-ForceAll level is considered complete as soon as ANY tech in it is
// researched (the player can only pick one sibling per level). A ForceAll
// level is complete when every tech in it is researched.
func firstAvailableTech(tree *model.TechTree, me *model.Player) (techName, subjectName string, cost uint64) {
	known := map[string]struct{}{}
	for _, t := range me.ResearchedTechs {
		known[t] = struct{}{}
	}
	for _, subject := range tree.Subjects {
		for _, level := range subject.Levels {
			if levelComplete(level, known, me) {
				continue
			}
			if levelUnlocksAllTechs(level, me) {
				return level.Name, subject.Name, level.Cost
			}
			for _, tech := range level.AvailableTechs {
				if _, ok := known[tech]; ok {
					continue
				}
				return tech, subject.Name, level.Cost
			}
		}
	}
	return "", "", 0
}

func levelComplete(level model.TechLevel, known map[string]struct{}, player *model.Player) bool {
	if len(level.AvailableTechs) == 0 {
		return false
	}
	if levelUnlocksAllTechs(level, player) {
		for _, t := range level.AvailableTechs {
			if _, ok := known[t]; !ok {
				return false
			}
		}
		return true
	}
	for _, t := range level.AvailableTechs {
		if _, ok := known[t]; ok {
			return true
		}
	}
	return false
}

func levelUnlocksAllTechs(level model.TechLevel, player *model.Player) bool {
	if player != nil {
		for _, traitID := range player.Race.TraitIDs {
			if traitID == "uncreative" {
				return false
			}
		}
	}
	if level.ForceAll {
		return true
	}
	if player == nil {
		return false
	}
	for _, traitID := range player.Race.TraitIDs {
		if traitID == "creative" {
			return true
		}
	}
	return false
}

func balanceLabor(me *model.Player, planets []*model.Planet, d *Decision) {
	for _, p := range planets {
		if p.Population <= 0 {
			continue
		}
		foodPerFarmer := p.Food.ProducedPerFarmer
		if foodPerFarmer <= 0 {
			continue
		}
		foodDemand := int64(math.Ceil(game.PopulationFoodDemand(me, p.Population)))
		if foodDemand <= 0 {
			continue
		}
		buildingFood := p.Food.ProducedPerBuilding
		totalFood := p.Food.Farmers*foodPerFarmer + buildingFood
		if totalFood >= foodDemand {
			continue
		}
		needed := foodDemand - totalFood
		extraFarmers := (needed + foodPerFarmer - 1) / foodPerFarmer
		maxShift := p.Industry.Workers
		if extraFarmers > maxShift {
			extraFarmers = maxShift
		}
		if extraFarmers <= 0 {
			continue
		}
		cmd := &model.MovePopulationCommand{
			PlanetID:   p.ID,
			Farmers:    p.Food.Farmers + extraFarmers,
			Workers:    p.Industry.Workers - extraFarmers,
			Scientists: p.Science.Scientists,
		}
		d.Commands.MovePopulation = append(d.Commands.MovePopulation, cmd)
		d.Notes = append(d.Notes, fmt.Sprintf("planet %d (%s): shifting %d worker(s) to farming to avert starvation", p.ID, p.Name, extraFarmers))
	}
}

func keepBuildQueueWarm(planets []*model.Planet, me *model.Player, d *Decision) {
	wanted := []string{"Automated Factory", "Research Lab", "Marine Barracks"}
	for _, p := range planets {
		planned := false
		for _, cmd := range d.Commands.Build {
			if cmd.PlanetID == p.ID {
				planned = true
				break
			}
		}
		if planned {
			continue
		}
		if p.CurrentProject != "" || len(p.BuildQueue) > 0 {
			continue
		}
		choice := ""
		for _, b := range wanted {
			if alreadyBuilt(p, b) {
				continue
			}
			if !techUnlocked(me, b) {
				continue
			}
			choice = b
			break
		}
		if choice == "" {
			d.Notes = append(d.Notes, fmt.Sprintf("planet %d (%s): no preferred building available; leaving idle", p.ID, p.Name))
			continue
		}
		d.Commands.Build = append(d.Commands.Build, &model.BuildCommand{
			PlanetID: p.ID,
			Building: choice,
		})
		d.Notes = append(d.Notes, fmt.Sprintf("planet %d (%s): queueing %s", p.ID, p.Name, choice))
	}
}

func alreadyBuilt(p *model.Planet, name string) bool {
	for _, b := range p.Buildings {
		if b == name {
			return true
		}
	}
	return false
}

func techUnlocked(me *model.Player, name string) bool {
	for _, b := range me.Available.Buildings {
		if b == name {
			return true
		}
	}
	return false
}

func resolvePendingColonies(snapshot *VisibleGame, playerID uint, d *Decision) {
	pending := snapshot.PendingColonyBases[playerID]
	if len(pending) > 0 {
		sourceID := pending[len(pending)-1]
		for _, star := range snapshot.Galaxy.Stars {
			if star == nil {
				continue
			}
			found := false
			for _, p := range star.Planets {
				if p != nil && p.ID == sourceID && p.Owner == playerID {
					found = true
					break
				}
			}
			if !found {
				continue
			}
			for _, p := range star.Planets {
				if colonizable(p) {
					d.Commands.ColonizeMany = append(d.Commands.ColonizeMany, &model.ColonizeCommand{PlanetID: p.ID})
					break
				}
			}
			break
		}
	}
	for _, fleetID := range snapshot.PendingColonyShips[playerID] {
		d.Commands.ColonyShip = &model.PrepareColonyShipCommand{FleetID: fleetID}
		d.Notes = append(d.Notes, fmt.Sprintf("preparing colony ship on fleet %d", fleetID))
	}
}

// issueInvasionsAtEnemyPlanets restores the prior auto-capture behavior for the
// AI now that transports no longer invade without an Invade order: any idle
// fleet with transports at a star with an enemy colony issues Invade.
func issueInvasionsAtEnemyPlanets(snapshot *VisibleGame, playerID uint, d *Decision) {
	invading := map[uint]bool{}
	for _, f := range snapshot.Fleets {
		if f == nil || f.Owner != playerID || f.DestinationStarID != 0 || f.TurnsRemaining != 0 {
			continue
		}
		var transportIndexes []uint
		for i, ship := range f.Ships {
			if ship.Type == "transport" {
				transportIndexes = append(transportIndexes, uint(i))
			}
		}
		if len(transportIndexes) == 0 {
			continue
		}
		star := findStarByID(snapshot, f.StarID)
		if star == nil {
			continue
		}
		for _, planet := range star.Planets {
			if planet == nil || planet.Owner == 0 || planet.Owner == playerID {
				continue
			}
			indexes := append([]uint(nil), transportIndexes...)
			d.Commands.Invade = append(d.Commands.Invade, &model.InvadeCommand{
				FleetID:        f.ID,
				TargetPlanetID: planet.ID,
				ShipIndexes:    &indexes,
			})
			invading[f.ID] = true
			d.Notes = append(d.Notes, fmt.Sprintf("fleet %d: invading planet %d", f.ID, planet.ID))
			break
		}
	}
	if len(invading) > 0 {
		d.Notes = append(d.Notes, fmt.Sprintf("invade: queued %d fleet(s)", len(invading)))
	}
}

func findStarByID(snapshot *VisibleGame, starID uint) *model.Star {
	if snapshot == nil {
		return nil
	}
	for _, star := range snapshot.Galaxy.Stars {
		if star != nil && star.ID == starID {
			return star
		}
	}
	return nil
}

// sendIdleScouts moves any stationary fleet (no destination) to the nearest
// unexplored star that is still inside the fleet's fuel range (same rule as
// game.validateFleetMoveFuelRange / FuelDestinationReachable). We treat
// "unexplored" as a star with no Name in the visible payload. Fleets already
// queued for colonization or invasion, or with a local colony target, stay put.
// If no unreserved reachable unexplored star exists, the fleet stays idle.
func sendIdleScouts(snapshot *VisibleGame, playerID uint, d *Decision) {
	if me := findPlayer(snapshot, playerID); me != nil {
		for _, traitID := range me.Race.TraitIDs {
			if traitID == "omniscient" {
				return
			}
		}
	}
	busy := map[uint]bool{}
	for _, cmd := range d.Commands.Invade {
		if cmd != nil {
			busy[cmd.FleetID] = true
		}
	}
	for _, move := range d.Commands.FleetMove {
		if move != nil {
			busy[move.FleetID] = true
		}
	}
	if d.Commands.ColonyShip != nil {
		busy[d.Commands.ColonyShip.FleetID] = true
	}
	if d.Commands.Colonize != nil && d.Commands.Colonize.FleetID != 0 {
		busy[d.Commands.Colonize.FleetID] = true
	}
	for _, cmd := range d.Commands.ColonizeMany {
		if cmd != nil && cmd.FleetID != 0 {
			busy[cmd.FleetID] = true
		}
	}
	supplies := ownedSupplyStarCoords(snapshot, playerID)
	// Avoid sending another scout to a destination already selected this turn
	// or already being visited by one of this player's fleets.
	reserved := make(map[uint]bool)
	for _, f := range snapshot.Fleets {
		if f != nil && f.Owner == playerID && f.DestinationStarID != 0 {
			reserved[f.DestinationStarID] = true
		}
	}
	for _, move := range d.Commands.FleetMove {
		if move != nil {
			reserved[move.DestinationStarID] = true
		}
	}
	for _, f := range snapshot.Fleets {
		if f == nil || f.Owner != playerID {
			continue
		}
		if f.DestinationStarID != 0 || f.TurnsRemaining != 0 {
			continue
		}
		if busy[f.ID] || hasLocalColonyTarget(snapshot, f) {
			continue
		}
		target := nearestReachableUnexploredStar(snapshot, playerID, f, supplies, reserved)
		if target == 0 {
			continue
		}
		reserved[target] = true
		d.Commands.FleetMove = append(d.Commands.FleetMove, &model.FleetMoveCommand{
			FleetID:           f.ID,
			DestinationStarID: target,
		})
		d.Notes = append(d.Notes, fmt.Sprintf("fleet %d: moving to nearest reachable unexplored star %d", f.ID, target))
	}
}

// A colony ship in orbit can colonize an unowned habitable planet here.
// Keep the fleet available rather than sending it away to scout.
func hasLocalColonyTarget(snapshot *VisibleGame, fleet *model.Fleet) bool {
	if fleet == nil {
		return false
	}
	hasColonyShip := false
	for _, ship := range fleet.Ships {
		if ship.Type == "colony_ship" {
			hasColonyShip = true
			break
		}
	}
	if !hasColonyShip {
		return false
	}
	star := findStarByID(snapshot, fleet.StarID)
	if star == nil {
		return false
	}
	for _, planet := range star.Planets {
		if planet != nil && planet.Owner == 0 && planet.Environment != model.PlanetEnvironmentInvalid && planet.Environment != model.PlanetEnvironmentGasGiant && planet.Environment != model.PlanetEnvironmentAsteroidBelt {
			return true
		}
	}
	return false
}

func ownedSupplyStarCoords(snapshot *VisibleGame, playerID uint) [][2]float64 {
	if snapshot == nil {
		return nil
	}
	var out [][2]float64
	for _, star := range snapshot.Galaxy.Stars {
		if star == nil {
			continue
		}
		for _, planet := range star.Planets {
			if planet != nil && planet.Owner == playerID {
				out = append(out, [2]float64{float64(star.X), float64(star.Y)})
				break
			}
		}
	}
	return out
}

// nearestReachableUnexploredStar picks the closest unexplored star (by fleet
// position) that FuelDestinationReachable accepts for this fleet's fuel range
// (same ResolveShipFuelRange chain as submit validation). Returns 0 when none
// are reachable or range cannot be determined.
func nearestReachableUnexploredStar(snapshot *VisibleGame, playerID uint, fleet *model.Fleet, supplies [][2]float64, reserved map[uint]bool) uint {
	if snapshot == nil || fleet == nil {
		return 0
	}
	rangeParsecs, unlimited, ok := game.ShipsFuelRange(fleet.Ships, visibleFuelRangeLookup(snapshot, playerID))
	if !ok {
		return 0
	}
	var bestID uint
	bestDist := math.MaxFloat64
	for _, s := range snapshot.Galaxy.Stars {
		if s == nil || s.Name != "" || reserved[s.ID] {
			continue
		}
		if !game.FuelDestinationReachable(float64(s.X), float64(s.Y), supplies, rangeParsecs, unlimited) {
			continue
		}
		dx := float64(s.X) - fleet.X
		dy := float64(s.Y) - fleet.Y
		d := dx*dx + dy*dy
		if d < bestDist {
			bestDist = d
			bestID = s.ID
		}
	}
	return bestID
}

func visibleFuelRangeLookup(snapshot *VisibleGame, playerID uint) game.FuelRangeLookup {
	lookup := game.FuelRangeLookup{}
	if snapshot != nil && snapshot.Resources != nil {
		if snapshot.Resources.ShipParts != nil {
			lookup.FuelCells = snapshot.Resources.ShipParts.FuelCells
		}
		if snapshot.Resources.ShipDesignCollection != nil {
			lookup.StockDesigns = snapshot.Resources.ShipDesignCollection.ShipDesigns
		}
	}
	if me := findPlayer(snapshot, playerID); me != nil {
		lookup.PlayerDesigns = me.ShipDesigns
		lookup.AvailableCellNames = me.Available.FuelCells
	}
	return lookup
}
