package game

import (
	"sort"
	. "ssg/model"
)

// maxBattleRounds bounds auto-resolve so under-armed fleets (e.g. designs with
// no beams installed yet) do not loop forever. A fight that reaches this cap
// without a kill is declared a draw and ships disperse intact.
const maxBattleRounds = 24

// battleStartRange is the opening squares-between distance used when applying
// beam dissipation. Picked at the middle of the dissipation table so beams are
// meaningfully but not punitively penalized in this first pass.
const battleStartRange int64 = 5

// resolveStarBattles runs auto-resolve combat at every star where two or more
// owners currently share orbit. Returns the new reports it produced.
func (g *Game) resolveStarBattles() []*BattleReport {
	if g == nil {
		return nil
	}
	byStar := map[uint][]*Fleet{}
	starIDs := []uint{}
	for _, fleet := range g.Fleets {
		if fleet == nil || fleet.DestinationStarID != 0 || fleet.TurnsRemaining != 0 {
			continue
		}
		if _, seen := byStar[fleet.StarID]; !seen {
			starIDs = append(starIDs, fleet.StarID)
		}
		byStar[fleet.StarID] = append(byStar[fleet.StarID], fleet)
	}
	sort.SliceStable(starIDs, func(i, j int) bool { return starIDs[i] < starIDs[j] })

	var reports []*BattleReport
	for _, starID := range starIDs {
		fleets := byStar[starID]
		if !multipleOwners(fleets) {
			continue
		}
		report := g.resolveBattle(starID, fleets)
		if report != nil {
			reports = append(reports, report)
		}
	}
	if len(reports) > 0 {
		g.BattleReports = append(g.BattleReports, reports...)
	}
	return reports
}

func multipleOwners(fleets []*Fleet) bool {
	seen := uint(0)
	for _, fleet := range fleets {
		if seen == 0 {
			seen = fleet.Owner
			continue
		}
		if fleet.Owner != seen {
			return true
		}
	}
	return false
}

// battleShip pairs a battle-state Ship with the FleetShip it came from so we
// can reconcile survivors back into the original fleets after combat.
type battleShip struct {
	ownerID   uint
	fleetID   uint
	shipIndex int
	original  FleetShip
	ship      *Ship
}

// battleSide groups one player's ships and supporting context for combat.
type battleSide struct {
	ownerID uint
	player  *Player
	ships   []*battleShip
}

func (g *Game) resolveBattle(starID uint, fleets []*Fleet) *BattleReport {
	sidesByOwner := map[uint]*battleSide{}
	ownerOrder := []uint{}
	originalShipsByOwner := map[uint][]FleetShip{}

	for _, fleet := range fleets {
		player, _ := g.getPlayer(fleet.Owner)
		side, ok := sidesByOwner[fleet.Owner]
		if !ok {
			side = &battleSide{ownerID: fleet.Owner, player: player}
			sidesByOwner[fleet.Owner] = side
			ownerOrder = append(ownerOrder, fleet.Owner)
		}
		for i, fs := range fleet.Ships {
			ship := materializeBattleShip(g, player, fs)
			side.ships = append(side.ships, &battleShip{
				ownerID:   fleet.Owner,
				fleetID:   fleet.ID,
				shipIndex: i,
				original:  fs,
				ship:      ship,
			})
			originalShipsByOwner[fleet.Owner] = append(originalShipsByOwner[fleet.Owner], fs)
		}
	}
	sort.SliceStable(ownerOrder, func(i, j int) bool { return ownerOrder[i] < ownerOrder[j] })

	sides := make([]*battleSide, 0, len(ownerOrder))
	for _, id := range ownerOrder {
		sides = append(sides, sidesByOwner[id])
	}

	rounds, winnerOwnerID, draw := runBattle(sides)

	star, _ := g.getStar(starID)
	report := &BattleReport{
		ID:            g.nextBattleReportID(),
		Turn:          g.CurrentTurn,
		StarID:        starID,
		WinnerOwnerID: winnerOwnerID,
		Rounds:        rounds,
		Draw:          draw,
	}
	if star != nil {
		report.X = float64(star.X)
		report.Y = float64(star.Y)
	}

	for _, side := range sides {
		survivors := []FleetShip{}
		destroyed := 0
		for _, bs := range side.ships {
			if shipAlive(bs.ship) {
				survivors = append(survivors, bs.original)
			} else {
				destroyed++
			}
		}
		report.Sides = append(report.Sides, BattleSide{
			OwnerID:        side.ownerID,
			OriginalShips:  originalShipsByOwner[side.ownerID],
			SurvivingShips: survivors,
			DestroyedCount: destroyed,
		})
	}

	g.applyBattleSurvivors(starID, fleets, sides)
	return report
}

// runBattle alternates fire between sides until only one side has surviving
// ships or maxBattleRounds is reached. Returns the round count, winner owner
// (zero if a draw), and a draw flag.
func runBattle(sides []*battleSide) (int, uint, bool) {
	if len(sides) < 2 {
		if len(sides) == 1 {
			return 0, sides[0].ownerID, false
		}
		return 0, 0, true
	}
	for round := 1; round <= maxBattleRounds; round++ {
		for i, attackerSide := range sides {
			defenders := otherSidesAlive(sides, i)
			if len(defenders) == 0 {
				return round, attackerSide.ownerID, false
			}
			for _, bs := range attackerSide.ships {
				if !shipAlive(bs.ship) {
					continue
				}
				target := pickTarget(defenders)
				if target == nil {
					break
				}
				FireOn(bs.ship, target.ship, battleStartRange, attackerSide.player, sideOf(sides, target.ownerID).player)
			}
		}
		if aliveSides := countSidesWithLiveShips(sides); aliveSides <= 1 {
			for _, side := range sides {
				if sideHasLiveShips(side) {
					return round, side.ownerID, false
				}
			}
			return round, 0, true
		}
	}
	return maxBattleRounds, 0, true
}

func otherSidesAlive(sides []*battleSide, attackerIdx int) []*battleShip {
	var live []*battleShip
	for i, side := range sides {
		if i == attackerIdx {
			continue
		}
		for _, bs := range side.ships {
			if shipAlive(bs.ship) {
				live = append(live, bs)
			}
		}
	}
	return live
}

func pickTarget(defenders []*battleShip) *battleShip {
	if len(defenders) == 0 {
		return nil
	}
	return defenders[randomIntn(len(defenders))]
}

func sideOf(sides []*battleSide, ownerID uint) *battleSide {
	for _, side := range sides {
		if side.ownerID == ownerID {
			return side
		}
	}
	return nil
}

func sideHasLiveShips(side *battleSide) bool {
	for _, bs := range side.ships {
		if shipAlive(bs.ship) {
			return true
		}
	}
	return false
}

func countSidesWithLiveShips(sides []*battleSide) int {
	count := 0
	for _, side := range sides {
		if sideHasLiveShips(side) {
			count++
		}
	}
	return count
}

func shipAlive(s *Ship) bool {
	if s == nil {
		return false
	}
	return s.CurrentStructureHP > 0
}

// materializeBattleShip turns a FleetShip into a battle-ready Ship. Ships built
// from a design use that design's parts; FleetShips without a design (colony
// ships, freighters) get a minimal unarmed shell so they can be targeted and
// destroyed but contribute no damage.
func materializeBattleShip(g *Game, player *Player, fs FleetShip) *Ship {
	if fs.DesignName != "" && g != nil {
		if design, ok := g.findShipDesignForFleetShip(player, fs); ok {
			ship := NewShip(fs.Name, design)
			return &ship
		}
	}
	ship := NewEmptyBattleShip(fs.Name)
	// Civilian ships are fragile — small hull, no weapons, easy to pop.
	ship.CurrentStructureHP = 10
	ship.MaxStructureHP = 10
	ship.CurrentArmorHP = 0
	ship.MaxArmorHP = 0
	return &ship
}

func (g *Game) applyBattleSurvivors(starID uint, fleets []*Fleet, sides []*battleSide) {
	// Index surviving FleetShip references by (fleetID, shipIndex) so we can
	// preserve which fleet a ship belonged to without disturbing the rest of
	// the world.
	survived := map[uint]map[int]FleetShip{}
	for _, side := range sides {
		for _, bs := range side.ships {
			if !shipAlive(bs.ship) {
				continue
			}
			byFleet, ok := survived[bs.fleetID]
			if !ok {
				byFleet = map[int]FleetShip{}
				survived[bs.fleetID] = byFleet
			}
			byFleet[bs.shipIndex] = bs.original
		}
	}

	for _, fleet := range fleets {
		surv := survived[fleet.ID]
		if len(surv) == 0 {
			g.removeFleet(fleet)
			continue
		}
		kept := make([]FleetShip, 0, len(surv))
		for i := range fleet.Ships {
			if ship, ok := surv[i]; ok {
				kept = append(kept, ship)
			}
		}
		fleet.Ships = kept
	}
	_ = starID
}

func (g *Game) nextBattleReportID() uint {
	max := uint(0)
	for _, report := range g.BattleReports {
		if report != nil && report.ID > max {
			max = report.ID
		}
	}
	return max + 1
}
