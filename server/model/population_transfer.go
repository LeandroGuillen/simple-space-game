package model

type PopulationTransfer struct {
	ID                  uint `json:"id"`
	Owner               uint `json:"owner"`
	SourcePlanetID      uint `json:"source_planet_id"`
	DestinationPlanetID uint `json:"destination_planet_id"`
	PopulationUnits     uint `json:"population_units"`
	FreightersReserved  uint `json:"freighters_reserved"`
	OriginStarID        uint `json:"origin_star_id"`
	DestinationStarID   uint `json:"destination_star_id"`
	TotalTurns          uint `json:"total_turns"`
	TurnsRemaining      uint `json:"turns_remaining"`
}

// TransferLossNotice records population lost when cargo cannot land or return.
type TransferLossNotice struct {
	Owner                 uint    `json:"owner"`
	Turn                  float64 `json:"turn"`
	TransferID            uint    `json:"transfer_id"`
	SourcePlanetID        uint    `json:"source_planet_id"`
	SourcePlanetName      string  `json:"source_planet_name,omitempty"`
	DestinationPlanetID   uint    `json:"destination_planet_id"`
	DestinationPlanetName string  `json:"destination_planet_name,omitempty"`
	PopulationLost        int64   `json:"population_lost"`
}
