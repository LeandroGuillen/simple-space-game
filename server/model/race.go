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
	// MoO2 ship combat picks (first level: +20%, 3 picks each).
	ShipAttackBonusPercent  int64 `json:"ship_attack_bonus_percent,omitempty"`
	ShipDefenseBonusPercent int64 `json:"ship_defense_bonus_percent,omitempty"`
	SpyBonus                int64 `json:"spy_bonus,omitempty"`
	// Stealthy Ships: reduces detection range when detection exists.
	DetectionModifier int64 `json:"detection_modifier,omitempty"`
	// Trans-Dimensional: bonus combat speed in battle.
	CombatSpeedBonus int64 `json:"combat_speed_bonus,omitempty"`
}
