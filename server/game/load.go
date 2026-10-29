package game

import (
	. "ssg/model"
	"sync"
)

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
	// Invade orders are turn-ephemeral: never replay them from a loaded/uploaded save.
	g.PendingInvasions = nil
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
