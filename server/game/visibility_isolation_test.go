package game

import (
	"bytes"
	"encoding/json"
	"sync"
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTwoPlayerIsolationGame(t *testing.T) *Game {
	t.Helper()
	resources, err := NewResourceManager()
	require.NoError(t, err)
	g, err := NewGame(resources, NewGameOptions{
		PlayerName:  "Ada",
		PlayerColor: model.DefaultPlayerColor(),
		NumStars:    6,
		Race:        DefaultRace("Human"),
		NumPlayers:  2,
	})
	require.NoError(t, err)

	// Seed mutable nested state so isolation checks have something to poke.
	player := g.Players[0]
	require.NotNil(t, player)
	player.Available.Buildings = append(player.Available.Buildings, "IsolationProbe")
	player.ShipDesigns = append(player.ShipDesigns, model.ShipDesign{
		Name:  "Isolation Frigate",
		Beams: []model.Beam{{Name: "Laser"}},
	})
	player.HyperAdvanced = map[string]uint64{"Physics": 1}

	for _, star := range g.Galaxy.Stars {
		for _, planet := range star.Planets {
			if planet != nil && planet.Owner == player.ID {
				planet.Buildings = append(planet.Buildings, "Isolation Hab")
				planet.BuildQueue = append([]string{"Isolation Queue"}, planet.BuildQueue...)
				planet.Morale.Sources = append(planet.Morale.Sources, model.PlanetMoraleSource{
					Name:  "Isolation Mood",
					Type:  "test",
					Bonus: 0.1,
				})
			}
		}
	}

	g.Fleets = append(g.Fleets, &model.Fleet{
		ID:     9001,
		Owner:  player.ID,
		StarID: player.ExploredStars[0],
		Ships:  []model.FleetShip{{Type: "colony_ship", Name: "Seed"}},
	})
	g.PendingColonyBases = map[uint][]uint{player.ID: {1, 2}}
	g.PendingBattles = []*model.TacticalBattle{{
		ID:            42,
		Status:        model.TacticalBattleActive,
		HumanOwnerIDs: []uint{player.ID},
		Ships: []*model.TacticalShip{{
			ID:      1,
			OwnerID: player.ID,
			Beams:   []model.Beam{{Name: "Laser"}},
		}},
		TurnOrder: []uint{1},
		Events:    []model.BattleEvent{{Type: "round_start", Round: 1}},
	}}
	g.BattleReports = []*model.BattleReport{{
		ID:     7,
		StarID: 1,
		Sides: []model.BattleSide{{
			OwnerID:       player.ID,
			OriginalShips: []model.FleetShip{{Type: "colony_ship", Name: "A"}},
		}},
	}}
	g.InvasionReports = []*model.InvasionReport{{
		ID:              3,
		AttackerOwnerID: player.ID,
		DefenderOwnerID: 2,
		Rounds:          []model.GroundBattleRound{{Round: 1}},
		InitialAttacker: model.GroundForce{OwnerID: player.ID, Marines: []model.GroundUnit{{Type: model.MarineUnit}}},
	}}
	return g
}

func snapshotLiveGameJSON(t *testing.T, g *Game) []byte {
	t.Helper()
	// Strip non-JSON runtime fields the same way encoding would for equality.
	data, err := json.Marshal(g)
	require.NoError(t, err)
	return data
}

func TestGetVisibleGameForPlayerDoesNotMutateLiveGame(t *testing.T) {
	g := newTwoPlayerIsolationGame(t)
	before := snapshotLiveGameJSON(t, g)

	visible, err := g.GetVisibleGameForPlayer(1)
	require.NoError(t, err)
	require.NotNil(t, visible)
	assert.Nil(t, visible.mu, "visible game must not share the live mutex")
	assert.NotZero(t, visible.Players[0].CommandPointsCapacity)

	after := snapshotLiveGameJSON(t, g)
	assert.True(t, bytes.Equal(before, after), "live game JSON must be identical before and after GetVisibleGameForPlayer")
}

func TestVisibleGameMutationsDoNotAffectLiveGame(t *testing.T) {
	g := newTwoPlayerIsolationGame(t)

	livePlayerBuildings := append([]string(nil), g.Players[0].Available.Buildings...)
	livePlanetBuildings := append([]string(nil), findOwnedPlanet(t, g, 1).Buildings...)
	liveFleetShips := append([]model.FleetShip(nil), g.Fleets[0].Ships...)
	livePendingBases := append([]uint(nil), g.PendingColonyBases[1]...)
	liveBattleShipBeams := append([]model.Beam(nil), g.PendingBattles[0].Ships[0].Beams...)
	liveReportShips := append([]model.FleetShip(nil), g.BattleReports[0].Sides[0].OriginalShips...)
	liveDesignBeams := append([]model.Beam(nil), g.Players[0].ShipDesigns[0].Beams...)
	liveExplored := append([]uint(nil), g.Players[0].ExploredStars...)
	liveHyper := g.Players[0].HyperAdvanced["Physics"]

	visible, err := g.GetVisibleGameForPlayer(1)
	require.NoError(t, err)

	vp := visible.Players[0]
	vp.Name = "Mutated"
	vp.Treasury = 99999
	vp.Available.Buildings[0] = "MUTATED"
	vp.Available.Buildings = append(vp.Available.Buildings, "Extra")
	vp.ExploredStars[0] = 999
	vp.ExploredStars = append(vp.ExploredStars, 888)
	vp.ShipDesigns[0].Name = "Mutated Design"
	vp.ShipDesigns[0].Beams[0].Name = "Mutated Beam"
	vp.ShipDesigns[0].Beams = append(vp.ShipDesigns[0].Beams, model.Beam{Name: "Extra"})
	vp.HyperAdvanced["Physics"] = 99
	vp.HyperAdvanced["New"] = 1
	vp.CommandPointsCapacity = 12345

	visPlanet := findOwnedPlanet(t, visible, 1)
	visPlanet.Name = "Mutated Planet"
	require.NotEmpty(t, visPlanet.Buildings)
	visPlanet.Buildings[0] = "MUTATED HAB"
	visPlanet.Buildings = append(visPlanet.Buildings, "Extra Hab")
	if len(visPlanet.BuildQueue) == 0 {
		visPlanet.BuildQueue = []string{"MUTATED QUEUE"}
	} else {
		visPlanet.BuildQueue[0] = "MUTATED QUEUE"
	}
	visPlanet.BuildQueue = append(visPlanet.BuildQueue, "Extra Queue")
	if len(visPlanet.Morale.Sources) == 0 {
		visPlanet.Morale.Sources = []model.PlanetMoraleSource{{Name: "Mutated Mood"}}
	} else {
		visPlanet.Morale.Sources[0].Name = "Mutated Mood"
	}
	visPlanet.Morale.Sources = append(visPlanet.Morale.Sources, model.PlanetMoraleSource{Name: "Extra Mood"})
	visPlanet.Population = 1

	visStar := visible.Galaxy.Stars[0]
	visStar.Name = "Mutated Star"
	visStar.X = -1

	visible.Fleets[0].Ships[0].Name = "Mutated Ship"
	visible.Fleets[0].Ships = append(visible.Fleets[0].Ships, model.FleetShip{Type: "extra"})
	visible.Fleets = append(visible.Fleets, &model.Fleet{ID: 1})

	visible.PendingColonyBases[1][0] = 999
	visible.PendingColonyBases[1] = append(visible.PendingColonyBases[1], 42)
	visible.PendingColonyBases[99] = []uint{1}

	visible.PendingBattles[0].Status = "mutated"
	visible.PendingBattles[0].Ships[0].Beams[0].Name = "Mutated"
	visible.PendingBattles[0].Ships[0].Beams = append(visible.PendingBattles[0].Ships[0].Beams, model.Beam{Name: "X"})
	visible.PendingBattles[0].Events = append(visible.PendingBattles[0].Events, model.BattleEvent{Type: "mutated"})

	visible.BattleReports[0].Sides[0].OriginalShips[0].Name = "Mutated"
	visible.BattleReports[0].Sides[0].OriginalShips = append(visible.BattleReports[0].Sides[0].OriginalShips, model.FleetShip{Name: "X"})

	visible.InvasionReports[0].Rounds[0].Round = 99
	visible.InvasionReports[0].InitialAttacker.Marines[0].Strength = 99

	visible.Players = append(visible.Players, &model.Player{ID: 99})
	visible.Galaxy.Stars = append(visible.Galaxy.Stars, &model.Star{ID: 99})

	assert.Equal(t, livePlayerBuildings, g.Players[0].Available.Buildings)
	assert.Equal(t, "Ada", g.Players[0].Name)
	assert.Equal(t, liveExplored, g.Players[0].ExploredStars)
	assert.Equal(t, liveHyper, g.Players[0].HyperAdvanced["Physics"])
	assert.NotContains(t, g.Players[0].HyperAdvanced, "New")
	assert.Equal(t, "Isolation Frigate", g.Players[0].ShipDesigns[0].Name)
	assert.Equal(t, liveDesignBeams, g.Players[0].ShipDesigns[0].Beams)

	livePlanet := findOwnedPlanet(t, g, 1)
	assert.Equal(t, livePlanetBuildings, livePlanet.Buildings)
	assert.NotEqual(t, "Mutated Planet", livePlanet.Name)
	assert.NotEqual(t, int64(1), livePlanet.Population)

	assert.Equal(t, liveFleetShips, g.Fleets[0].Ships)
	assert.Equal(t, livePendingBases, g.PendingColonyBases[1])
	assert.NotContains(t, g.PendingColonyBases, uint(99))
	assert.Equal(t, model.TacticalBattleActive, g.PendingBattles[0].Status)
	assert.Equal(t, liveBattleShipBeams, g.PendingBattles[0].Ships[0].Beams)
	assert.Equal(t, liveReportShips, g.BattleReports[0].Sides[0].OriginalShips)
	assert.Equal(t, 1, g.InvasionReports[0].Rounds[0].Round)
	assert.Len(t, g.Players, 2)
}

func findOwnedPlanet(t *testing.T, g *Game, ownerID uint) *model.Planet {
	t.Helper()
	for _, star := range g.Galaxy.Stars {
		for _, planet := range star.Planets {
			if planet != nil && planet.Owner == ownerID {
				return planet
			}
		}
	}
	t.Fatalf("no planet owned by %d", ownerID)
	return nil
}

func TestConcurrentSubmitTurnAndVisibleReadsNoRace(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)
	g, err := NewGame(resources, NewGameOptions{
		PlayerName:  "Ada",
		PlayerColor: model.DefaultPlayerColor(),
		NumStars:    4,
		Race:        DefaultRace("Human"),
		NumPlayers:  2,
	})
	require.NoError(t, err)

	const readers = 8
	const iterations = 40
	var wg sync.WaitGroup

	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func(playerID uint) {
			defer wg.Done()
			for n := 0; n < iterations*4; n++ {
				visible, err := g.GetVisibleGameForPlayer(playerID)
				if err == nil {
					_ = visible.CurrentTurn
					if len(visible.Players) > 0 {
						_ = visible.Players[0].CommandPointsCapacity
					}
				}
			}
		}(uint(i%2 + 1))
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			_, _, err := g.SubmitTurn(1, model.Commands{})
			require.NoError(t, err)
			_, advanced, err := g.SubmitTurn(2, model.Commands{})
			require.NoError(t, err)
			require.True(t, advanced)
		}
	}()

	wg.Wait()
}
