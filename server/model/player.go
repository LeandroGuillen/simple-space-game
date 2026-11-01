package model

type Player struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`

	// IsHuman marks players whose battles pause for interactive tactical
	// combat instead of instant auto-resolve.
	IsHuman    bool   `json:"is_human,omitempty"`
	Eliminated bool   `json:"eliminated,omitempty"`
	Controller string `json:"controller,omitempty"`

	Race `json:"race"`

	Treasury        float64           `json:"treasury"`
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

	CommandPointsCapacity      int64 `json:"command_points_capacity"`
	CommandPointsUsed          int64 `json:"command_points_used"`
	CommandPointsRemaining     int64 `json:"command_points_remaining"`
	CommandOverdrawBC          int64 `json:"command_overdraw_bc"`
	CommandPointsStartingBonus int64 `json:"command_points_starting_bonus"`

	EvolutionaryMutationPoints  int  `json:"evolutionary_mutation_points,omitempty"`
	HasUsedEvolutionaryMutation bool `json:"has_used_evolutionary_mutation,omitempty"`
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
