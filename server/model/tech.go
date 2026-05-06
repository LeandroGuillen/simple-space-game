package model

type TechTree struct {
	Subjects []TechSubject `json:"subjects"`
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

type Tech struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Cost        int64    `json:"cost,omitempty"`
	Maintenance int64    `json:"maintenance,omitempty"`
	Effects     any      `json:"effects,omitempty"`
	BuiltOn     []string `json:"built_on,omitempty"`
	Unique      bool     `json:"unique,omitempty"`
}
