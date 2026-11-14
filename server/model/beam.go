package model

// Beam JSON tags must stay camelCase: game/resources/beams.json uses
// minDamage/maxDamage and loads straight into this struct.
type Beam struct {
	Name      string `json:"name"`
	MinDamage int64  `json:"minDamage"`
	MaxDamage int64  `json:"maxDamage"`
	// RequiredTech names the research level that unlocks this beam. It is
	// only filled on the editor's ship-parts options, never on designs.
	RequiredTech string `json:"required_tech,omitempty"`
}
