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
	// AtWar is set on both sides when either empire declares war (or a fight
	// auto-declares it). Omitted (older saves) means not at war. A war ends
	// when either empire is eliminated.
	AtWar bool `json:"at_war,omitempty"`
	// WarSince is the stardate of the turn in which the current war was
	// declared; omitted when not at war.
	WarSince float64 `json:"war_since,omitempty"`
	// PeaceTreaty is set on both sides while an accepted peace treaty is in
	// effect: no battles or invasions happen between the empires until war is
	// declared again (which clears it). Omitted means no treaty, the default
	// state, in which a fight that would start auto-declares war first.
	PeaceTreaty bool `json:"peace_treaty,omitempty"`
}

// HasPeaceTreatyWith reports whether p has a peace treaty in effect with
// otherID (and is therefore not at war with it).
func (p *Player) HasPeaceTreatyWith(otherID uint) bool {
	if p == nil {
		return false
	}
	for i := range p.Contacts {
		if p.Contacts[i].PlayerID == otherID {
			return p.Contacts[i].PeaceTreaty && !p.Contacts[i].AtWar
		}
	}
	return false
}
