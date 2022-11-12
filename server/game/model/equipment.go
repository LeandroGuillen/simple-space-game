package model

//https://gamefaqs.gamespot.com/pc/197873-master-of-orion-ii-battle-at-antares/faqs/16743

type HullSize struct {
	Name       string `json:"name"`
	SizeRating int64  `json:"size_rating"`
	Space      int64  `json:"space"`
	Marines    int64  `json:"marines"`
	BaseSpeed  int64  `json:"base_speed"` // squares/turn

	// hit points
	BaseHP     int64 `json:"base_hp"`
	ComputerHP int64 `json:"computer_hp"`
	DriveHP    int64 `json:"drive_hp"`
	ShieldHP   int64 `json:"shield_hp"`
}

type Drive struct {
	Name              string `json:"name"`
	InterstellarSpeed int64  `json:"interstellar_speed"` // parsecs/turn
	CombatSpeed       int64  `json:"combat_speed"`       // squares/turn
}

type Armor struct {
	Name                string `json:"name"`
	Multiplier          int64  `json:"multiplier"`
	GroundTroopStrength int64  `json:"ground_troop_strength"`
}

type FuelCell struct {
	Name     string `json:"name"`
	Autonomy int64  `json:"autonomy"` // in parsecs
}

type Shield struct {
	Name            string  `json:"name"`
	RechargeRate    float64 `json:"recharge_rate"`
	ReduceDamage    int64   `json:"reduce_damage"`
	AbsortionFactor int64   `json:"absortion_factor"` // Absorbs AbsortionFactor * BaseHP
}

type ShipSystem struct {
	Name             string `json:"name"`
	BeamAttackBonus  int64  `json:"beam_attack_bonus"`
	BeamDefenseBonus int64  `json:"beam_defense_bonus"`
}
