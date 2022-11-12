package model

type Building struct {
	ID          uint     `json:"id,omitempty"`
	Name        string   `json:"name,omitempty"`
	Cost        int64    `json:"cost,omitempty"`
	Maintenance int64    `json:"maintenance,omitempty"`
	Effects     []Effect `json:"effects,omitempty"`
}
