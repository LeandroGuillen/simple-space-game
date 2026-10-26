package game

import (
	"fmt"
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuyProductionCostUsesFourBcPerPpBelowFiftyPercent(t *testing.T) {
	assert.Equal(t, int64(400), BuyProductionCost(100, 100), "0% complete: 100 remaining × 4 BC = 400 BC")
	assert.Equal(t, int64(204), BuyProductionCost(100, 51), "49% complete: 51 remaining × 4 BC = 204 BC")
}

func TestBuyProductionCostUsesTwoBcPerPpAtOrAboveFiftyPercent(t *testing.T) {
	assert.Equal(t, int64(100), BuyProductionCost(100, 50), "exactly 50% complete falls into the cheaper tier")
	assert.Equal(t, int64(80), BuyProductionCost(100, 40), "60% complete: 40 remaining × 2 BC = 80 BC")
	assert.Equal(t, int64(2), BuyProductionCost(100, 1), "99% complete: 1 remaining × 2 BC = 2 BC")
}

func TestBuyProductionCostReturnsZeroForNonPositiveInputs(t *testing.T) {
	assert.Equal(t, int64(0), BuyProductionCost(100, 0))
	assert.Equal(t, int64(0), BuyProductionCost(100, -5))
	assert.Equal(t, int64(0), BuyProductionCost(0, 50))
}

func TestTurnBuyProductionFinishesProjectAndDeductsTreasury(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: DefaultRace("Human")})
	require.NoError(t, err)
	player := g.Players[0]
	player.Available.Buildings = append(player.Available.Buildings, "Automated Factory")

	// kick off a real construction so a building cost can be observed.
	require.NoError(t, g.Turn(player.ID, model.Commands{
		Build: []*model.BuildCommand{{PlanetID: 1, Building: "Automated Factory"}},
	}))
	colonies, err := g.GetColonyList(player.ID)
	require.NoError(t, err)
	require.NotEmpty(t, colonies)
	capital := colonies[0]
	require.True(t, capital.IsBuilding)
	require.Equal(t, "Automated Factory", capital.CurrentProject)

	building, err := getBuildingFromName(g, capital.CurrentProject)
	require.NoError(t, err)
	effectiveCost := effectiveBuildingCost(capital, building, g)
	remainingBeforeBuy := capital.BuildBucket
	expectedCost := BuyProductionCost(effectiveCost, remainingBeforeBuy)

	// guarantee the player can afford the buyout regardless of starting treasury
	player.Treasury = float64(expectedCost + 50)
	treasuryBeforeBuy := player.Treasury

	require.NoError(t, g.Turn(player.ID, model.Commands{
		BuyProduction: []*model.BuyProductionCommand{{PlanetID: capital.ID}},
	}))

	colonies, err = g.GetColonyList(player.ID)
	require.NoError(t, err)
	finished := colonies[0]
	assert.False(t, finished.IsBuilding, "project should complete the same turn it is bought")
	assert.Contains(t, finished.Buildings, "Automated Factory")
	// treasury at end of turn = pre-turn - buy cost + this turn's income.
	assert.Less(t, player.Treasury, treasuryBeforeBuy, "buyout must reduce treasury net of income")
	assert.GreaterOrEqual(t, player.Treasury, treasuryBeforeBuy-float64(expectedCost), "treasury should never drop more than the buy cost (income only adds back)")
}

func TestTurnBuyProductionRejectsCombinedPurchasesWithoutMutatingState(t *testing.T) {
	g := newSinglePlanetBuyHarness(t, 400)
	player := g.Players[0]
	first := g.Galaxy.Stars[0].Planets[0]
	second := *first
	second.ID = 2
	g.Galaxy.Stars[0].Planets = append(g.Galaxy.Stars[0].Planets, &second)

	firstPop, secondPop := first.Population, second.Population
	firstBucket, secondBucket := first.BuildBucket, second.BuildBucket
	treasury := player.Treasury

	err := g.Turn(player.ID, model.Commands{BuyProduction: []*model.BuyProductionCommand{
		{PlanetID: first.ID},
		{PlanetID: second.ID},
	}})

	require.Error(t, err, "two 400 BC buys must be rejected with only 400 BC available")
	assert.Equal(t, treasury, player.Treasury)
	assert.Equal(t, firstPop, first.Population)
	assert.Equal(t, secondPop, second.Population)
	assert.Equal(t, firstBucket, first.BuildBucket)
	assert.Equal(t, secondBucket, second.BuildBucket)
	assert.Equal(t, "Cheap Lab", first.CurrentProject)
	assert.Equal(t, "Cheap Lab", second.CurrentProject)
}

func TestTurnBuyProductionCompletesBothAffordablePurchasesAndChargesTheirSum(t *testing.T) {
	g := newSinglePlanetBuyHarness(t, 800)
	player := g.Players[0]
	first := g.Galaxy.Stars[0].Planets[0]
	second := *first
	second.ID = 2
	g.Galaxy.Stars[0].Planets = append(g.Galaxy.Stars[0].Planets, &second)
	startingTreasury := player.Treasury

	require.NoError(t, g.Turn(player.ID, model.Commands{BuyProduction: []*model.BuyProductionCommand{
		{PlanetID: first.ID},
		{PlanetID: second.ID},
	}}))

	income := planetIncomeAfterTurn(t, g, player.ID, first.ID) + planetIncomeAfterTurn(t, g, player.ID, second.ID)
	assert.Equal(t, float64(800), startingTreasury-player.Treasury+float64(income))
	assert.Contains(t, first.Buildings, "Cheap Lab")
	assert.Contains(t, second.Buildings, "Cheap Lab")
}

func TestTurnBuyProductionDuplicatePlanetCommandIsChargedOnce(t *testing.T) {
	g := newSinglePlanetBuyHarness(t, 400)
	player := g.Players[0]
	planet := g.Galaxy.Stars[0].Planets[0]
	startingTreasury := player.Treasury

	require.NoError(t, g.Turn(player.ID, model.Commands{BuyProduction: []*model.BuyProductionCommand{
		{PlanetID: planet.ID},
		{PlanetID: planet.ID},
	}}))

	income := planetIncomeAfterTurn(t, g, player.ID, planet.ID)
	assert.Equal(t, float64(400), startingTreasury-player.Treasury+float64(income))
	assert.Contains(t, planet.Buildings, "Cheap Lab")
}

func TestTurnBuyProductionSameTurnBuildUsesNewProjectCost(t *testing.T) {
	for _, treasury := range []int64{799, 800} {
		t.Run(fmt.Sprintf("treasury_%d", treasury), func(t *testing.T) {
			g := newSinglePlanetBuyHarness(t, treasury)
			player := g.Players[0]
			planet := g.Galaxy.Stars[0].Planets[0]
			g.Resources.BuildingCollection.Buildings = append(g.Resources.BuildingCollection.Buildings, model.Building{Name: "Expensive Lab", Cost: 200})
			player.Available.Buildings = append(player.Available.Buildings, "Expensive Lab")
			startingPopulation, startingBucket := planet.Population, planet.BuildBucket

			err := g.Turn(player.ID, model.Commands{
				Build:         []*model.BuildCommand{{PlanetID: planet.ID, Building: "Expensive Lab"}},
				BuyProduction: []*model.BuyProductionCommand{{PlanetID: planet.ID}},
			})
			if treasury == 799 {
				require.Error(t, err, "the new project's 800 BC cost must be checked before turn processing")
				assert.Equal(t, float64(799), player.Treasury)
				assert.Equal(t, startingPopulation, planet.Population)
				assert.Equal(t, startingBucket, planet.BuildBucket)
				assert.Equal(t, "Cheap Lab", planet.CurrentProject)
				return
			}
			require.NoError(t, err)
			income := planetIncomeAfterTurn(t, g, player.ID, planet.ID)
			assert.Equal(t, float64(800), float64(treasury)-player.Treasury+float64(income))
			assert.Contains(t, planet.Buildings, "Expensive Lab")
		})
	}
}

func TestTurnBuyProductionRejectsInsufficientTreasury(t *testing.T) {
	g := newSinglePlanetBuyHarness(t, 0 /*treasury*/)
	player := g.Players[0]
	planet := g.Galaxy.Stars[0].Planets[0]
	require.Equal(t, int64(100), planet.BuildBucket)

	err := g.Turn(player.ID, model.Commands{
		BuyProduction: []*model.BuyProductionCommand{{PlanetID: planet.ID}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot buy production")

	assert.Equal(t, int64(100), planet.BuildBucket, "failed buy must not change bucket")
	assert.Equal(t, float64(0), player.Treasury)
}

func TestTurnBuyProductionRejectsTradeGoods(t *testing.T) {
	g := newSinglePlanetBuyHarness(t, 1000)
	player := g.Players[0]
	planet := g.Galaxy.Stars[0].Planets[0]
	planet.CurrentProject = "Trade Goods"
	planet.IsBuilding = false
	planet.BuildBucket = 0

	err := g.Turn(player.ID, model.Commands{
		BuyProduction: []*model.BuyProductionCommand{{PlanetID: planet.ID}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Trade Goods")
	assert.Equal(t, float64(1000), player.Treasury, "treasury must be unchanged when buy fails")
}

func TestTurnBuyProductionIgnoresOtherPlayersPlanets(t *testing.T) {
	g := newSinglePlanetBuyHarness(t, 500)
	player := g.Players[0]

	// command targets a non-existent planet ID owned by no one
	err := g.Turn(player.ID, model.Commands{
		BuyProduction: []*model.BuyProductionCommand{{PlanetID: 9999}},
	})
	require.NoError(t, err, "buy command for a non-owned planet should be silently ignored")
	planet := g.Galaxy.Stars[0].Planets[0]
	assert.Equal(t, int64(100), planet.BuildBucket-planetIndustryProduced(g, planet), "bucket only drains by normal industry")
}

func TestTurnBuyProductionCostMatchesBelowFiftyTier(t *testing.T) {
	g := newSinglePlanetBuyHarness(t, 10_000)
	player := g.Players[0]
	planet := g.Galaxy.Stars[0].Planets[0]
	// 100 remaining out of 100 total => 0% complete => 4 BC/PP => 400 BC
	startingTreasury := player.Treasury

	require.NoError(t, g.Turn(player.ID, model.Commands{
		BuyProduction: []*model.BuyProductionCommand{{PlanetID: planet.ID}},
	}))

	// turn income gets added on top, so isolate the buy-cost delta.
	income := planetIncomeAfterTurn(t, g, player.ID, planet.ID)
	netTreasuryChange := player.Treasury - startingTreasury
	buyCost := float64(income) - netTreasuryChange
	assert.Equal(t, float64(400), buyCost, "below-50%% rate should be 4 BC/PP")
	assert.False(t, planet.IsBuilding)
	assert.Contains(t, planet.Buildings, "Cheap Lab")
}

func TestTurnBuyProductionCostMatchesAboveFiftyTier(t *testing.T) {
	g := newSinglePlanetBuyHarness(t, 10_000)
	player := g.Players[0]
	planet := g.Galaxy.Stars[0].Planets[0]
	// pre-invest 60% by lowering bucket from 100 to 40
	planet.BuildBucket = 40
	startingTreasury := player.Treasury

	require.NoError(t, g.Turn(player.ID, model.Commands{
		BuyProduction: []*model.BuyProductionCommand{{PlanetID: planet.ID}},
	}))

	income := planetIncomeAfterTurn(t, g, player.ID, planet.ID)
	netTreasuryChange := player.Treasury - startingTreasury
	buyCost := float64(income) - netTreasuryChange
	assert.Equal(t, float64(80), buyCost, "above-50%% rate should be 2 BC/PP")
	assert.Contains(t, planet.Buildings, "Cheap Lab")
}

func TestTurnBuyProductionUsesStackedProduction(t *testing.T) {
	g := newSinglePlanetBuyHarness(t, 10_000)
	player := g.Players[0]
	planet := g.Galaxy.Stars[0].Planets[0]
	planet.IndustryStack = 60
	startingTreasury := player.Treasury

	require.NoError(t, g.Turn(player.ID, model.Commands{
		BuyProduction: []*model.BuyProductionCommand{{PlanetID: planet.ID}},
	}))

	income := planetIncomeAfterTurn(t, g, player.ID, planet.ID)
	netTreasuryChange := player.Treasury - startingTreasury
	buyCost := float64(income) - netTreasuryChange
	assert.Equal(t, float64(80), buyCost, "stacked PP should reduce the remaining buyout cost")
	assert.Equal(t, int64(0), planet.IndustryStack)
	assert.Contains(t, planet.Buildings, "Cheap Lab")
}

// newSinglePlanetBuyHarness builds a minimal game with one player and one planet currently
// constructing a 100-PP building called "Cheap Lab". The planet has no industry workers so
// the only way for the project to complete is via a Buy command.
func newSinglePlanetBuyHarness(t *testing.T, treasury int64) *Game {
	t.Helper()
	building := model.Building{Name: "Cheap Lab", Cost: 100}
	planet := &model.Planet{
		ID:             1,
		Owner:          1,
		Population:     1000,
		Size:           model.PlanetSizeMedium,
		CurrentProject: "Cheap Lab",
		IsBuilding:     true,
		BuildBucket:    100,
	}
	return &Game{
		Players: []*model.Player{{
			ID:        1,
			Treasury:  float64(treasury),
			Available: model.AvailableTech{Buildings: []string{"Cheap Lab"}},
		}},
		Resources: &Resources{
			BuildingCollection: &BuildingCollection{Buildings: []model.Building{building}},
		},
		Galaxy: model.Galaxy{Stars: []*model.Star{{Planets: []*model.Planet{planet}}}},
	}
}

func planetIndustryProduced(g *Game, p *model.Planet) int64 {
	return GamePlanetIndustryProduced(g, p)
}

func planetIncomeAfterTurn(t *testing.T, g *Game, playerID, planetID uint) int64 {
	t.Helper()
	colonies, err := g.GetColonyList(playerID)
	require.NoError(t, err)
	for _, p := range colonies {
		if p.ID == planetID {
			return p.Income.Total
		}
	}
	t.Fatalf("planet %d not found", planetID)
	return 0
}
