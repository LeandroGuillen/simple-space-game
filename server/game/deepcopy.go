package game

import (
	"ssg/model"
)

// cloneSlice returns a shallow copy of src, preserving nil vs empty:
// nil stays nil; a non-nil empty slice stays a non-nil empty slice.
func cloneSlice[T any](src []T) []T {
	if src == nil {
		return nil
	}
	out := make([]T, len(src))
	copy(out, src)
	return out
}

// cloneMap returns a shallow copy of src, preserving nil vs empty maps.
func cloneMap[K comparable, V any](src map[K]V) map[K]V {
	if src == nil {
		return nil
	}
	out := make(map[K]V, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func copyPlanet(p *model.Planet) *model.Planet {
	if p == nil {
		return nil
	}
	cp := *p
	cp.Buildings = cloneSlice(p.Buildings)
	cp.BuildQueue = cloneSlice(p.BuildQueue)
	cp.Morale.Sources = cloneSlice(p.Morale.Sources)
	return &cp
}

func copyShipDesign(d model.ShipDesign) model.ShipDesign {
	cp := d
	cp.Beams = cloneSlice(d.Beams)
	cp.Missiles = cloneSlice(d.Missiles)
	cp.SpecialSystems = cloneSlice(d.SpecialSystems)
	return cp
}

func copyTacticalShip(s *model.TacticalShip) *model.TacticalShip {
	if s == nil {
		return nil
	}
	cp := *s
	cp.Beams = cloneSlice(s.Beams)
	cp.Missiles = cloneSlice(s.Missiles)
	return &cp
}

func copyTacticalBattle(b *model.TacticalBattle) *model.TacticalBattle {
	if b == nil {
		return nil
	}
	cp := *b
	if b.Ships == nil {
		cp.Ships = nil
	} else {
		cp.Ships = make([]*model.TacticalShip, len(b.Ships))
		for i, ship := range b.Ships {
			cp.Ships[i] = copyTacticalShip(ship)
		}
	}
	cp.TurnOrder = cloneSlice(b.TurnOrder)
	cp.HumanOwnerIDs = cloneSlice(b.HumanOwnerIDs)
	cp.Events = cloneSlice(b.Events)
	return &cp
}

func copyBattleSide(side model.BattleSide) model.BattleSide {
	cp := side
	cp.OriginalShips = cloneSlice(side.OriginalShips)
	cp.SurvivingShips = cloneSlice(side.SurvivingShips)
	return cp
}

func copyBattleReport(r *model.BattleReport) *model.BattleReport {
	if r == nil {
		return nil
	}
	cp := *r
	if r.Sides == nil {
		cp.Sides = nil
	} else {
		cp.Sides = make([]model.BattleSide, len(r.Sides))
		for i, side := range r.Sides {
			cp.Sides[i] = copyBattleSide(side)
		}
	}
	return &cp
}

func copyInvasionReport(r *model.InvasionReport) *model.InvasionReport {
	if r == nil {
		return nil
	}
	cp := *r
	cp.Rounds = cloneSlice(r.Rounds)
	cp.InitialAttacker = copyGroundForce(r.InitialAttacker)
	cp.InitialDefender = copyGroundForce(r.InitialDefender)
	cp.SurvivingUnits = copyGroundForce(r.SurvivingUnits)
	if r.CaptureEffects != nil {
		effects := *r.CaptureEffects
		effects.DestroyedBuildings = cloneSlice(effects.DestroyedBuildings)
		effects.RemovedBuildings = cloneSlice(effects.RemovedBuildings)
		cp.CaptureEffects = &effects
	}
	return &cp
}

func copyTurnStatus(ts TurnStatus) TurnStatus {
	return TurnStatus{
		Mode:              ts.Mode,
		ReadyPlayerIDs:    cloneSlice(ts.ReadyPlayerIDs),
		WaitingPlayerIDs:  cloneSlice(ts.WaitingPlayerIDs),
		RequiredPlayerIDs: cloneSlice(ts.RequiredPlayerIDs),
	}
}

// copyUintSlice preserves nil vs empty; kept as a named alias for call sites
// that already use it.
func copyUintSlice(src []uint) []uint {
	return cloneSlice(src)
}
