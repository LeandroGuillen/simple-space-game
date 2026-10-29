package game

import (
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOmniscientPlayerSeesAllStars(t *testing.T) {
	g := newGameWithRaceTraits(t, []string{"omniscient"})
	require.GreaterOrEqual(t, len(g.Galaxy.Stars), 2)

	player := g.Players[0]
	assert.Len(t, player.ExploredStars, 1)

	visible, err := g.GetVisibleGameForPlayer(player.ID)
	require.NoError(t, err)

	named := exploredStarIDs(visible)
	assert.Len(t, named, len(g.Galaxy.Stars))
}

func TestNonOmniscientPlayerOnlySeesExploredStars(t *testing.T) {
	g := newGameWithRaceTraits(t, []string{"creative"})
	require.GreaterOrEqual(t, len(g.Galaxy.Stars), 2)

	player := g.Players[0]
	visible, err := g.GetVisibleGameForPlayer(player.ID)
	require.NoError(t, err)

	assert.Len(t, exploredStarIDs(visible), 1)
}

func TestOmniscientSeesEnemyColoniesWithoutViewerHydration(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	omniscient, err := BuildCustomRace(resources, CustomRaceDesign{Name: "Seers", TraitIDs: []string{"omniscient"}})
	require.NoError(t, err)
	g, err := NewGame(resources, NewGameOptions{
		PlayerName:  "Ada",
		PlayerColor: model.DefaultPlayerColor(),
		NumStars:    4,
		Race:        omniscient,
		NumPlayers:  2,
	})
	require.NoError(t, err)

	var enemyPlanet *model.Planet
	for _, star := range g.Galaxy.Stars {
		for _, planet := range star.Planets {
			if planet.Owner == 2 {
				enemyPlanet = planet
				break
			}
		}
	}
	require.NotNil(t, enemyPlanet)

	visible, err := g.GetVisibleGameForPlayer(1)
	require.NoError(t, err)

	var visibleEnemy *model.Planet
	for _, star := range visible.Galaxy.Stars {
		require.NotEmpty(t, star.Name)
		for _, planet := range star.Planets {
			if planet.ID == enemyPlanet.ID {
				visibleEnemy = planet
				break
			}
		}
	}
	require.NotNil(t, visibleEnemy)
	assert.Equal(t, uint(2), visibleEnemy.Owner)
	assert.Equal(t, enemyPlanet.Name, visibleEnemy.Name)
	assert.Equal(t, enemyPlanet.Government.Name, visibleEnemy.Government.Name)

	var ownPlanet *model.Planet
	for _, star := range visible.Galaxy.Stars {
		for _, planet := range star.Planets {
			if planet.Owner == 1 {
				ownPlanet = planet
				break
			}
		}
	}
	require.NotNil(t, ownPlanet)
	assert.NotZero(t, ownPlanet.Income.Total)
	assert.Equal(t, g.Players[0].Race.Government, ownPlanet.Government.Name)
}

func TestVisibleGameStripsRivalPrivateState(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, NewGameOptions{
		PlayerName:  "Ada",
		PlayerColor: model.DefaultPlayerColor(),
		NumStars:    8,
		Race:        DefaultRace("Human"),
		NumPlayers:  2,
	})
	require.NoError(t, err)

	p1 := g.Players[0]
	p2 := g.Players[1]
	p1.Treasury = 123
	p1.ResearchPoints = 45
	p1.ResearchedTechs = append(p1.ResearchedTechs, "Anti-Missile Rockets")
	p2.Treasury = 999
	p2.ResearchPoints = 77
	p2.ResearchedTechs = append([]string{}, "Freighters", "Colony Ship")
	p2.CurrentResearch = &model.ResearchProject{TechName: "Nuclear Drive", Subject: "Nuclear Drive", LevelName: "Nuclear Fission", Progress: 10}
	p2.HyperAdvanced = map[string]uint64{"Construction": 2}
	p2.ExploredStars = append(p2.ExploredStars, 3, 4, 5)
	p2.CommandPointsCapacity = 50
	p2.CommandPointsUsed = 12
	p2.CommandPointsRemaining = 38
	p2.Freighters = 8
	p2.ShipDesigns = []model.ShipDesign{{Name: "Secret Scout"}}
	p2.Available.Buildings = append(p2.Available.Buildings, "Secret Lab")

	// Own fleet at home; enemy fleet at enemy home; enemy fleet at an unoccupied explored-looking star far away.
	p1Home := getPlayerHomeworld(g, 1)
	p2Home := getPlayerHomeworld(g, 2)
	require.NotNil(t, p1Home)
	require.NotNil(t, p2Home)

	var remoteStar uint
	for _, star := range g.Galaxy.Stars {
		if star.ID != p1Home.ParentStar && star.ID != p2Home.ParentStar {
			remoteStar = star.ID
			break
		}
	}
	require.NotZero(t, remoteStar)
	ownFleet := &model.Fleet{
		ID: g.nextFleetID(), Owner: 1, StarID: p1Home.ParentStar,
		Ships: []model.FleetShip{{Type: "ship_design", Name: "Own Scout"}},
	}
	g.Fleets = append(g.Fleets, ownFleet)
	enemyAtHome := &model.Fleet{
		ID: g.nextFleetID(), Owner: 2, StarID: p2Home.ParentStar,
		DestinationStarID: 99, TurnsRemaining: 4, TotalTurns: 5,
		OriginStarID: p2Home.ParentStar, OriginX: 1, OriginY: 2,
		Ships: []model.FleetShip{{Type: "ship_design", Name: "Enemy Raider"}},
	}
	g.Fleets = append(g.Fleets, enemyAtHome)
	enemyRemote := &model.Fleet{
		ID: g.nextFleetID(), Owner: 2, StarID: remoteStar,
		DestinationStarID: p1Home.ParentStar, TurnsRemaining: 2, TotalTurns: 3,
		Ships: []model.FleetShip{{Type: "ship_design", Name: "Hidden Fleet"}},
	}
	g.Fleets = append(g.Fleets, enemyRemote)

	// Battle involving only player 2 should be hidden from player 1.
	foreignBattle := &model.TacticalBattle{
		ID: 1, StarID: p2Home.ParentStar, Status: model.TacticalBattleActive,
		HumanOwnerIDs: []uint{2},
		Ships:         []*model.TacticalShip{{ID: 1, OwnerID: 2}},
	}
	ownBattle := &model.TacticalBattle{
		ID: 2, StarID: p1Home.ParentStar, Status: model.TacticalBattleActive,
		HumanOwnerIDs: []uint{1},
		Ships:         []*model.TacticalShip{{ID: 2, OwnerID: 1}, {ID: 3, OwnerID: 2}},
	}
	g.PendingBattles = []*model.TacticalBattle{foreignBattle, ownBattle}

	g.BattleReports = []*model.BattleReport{
		{ID: 1, StarID: p2Home.ParentStar, Sides: []model.BattleSide{{OwnerID: 2}}},
		{ID: 2, StarID: p1Home.ParentStar, Sides: []model.BattleSide{{OwnerID: 1}, {OwnerID: 2}}},
	}
	g.InvasionReports = []*model.InvasionReport{
		{ID: 1, AttackerOwnerID: 2, DefenderOwnerID: 2},
		{ID: 2, AttackerOwnerID: 1, DefenderOwnerID: 2},
	}
	g.PopulationTransfers = []model.PopulationTransfer{
		{ID: 1, Owner: 2, PopulationUnits: 5},
		{ID: 2, Owner: 1, PopulationUnits: 3},
	}
	g.PendingColonyBases = map[uint][]uint{1: {p1Home.ID}, 2: {p2Home.ID}}
	g.PendingColonyShips = map[uint][]uint{2: {enemyAtHome.ID}}

	visible, err := g.GetVisibleGameForPlayer(1)
	require.NoError(t, err)

	require.Len(t, visible.Players, 2)
	var ownView, rivalView *model.Player
	for _, p := range visible.Players {
		switch p.ID {
		case 1:
			ownView = p
		case 2:
			rivalView = p
		}
	}
	require.NotNil(t, ownView)
	require.NotNil(t, rivalView)

	assert.Equal(t, float64(123), ownView.Treasury)
	assert.Equal(t, uint64(45), ownView.ResearchPoints)
	assert.Contains(t, ownView.ResearchedTechs, "Anti-Missile Rockets")
	assert.NotEmpty(t, ownView.ExploredStars)

	assert.Equal(t, "Player 2", rivalView.Name)
	assert.Equal(t, p2.Color, rivalView.Color)
	assert.Equal(t, p2.Race.Name, rivalView.Race.Name)
	assert.Zero(t, rivalView.Treasury)
	assert.Zero(t, rivalView.ResearchPoints)
	assert.Nil(t, rivalView.CurrentResearch)
	assert.Empty(t, rivalView.ResearchedTechs)
	assert.Empty(t, rivalView.ExploredStars)
	assert.Nil(t, rivalView.HyperAdvanced)
	assert.Empty(t, rivalView.Available.Buildings)
	assert.Empty(t, rivalView.ShipDesigns)
	assert.Zero(t, rivalView.CommandPointsCapacity)
	assert.Zero(t, rivalView.CommandPointsRemaining)
	assert.Zero(t, rivalView.Freighters)

	// Live game must be unchanged.
	assert.Equal(t, float64(999), g.Players[1].Treasury)
	assert.Equal(t, uint64(77), g.Players[1].ResearchPoints)
	assert.NotEmpty(t, g.Players[1].ResearchedTechs)

	seenFleetIDs := map[uint]bool{}
	for _, fleet := range visible.Fleets {
		seenFleetIDs[fleet.ID] = true
		if fleet.Owner == 2 {
			assert.Equal(t, p1Home.ParentStar, fleet.StarID, "enemy fleets only at stars player 1 currently sees")
			assert.Zero(t, fleet.DestinationStarID)
			assert.Zero(t, fleet.TurnsRemaining)
			assert.Zero(t, fleet.TotalTurns)
			assert.Zero(t, fleet.OriginStarID)
		}
	}
	assert.True(t, seenFleetIDs[ownFleet.ID])
	assert.False(t, seenFleetIDs[enemyAtHome.ID], "enemy home fleet not visible without presence")
	assert.False(t, seenFleetIDs[enemyRemote.ID], "remote enemy fleet not visible")

	// Place an enemy fleet at player 1's star — should appear without orders.
	enemyAtP1 := &model.Fleet{
		ID: g.nextFleetID(), Owner: 2, StarID: p1Home.ParentStar,
		DestinationStarID: remoteStar, TurnsRemaining: 3, TotalTurns: 3,
		Ships: []model.FleetShip{{Type: "ship_design", Name: "Visitor"}},
	}
	g.Fleets = append(g.Fleets, enemyAtP1)
	visible2, err := g.GetVisibleGameForPlayer(1)
	require.NoError(t, err)
	var visitor *model.Fleet
	for _, fleet := range visible2.Fleets {
		if fleet.ID == enemyAtP1.ID {
			visitor = fleet
		}
	}
	require.NotNil(t, visitor)
	assert.Zero(t, visitor.DestinationStarID)
	assert.Zero(t, visitor.TurnsRemaining)
	assert.Equal(t, "Visitor", visitor.Ships[0].Name)

	require.Len(t, visible.PendingBattles, 1)
	assert.Equal(t, uint(2), visible.PendingBattles[0].ID)

	require.Len(t, visible.BattleReports, 1)
	assert.Equal(t, uint(2), visible.BattleReports[0].ID)
	require.Len(t, visible.InvasionReports, 1)
	assert.Equal(t, uint(2), visible.InvasionReports[0].ID)
	require.Len(t, visible.PopulationTransfers, 1)
	assert.Equal(t, uint(2), visible.PopulationTransfers[0].ID)
	assert.Equal(t, []uint{p1Home.ID}, visible.PendingColonyBases[1])
	assert.Empty(t, visible.PendingColonyBases[2])
	assert.Empty(t, visible.PendingColonyShips[2])
}

func TestVisibleGameOmniscientStillSeesStars(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)
	omniscient, err := BuildCustomRace(resources, CustomRaceDesign{Name: "Seers", TraitIDs: []string{"omniscient"}})
	require.NoError(t, err)
	g, err := NewGame(resources, NewGameOptions{
		PlayerName:  "Ada",
		PlayerColor: model.DefaultPlayerColor(),
		NumStars:    6,
		Race:        omniscient,
		NumPlayers:  2,
	})
	require.NoError(t, err)

	visible, err := g.GetVisibleGameForPlayer(1)
	require.NoError(t, err)
	assert.Len(t, exploredStarIDs(visible), len(g.Galaxy.Stars))

	// Rival private fields still stripped for omniscient viewers.
	var rival *model.Player
	for _, p := range visible.Players {
		if p.ID == 2 {
			rival = p
		}
	}
	require.NotNil(t, rival)
	g.Players[1].Treasury = 500
	visible, err = g.GetVisibleGameForPlayer(1)
	require.NoError(t, err)
	for _, p := range visible.Players {
		if p.ID == 2 {
			assert.Zero(t, p.Treasury)
		}
	}
}
