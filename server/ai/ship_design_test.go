package ai

import (
	"github.com/stretchr/testify/require"
	"ssg/model"
	"testing"
)

func TestShipDesignPlansUpgradeWithoutChangingExistingDesign(t *testing.T) {
	p := &model.Player{ID: 2, Available: model.AvailableTech{HullSizes: []string{"Frigate"}, Drives: []string{"Nuclear", "Fusion"}, Armors: []string{"Titanium"}, Computers: []string{"None"}, FuelCells: []string{"Standard"}, Shields: []string{"None"}, Beams: []string{"Laser"}}}
	p.ShipDesigns = []model.ShipDesign{{Name: "AI Frigate 1", HullSize: model.HullSize{Name: "Frigate"}, Drive: model.Drive{Name: "Nuclear"}, Armor: model.Armor{Name: "Titanium"}, Computer: model.Computer{Name: "None"}, FuelCell: model.FuelCell{Name: "Standard"}, Shield: model.Shield{Name: "None"}, Beams: []model.Beam{{Name: "Laser"}}}}
	s := &VisibleGame{Resources: &VisibleResources{ShipParts: &VisibleShipParts{HullSizes: []model.HullSize{{Name: "Frigate", Space: 10}}, Drives: []model.Drive{{Name: "Nuclear", InterstellarSpeed: 1}, {Name: "Fusion", InterstellarSpeed: 2}}, Armors: []model.Armor{{Name: "Titanium", Multiplier: 1}}, Computers: []model.Computer{{Name: "None"}}, FuelCells: []model.FuelCell{{Name: "Standard", Autonomy: 2}}, Shields: []model.Shield{{Name: "None"}}}, BeamCollection: &VisibleBeams{Beams: []model.Beam{{Name: "Laser", MaxDamage: 4}}}}}
	d := Decision{}
	planShipDesign(s, p, nil, &d)
	require.NotNil(t, d.Commands.DesignShip)
	require.Equal(t, "AI Frigate 2", d.Commands.DesignShip.Name)
	require.Equal(t, "Fusion", d.Commands.DesignShip.Drive)
	require.Equal(t, "Nuclear", p.ShipDesigns[0].Drive.Name)
}
