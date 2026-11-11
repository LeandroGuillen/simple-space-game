package game

import (
	"encoding/json"
	"fmt"
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

// MarshalSinglePlayerSave allows full-state export only for a player with server AI opponents.
func (g *Game) MarshalSinglePlayerSave(playerID uint) ([]byte, error) {
	g.ensureMu()
	g.mu.RLock()
	defer g.mu.RUnlock()
	found := false
	for _, player := range g.Players {
		if player == nil {
			return nil, fmt.Errorf("full saves require a valid player and only server AI opponents")
		}
		if player.ID == playerID {
			found = true
		} else if player.Controller != "server_ai" {
			return nil, fmt.Errorf("full saves require a valid player and only server AI opponents")
		}
	}
	if !found {
		return nil, fmt.Errorf("full saves require a valid player and only server AI opponents")
	}
	return json.Marshal(g)
}

// HydrateLoadedGame re-attaches the live resources and reinitializes the
// non-serialized runtime state on a Game loaded from a save file.
func HydrateLoadedGame(g *Game, resources *ResourceManager) {
	if g == nil {
		return
	}
	if g.Status == "" {
		g.Status = "active"
	}
	if resources != nil {
		g.Resources = &resources.Resources
	}
	// Normalize legacy beam names while preserving saved damage and all other
	// parts (including the distinct Fusion drive). This keeps unlock checks and
	// AI design comparisons consistent with the current beam catalog.
	for _, player := range g.Players {
		if player == nil {
			continue
		}
		for i := range player.Available.Beams {
			player.Available.Beams[i] = canonicalBeamName(player.Available.Beams[i])
		}
		for i := range player.ShipDesigns {
			for j := range player.ShipDesigns[i].Beams {
				beam := &player.ShipDesigns[i].Beams[j]
				beam.Name = canonicalBeamName(beam.Name)
			}
		}
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

// ResetTurnStatusAfterRestart discards readiness because submitted orders are
// intentionally not serialized in saves.
func (g *Game) ResetTurnStatusAfterRestart() {
	g.TurnSubmissions = make(map[uint]Commands)
	g.resetTurnStatus()
}
