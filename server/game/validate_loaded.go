package game

import (
	"fmt"
	"math"
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
	return nil
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
		for _, d := range resources.ShipDesigns {
			if d.Name == name {
				return true
			}
		}
	}
	return false
}
