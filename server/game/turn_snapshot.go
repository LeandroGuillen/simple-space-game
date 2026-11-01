package game

import (
	"ssg/model"
	"time"
)

// gameTurnSnapshot captures every mutable Game field that AdvanceTurn
// (applyPlayerTurn / planetTurn / autoResolveLeftoverBattles / etc.) may change.
// Resources and mu are intentionally omitted: Resources is a shared immutable
// catalog; mu must stay the live lock on the same *Game so external holders of
// *Game keep a valid pointer after restore.
type gameTurnSnapshot struct {
	Galaxy                   model.Galaxy
	Players                  []*model.Player
	Fleets                   []*model.Fleet
	PopulationTransfers      []model.PopulationTransfer
	TransferLosses           []model.TransferLossNotice
	TurnEvents               []model.TurnEvent
	ID                       string
	CurrentTurn              float64
	Status                   string
	WinnerID                 uint
	TurnsPlayed              uint
	CreatedAt                time.Time
	LastTurnAt               *time.Time
	PendingColonyBases       map[uint][]uint
	PendingColonyShips       map[uint][]uint
	PendingArtificialPlanets map[uint][]uint
	BattleReports            []*model.BattleReport
	PendingBattles           []*model.TacticalBattle
	InvasionReports          []*model.InvasionReport
	PendingInvasions         []PendingInvasion
	PostponedInvasions       []PendingInvasion
	TurnSubmissions          map[uint]model.Commands
	TurnStatus               TurnStatus
	Debug                    bool
}

// gameTurnSnapshotFieldNames lists every Game field the snapshot must cover
// (except Resources and mu). The reflect guard test fails if Game gains a
// field that is not listed here or in the intentionally-shared set.
var gameTurnSnapshotFieldNames = []string{
	"Galaxy",
	"Players",
	"Fleets",
	"PopulationTransfers",
	"TransferLosses",
	"TurnEvents",
	"ID",
	"CurrentTurn",
	"Status",
	"WinnerID",
	"TurnsPlayed",
	"CreatedAt",
	"LastTurnAt",
	"PendingColonyBases",
	"PendingColonyShips",
	"PendingArtificialPlanets",
	"BattleReports",
	"PendingBattles",
	"InvasionReports",
	"PendingInvasions",
	"PostponedInvasions",
	"TurnSubmissions",
	"TurnStatus",
	"Debug",
}

var gameTurnSnapshotSharedFieldNames = []string{
	"Resources",
	"mu",
}

func snapshotGameState(g *Game) *gameTurnSnapshot {
	if g == nil {
		return nil
	}
	snap := &gameTurnSnapshot{
		ID:          g.ID,
		CurrentTurn: g.CurrentTurn,
		Status:      g.Status,
		WinnerID:    g.WinnerID,
		TurnsPlayed: g.TurnsPlayed,
		CreatedAt:   g.CreatedAt,
		Debug:       g.Debug,
		TurnStatus:  copyTurnStatus(g.TurnStatus),
	}
	if g.LastTurnAt != nil {
		t := *g.LastTurnAt
		snap.LastTurnAt = &t
	}
	snap.Galaxy = copyGalaxy(g.Galaxy)
	snap.Players = copyPlayers(g.Players)
	snap.Fleets = copyFleets(g.Fleets)
	snap.PopulationTransfers = copyPopulationTransfers(g.PopulationTransfers)
	snap.TransferLosses = cloneSlice(g.TransferLosses)
	snap.TurnEvents = cloneSlice(g.TurnEvents)
	snap.PendingColonyBases = copyPendingUintMap(g.PendingColonyBases)
	snap.PendingColonyShips = copyPendingUintMap(g.PendingColonyShips)
	snap.PendingArtificialPlanets = copyPendingUintMap(g.PendingArtificialPlanets)
	// Reports are append-only during turns: retain their original slice headers.
	// Rollback restores their original lengths (and nil-versus-empty shape).
	snap.BattleReports = g.BattleReports
	snap.PendingBattles = copyPendingBattles(g.PendingBattles)
	snap.InvasionReports = g.InvasionReports
	snap.PendingInvasions = copyPendingInvasions(g.PendingInvasions)
	snap.PostponedInvasions = copyPendingInvasions(g.PostponedInvasions)
	snap.TurnSubmissions = copyTurnSubmissions(g.TurnSubmissions)
	return snap
}

// restoreGameState writes snap back into the same *Game (preserving g.mu and
// the shared Resources pointer).
func restoreGameState(g *Game, snap *gameTurnSnapshot) {
	if g == nil || snap == nil {
		return
	}
	g.Galaxy = snap.Galaxy
	g.Players = snap.Players
	g.Fleets = snap.Fleets
	g.PopulationTransfers = snap.PopulationTransfers
	g.TransferLosses = snap.TransferLosses
	g.TurnEvents = snap.TurnEvents
	g.ID = snap.ID
	g.CurrentTurn = snap.CurrentTurn
	g.Status = snap.Status
	g.WinnerID = snap.WinnerID
	g.TurnsPlayed = snap.TurnsPlayed
	g.CreatedAt = snap.CreatedAt
	g.LastTurnAt = snap.LastTurnAt
	g.PendingColonyBases = snap.PendingColonyBases
	g.PendingColonyShips = snap.PendingColonyShips
	g.PendingArtificialPlanets = snap.PendingArtificialPlanets
	g.BattleReports = snap.BattleReports
	g.PendingBattles = snap.PendingBattles
	g.InvasionReports = snap.InvasionReports
	g.PendingInvasions = snap.PendingInvasions
	g.PostponedInvasions = snap.PostponedInvasions
	g.TurnSubmissions = snap.TurnSubmissions
	g.TurnStatus = snap.TurnStatus
	g.Debug = snap.Debug
}

func copyGalaxy(galaxy model.Galaxy) model.Galaxy {
	out := model.Galaxy{
		Width:  galaxy.Width,
		Height: galaxy.Height,
	}
	if galaxy.Stars == nil {
		return out
	}
	out.Stars = make([]*model.Star, len(galaxy.Stars))
	for i, star := range galaxy.Stars {
		out.Stars[i] = copyStar(star)
	}
	return out
}

func copyStar(star *model.Star) *model.Star {
	if star == nil {
		return nil
	}
	cp := *star
	if star.Planets == nil {
		cp.Planets = nil
	} else {
		cp.Planets = make([]*model.Planet, len(star.Planets))
		for i, planet := range star.Planets {
			cp.Planets[i] = copyPlanet(planet)
		}
	}
	return &cp
}

func copyPlayers(players []*model.Player) []*model.Player {
	if players == nil {
		return nil
	}
	out := make([]*model.Player, len(players))
	for i, p := range players {
		if p == nil {
			continue
		}
		out[i] = copyPlayerPrivate(p)
	}
	return out
}

func copyFleets(fleets []*model.Fleet) []*model.Fleet {
	if fleets == nil {
		return nil
	}
	out := make([]*model.Fleet, len(fleets))
	for i, f := range fleets {
		if f == nil {
			continue
		}
		out[i] = copyFleet(f)
	}
	return out
}

func copyPopulationTransfers(src []model.PopulationTransfer) []model.PopulationTransfer {
	return cloneSlice(src)
}

func copyPendingUintMap(src map[uint][]uint) map[uint][]uint {
	if src == nil {
		return nil
	}
	out := make(map[uint][]uint, len(src))
	for k, v := range src {
		out[k] = cloneSlice(v)
	}
	return out
}

func copyBattleReports(reports []*model.BattleReport) []*model.BattleReport {
	if reports == nil {
		return nil
	}
	out := make([]*model.BattleReport, len(reports))
	for i, r := range reports {
		out[i] = copyBattleReport(r)
	}
	return out
}

func copyPendingBattles(battles []*model.TacticalBattle) []*model.TacticalBattle {
	if battles == nil {
		return nil
	}
	out := make([]*model.TacticalBattle, len(battles))
	for i, b := range battles {
		out[i] = copyTacticalBattle(b)
	}
	return out
}

func copyInvasionReports(reports []*model.InvasionReport) []*model.InvasionReport {
	if reports == nil {
		return nil
	}
	out := make([]*model.InvasionReport, len(reports))
	for i, r := range reports {
		out[i] = copyInvasionReport(r)
	}
	return out
}

func copyPendingInvasions(orders []PendingInvasion) []PendingInvasion {
	if orders == nil {
		return nil
	}
	out := make([]PendingInvasion, len(orders))
	for i, order := range orders {
		out[i] = PendingInvasion{
			OwnerID:        order.OwnerID,
			FleetID:        order.FleetID,
			TargetPlanetID: order.TargetPlanetID,
			ShipIndexes:    copyUintIndexList(order.ShipIndexes),
		}
	}
	return out
}

func copyTurnSubmissions(src map[uint]model.Commands) map[uint]model.Commands {
	if src == nil {
		return nil
	}
	out := make(map[uint]model.Commands, len(src))
	for id, cmds := range src {
		out[id] = copyCommands(cmds)
	}
	return out
}

func copyCommands(cmds model.Commands) model.Commands {
	out := model.Commands{}
	if cmds.Build != nil {
		out.Build = make([]*model.BuildCommand, len(cmds.Build))
		for i, c := range cmds.Build {
			if c == nil {
				continue
			}
			cp := *c
			cp.BuildQueue = cloneSlice(c.BuildQueue)
			out.Build[i] = &cp
		}
	}
	if cmds.MovePopulation != nil {
		out.MovePopulation = make([]*model.MovePopulationCommand, len(cmds.MovePopulation))
		for i, c := range cmds.MovePopulation {
			if c == nil {
				continue
			}
			cp := *c
			out.MovePopulation[i] = &cp
		}
	}
	if cmds.TransferPopulation != nil {
		out.TransferPopulation = make([]*model.TransferPopulationCommand, len(cmds.TransferPopulation))
		for i, c := range cmds.TransferPopulation {
			if c == nil {
				continue
			}
			cp := *c
			out.TransferPopulation[i] = &cp
		}
	}
	if cmds.SelectResearch != nil {
		cp := *cmds.SelectResearch
		out.SelectResearch = &cp
	}
	if cmds.Colonize != nil {
		cp := *cmds.Colonize
		out.Colonize = &cp
	}
	if cmds.ColonizeMany != nil {
		out.ColonizeMany = make([]*model.ColonizeCommand, len(cmds.ColonizeMany))
		for i, c := range cmds.ColonizeMany {
			if c == nil {
				continue
			}
			cp := *c
			out.ColonizeMany[i] = &cp
		}
	}
	if cmds.ArtificialPlanet != nil {
		cp := *cmds.ArtificialPlanet
		out.ArtificialPlanet = &cp
	}
	if cmds.ArtificialPlanetMany != nil {
		out.ArtificialPlanetMany = make([]*model.ArtificialPlanetCommand, len(cmds.ArtificialPlanetMany))
		for i, c := range cmds.ArtificialPlanetMany {
			if c == nil {
				continue
			}
			cp := *c
			out.ArtificialPlanetMany[i] = &cp
		}
	}
	if cmds.ColonyShip != nil {
		cp := *cmds.ColonyShip
		out.ColonyShip = &cp
	}
	if cmds.FleetMove != nil {
		out.FleetMove = make([]*model.FleetMoveCommand, len(cmds.FleetMove))
		for i, c := range cmds.FleetMove {
			if c == nil {
				continue
			}
			cp := *c
			cp.ShipIndexes = copyUintIndexList(c.ShipIndexes)
			out.FleetMove[i] = &cp
		}
	}
	if cmds.Invade != nil {
		out.Invade = make([]*model.InvadeCommand, len(cmds.Invade))
		for i, c := range cmds.Invade {
			if c == nil {
				continue
			}
			cp := *c
			cp.ShipIndexes = copyUintIndexList(c.ShipIndexes)
			out.Invade[i] = &cp
		}
	}
	if cmds.BuyProduction != nil {
		out.BuyProduction = make([]*model.BuyProductionCommand, len(cmds.BuyProduction))
		for i, c := range cmds.BuyProduction {
			if c == nil {
				continue
			}
			cp := *c
			out.BuyProduction[i] = &cp
		}
	}
	if cmds.DesignShip != nil {
		cp := *cmds.DesignShip
		out.DesignShip = &cp
	}
	if cmds.MutateRace != nil {
		cp := *cmds.MutateRace
		cp.TraitIDs = cloneSlice(cmds.MutateRace.TraitIDs)
		out.MutateRace = &cp
	}
	return out
}
