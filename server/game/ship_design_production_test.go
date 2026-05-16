package game

import (
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildStarterFrigateDesignStartsAtFixedCost(t *testing.T) {
	g, planet := newFrigateProductionGame(t, DefaultRace("Ada"))

	require.True(t, Build(planet, "Flunker I", g.Players[0].Available.Buildings, g))

	assert.True(t, planet.IsBuilding)
	assert.Equal(t, "Flunker I", planet.CurrentProject)
	assert.Equal(t, int64(100), planet.BuildBucket)
}

func TestBuildStarterFrigateRequiresUnlockedFuelCell(t *testing.T) {
	g, planet := newFrigateProductionGame(t, DefaultRace("Ada"))
	g.Players[0].Available.FuelCells = removeString(g.Players[0].Available.FuelCells, "Standard")

	assert.False(t, Build(planet, "Flunker I", g.Players[0].Available.Buildings, g))
	assert.False(t, planet.IsBuilding)
}

func TestBuildStarterFrigateRequiresUnlockedDrive(t *testing.T) {
	g, planet := newFrigateProductionGame(t, DefaultRace("Ada"))
	g.Players[0].Available.Drives = removeString(g.Players[0].Available.Drives, "Nuclear")

	assert.False(t, Build(planet, "Flunker I", g.Players[0].Available.Buildings, g))
	assert.False(t, planet.IsBuilding)
}

func TestBuildUnknownShipDesignIsRejected(t *testing.T) {
	g, planet := newFrigateProductionGame(t, DefaultRace("Ada"))

	assert.False(t, Build(planet, "Missing Frigate", g.Players[0].Available.Buildings, g))
	assert.False(t, planet.IsBuilding)
}

func TestTurnCompletingStarterFrigateCreatesOrbitFleetShip(t *testing.T) {
	g, planet := newFrigateProductionGame(t, DefaultRace("Ada"))
	boostPlanetIndustry(g, planet, 100)

	require.NoError(t, g.Turn(1, model.Commands{
		Build: []*model.BuildCommand{{PlanetID: planet.ID, Building: "Flunker I"}},
	}))

	require.Len(t, g.Fleets, 1)
	assert.Equal(t, uint(1), g.Fleets[0].Owner)
	assert.Equal(t, planet.ParentStar, g.Fleets[0].StarID)
	require.Len(t, g.Fleets[0].Ships, 1)
	ship := g.Fleets[0].Ships[0]
	assert.Equal(t, "ship_design", ship.Type)
	assert.Equal(t, "Flunker I", ship.Name)
	assert.Equal(t, "Flunker I", ship.DesignName)
	assert.Equal(t, "Frigate", ship.HullSize)
}

func TestTurnCompletingMultipleStarterFrigatesReusesOrbitFleet(t *testing.T) {
	g, planet := newFrigateProductionGame(t, DefaultRace("Ada"))
	boostPlanetIndustry(g, planet, 100)

	require.NoError(t, g.Turn(1, model.Commands{
		Build: []*model.BuildCommand{{
			PlanetID:   planet.ID,
			Building:   "Flunker I",
			BuildQueue: []string{"Flunker I", "Flunker I"},
		}},
	}))
	require.True(t, planet.IsBuilding)

	require.NoError(t, g.Turn(1, model.Commands{}))

	require.Len(t, g.Fleets, 1)
	require.Len(t, g.Fleets[0].Ships, 2)
	assert.Equal(t, "Flunker I", g.Fleets[0].Ships[0].Name)
	assert.Equal(t, "Flunker I", g.Fleets[0].Ships[1].Name)
}

func TestFrigateDesignCostUsesGovernmentShipCostMultiplier(t *testing.T) {
	resources := testResources(t)
	feudal, err := BuildCustomRace(resources, CustomRaceDesign{Name: "Ada", Government: "Feudal"})
	require.NoError(t, err)
	g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: feudal})
	require.NoError(t, err)
	planet, err := g.getPlanet(1)
	require.NoError(t, err)

	require.True(t, Build(planet, "Flunker I", g.Players[0].Available.Buildings, g))

	assert.Equal(t, int64(67), planet.BuildBucket)
}

func TestCreateShipDesignRequiresResearchedParts(t *testing.T) {
	g, _ := newFrigateProductionGame(t, DefaultRace("Ada"))

	err := g.CreateShipDesign(1, "Fast One", "Fusion", "None", "None", "Titanium", "Frigate", "Standard")

	require.Error(t, err)
	assert.Contains(t, err.Error(), `drive "Fusion" has not been researched`)
}

func TestCreateShipDesignAcceptsUnlockedParts(t *testing.T) {
	g, _ := newFrigateProductionGame(t, DefaultRace("Ada"))

	require.NoError(t, g.CreateShipDesign(1, "Bare One", "Nuclear", "None", "None", "Titanium", "Frigate", "Standard"))

	require.Len(t, g.Players[0].ShipDesigns, 1)
	assert.Equal(t, "Bare One", g.Players[0].ShipDesigns[0].Name)
}

func TestCreateShipDesignRejectsBuiltInDesignName(t *testing.T) {
	g, _ := newFrigateProductionGame(t, DefaultRace("Ada"))

	err := g.CreateShipDesign(1, "Flunker I", "Nuclear", "None", "None", "Titanium", "Frigate", "Standard")

	require.Error(t, err)
	assert.Contains(t, err.Error(), `name already exists`)
}

func TestUpdateShipDesignRequiresResearchedParts(t *testing.T) {
	g, _ := newFrigateProductionGame(t, DefaultRace("Ada"))
	require.NoError(t, g.CreateShipDesign(1, "Bare One", "Nuclear", "None", "None", "Titanium", "Frigate", "Standard"))

	err := g.UpdateShipDesign(1, "Bare One", "Fusion", "None", "None", "Titanium", "Frigate", "Standard")

	require.Error(t, err)
	assert.Contains(t, err.Error(), `drive "Fusion" has not been researched`)
}

func newFrigateProductionGame(t *testing.T, race model.Race) (*Game, *model.Planet) {
	t.Helper()
	resources := testResources(t)
	g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: race})
	require.NoError(t, err)
	planet, err := g.getPlanet(1)
	require.NoError(t, err)
	return g, planet
}

func removeString(values []string, value string) []string {
	next := values[:0]
	for _, candidate := range values {
		if candidate != value {
			next = append(next, candidate)
		}
	}
	return next
}

func boostPlanetIndustry(g *Game, planet *model.Planet, amount int64) {
	g.Resources.Buildings = append(g.Resources.Buildings, model.Building{
		Name: "Test Foundry",
		Grants: model.Grants{
			Modifiers: []model.Modifier{{Name: "industry_flat", Value: float64(amount)}},
		},
	})
	planet.Buildings = append(planet.Buildings, "Test Foundry")
}
