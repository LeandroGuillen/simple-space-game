package model

type BuildCommand struct {
	PlanetID   uint     `json:"planetId"`
	Building   string   `json:"building"`
	BuildQueue []string `json:"build_queue,omitempty"`
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

type ColonizeCommand struct {
	PlanetID uint `json:"planet_id"`
}

type FleetMoveCommand struct {
	FleetID           uint    `json:"fleet_id"`
	DestinationStarID uint    `json:"destination_star_id"`
	ShipIndexes       *[]uint `json:"ship_indexes,omitempty"`
}

type Commands struct {
	Build          []*BuildCommand          `json:"build"`
	MovePopulation []*MovePopulationCommand `json:"movePopulation,omitempty"`
	SelectResearch *SelectResearchCommand   `json:"selectResearch,omitempty"`
	Colonize       *ColonizeCommand         `json:"colonize,omitempty"`
	FleetMove      []*FleetMoveCommand      `json:"fleet_move,omitempty"`
}
