package game

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	. "ssg/model"
	"testing"
)

func TestConquestEffectsPersistAndLiberation(t *testing.T) {
	g, attacker, p := invasionHarness(t)
	defender := g.Players[1]
	p.Marines = 0
	p.Buildings = []string{"Capitol", "Automated Factory", "Hydroponic Farm", "Research Laboratory", "Marine Barracks"}
	p.CurrentProject, p.BuildQueue, p.IsBuilding, p.BuildBucket, p.IndustryStack, p.Autobuild = "Automated Factory", []string{"Marine Barracks"}, true, 30, 7, true
	fleet := &Fleet{ID: 45, Owner: attacker.ID, StarID: p.ParentStar, Ships: []FleetShip{{Type: "transport"}}}
	g.Fleets = append(g.Fleets, fleet)
	g.PendingInvasions = []PendingInvasion{{OwnerID: attacker.ID, FleetID: fleet.ID, TargetPlanetID: p.ID}}
	reports := g.resolveGroundInvasions()
	require.Len(t, reports, 1)
	r := reports[0]
	require.True(t, r.PlanetCaptured)
	require.NotNil(t, r.CaptureEffects)
	assert.Equal(t, defender.Race.Name, p.PopulationRace)
	assert.Equal(t, int64(2000), p.Population)
	assert.Equal(t, attacker.ID, p.Owner)
	assert.Equal(t, []string{"Automated Factory"}, r.CaptureEffects.DestroyedBuildings)
	assert.Equal(t, []string{"Capitol"}, r.CaptureEffects.RemovedBuildings)
	assert.InDelta(t, 0.2, r.CaptureEffects.UnrestPenalty, 1e-9)
	assert.Contains(t, sourceNames(PlanetMoraleSources(g, p, attacker)), "Conquered population")
	foreignBonus := PlanetMoraleBonus(g, p, attacker)
	originalRace := p.PopulationRace
	p.PopulationRace = attacker.Race.Name
	assert.InDelta(t, foreignBonus+0.2, PlanetMoraleBonus(g, p, attacker), 1e-9)
	p.PopulationRace = originalRace
	assert.Equal(t, "", p.CurrentProject)
	assert.Empty(t, p.BuildQueue)
	assert.False(t, p.IsBuilding)
	assert.False(t, p.Autobuild)
	assert.Zero(t, p.BuildBucket)
	assert.Zero(t, p.IndustryStack)
	assert.True(t, hasExploredStar(attacker.ExploredStars, p.ParentStar))
	visible, err := g.GetVisibleGameForPlayer(attacker.ID)
	require.NoError(t, err)
	vp, err := visible.getPlanet(p.ID)
	require.NoError(t, err)
	assert.Equal(t, attacker.ID, vp.Owner)
	visible.InvasionReports[0].CaptureEffects.DestroyedBuildings[0] = "changed"
	assert.Equal(t, "Automated Factory", r.CaptureEffects.DestroyedBuildings[0])
	encoded, err := json.Marshal(p)
	require.NoError(t, err)
	var loaded Planet
	require.NoError(t, json.Unmarshal(encoded, &loaded))
	assert.Equal(t, defender.Race.Name, loaded.PopulationRace)
	encoded, err = json.Marshal(r)
	require.NoError(t, err)
	var loadedReport InvasionReport
	require.NoError(t, json.Unmarshal(encoded, &loadedReport))
	require.NotNil(t, loadedReport.CaptureEffects)
	assert.Equal(t, []string{"Automated Factory"}, loadedReport.CaptureEffects.DestroyedBuildings)
	effects := g.capturePlanet(p, defender.ID, &GroundForce{})
	assert.Equal(t, defender.Race.Name, p.PopulationRace)
	assert.Zero(t, effects.UnrestPenalty)
	assert.NotContains(t, sourceNames(PlanetMoraleSources(g, p, defender)), "Conquered population")
}

func TestConquestPreservesOriginalRaceAcrossOccupations(t *testing.T) {
	g, attacker, p := invasionHarness(t)
	p.PopulationRace = "Psilon"
	p.Buildings = []string{"Marine Barracks", "Research Laboratory", "Hydroponic Farm"}
	effects := g.capturePlanet(p, attacker.ID, &GroundForce{})
	assert.Equal(t, "Psilon", p.PopulationRace)
	assert.Empty(t, effects.DestroyedBuildings)
	assert.Len(t, p.Buildings, 3)
}
