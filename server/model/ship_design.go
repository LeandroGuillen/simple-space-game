package model

type ShipDesign struct {
	Name  string `json:"name"`
	Beams []Beam `json:"weapons"`
	// Missiles is the design's missile rack. A design mounts a beam or a
	// missile rack in its one weapon slot, never both. Designs saved before
	// missiles existed have no rack.
	Missiles       []Missile       `json:"missiles,omitempty"`
	SpecialSystems []SpecialSystem `json:"special_systems"`
	BaseSpace      int64           `json:"base_space"`
	Drive          `json:"drive"`
	Computer       `json:"computer"`
	Shield         `json:"shield"`
	Armor          `json:"armor"`
	HullSize       `json:"hull_size"`
	FuelCell       `json:"fuel_cell"`
}

type ShipDesignsDef struct {
	Beam     string `json:"beam,omitempty"`
	Missile  string `json:"missile,omitempty"`
	Name     string `json:"name"`
	Drive    string `json:"drive"`
	Computer string `json:"computer"`
	Shield   string `json:"shield"`
	Armor    string `json:"armor"`
	HullSize string `json:"hull_size"`
	FuelCell string `json:"fuel_cell"`
}
