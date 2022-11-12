package model

type Race struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`

	Stats struct {
		PopulationGrowth int64
		Farming          int64
	}
}
