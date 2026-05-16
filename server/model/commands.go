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
	FleetID  uint `json:"fleet_id,omitempty"`
}

type ArtificialPlanetCommand struct {
	SourcePlanetID uint `json:"source_planet_id,omitempty"`
	TargetPlanetID uint `json:"target_planet_id"`
}

type PrepareColonyShipCommand struct {
	FleetID uint `json:"fleet_id"`
}

type FleetMoveCommand struct {
	FleetID           uint    `json:"fleet_id"`
	DestinationStarID uint    `json:"destination_star_id"`
	ShipIndexes       *[]uint `json:"ship_indexes,omitempty"`
}

type BuyProductionCommand struct {
	PlanetID uint `json:"planetId"`
}

type TransferPopulationCommand struct {
	SourcePlanetID      uint `json:"source_planet_id"`
	DestinationPlanetID uint `json:"destination_planet_id"`
	PopulationUnits     uint `json:"population_units"`
}

type Commands struct {
	Build                []*BuildCommand              `json:"build"`
	MovePopulation       []*MovePopulationCommand     `json:"movePopulation,omitempty"`
	TransferPopulation   []*TransferPopulationCommand `json:"transfer_population,omitempty"`
	SelectResearch       *SelectResearchCommand       `json:"selectResearch,omitempty"`
	Colonize             *ColonizeCommand             `json:"colonize,omitempty"`
	ColonizeMany         []*ColonizeCommand           `json:"colonize_many,omitempty"`
	ArtificialPlanet     *ArtificialPlanetCommand     `json:"artificial_planet,omitempty"`
	ArtificialPlanetMany []*ArtificialPlanetCommand   `json:"artificial_planet_many,omitempty"`
	ColonyShip           *PrepareColonyShipCommand    `json:"colony_ship,omitempty"`
	FleetMove            []*FleetMoveCommand          `json:"fleet_move,omitempty"`
	BuyProduction        []*BuyProductionCommand      `json:"buy_production,omitempty"`
}
