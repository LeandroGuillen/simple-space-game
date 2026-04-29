package model

import (
	"fmt"
)

//https://gamefaqs.gamespot.com/pc/197873-master-of-orion-ii-battle-at-antares/faqs/16743

type Ship struct {
	Name string
	ShipDesign
	ShipStatus
	SpaceAvailable int64
	BeamDefense    int64
	MissileEvasion int64
}

func (s Ship) String() string {
	return fmt.Sprintf("[%s %s] Armor: %d/%d - Hull %d/%d",
		s.HullSize.Name,
		s.Name,
		s.ShipStatus.CurrentArmorHP,
		s.ShipStatus.MaxArmorHP,
		s.ShipStatus.CurrentStructureHP,
		s.ShipStatus.MaxStructureHP,
	)
}
