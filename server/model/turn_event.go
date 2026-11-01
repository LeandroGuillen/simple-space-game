package model

// TurnEvent describes a player-owned event from the most recently resolved turn.
type TurnEvent struct {
	Owner    uint   `json:"owner"`
	Kind     string `json:"kind"`
	Text     string `json:"text"`
	PlanetID uint   `json:"planet_id,omitempty"`
	StarID   uint   `json:"star_id,omitempty"`
	FleetID  uint   `json:"fleet_id,omitempty"`
}
