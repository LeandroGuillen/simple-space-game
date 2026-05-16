package model

type FleetShip struct {
	Type               string `json:"type"`
	Name               string `json:"name"`
	Icon               string `json:"icon,omitempty"`
	DesignName         string `json:"design_name,omitempty"`
	HullSize           string `json:"hull_size,omitempty"`
	Drive              string `json:"drive,omitempty"`
	InterstellarSpeed  int64  `json:"interstellar_speed,omitempty"`
	FuelCell           string `json:"fuel_cell,omitempty"`
	FuelRangeParsecs   int64  `json:"fuel_range_parsecs,omitempty"`
	UnlimitedFuelRange bool   `json:"unlimited_fuel_range,omitempty"`
}

type Fleet struct {
	ID                uint        `json:"id"`
	Owner             uint        `json:"owner"`
	StarID            uint        `json:"star_id"`
	X                 float64     `json:"x"`
	Y                 float64     `json:"y"`
	OriginStarID      uint        `json:"origin_star_id,omitempty"`
	OriginX           float64     `json:"origin_x,omitempty"`
	OriginY           float64     `json:"origin_y,omitempty"`
	DestinationStarID uint        `json:"destination_star_id,omitempty"`
	TotalTurns        uint        `json:"total_turns,omitempty"`
	TurnsRemaining    uint        `json:"turns_remaining,omitempty"`
	Ships             []FleetShip `json:"ships"`
}
