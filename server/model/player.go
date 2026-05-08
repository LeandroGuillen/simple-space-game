package model

type Player struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`

	Race `json:"race"`

	Treasury        int64            `json:"treasury"`
	TaxRate         float64          `json:"tax_rate"`
	ResearchPoints  uint64           `json:"research_points"`
	CurrentResearch *ResearchProject `json:"current_research,omitempty"`
	KnownTechs      []string         `json:"known_techs,omitempty"`
	Available       AvailableTech    `json:"available"`

	Fleets         []uint `json:"fleets"`
	Freighters     uint   `json:"freighters"`
	FreightersUsed uint   `json:"freighters_used"`

	ShipDesigns   []ShipDesign `json:"ship_designs"`
	ExploredStars []uint       `json:"explored_stars"`
}

type AvailableTech struct {
	Buildings []string `json:"buildings"`
}
