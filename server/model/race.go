package model

type Race struct {
	ID         uint      `json:"id"`
	Name       string    `json:"name"`
	Government string    `json:"government"`
	TraitIDs   []string  `json:"trait_ids,omitempty"`
	Grants     Grants    `json:"grants"`
	Stats      RaceStats `json:"stats"`
}

type RaceStats struct {
	PopulationGrowthPercent int64   `json:"population_growth_percent,omitempty"`
	Farming                 int64   `json:"farming,omitempty"`
	Production              int64   `json:"production,omitempty"`
	Research                int64   `json:"research,omitempty"`
	MoneyBonus              float64 `json:"money_bonus,omitempty"`
}
