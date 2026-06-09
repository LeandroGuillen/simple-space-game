package model

// GroundUnitType defines the type of ground unit
type GroundUnitType string

const (
	MarineUnit  GroundUnitType = "marine"
	ArmoredUnit GroundUnitType = "armored"
)

// GroundUnit represents a single ground combat unit
type GroundUnit struct {
	Type     GroundUnitType `json:"type"`
	Strength int64          `json:"strength"` // base combat strength
	Health   int64          `json:"health"`   // current health points
	MaxHP    int64          `json:"max_hp"`   // maximum health points
}

// GroundForce represents a collection of ground units for one side
type GroundForce struct {
	OwnerID uint         `json:"owner_id"`
	Marines []GroundUnit `json:"marines"`
	Armored []GroundUnit `json:"armored"`
}

// GroundBattleRound represents one round of ground combat
type GroundBattleRound struct {
	Round           int   `json:"round"`
	AttackerLosses  int   `json:"attacker_losses"`
	DefenderLosses  int   `json:"defender_losses"`
	AttackerDamage  int64 `json:"attacker_damage"`
	DefenderDamage  int64 `json:"defender_damage"`
	AttackerSurvive int   `json:"attacker_survive"`
	DefenderSurvive int   `json:"defender_survive"`
}

// InvasionReport represents the result of a ground combat invasion
type InvasionReport struct {
	ID              uint                 `json:"id"`
	Turn            int64                `json:"turn"`
	PlanetID        uint                 `json:"planet_id"`
	PlanetName      string               `json:"planet_name"`
	X               float64              `json:"x"`
	Y               float64              `json:"y"`
	AttackerOwnerID uint                 `json:"attacker_owner_id"`
	DefenderOwnerID uint                 `json:"defender_owner_id"`
	Success         bool                 `json:"success"`
	PlanetCaptured  bool                 `json:"planet_captured"`
	Rounds          []GroundBattleRound  `json:"rounds"`
	InitialAttacker GroundForce          `json:"initial_attacker"`
	InitialDefender GroundForce          `json:"initial_defender"`
	SurvivingUnits  GroundForce          `json:"surviving_units"`
}

// NewGroundUnit creates a new ground unit with default stats
func NewGroundUnit(unitType GroundUnitType) GroundUnit {
	switch unitType {
	case MarineUnit:
		return GroundUnit{
			Type:     MarineUnit,
			Strength: 10, // base marine strength
			Health:   20, // marine HP
			MaxHP:    20,
		}
	case ArmoredUnit:
		return GroundUnit{
			Type:     ArmoredUnit,
			Strength: 25, // armored units are stronger
			Health:   40, // but have more HP
			MaxHP:    40,
		}
	default:
		return GroundUnit{Type: MarineUnit, Strength: 10, Health: 20, MaxHP: 20}
	}
}

// IsAlive returns true if the unit has health remaining
func (u *GroundUnit) IsAlive() bool {
	return u.Health > 0
}

// TotalStrength calculates the total combat strength of a ground force
func (gf *GroundForce) TotalStrength() int64 {
	total := int64(0)
	for _, marine := range gf.Marines {
		if marine.IsAlive() {
			total += marine.Strength
		}
	}
	for _, armored := range gf.Armored {
		if armored.IsAlive() {
			total += armored.Strength
		}
	}
	return total
}

// TotalUnits returns the count of living units in the force
func (gf *GroundForce) TotalUnits() int {
	count := 0
	for _, marine := range gf.Marines {
		if marine.IsAlive() {
			count++
		}
	}
	for _, armored := range gf.Armored {
		if armored.IsAlive() {
			count++
		}
	}
	return count
}

// HasLivingUnits returns true if the force has any units still alive
func (gf *GroundForce) HasLivingUnits() bool {
	return gf.TotalUnits() > 0
}