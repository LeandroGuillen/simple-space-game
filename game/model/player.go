package model

import "ssg/game/combat"

type Player struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`

	Race string `json:"race"`

	Treasury       int64   `json:"treasury"`
	TaxRate        float64 `json:"tax_rate"`
	ResearchPoints uint    `json:"research_points"`

	Fleets         []uint `json:"fleets"`
	Freighters     uint   `json:"freighters"`
	FreightersUsed uint   `json:"freighters_used"`

	ShipDesigns []combat.ShipDesign `json:"ship_designs"`
}
