package model

type TechTree struct {
	Subjects []TechSubject `json:"subjects"`
	AllTechs []Tech        `json:"all_techs"`
}

type TechSubject struct {
	Name   string      `json:"name"`
	Levels []TechLevel `json:"levels"`
}

type TechLevel struct {
	Name           string   `json:"name"`
	Cost           uint64   `json:"cost"`
	ForceAll       bool     `json:"force_all,omitempty"`
	AvailableTechs []string `json:"available_techs"`
}

type ResearchProject struct {
	Subject   string   `json:"subject"`
	TechName  string   `json:"tech_name,omitempty"`
	TechNames []string `json:"tech_names,omitempty"`
	LevelName string   `json:"level_name,omitempty"`
	Progress  uint64   `json:"progress"`
}

type Tech struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Hidden      bool     `json:"hidden,omitempty"`
	Cost        int64    `json:"cost,omitempty"`
	Maintenance int64    `json:"maintenance,omitempty"`
	BuiltOn     []string `json:"built_on,omitempty"`
	Unique      bool     `json:"unique,omitempty"`
	Grants      Grants   `json:"grants,omitempty"`
}

type Grants struct {
	Unlocks   Unlocks    `json:"unlocks,omitempty"`
	Modifiers []Modifier `json:"modifiers,omitempty"`
}

type Unlocks struct {
	Buildings   []string `json:"buildings,omitempty"`
	HullSizes   []string `json:"hull_sizes,omitempty"`
	Drives      []string `json:"drives,omitempty"`
	Armors      []string `json:"armors,omitempty"`
	Computers   []string `json:"computers,omitempty"`
	FuelCells   []string `json:"fuel_cells,omitempty"`
	Shields     []string `json:"shields,omitempty"`
	Beams       []string `json:"beams,omitempty"`
	ShipSystems []string `json:"ship_systems,omitempty"`
}

type Modifier struct {
	Name  string  `json:"name"`
	Value float64 `json:"value,omitempty"`
}

type ModifierDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Scope       string `json:"scope,omitempty"`
	Stat        string `json:"stat,omitempty"`
	Operation   string `json:"operation,omitempty"`
}
