package model

type Player struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`

	Race `json:"race"`

	Treasury        int64             `json:"treasury"`
	TaxRate         float64           `json:"tax_rate"`
	ResearchPoints  uint64            `json:"research_points"`
	CurrentResearch *ResearchProject  `json:"current_research,omitempty"`
	ResearchedTechs []string          `json:"researched_techs,omitempty"`
	HyperAdvanced   map[string]uint64 `json:"hyper_advanced,omitempty"`
	Available       AvailableTech     `json:"available"`

	Fleets         []uint `json:"fleets"`
	Freighters     uint   `json:"freighters"`
	FreightersUsed uint   `json:"freighters_used"`

	ShipDesigns   []ShipDesign `json:"ship_designs"`
	ExploredStars []uint       `json:"explored_stars"`
}

type AvailableTech struct {
	Buildings   []string `json:"buildings"`
	HullSizes   []string `json:"hull_sizes"`
	Drives      []string `json:"drives"`
	Armors      []string `json:"armors"`
	Computers   []string `json:"computers"`
	FuelCells   []string `json:"fuel_cells"`
	Shields     []string `json:"shields"`
	Beams       []string `json:"beams"`
	ShipSystems []string `json:"ship_systems"`
}
