package model

// BattleSide is one belligerent's contribution to a battle: the player it
// belongs to and the ships that survived the engagement. Ships that were lost
// can be derived by comparing OriginalShips and SurvivingShips.
type BattleSide struct {
	OwnerID        uint        `json:"owner_id"`
	OriginalShips  []FleetShip `json:"original_ships"`
	SurvivingShips []FleetShip `json:"surviving_ships"`
	DestroyedCount int         `json:"destroyed_count"`
}

// BattleReport is the post-resolution record of a single battle. It is stored
// on the game and surfaced to participating players for after-action review.
type BattleReport struct {
	ID            uint         `json:"id"`
	Turn          float64      `json:"turn"`
	StarID        uint         `json:"star_id"`
	X             float64      `json:"x"`
	Y             float64      `json:"y"`
	Sides         []BattleSide `json:"sides"`
	WinnerOwnerID uint         `json:"winner_owner_id,omitempty"`
	Rounds        int          `json:"rounds"`
	Draw          bool         `json:"draw,omitempty"`
}
