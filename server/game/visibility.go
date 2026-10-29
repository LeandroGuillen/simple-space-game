package game

import (
	"ssg/model"
)

func (g *Game) ExploreStar(playerID, starID uint) error {
	player, err := g.getPlayer(playerID)
	if err != nil {
		return err
	}
	if _, err := g.getStar(starID); err != nil {
		return err
	}
	if hasExploredStar(player.ExploredStars, starID) {
		return nil
	}
	player.ExploredStars = append(player.ExploredStars, starID)
	return nil
}

func (g *Game) GetVisibleGameForPlayer(playerID uint) (*Game, error) {
	player, err := g.getPlayer(playerID)
	if err != nil {
		return nil, err
	}

	g.HydrateCommandPoints()

	visible := &Game{
		ID:          g.ID,
		CurrentTurn: g.CurrentTurn,
		CreatedAt:   g.CreatedAt,
		LastTurnAt:  g.LastTurnAt,
		Resources:   g.Resources,
		TurnStatus:  g.TurnStatus,
		// Do not share submission map or mutex with the live game.
		TurnSubmissions: nil,
		mu:              nil,
	}

	visible.Galaxy = model.Galaxy{
		Width:  g.Galaxy.Width,
		Height: g.Galaxy.Height,
		Stars:  make([]*model.Star, 0, len(g.Galaxy.Stars)),
	}
	for _, star := range g.Galaxy.Stars {
		starCopy := &model.Star{
			ID:    star.ID,
			Color: star.Color,
			X:     star.X,
			Y:     star.Y,
		}
		if isStarVisibleToPlayer(player, star.ID) {
			starCopy.Name = star.Name
			starCopy.Planets = make([]*model.Planet, 0, len(star.Planets))
			for _, planet := range star.Planets {
				planetCopy := *planet
				if planet.Owner != 0 && planet.Owner == playerID {
					hydratePlanetGovernment(g, &planetCopy, player)
					hydratePlanetMorale(g, &planetCopy, player)
					hydratePlanetOutputs(g, &planetCopy)
					hydratePlanetPopulationCapacity(g, &planetCopy, player)
					hydratePlanetIncome(g, &planetCopy, GamePlanetIndustryProduced(g, &planetCopy))
				}
				starCopy.Planets = append(starCopy.Planets, &planetCopy)
			}
		}
		visible.Galaxy.Stars = append(visible.Galaxy.Stars, starCopy)
	}

	visible.Players = make([]*model.Player, 0, len(g.Players))
	for _, p := range g.Players {
		if p == nil {
			continue
		}
		if p.ID == playerID {
			visible.Players = append(visible.Players, copyPlayerPrivate(p))
		} else {
			visible.Players = append(visible.Players, publicPlayerView(p))
		}
	}

	seenStars := g.starsPlayerCurrentlySees(playerID)
	visible.Fleets = make([]*model.Fleet, 0, len(g.Fleets))
	for _, fleet := range g.Fleets {
		if fleet == nil {
			continue
		}
		if fleet.Owner == playerID {
			visible.Fleets = append(visible.Fleets, copyFleet(fleet))
			continue
		}
		if !seenStars[fleet.StarID] {
			continue
		}
		visible.Fleets = append(visible.Fleets, sanitizeEnemyFleet(fleet))
	}

	visible.PendingBattles = filterPendingBattlesForPlayer(g.PendingBattles, playerID)
	visible.BattleReports = filterBattleReportsForPlayer(g.BattleReports, playerID)
	visible.InvasionReports = filterInvasionReportsForPlayer(g.InvasionReports, playerID)
	visible.PopulationTransfers = filterPopulationTransfersForPlayer(g.PopulationTransfers, playerID)
	visible.PendingColonyBases = filterPendingMapForPlayer(g.PendingColonyBases, playerID)
	visible.PendingColonyShips = filterPendingMapForPlayer(g.PendingColonyShips, playerID)
	visible.PendingArtificialPlanets = filterPendingMapForPlayer(g.PendingArtificialPlanets, playerID)

	return visible, nil
}

// starsPlayerCurrentlySees returns stars where the player has a colony or an
// orbiting fleet. Scanner-range buildings exist in data but are not wired into
// runtime detection yet.
func (g *Game) starsPlayerCurrentlySees(playerID uint) map[uint]bool {
	seen := make(map[uint]bool)
	for _, star := range g.Galaxy.Stars {
		if star == nil {
			continue
		}
		for _, planet := range star.Planets {
			if planet != nil && planet.Owner == playerID {
				seen[star.ID] = true
				break
			}
		}
	}
	for _, fleet := range g.Fleets {
		if fleet == nil || fleet.Owner != playerID {
			continue
		}
		if fleet.DestinationStarID != 0 || fleet.TurnsRemaining != 0 {
			continue
		}
		seen[fleet.StarID] = true
	}
	return seen
}

func copyPlayerPrivate(p *model.Player) *model.Player {
	cp := *p
	if p.CurrentResearch != nil {
		research := *p.CurrentResearch
		research.TechNames = append([]string(nil), p.CurrentResearch.TechNames...)
		cp.CurrentResearch = &research
	}
	cp.ResearchedTechs = append([]string(nil), p.ResearchedTechs...)
	if p.HyperAdvanced != nil {
		cp.HyperAdvanced = make(map[string]uint64, len(p.HyperAdvanced))
		for k, v := range p.HyperAdvanced {
			cp.HyperAdvanced[k] = v
		}
	}
	cp.Available = model.AvailableTech{
		Buildings:   append([]string(nil), p.Available.Buildings...),
		HullSizes:   append([]string(nil), p.Available.HullSizes...),
		Drives:      append([]string(nil), p.Available.Drives...),
		Armors:      append([]string(nil), p.Available.Armors...),
		Computers:   append([]string(nil), p.Available.Computers...),
		FuelCells:   append([]string(nil), p.Available.FuelCells...),
		Shields:     append([]string(nil), p.Available.Shields...),
		Beams:       append([]string(nil), p.Available.Beams...),
		ShipSystems: append([]string(nil), p.Available.ShipSystems...),
	}
	cp.Fleets = append([]uint(nil), p.Fleets...)
	cp.ShipDesigns = append([]model.ShipDesign(nil), p.ShipDesigns...)
	cp.ExploredStars = append([]uint(nil), p.ExploredStars...)
	cp.Race = p.Race
	cp.Race.TraitIDs = append([]string(nil), p.Race.TraitIDs...)
	return &cp
}

// publicPlayerView keeps only identity / diplomacy fields for rivals.
func publicPlayerView(p *model.Player) *model.Player {
	return &model.Player{
		ID:      p.ID,
		Name:    p.Name,
		Color:   p.Color,
		IsHuman: p.IsHuman,
		Race: model.Race{
			Name:       p.Race.Name,
			Government: p.Race.Government,
			TraitIDs:   append([]string(nil), p.Race.TraitIDs...),
		},
	}
}

func copyFleet(f *model.Fleet) *model.Fleet {
	cp := *f
	cp.Ships = append([]model.FleetShip(nil), f.Ships...)
	return &cp
}

// sanitizeEnemyFleet exposes presence/composition at a visible star without
// destination or movement orders.
func sanitizeEnemyFleet(f *model.Fleet) *model.Fleet {
	return &model.Fleet{
		ID:     f.ID,
		Owner:  f.Owner,
		StarID: f.StarID,
		X:      f.X,
		Y:      f.Y,
		Ships:  append([]model.FleetShip(nil), f.Ships...),
	}
}

func filterPendingBattlesForPlayer(battles []*model.TacticalBattle, playerID uint) []*model.TacticalBattle {
	if len(battles) == 0 {
		return nil
	}
	out := make([]*model.TacticalBattle, 0, len(battles))
	for _, battle := range battles {
		if battle == nil || !tacticalBattleInvolvesPlayer(battle, playerID) {
			continue
		}
		out = append(out, battle)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func tacticalBattleInvolvesPlayer(battle *model.TacticalBattle, playerID uint) bool {
	if battle.IsHumanOwner(playerID) {
		return true
	}
	for _, ship := range battle.Ships {
		if ship != nil && ship.OwnerID == playerID {
			return true
		}
	}
	return false
}

func filterBattleReportsForPlayer(reports []*model.BattleReport, playerID uint) []*model.BattleReport {
	if len(reports) == 0 {
		return nil
	}
	out := make([]*model.BattleReport, 0, len(reports))
	for _, report := range reports {
		if report == nil {
			continue
		}
		for _, side := range report.Sides {
			if side.OwnerID == playerID {
				out = append(out, report)
				break
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func filterInvasionReportsForPlayer(reports []*model.InvasionReport, playerID uint) []*model.InvasionReport {
	if len(reports) == 0 {
		return nil
	}
	out := make([]*model.InvasionReport, 0, len(reports))
	for _, report := range reports {
		if report == nil {
			continue
		}
		if report.AttackerOwnerID == playerID || report.DefenderOwnerID == playerID {
			out = append(out, report)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func filterPopulationTransfersForPlayer(transfers []model.PopulationTransfer, playerID uint) []model.PopulationTransfer {
	if len(transfers) == 0 {
		return nil
	}
	out := make([]model.PopulationTransfer, 0, len(transfers))
	for _, transfer := range transfers {
		if transfer.Owner == playerID {
			out = append(out, transfer)
		}
	}
	return out
}

func filterPendingMapForPlayer(pending map[uint][]uint, playerID uint) map[uint][]uint {
	if pending == nil {
		return nil
	}
	own, ok := pending[playerID]
	if !ok || len(own) == 0 {
		return map[uint][]uint{}
	}
	return map[uint][]uint{playerID: append([]uint(nil), own...)}
}

func isStarVisibleToPlayer(player *model.Player, starID uint) bool {
	if playerHasRaceTrait(player, "omniscient") {
		return true
	}
	return hasExploredStar(player.ExploredStars, starID)
}

func hasExploredStar(explored []uint, starID uint) bool {
	for _, id := range explored {
		if id == starID {
			return true
		}
	}
	return false
}
