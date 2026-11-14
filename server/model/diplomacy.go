package model

// Contact records that the owning player has met another empire. Contacts are
// created during turn resolution (first contact) and never removed; an
// eliminated empire stays on the list.
type Contact struct {
	PlayerID uint `json:"player_id"`
	// MetTurn is the stardate of the turn in which the empires first met.
	MetTurn float64 `json:"met_turn,omitempty"`
	// StarID is the system where the first contact happened.
	StarID uint `json:"star_id,omitempty"`
	// Relation is the diplomatic relation score. Relation modifiers are not
	// implemented yet, so it is always 0; omitted means 0.
	Relation int `json:"relation,omitempty"`
	// AtWar is set on both sides when either empire declares war. Omitted
	// (older saves) means peace. A war ends when either empire is eliminated.
	AtWar bool `json:"at_war,omitempty"`
	// WarSince is the stardate of the turn in which the current war was
	// declared; omitted when at peace.
	WarSince float64 `json:"war_since,omitempty"`
}
