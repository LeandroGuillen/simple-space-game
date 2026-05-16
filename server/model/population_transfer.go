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
