package ai

import (
	"github.com/stretchr/testify/require"
	"ssg/model"
	"testing"
)

func militaryFixture() (*VisibleGame, *model.Player, *model.Fleet) {
	design := model.ShipDesign{Name: "Warship", Drive: model.Drive{Name: "Nuclear", InterstellarSpeed: 2}, FuelCell: model.FuelCell{Name: "Standard", Autonomy: 2}, Beams: []model.Beam{{Name: "Laser", MaxDamage: 4}}}
	p := &model.Player{ID: 1, ShipDesigns: []model.ShipDesign{design}}
	home := &model.Star{ID: 1, Name: "Home", Planets: []*model.Planet{{ID: 1, Owner: 1}}}
	enemy := &model.Star{ID: 2, Name: "Enemy", X: 60, Planets: []*model.Planet{{ID: 2, Owner: 2}}}
	ship := model.FleetShip{Type: "ship_design", DesignName: "Warship", FuelRangeParsecs: 2, InterstellarSpeed: 2}
	fleet := &model.Fleet{ID: 1, Owner: 1, StarID: 1, Ships: []model.FleetShip{ship, ship}}
	s := &VisibleGame{Players: []*model.Player{p}, Galaxy: Galaxy{Stars: []*model.Star{home, enemy}}, Fleets: []*model.Fleet{fleet}}
	return s, p, fleet
}
func TestAttackLeavesDefenderAndMovesOnlyArmedShips(t *testing.T) {
	s, p, f := militaryFixture()
	f.Ships = append(f.Ships, model.FleetShip{Type: "colony_ship"})
	d := Decision{}
	planFleetAttacks(s, p, &d)
	require.Len(t, d.Commands.FleetMove, 1)
	require.Equal(t, uint(2), d.Commands.FleetMove[0].DestinationStarID)
	require.Equal(t, []uint{1}, *d.Commands.FleetMove[0].ShipIndexes)
}
func TestAttackSkipsStrongInvisibleOrUnreachableTargets(t *testing.T) {
	s, p, _ := militaryFixture()
	enemy := s.Galaxy.Stars[1]
	d := Decision{}
	s.Fleets = append(s.Fleets, &model.Fleet{Owner: 2, StarID: 2, Ships: []model.FleetShip{{Type: "ship_design"}}})
	planFleetAttacks(s, p, &d)
	require.Empty(t, d.Commands.FleetMove)
	s.Fleets = s.Fleets[:1]
	enemy.Name = ""
	planFleetAttacks(s, p, &d)
	require.Empty(t, d.Commands.FleetMove)
	enemy.Name = "Enemy"
	enemy.X = 600
	planFleetAttacks(s, p, &d)
	require.Empty(t, d.Commands.FleetMove)
}
func TestAttackAvoidsBusyFleet(t *testing.T) {
	s, p, _ := militaryFixture()
	d := Decision{Commands: model.Commands{Invade: []*model.InvadeCommand{{FleetID: 1}}}}
	planFleetAttacks(s, p, &d)
	require.Empty(t, d.Commands.FleetMove)
}

func TestTransportMovesOnlyToSecuredOrbit(t *testing.T) {
	s, p, f := militaryFixture()
	p.Available.Drives = []string{"Nuclear"}
	p.Available.FuelCells = []string{"Standard"}
	s.Resources = &VisibleResources{ShipParts: &VisibleShipParts{Drives: []model.Drive{{Name: "Nuclear", InterstellarSpeed: 2}}, FuelCells: []model.FuelCell{{Name: "Standard", Autonomy: 2}}}}
	f.Ships = append(f.Ships, model.FleetShip{Type: "transport", FuelRangeParsecs: 2})
	escort := &model.Fleet{ID: 2, Owner: 1, StarID: 2, Ships: []model.FleetShip{f.Ships[0]}}
	s.Fleets = append(s.Fleets, escort)
	d := Decision{}
	planTransportMoves(s, p, &d)
	require.Len(t, d.Commands.FleetMove, 1)
	require.Equal(t, []uint{2}, *d.Commands.FleetMove[0].ShipIndexes)
	s.Fleets = append(s.Fleets, &model.Fleet{ID: 3, Owner: 2, StarID: 2, Ships: []model.FleetShip{{Type: "ship_design"}}})
	d = Decision{}
	planTransportMoves(s, p, &d)
	require.Empty(t, d.Commands.FleetMove)
}
func TestInvasionWaitsForHostileOrbit(t *testing.T) {
	s, _, f := militaryFixture()
	f.StarID = 2
	f.Ships = append(f.Ships, model.FleetShip{Type: "transport"})
	s.Fleets = append(s.Fleets, &model.Fleet{ID: 3, Owner: 2, StarID: 2, Ships: []model.FleetShip{{Type: "ship_design"}}})
	d := Decision{}
	issueInvasionsAtEnemyPlanets(s, 1, &d)
	require.Empty(t, d.Commands.Invade)
	s.Fleets = s.Fleets[:1]
	issueInvasionsAtEnemyPlanets(s, 1, &d)
	require.Len(t, d.Commands.Invade, 1)
}
