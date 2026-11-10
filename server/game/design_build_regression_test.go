package game

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"ssg/model"
	"testing"
)

func TestEditShipDesignDuringProduction(t *testing.T) {
	for _, queued := range []bool{false, true} {
		t.Run(map[bool]string{false: "active", true: "queued"}[queued], func(t *testing.T) {
			resources := testResources(t)
			g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 4, PlayerColor: model.DefaultPlayerColor(), Race: DefaultRace("Ada")})
			require.NoError(t, err)
			require.NoError(t, g.CreateShipDesign(1, "Raider", "Nuclear", "None", "None", "Titanium", "Frigate", "Standard"))
			command := &model.BuildCommand{PlanetID: 1, Building: "Raider"}
			if queued {
				command.Building = "Flunker I"
				command.BuildQueue = []string{"Raider"}
			}
			require.NoError(t, g.Turn(1, model.Commands{Build: []*model.BuildCommand{command}}))
			err = g.UpdateShipDesign(1, "Raider", "Nuclear", "None", "None", "Titanium", "Destroyer", "Standard")
			assert.Error(t, err, "editing an active or queued design must be rejected")
			for i := 0; i < 100; i++ {
				p, _ := g.getPlanet(1)
				if !p.IsBuilding && len(p.BuildQueue) == 0 {
					break
				}
				require.NoError(t, g.Turn(1, model.Commands{}))
			}
			p, _ := g.getPlanet(1)
			assert.NotContains(t, p.Buildings, "Raider")
			found := false
			for _, fleet := range g.Fleets {
				for _, ship := range fleet.Ships {
					if ship.DesignName == "Raider" {
						found = true
						assert.Equal(t, "Frigate", ship.HullSize)
					}
				}
			}
			assert.True(t, found, "original frigate must complete")
			assert.NoError(t, ValidateLoadedGame(g, resources))
			data, err := json.Marshal(g)
			require.NoError(t, err)
			var loaded Game
			require.NoError(t, json.Unmarshal(data, &loaded))
			assert.NoError(t, ValidateLoadedGame(&loaded, resources))
		})
	}
}

func TestBuildOnPlanetRejectsUnknownBuilding(t *testing.T) {
	g, p := newFrigateProductionGame(t, DefaultRace("Ada"))
	before := append([]string(nil), p.Buildings...)
	buildOnPlanet(g, p, "Raider")
	assert.Equal(t, before, p.Buildings)
}

func TestUnresolvedShipCompletionKeepsProduction(t *testing.T) {
	g, p := newFrigateProductionGame(t, DefaultRace("Ada"))
	require.NoError(t, g.CreateShipDesign(1, "Raider", "Nuclear", "None", "None", "Titanium", "Frigate", "Standard"))
	require.True(t, Build(p, "Raider", g.Players[0].Available.Buildings, g))
	// Simulate a stale catalog / old save bypassing the mutation guard.
	g.Players[0].ShipDesigns[0].HullSize.Name = "Destroyer"
	p.BuildBucket = 1
	p.IndustryStack = 100
	require.NoError(t, g.Turn(1, model.Commands{}))
	assert.NotContains(t, p.Buildings, "Raider")
	assert.True(t, p.IsBuilding)
	assert.Equal(t, int64(1), p.BuildBucket)
	assert.GreaterOrEqual(t, p.IndustryStack, int64(100))
	found := false
	for _, event := range g.TurnEvents {
		if event.Kind == "production_blocked" {
			found = true
			assert.Contains(t, event.Text, "production kept")
		}
	}
	assert.True(t, found)
}

func TestShipDesignEditRejectsSubmittedQueue(t *testing.T) {
	resources := testResources(t)
	g, err := NewGame(resources, NewGameOptions{PlayerName: "Ada", NumStars: 8, NumPlayers: 2, PlayerColor: model.DefaultPlayerColor(), Race: DefaultRace("Ada")})
	require.NoError(t, err)
	require.NoError(t, g.CreateShipDesign(1, "Raider", "Nuclear", "None", "None", "Titanium", "Frigate", "Standard"))
	_, advanced, err := g.SubmitTurn(1, model.Commands{Build: []*model.BuildCommand{{PlanetID: 1, Building: "Flunker I", BuildQueue: []string{"Raider"}}}})
	require.NoError(t, err)
	require.False(t, advanced)
	assert.Error(t, g.UpdateShipDesign(1, "Raider", "Nuclear", "None", "None", "Titanium", "Destroyer", "Standard"))
}
