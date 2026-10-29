package game

import (
	"ssg/model"
)

func copyPlanet(p *model.Planet) *model.Planet {
	if p == nil {
		return nil
	}
	cp := *p
	cp.Buildings = append([]string(nil), p.Buildings...)
	cp.BuildQueue = append([]string(nil), p.BuildQueue...)
	if p.Morale.Sources != nil {
		cp.Morale.Sources = append([]model.PlanetMoraleSource(nil), p.Morale.Sources...)
	}
	return &cp
}

func copyShipDesign(d model.ShipDesign) model.ShipDesign {
	cp := d
	cp.Beams = append([]model.Beam(nil), d.Beams...)
	cp.SpecialSystems = append([]model.SpecialSystem(nil), d.SpecialSystems...)
	return cp
}

func copyTacticalShip(s *model.TacticalShip) *model.TacticalShip {
	if s == nil {
		return nil
	}
	cp := *s
	cp.Beams = append([]model.Beam(nil), s.Beams...)
	return &cp
}

func copyTacticalBattle(b *model.TacticalBattle) *model.TacticalBattle {
	if b == nil {
		return nil
	}
	cp := *b
	if b.Ships != nil {
		cp.Ships = make([]*model.TacticalShip, len(b.Ships))
		for i, ship := range b.Ships {
			cp.Ships[i] = copyTacticalShip(ship)
		}
	}
	cp.TurnOrder = append([]uint(nil), b.TurnOrder...)
	cp.HumanOwnerIDs = append([]uint(nil), b.HumanOwnerIDs...)
	cp.Events = append([]model.BattleEvent(nil), b.Events...)
	return &cp
}

func copyBattleSide(side model.BattleSide) model.BattleSide {
	cp := side
	cp.OriginalShips = append([]model.FleetShip(nil), side.OriginalShips...)
	cp.SurvivingShips = append([]model.FleetShip(nil), side.SurvivingShips...)
	return cp
}

func copyBattleReport(r *model.BattleReport) *model.BattleReport {
	if r == nil {
		return nil
	}
	cp := *r
	if r.Sides != nil {
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
	cp.Rounds = append([]model.GroundBattleRound(nil), r.Rounds...)
	cp.InitialAttacker = copyGroundForce(r.InitialAttacker)
	cp.InitialDefender = copyGroundForce(r.InitialDefender)
	cp.SurvivingUnits = copyGroundForce(r.SurvivingUnits)
	return &cp
}

func copyTurnStatus(ts TurnStatus) TurnStatus {
	return TurnStatus{
		Mode:              ts.Mode,
		ReadyPlayerIDs:    append([]uint(nil), ts.ReadyPlayerIDs...),
		WaitingPlayerIDs:  append([]uint(nil), ts.WaitingPlayerIDs...),
		RequiredPlayerIDs: append([]uint(nil), ts.RequiredPlayerIDs...),
	}
}
