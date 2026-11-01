package game

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	. "ssg/model"
	"testing"
)

func eventKinds(events []TurnEvent) []string {
	out := []string{}
	for _, event := range events {
		out = append(out, event.Kind)
	}
	return out
}
func TestTurnEventsForProductionResearchArrivalAndPrivacy(t *testing.T) {
	g, attacker, defenderPlanet := invasionHarness(t)
	defender := g.Players[1]
	var home *Planet
	for _, star := range g.Galaxy.Stars {
		for _, p := range star.Planets {
			if p.Owner == attacker.ID {
				home = p
				break
			}
		}
		if home != nil {
			break
		}
	}
	require.NotNil(t, home)
	home.CurrentProject = "Colony Ship"
	home.IsBuilding = true
	home.BuildBucket = 1
	home.IndustryStack = 10
	attacker.ResearchPoints = 1000
	target, err := g.getStar(defenderPlanet.ParentStar)
	require.NoError(t, err)
	g.Fleets = append(g.Fleets, &Fleet{ID: 907, Owner: attacker.ID, StarID: home.ParentStar, DestinationStarID: target.ID, OriginStarID: home.ParentStar, TurnsRemaining: 1, TotalTurns: 1, Ships: []FleetShip{{Type: "colony_ship"}}})
	_, _, err = g.SubmitTurn(attacker.ID, Commands{SelectResearch: &SelectResearchCommand{TechName: "Physics"}})
	require.NoError(t, err)
	_, advanced, err := g.SubmitTurn(defender.ID, Commands{})
	require.NoError(t, err)
	require.True(t, advanced)
	visible, err := g.GetVisibleGameForPlayer(attacker.ID)
	require.NoError(t, err)
	kinds := eventKinds(visible.TurnEvents)
	assert.Contains(t, kinds, "ship")
	assert.Contains(t, kinds, "research")
	assert.Contains(t, kinds, "arrival")
	other, err := g.GetVisibleGameForPlayer(defender.ID)
	require.NoError(t, err)
	for _, event := range other.TurnEvents {
		assert.Equal(t, defender.ID, event.Owner)
	}
}
func TestTurnEventsRestoreAfterFailedTurn(t *testing.T) {
	g := newTestGame(t)
	g.TurnEvents = []TurnEvent{{Owner: 1, Kind: "old", Text: "previous"}}
	advanceTurnTestHook = func(stage string) error {
		if stage == "before_fleet_movements" {
			return errors.New("injected")
		}
		return nil
	}
	t.Cleanup(func() { advanceTurnTestHook = nil })
	_, _, err := g.SubmitTurn(1, Commands{})
	require.Error(t, err)
	require.Equal(t, []TurnEvent{{Owner: 1, Kind: "old", Text: "previous"}}, g.TurnEvents)
}
func TestCombatEventsAreScopedToParticipants(t *testing.T) {
	g := newTestGame(t)
	g.recordBattleEvent(&BattleReport{StarID: 2, Sides: []BattleSide{{OwnerID: 1}, {OwnerID: 2}}})
	g.recordInvasionEvent(&InvasionReport{PlanetName: "Sirius", AttackerOwnerID: 1, DefenderOwnerID: 2, PlanetCaptured: true})
	require.Len(t, g.TurnEvents, 4)
	assert.Equal(t, []string{"battle", "battle", "invasion", "invasion"}, eventKinds(g.TurnEvents))
}

func TestTurnEventsReportBuildingAndColony(t *testing.T) {
	g := newTestGame(t)
	player := g.Players[0]
	var home *Planet
	var star *Star
	for _, s := range g.Galaxy.Stars {
		for _, p := range s.Planets {
			if p.Owner == player.ID {
				home = p
				star = s
				break
			}
		}
		if home != nil {
			break
		}
	}
	require.NotNil(t, home)
	target := &Planet{ID: 99999, ParentStar: star.ID, Name: "New World", Environment: PlanetEnvironmentTerran, Size: PlanetSizeMedium, Gravity: PlanetGravityNormal, MineralClass: PlanetMineralAbundant}
	star.Planets = append(star.Planets, target)
	home.CurrentProject = "Automated Factory"
	home.IsBuilding = true
	home.BuildBucket = 1
	home.IndustryStack = 10
	g.PendingColonyBases[player.ID] = []uint{home.ID}
	_, advanced, err := g.SubmitTurn(player.ID, Commands{ColonizeMany: []*ColonizeCommand{{PlanetID: target.ID}}})
	require.NoError(t, err)
	require.True(t, advanced)
	kinds := eventKinds(g.TurnEvents)
	assert.Contains(t, kinds, "building")
	assert.Contains(t, kinds, "colony")
}
func TestTurnEventsReportStarvation(t *testing.T) {
	g := newTestGame(t)
	player := g.Players[0]
	var home *Planet
	for _, s := range g.Galaxy.Stars {
		for _, p := range s.Planets {
			if p.Owner == player.ID {
				home = p
				break
			}
		}
		if home != nil {
			break
		}
	}
	require.NotNil(t, home)
	home.Food.Farmers = 0
	home.Industry.Workers = 8
	_, _, err := g.SubmitTurn(player.ID, Commands{})
	require.NoError(t, err)
	assert.Contains(t, eventKinds(g.TurnEvents), "starvation")
}
