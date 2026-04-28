package model

type PopCategory string

const (
	PopCategoryFarmers    PopCategory = "farmers"
	PopCategoryWorkers   PopCategory = "workers"
	PopCategoryScientists PopCategory = "scientists"
	PopCategoryMarines   PopCategory = "marines"
)

type MovePopulationCommand struct {
	PlanetID uint       `json:"planetId"`
	From    PopCategory `json:"from"`
	To      PopCategory `json:"to"`
	Count   int64     `json:"count"`
}

type Commands struct {
	MovePopulation []MovePopulationCommand `json:"movePopulation"`
}