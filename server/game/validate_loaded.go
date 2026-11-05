package game

import (
	"fmt"
	"math"
	"slices"
	. "ssg/model"
)

// ValidateLoadedGame checks structural and catalog invariants on a Game decoded
// from an uploaded save or on-disk scenario. It does not mutate g and does not
// depend on HTTP. Both LoadUploadedSave and LoadScenario call this before
// HydrateLoadedGame.
func ValidateLoadedGame(g *Game, resources *ResourceManager) error {
	if g == nil {
		return fmt.Errorf("game is nil")
	}
	if g.CurrentTurn < 0 || math.IsNaN(g.CurrentTurn) || math.IsInf(g.CurrentTurn, 0) {
		return fmt.Errorf("current turn is negative or invalid")
	}

	if len(g.Players) == 0 {
		return fmt.Errorf("no players")
	}
	if len(g.Players) > MaxPlayers {
		return fmt.Errorf("too many players: %d (max %d)", len(g.Players), MaxPlayers)
	}

	playerIDs := make(map[uint]struct{}, len(g.Players))
	for i, p := range g.Players {
		if p == nil {
			return fmt.Errorf("player at index %d is nil", i)
		}
		if p.ID == 0 {
			return fmt.Errorf("player has zero id")
		}
		if _, exists := playerIDs[p.ID]; exists {
			return fmt.Errorf("duplicate player id %d", p.ID)
		}
		playerIDs[p.ID] = struct{}{}

		if math.IsNaN(p.Treasury) || math.IsInf(p.Treasury, 0) {
			return fmt.Errorf("player %d has absurd treasury value", p.ID)
		}
		// Negative treasury is reachable in normal play (command-point overdraw
		// and freighter operating costs are subtracted with no floor), so do
		// not reject it on load — only NaN/Inf are absurd.
		// TaxRate is unused by turn logic today (no SetTaxRate handler); treat
		// it as a unit-interval rate matching the field name.
		if math.IsNaN(p.TaxRate) || math.IsInf(p.TaxRate, 0) || p.TaxRate < 0 || p.TaxRate > 1 {
			return fmt.Errorf("player %d has out-of-range tax_rate %v (want 0..1)", p.ID, p.TaxRate)
		}
		if err := validatePlayerRace(p, resources); err != nil {
			return err
		}
		if err := validatePlayerTechs(p, resources); err != nil {
			return err
		}
		if err := validatePlayerCurrentResearch(p, resources); err != nil {
			return err
		}
	}

	if g.Status != "" && g.Status != "active" && g.Status != "finished" {
		return fmt.Errorf("invalid game status %q", g.Status)
	}
	if g.WinnerID != 0 {
		winner := findPlayerByID(g.Players, g.WinnerID)
		if g.Status != "finished" || winner == nil || winner.Eliminated {
			return fmt.Errorf("invalid game winner %d", g.WinnerID)
		}
	}

	starIDs := make(map[uint]struct{}, len(g.Galaxy.Stars))
	planetIDs := make(map[uint]struct{})
	for i, star := range g.Galaxy.Stars {
		if star == nil {
			return fmt.Errorf("star at index %d is nil", i)
		}
		if star.ID == 0 {
			return fmt.Errorf("star has zero id")
		}
		if _, exists := starIDs[star.ID]; exists {
			return fmt.Errorf("duplicate star id %d", star.ID)
		}
		starIDs[star.ID] = struct{}{}

		for j, planet := range star.Planets {
			if planet == nil {
				return fmt.Errorf("planet at index %d of star %d is nil", j, star.ID)
			}
			if planet.ID == 0 {
				return fmt.Errorf("planet has zero id")
			}
			if _, exists := planetIDs[planet.ID]; exists {
				return fmt.Errorf("duplicate planet id %d", planet.ID)
			}
			planetIDs[planet.ID] = struct{}{}

			if planet.Population < 0 {
				return fmt.Errorf("planet %d has negative population", planet.ID)
			}
			if planet.MaxPopulation < 0 {
				return fmt.Errorf("planet %d has negative max population", planet.ID)
			}
			if planet.Owner != 0 {
				if _, ok := playerIDs[planet.Owner]; !ok {
					return fmt.Errorf("planet %d references missing player %d", planet.ID, planet.Owner)
				}
			}
			if err := validatePlanetRanges(planet); err != nil {
				return err
			}
			owner := findPlayerByID(g.Players, planet.Owner)
			if err := validatePlanetBuildings(planet, owner, resources); err != nil {
				return err
			}
		}
	}

	// Second pass: parent_star must exist (planets can only belong to listed stars).
	for _, star := range g.Galaxy.Stars {
		if star == nil {
			continue
		}
		for _, planet := range star.Planets {
			if planet == nil {
				continue
			}
			parent := planet.ParentStar
			if parent == 0 {
				parent = star.ID
			}
			if _, ok := starIDs[parent]; !ok {
				return fmt.Errorf("planet %d references missing star %d", planet.ID, parent)
			}
		}
	}

	fleetIDs := make(map[uint]struct{}, len(g.Fleets))
	for i, fleet := range g.Fleets {
		if fleet == nil {
			return fmt.Errorf("fleet at index %d is nil", i)
		}
		if fleet.ID == 0 {
			return fmt.Errorf("fleet has zero id")
		}
		if _, exists := fleetIDs[fleet.ID]; exists {
			return fmt.Errorf("duplicate fleet id %d", fleet.ID)
		}
		fleetIDs[fleet.ID] = struct{}{}

		if _, ok := playerIDs[fleet.Owner]; !ok {
			return fmt.Errorf("fleet %d references missing player %d", fleet.ID, fleet.Owner)
		}
		if fleet.StarID != 0 {
			if _, ok := starIDs[fleet.StarID]; !ok {
				return fmt.Errorf("fleet %d references missing star %d", fleet.ID, fleet.StarID)
			}
		}
		if fleet.DestinationStarID != 0 {
			if _, ok := starIDs[fleet.DestinationStarID]; !ok {
				return fmt.Errorf("fleet %d references missing destination star %d", fleet.ID, fleet.DestinationStarID)
			}
		}
		if fleet.OriginStarID != 0 {
			if _, ok := starIDs[fleet.OriginStarID]; !ok {
				return fmt.Errorf("fleet %d references missing origin star %d", fleet.ID, fleet.OriginStarID)
			}
		}

		owner := findPlayerByID(g.Players, fleet.Owner)
		for _, ship := range fleet.Ships {
			if ship.DesignName == "" {
				continue
			}
			if !shipDesignKnown(owner, resources, ship.DesignName) {
				return fmt.Errorf("fleet %d references unknown ship design %q", fleet.ID, ship.DesignName)
			}
		}
	}

	// Postponed invasions must reference live fleets/planets/owners and a
	// pending battle at the target star; otherwise reject the upload (Hydrate
	// would prune them on trusted loads, but uploads must not smuggle orphans).
	pendingBattleStars := make(map[uint]struct{}, len(g.PendingBattles))
	for i, b := range g.PendingBattles {
		if b == nil {
			return fmt.Errorf("pending battle at index %d is nil", i)
		}
		if b.Status == TacticalBattleActive {
			pendingBattleStars[b.StarID] = struct{}{}
		}
	}
	seenPostponed := make(map[uint]map[uint]bool)
	for i, order := range g.PostponedInvasions {
		if _, ok := playerIDs[order.OwnerID]; !ok {
			return fmt.Errorf("postponed invasion %d references missing player %d", i, order.OwnerID)
		}
		if _, ok := fleetIDs[order.FleetID]; !ok {
			return fmt.Errorf("postponed invasion %d references missing fleet %d", i, order.FleetID)
		}
		if _, ok := planetIDs[order.TargetPlanetID]; !ok {
			return fmt.Errorf("postponed invasion %d references missing planet %d", i, order.TargetPlanetID)
		}
		planetStar := postponedInvasionStarID(g, order.TargetPlanetID)
		if planetStar == 0 {
			return fmt.Errorf("postponed invasion %d references planet %d with unknown star", i, order.TargetPlanetID)
		}
		if _, ok := pendingBattleStars[planetStar]; !ok {
			return fmt.Errorf("postponed invasion %d has no pending battle at star %d", i, planetStar)
		}
		if seenPostponed[order.FleetID] == nil {
			seenPostponed[order.FleetID] = map[uint]bool{}
		}
		if seenPostponed[order.FleetID][order.TargetPlanetID] {
			return fmt.Errorf("duplicate postponed invasion for fleet %d planet %d", order.FleetID, order.TargetPlanetID)
		}
		seenPostponed[order.FleetID][order.TargetPlanetID] = true
	}

	return nil
}

func postponedInvasionStarID(g *Game, planetID uint) uint {
	for _, star := range g.Galaxy.Stars {
		if star == nil {
			continue
		}
		for _, planet := range star.Planets {
			if planet != nil && planet.ID == planetID {
				if planet.ParentStar != 0 {
					return planet.ParentStar
				}
				return star.ID
			}
		}
	}
	return 0
}

func findPlayerByID(players []*Player, id uint) *Player {
	for _, p := range players {
		if p != nil && p.ID == id {
			return p
		}
	}
	return nil
}

func validatePlayerRace(p *Player, resources *ResourceManager) error {
	gov := p.Race.Government
	if gov != "" && resources != nil && resources.TechTree != nil {
		if _, ok := findGovernmentTech(&resources.Resources, gov); !ok {
			return fmt.Errorf("player %d has unknown government %q", p.ID, gov)
		}
	}
	if len(p.Race.TraitIDs) == 0 {
		return nil
	}
	byID := raceTraitDefinitionsByID()
	for _, traitID := range p.Race.TraitIDs {
		if _, ok := byID[traitID]; !ok {
			return fmt.Errorf("player %d has unknown race trait %q", p.ID, traitID)
		}
	}
	return nil
}

func validatePlayerTechs(p *Player, resources *ResourceManager) error {
	if resources == nil || resources.TechTree == nil || len(p.ResearchedTechs) == 0 {
		return nil
	}
	known := make(map[string]struct{}, len(resources.TechTree.AllTechs))
	for _, tech := range resources.TechTree.AllTechs {
		known[tech.Name] = struct{}{}
	}
	for _, name := range p.ResearchedTechs {
		if name == "" {
			continue
		}
		if _, ok := known[name]; !ok {
			return fmt.Errorf("player %d has unknown tech %q", p.ID, name)
		}
	}
	return nil
}

func validatePlayerCurrentResearch(p *Player, resources *ResourceManager) error {
	if p == nil || p.CurrentResearch == nil || resources == nil {
		return nil
	}
	// Lookups live on *Game and read Resources; reuse them so load-time checks
	// match applyResearch / validateSelectResearchOrder.
	g := &Game{Resources: &resources.Resources}
	if err := g.validateCurrentResearchProject(p); err != nil {
		return fmt.Errorf("player %d: %w", p.ID, err)
	}
	proj := p.CurrentResearch
	if subjectName, ok := parseHyperAdvancedResearch(proj.TechName); ok {
		if err := g.validateHyperAdvancedResearchOrder(p, subjectName); err != nil {
			return fmt.Errorf("player %d: %w", p.ID, err)
		}
		return nil
	}
	level, _ := g.findResearchLevel(proj.LevelName)
	if isResearchLevelComplete(p, level) {
		return fmt.Errorf("player %d: research level %q is already complete", p.ID, level.Name)
	}
	if len(selectedResearchTechs(proj)) == 0 {
		return fmt.Errorf("player %d: research project has no selected techs", p.ID)
	}
	if proj.TechName != "" && len(proj.TechNames) > 0 && !slices.Contains(proj.TechNames, proj.TechName) {
		return fmt.Errorf("player %d: research tech %q is not in selected techs", p.ID, proj.TechName)
	}
	return nil
}

// validatePlanetRanges enforces enum and non-negative stock bounds used by the
// rules (Planet*Max sentinels in model; stocks that planetTurn indexes/adds).
func validatePlanetRanges(planet *Planet) error {
	if planet == nil {
		return nil
	}
	id := planet.ID
	if !validPlanetSize(planet.Size) {
		return fmt.Errorf("planet %d has out-of-range size %d (want %d..%d)", id, planet.Size, PlanetSizeInvalid+1, PlanetSizeMax-1)
	}
	if !validPlanetGravity(planet.Gravity) {
		return fmt.Errorf("planet %d has out-of-range gravity %d (want %d..%d)", id, planet.Gravity, PlanetGravityInvalid+1, PlanetGravityMax-1)
	}
	if !validPlanetEnvironment(planet.Environment) {
		return fmt.Errorf("planet %d has out-of-range environment %d (want %d..%d)", id, planet.Environment, PlanetEnvironmentInvalid+1, PlanetEnvironmentMax-1)
	}
	if !validPlanetMineral(planet.MineralClass) {
		return fmt.Errorf("planet %d has out-of-range mineral_class %d (want %d..%d)", id, planet.MineralClass, PlanetMineralInvalid+1, PlanetMineralMax-1)
	}
	if math.IsNaN(planet.Morale.Bonus) || math.IsInf(planet.Morale.Bonus, 0) {
		return fmt.Errorf("planet %d has invalid morale bonus", id)
	}
	if math.IsNaN(planet.Morale.Multiplier) || math.IsInf(planet.Morale.Multiplier, 0) || planet.Morale.Multiplier < 0 {
		// moraleMultiplier clamps to >= 0; negative multipliers are never produced.
		return fmt.Errorf("planet %d has invalid morale multiplier", id)
	}
	if planet.Pollution < 0 {
		return fmt.Errorf("planet %d has negative pollution", id)
	}
	if planet.IndustryStack < 0 {
		return fmt.Errorf("planet %d has negative industry_stack", id)
	}
	if planet.BuildBucket < 0 {
		return fmt.Errorf("planet %d has negative build_bucket", id)
	}
	if planet.Marines < 0 {
		return fmt.Errorf("planet %d has negative marines", id)
	}
	if planet.Armored < 0 {
		return fmt.Errorf("planet %d has negative armor", id)
	}
	if planet.TerraformingCount < 0 {
		return fmt.Errorf("planet %d has negative terraforming_count", id)
	}
	if math.IsNaN(planet.GravityMultiplier) || math.IsInf(planet.GravityMultiplier, 0) {
		return fmt.Errorf("planet %d has invalid gravity_multiplier", id)
	}
	return nil
}

func validPlanetSize(v PlanetSize) bool {
	return v > PlanetSizeInvalid && v < PlanetSizeMax
}

func validPlanetGravity(v PlanetGravity) bool {
	return v > PlanetGravityInvalid && v < PlanetGravityMax
}

func validPlanetEnvironment(v PlanetEnvironment) bool {
	return v > PlanetEnvironmentInvalid && v < PlanetEnvironmentMax
}

func validPlanetMineral(v PlanetMineral) bool {
	return v > PlanetMineralInvalid && v < PlanetMineralMax
}

func validatePlanetBuildings(planet *Planet, owner *Player, resources *ResourceManager) error {
	if planet == nil {
		return nil
	}
	for _, name := range planet.Buildings {
		if name == "" {
			continue
		}
		if !catalogBuildingKnown(resources, name) {
			return fmt.Errorf("planet %d has unknown building %q", planet.ID, name)
		}
	}
	for _, name := range planet.BuildQueue {
		if name == "" {
			continue
		}
		if !planetProjectKnown(name, owner, resources) {
			return fmt.Errorf("planet %d build_queue has unknown project %q", planet.ID, name)
		}
	}
	if planet.CurrentProject != "" && !planetProjectKnown(planet.CurrentProject, owner, resources) {
		return fmt.Errorf("planet %d has unknown current_project %q", planet.ID, planet.CurrentProject)
	}
	return nil
}

// planetProjectKnown accepts catalog buildings, Trade Goods / Housing (see
// isSpecialNonBuildingProject), and known ship designs (production projects).
func planetProjectKnown(name string, owner *Player, resources *ResourceManager) bool {
	if isSpecialNonBuildingProject(name) {
		return true
	}
	if catalogBuildingKnown(resources, name) {
		return true
	}
	return shipDesignKnown(owner, resources, name)
}

func catalogBuildingKnown(resources *ResourceManager, name string) bool {
	if resources == nil || resources.BuildingCollection == nil {
		return false
	}
	return buildingExists(resources.Buildings, name)
}

func shipDesignKnown(owner *Player, resources *ResourceManager, name string) bool {
	if owner != nil {
		for _, d := range owner.ShipDesigns {
			if d.Name == name {
				return true
			}
		}
	}
	if resources != nil {
		for _, d := range resources.ShipDesignCatalog() {
			if d.Name == name {
				return true
			}
		}
	}
	return false
}
