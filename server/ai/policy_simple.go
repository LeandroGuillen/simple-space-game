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
	keepBuildQueueWarm(planets, me, &d)
	resolvePendingColonies(snapshot, playerID, &d)
	issueInvasionsAtEnemyPlanets(snapshot, playerID, &d)
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
	for _, planetID := range snapshot.PendingColonyBases[playerID] {
		d.Commands.ColonizeMany = append(d.Commands.ColonizeMany, &model.ColonizeCommand{
			PlanetID: planetID,
		})
		d.Notes = append(d.Notes, fmt.Sprintf("colonizing pending base on planet %d", planetID))
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
// queued to invade this turn stay put. If no reachable unexplored star exists,
// the fleet stays idle.
func sendIdleScouts(snapshot *VisibleGame, playerID uint, d *Decision) {
	if me := findPlayer(snapshot, playerID); me != nil {
		for _, traitID := range me.Race.TraitIDs {
			if traitID == "omniscient" {
				return
			}
		}
	}
	invading := map[uint]bool{}
	for _, cmd := range d.Commands.Invade {
		if cmd != nil {
			invading[cmd.FleetID] = true
		}
	}
	supplies := ownedSupplyStarCoords(snapshot, playerID)
	for _, f := range snapshot.Fleets {
		if f == nil || f.Owner != playerID {
			continue
		}
		if f.DestinationStarID != 0 {
			continue
		}
		if invading[f.ID] {
			continue
		}
		target := nearestReachableUnexploredStar(snapshot, playerID, f, supplies)
		if target == 0 {
			continue
		}
		d.Commands.FleetMove = append(d.Commands.FleetMove, &model.FleetMoveCommand{
			FleetID:           f.ID,
			DestinationStarID: target,
		})
		d.Notes = append(d.Notes, fmt.Sprintf("fleet %d: moving to nearest reachable unexplored star %d", f.ID, target))
	}
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
func nearestReachableUnexploredStar(snapshot *VisibleGame, playerID uint, fleet *model.Fleet, supplies [][2]float64) uint {
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
		if s == nil || s.Name != "" {
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
