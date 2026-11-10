package game

import (
	"fmt"
	"slices"
)

// ShipDesignInUseError means changing a design would invalidate production or ships.
type ShipDesignInUseError struct{ Name, Use string }

func (e *ShipDesignInUseError) Error() string {
	return fmt.Sprintf("ship design %q is in use %s; cancel its build orders and remove ships using it, or save the changes under a new name", e.Name, e.Use)
}

// GuardShipDesignChanges holds the game lock until the returned release function
// is called. Catalog writers must keep all games guarded through publication so
// a turn cannot start using a design between the usage check and its removal.
func (g *Game) GuardShipDesignChanges(names ...string) (release func(), err error) {
	g.ensureMu()
	g.mu.Lock()
	for _, name := range names {
		if err := g.shipDesignUsage(name, 0); err != nil {
			g.mu.Unlock()
			return nil, err
		}
	}
	return g.mu.Unlock, nil
}

// Caller holds g.mu. owner == 0 checks every player for a shared catalog change.
func (g *Game) shipDesignUsage(name string, owner uint) error {
	if name == "" {
		return nil
	}
	conflict := func(use string) error { return &ShipDesignInUseError{Name: name, Use: use} }
	for _, star := range g.Galaxy.Stars {
		for _, p := range star.Planets {
			if p == nil || (owner != 0 && p.Owner != owner) {
				continue
			}
			if p.IsBuilding && p.CurrentProject == name {
				return conflict(fmt.Sprintf("under construction at %s", p.Name))
			}
			if slices.Contains(p.BuildQueue, name) {
				return conflict(fmt.Sprintf("in the build queue at %s", p.Name))
			}
		}
	}
	for playerID, commands := range g.TurnSubmissions {
		if owner != 0 && playerID != owner {
			continue
		}
		for _, c := range commands.Build {
			if c != nil && (c.Building == name || slices.Contains(c.BuildQueue, name)) {
				return conflict("in submitted build orders")
			}
		}
	}
	for _, fleet := range g.Fleets {
		if fleet == nil || (owner != 0 && fleet.Owner != owner) {
			continue
		}
		for _, ship := range fleet.Ships {
			if ship.DesignName == name {
				return conflict("by existing ships")
			}
		}
	}
	return nil
}
