package ai

import "ssg/model"

// Decision is a single output of a Policy.Decide call: the commands to submit
// plus the human-readable reasoning lines to record in the agent's log.
type Decision struct {
	Commands model.Commands
	Notes    []string
}

// Policy turns a visible game snapshot into the commands the agent will
// submit. Implementations should be pure functions of their inputs.
type Policy interface {
	Name() string
	Decide(snapshot *VisibleGame, playerID uint) Decision
}

// findPlayer returns the player record for id, or nil if not present.
func findPlayer(snapshot *VisibleGame, id uint) *model.Player {
	for _, p := range snapshot.Players {
		if p != nil && p.ID == id {
			return p
		}
	}
	return nil
}

// myPlanets returns owned planets for the given player.
func myPlanets(snapshot *VisibleGame, playerID uint) []*model.Planet {
	var out []*model.Planet
	for _, star := range snapshot.Galaxy.Stars {
		for _, p := range star.Planets {
			if p != nil && p.Owner == playerID {
				out = append(out, p)
			}
		}
	}
	return out
}

// myFleets returns fleets owned by the player.
func myFleets(snapshot *VisibleGame, playerID uint) []*model.Fleet {
	var out []*model.Fleet
	for _, f := range snapshot.Fleets {
		if f != nil && f.Owner == playerID {
			out = append(out, f)
		}
	}
	return out
}
