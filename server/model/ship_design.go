package model

type ShipDesign struct {
	Name           string          `json:"name"`
	Beams          []Beam          `json:"weapons"`
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
	Name     string `json:"name"`
	Drive    string `json:"drive"`
	Computer string `json:"computer"`
	Shield   string `json:"shield"`
	Armor    string `json:"armor"`
	HullSize string `json:"hull_size"`
	FuelCell string `json:"fuel_cell"`
}
