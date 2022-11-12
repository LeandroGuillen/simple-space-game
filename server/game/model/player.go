package model

type Player struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`

	Race `json:"race"`

	Treasury       int64   `json:"treasury"`
	TaxRate        float64 `json:"tax_rate"`
	ResearchPoints uint    `json:"research_points"`

	Fleets         []uint `json:"fleets"`
	Freighters     uint   `json:"freighters"`
	FreightersUsed uint   `json:"freighters_used"`

	ShipDesigns []ShipDesign `json:"ship_designs"`
}
