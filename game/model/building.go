package model

type Project interface {
	GetCost() int64
	GetMaintenance() int64
}

type Building struct {
	ID          uint   `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Cost        int64  `json:"cost,omitempty"`
	Maintenance int64  `json:"maintenance,omitempty"`
}

func (b Building) GetCost() int64 {
	return b.Cost
}
func (b Building) GetMaintenance() int64 {
	return b.Maintenance
}
