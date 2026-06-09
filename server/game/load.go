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
	g.mu = &sync.Mutex{}
}
