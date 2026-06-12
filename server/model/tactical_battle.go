package model

// Tactical battle status values.
const (
	TacticalBattleActive   = "active"
	TacticalBattleFinished = "finished"
)

// Tactical battle grid dimensions, MoO2-ish proportions. Small enough that
// every square is within beam dissipation range (max Chebyshev distance 13).
const (
	TacticalGridWidth  = 14
	TacticalGridHeight = 10
)

// TacticalShip is one ship materialized onto the battle grid. FleetID and
// ShipIndex keep fleet provenance so survivors can be reconciled back into
// their fleets after the fight.
type TacticalShip struct {
	ID      uint `json:"id"`
	OwnerID uint `json:"owner_id"`

	FleetID   uint `json:"fleet_id"`
	ShipIndex int  `json:"ship_index"`

	Name       string `json:"name"`
	DesignName string `json:"design_name,omitempty"`
	HullSize   string `json:"hull_size,omitempty"`
	SizeRating int64  `json:"size_rating"`

	X int `json:"x"`
	Y int `json:"y"`

	CombatSpeed int   `json:"combat_speed"`
	Initiative  int64 `json:"initiative"`

	Beams           []Beam `json:"beams,omitempty"`
	BeamAttackBonus int64  `json:"beam_attack_bonus"`
	BeamDefense     int64  `json:"beam_defense"`

	ShipStatus `json:"status"`

	// per-activation state
	MovesLeft int  `json:"moves_left"`
	HasFired  bool `json:"has_fired"`
	Destroyed bool `json:"destroyed"`
}

// BattleEvent is one entry of the append-only battle log. Clients animate the
// battle by consuming events in sequence order.
type BattleEvent struct {
	Seq  int    `json:"seq"`
	Type string `json:"type"` // round_start | activation | move | fire | destroyed | finished

	Round    int   `json:"round,omitempty"`
	ShipID   uint  `json:"ship_id,omitempty"`
	TargetID uint  `json:"target_id,omitempty"`
	FromX    int   `json:"from_x,omitempty"`
	FromY    int   `json:"from_y,omitempty"`
	X        int   `json:"x,omitempty"`
	Y        int   `json:"y,omitempty"`
	Damage   int64 `json:"damage,omitempty"`
	Hits     int   `json:"hits,omitempty"`
	Misses   int   `json:"misses,omitempty"`

	WinnerOwnerID uint `json:"winner_owner_id,omitempty"`
	Draw          bool `json:"draw,omitempty"`
}

// TacticalBattle is a server-authoritative interactive battle at a star. It is
// created during turn processing when a human player's fleet shares orbit with
// a hostile fleet, and resolved through battle action API calls.
type TacticalBattle struct {
	ID       uint    `json:"id"`
	GameTurn float64 `json:"game_turn"`
	StarID   uint    `json:"star_id"`
	Status   string  `json:"status"`

	GridWidth  int `json:"grid_width"`
	GridHeight int `json:"grid_height"`
	Round      int `json:"round"`

	Ships     []*TacticalShip `json:"ships"`
	TurnOrder []uint          `json:"turn_order"` // ship IDs in initiative order
	TurnIndex int             `json:"turn_index"` // position in TurnOrder of the acting ship

	HumanOwnerIDs []uint `json:"human_owner_ids"`

	WinnerOwnerID uint `json:"winner_owner_id,omitempty"`
	Draw          bool `json:"draw,omitempty"`

	Events []BattleEvent `json:"events"`
}

// AppendEvent adds an event to the battle log with the next sequence number.
func (b *TacticalBattle) AppendEvent(e BattleEvent) {
	e.Seq = len(b.Events) + 1
	b.Events = append(b.Events, e)
}

// ShipByID finds a battle ship by its battle-local ID.
func (b *TacticalBattle) ShipByID(id uint) *TacticalShip {
	for _, ts := range b.Ships {
		if ts.ID == id {
			return ts
		}
	}
	return nil
}

// CurrentShip returns the ship whose activation it is. Nil once the battle
// ended or if the order entry is stale (callers keep the index on live ships).
func (b *TacticalBattle) CurrentShip() *TacticalShip {
	if b.Status != TacticalBattleActive || b.TurnIndex >= len(b.TurnOrder) {
		return nil
	}
	ts := b.ShipByID(b.TurnOrder[b.TurnIndex])
	if ts == nil || ts.Destroyed {
		return nil
	}
	return ts
}

// ShipAt returns the living ship occupying the square, if any.
func (b *TacticalBattle) ShipAt(x, y int) *TacticalShip {
	for _, ts := range b.Ships {
		if !ts.Destroyed && ts.X == x && ts.Y == y {
			return ts
		}
	}
	return nil
}

// AnnounceActivation logs whose ship is up, for client turn indicators.
func (b *TacticalBattle) AnnounceActivation() {
	if ts := b.CurrentShip(); ts != nil {
		b.AppendEvent(BattleEvent{Type: "activation", ShipID: ts.ID, Round: b.Round})
	}
}

// OwnersWithLiveShips lists owner IDs that still have ships in the fight.
func (b *TacticalBattle) OwnersWithLiveShips() []uint {
	seen := map[uint]bool{}
	owners := []uint{}
	for _, ts := range b.Ships {
		if !ts.Destroyed && !seen[ts.OwnerID] {
			seen[ts.OwnerID] = true
			owners = append(owners, ts.OwnerID)
		}
	}
	return owners
}

// IsHumanOwner reports whether ownerID fights this battle interactively.
func (b *TacticalBattle) IsHumanOwner(ownerID uint) bool {
	for _, id := range b.HumanOwnerIDs {
		if id == ownerID {
			return true
		}
	}
	return false
}

// BattleAction is a player's order for the currently acting ship.
type BattleAction struct {
	Type     string `json:"type"` // move | fire | done | wait | auto
	ShipID   uint   `json:"ship_id"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	TargetID uint   `json:"target_id"`
}
