package game

import (
	"encoding/json"
	. "ssg/model"
	"sync"
)

// MarshalSave returns the complete serializable game state for a save file.
func (g *Game) MarshalSave() ([]byte, error) {
	g.ensureMu()
	g.mu.RLock()
	defer g.mu.RUnlock()
	return json.Marshal(g)
}

// HydrateLoadedGame re-attaches the live resources and reinitializes the
// non-serialized runtime state on a Game loaded from a save file.
func HydrateLoadedGame(g *Game, resources *ResourceManager) {
	if g == nil {
		return
	}
	if resources != nil {
		g.Resources = &resources.Resources
	}
	if g.TurnSubmissions == nil {
		g.TurnSubmissions = make(map[uint]Commands)
	}
	if g.PendingColonyBases == nil {
		g.PendingColonyBases = make(map[uint][]uint)
	}
	if g.PendingColonyShips == nil {
		g.PendingColonyShips = make(map[uint][]uint)
	}
	if g.PendingArtificialPlanets == nil {
		g.PendingArtificialPlanets = make(map[uint][]uint)
	}
	// Invade orders recorded mid-AdvanceTurn are turn-ephemeral: never replay
	// them from a loaded/uploaded save.
	g.PendingInvasions = nil
	// Postponed invasions survive only while a matching pending tactical battle
	// still exists at the target star (and the fleet/planet refs are live).
	// Orphans are dropped so a battle-less save cannot resurrect landings.
	g.prunePostponedInvasions()
	// Saves from before interactive combat never marked a human player;
	// default to player 1 so loaded games get tactical battles too.
	if len(g.Players) > 0 {
		anyHuman := false
		for _, p := range g.Players {
			if p != nil && p.IsHuman {
				anyHuman = true
				break
			}
		}
		if !anyHuman {
			g.Players[0].IsHuman = true
		}
	}
	g.mu = &sync.RWMutex{}
}

// prunePostponedInvasions drops postponed Invade orders that cannot still be
// honored: missing fleet/planet, or no active pending battle at the planet's
// star. Kept entries are those waiting on a live orbital fight.
func (g *Game) prunePostponedInvasions() {
	if g == nil || len(g.PostponedInvasions) == 0 {
		g.PostponedInvasions = nil
		return
	}
	kept := make([]PendingInvasion, 0, len(g.PostponedInvasions))
	seen := make(map[uint]map[uint]bool) // fleetID -> planetID
	for _, order := range g.PostponedInvasions {
		if _, err := g.getFleet(order.FleetID); err != nil {
			continue
		}
		planet, err := g.getPlanet(order.TargetPlanetID)
		if err != nil || planet == nil {
			continue
		}
		if !g.hasPendingBattleAtStar(planet.ParentStar) {
			continue
		}
		if seen[order.FleetID] == nil {
			seen[order.FleetID] = map[uint]bool{}
		}
		if seen[order.FleetID][order.TargetPlanetID] {
			continue // dedupe
		}
		seen[order.FleetID][order.TargetPlanetID] = true
		kept = append(kept, order)
	}
	g.PostponedInvasions = kept
}
