package model

type BuildCommand struct {
	PlanetID uint   `json:"planetId"`
	Building string `json:"building"`
}
type MovePopulationCommand struct {
	PlanetID   uint  `json:"planetId"`
	Farmers    int64 `json:"farmers"`
	Workers    int64 `json:"workers"`
	Scientists int64 `json:"scientists"`
}

type SelectResearchCommand struct {
	TechName string `json:"tech_name"`
}

type Commands struct {
	Build          []*BuildCommand        `json:"build"`
	MovePopulation *MovePopulationCommand `json:"movePopulation,omitempty"`
	SelectResearch *SelectResearchCommand `json:"selectResearch,omitempty"`
}
