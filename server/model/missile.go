package model

// Missile is a missile rack's warhead as loaded from
// game/resources/missiles.json. Missiles home on their target: they ignore
// range and the attacker's computer, and a battle gives each rack Ammo
// shots. Speed is stored for future in-flight missiles and is unused today.
type Missile struct {
	Name   string `json:"name"`
	Damage int64  `json:"damage"`
	Ammo   int    `json:"ammo"`
	Speed  int64  `json:"speed,omitempty"`
}

// MissileRack is one missile rack's per-battle state. Ammo lives only on
// the battle copy of a ship, so fleets carry no ammo between battles.
type MissileRack struct {
	Name     string `json:"name"`
	Damage   int64  `json:"damage"`
	AmmoLeft int    `json:"ammo_left"`
	AmmoMax  int    `json:"ammo_max"`
}
