package ai

import (
	"github.com/stretchr/testify/require"
	"ssg/model"
	"testing"
)

func TestExpansionReservesLocalTargetForOneFleet(t *testing.T) {
	p := &model.Player{ID: 1}
	star := &model.Star{ID: 1, Name: "Sol", Planets: []*model.Planet{{ID: 1, Owner: 1, ParentStar: 1}, {ID: 2, ParentStar: 1, Environment: model.PlanetEnvironmentTerran}}}
	fleets := []*model.Fleet{{ID: 10, Owner: 1, StarID: 1, Ships: []model.FleetShip{{Type: "colony_ship"}}}, {ID: 11, Owner: 1, StarID: 1, Ships: []model.FleetShip{{Type: "colony_ship"}}}}
	s := &VisibleGame{Players: []*model.Player{p}, Galaxy: Galaxy{Stars: []*model.Star{star}}, Fleets: fleets}
	d := Decision{}
	planExpansion(s, p, []*model.Planet{star.Planets[0]}, &d)
	require.Len(t, d.Commands.ColonizeMany, 1)
	require.Equal(t, uint(2), d.Commands.ColonizeMany[0].PlanetID)
}
func TestExpansionTargetsOnlyVisibleAndFuelReachableStars(t *testing.T) {
	home := &model.Star{ID: 1, Name: "Sol", X: 0, Y: 0, Planets: []*model.Planet{{ID: 1, Owner: 1}}}
	nearby := &model.Star{ID: 2, Name: "Near", X: 60, Y: 0, Planets: []*model.Planet{{ID: 2, Environment: model.PlanetEnvironmentTerran}}}
	far := &model.Star{ID: 3, Name: "Far", X: 600, Y: 0, Planets: []*model.Planet{{ID: 3, Environment: model.PlanetEnvironmentTerran}}}
	hidden := &model.Star{ID: 4, X: 30, Y: 0, Planets: []*model.Planet{{ID: 4, Environment: model.PlanetEnvironmentTerran}}}
	p := &model.Player{ID: 1}
	f := &model.Fleet{ID: 10, Owner: 1, StarID: 1, Ships: []model.FleetShip{{Type: "colony_ship", FuelRangeParsecs: 2}}}
	s := &VisibleGame{Players: []*model.Player{p}, Galaxy: Galaxy{Stars: []*model.Star{home, nearby, far, hidden}}, Fleets: []*model.Fleet{f}}
	require.Equal(t, uint(2), expansionTarget(s, 1, f, map[uint]bool{}))
	require.Zero(t, expansionTarget(s, 1, f, map[uint]bool{2: true}))
}
