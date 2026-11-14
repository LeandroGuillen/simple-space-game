package game

import (
	"fmt"
	"slices"

	. "ssg/model"
)

// TurnEventFirstContact is the turn-report kind recorded for both empires when
// they meet for the first time.
const TurnEventFirstContact = "first_contact"

// TurnEventDeclaredWar is the turn-report kind recorded for both empires when
// a war starts.
const TurnEventDeclaredWar = "declared_war"

// hasContact reports whether player has met otherID.
func hasContact(player *Player, otherID uint) bool {
	return contactWith(player, otherID) != nil
}

// contactWith returns player's contact entry for otherID, or nil.
func contactWith(player *Player, otherID uint) *Contact {
	if player == nil {
		return nil
	}
	for i := range player.Contacts {
		if player.Contacts[i].PlayerID == otherID {
			return &player.Contacts[i]
		}
	}
	return nil
}

// atWar reports whether player is at war with otherID.
func atWar(player *Player, otherID uint) bool {
	contact := contactWith(player, otherID)
	return contact != nil && contact.AtWar
}

// warTarget checks that player may be at war with targetID: the target must be
// a contacted, existing, non-eliminated empire.
func (g *Game) warTarget(player *Player, targetID uint) (*Player, error) {
	if targetID == 0 || targetID == player.ID {
		return nil, fmt.Errorf("cannot declare war on player %d", targetID)
	}
	if !hasContact(player, targetID) {
		return nil, fmt.Errorf("cannot declare war on an empire you have not met")
	}
	target, err := g.getPlayer(targetID)
	if err != nil || target == nil {
		return nil, fmt.Errorf("cannot declare war on unknown player %d", targetID)
	}
	if !hasContact(target, player.ID) {
		return nil, fmt.Errorf("cannot declare war on an empire you have not met")
	}
	if target.Eliminated {
		return nil, fmt.Errorf("cannot declare war on an eliminated empire")
	}
	return target, nil
}

// validateDeclareWarCommands checks DeclareWar orders at submit time. Each
// target must be a contacted, non-eliminated empire not already at war with
// the player, and may appear only once.
func (g *Game) validateDeclareWarCommands(player *Player, commands []*DeclareWarCommand) error {
	seen := make(map[uint]bool, len(commands))
	for _, command := range commands {
		if command == nil {
			return fmt.Errorf("invalid declare war command")
		}
		if seen[command.PlayerID] {
			return fmt.Errorf("duplicate declaration of war on player %d", command.PlayerID)
		}
		seen[command.PlayerID] = true
		if _, err := g.warTarget(player, command.PlayerID); err != nil {
			return err
		}
		if atWar(player, command.PlayerID) {
			return fmt.Errorf("already at war with player %d", command.PlayerID)
		}
	}
	return nil
}

// resolveWarDeclarations applies every DeclareWar order of the turn at once.
// A declaration against an empire the player is already at war with is a
// no-op, and two empires declaring on each other in the same turn simply go to
// war. Both sides get a turn-report event. Caller holds the write lock.
func (g *Game) resolveWarDeclarations(submissions map[uint]Commands, playerIDs []uint) error {
	type pair struct{ a, b uint }
	declared := map[pair]map[uint]bool{} // ordered pair -> declarers
	order := []pair{}
	for _, playerID := range playerIDs {
		player, err := g.getPlayer(playerID)
		if err != nil {
			return err
		}
		for _, command := range submissions[playerID].DeclareWar {
			if command == nil {
				continue
			}
			if _, err := g.warTarget(player, command.PlayerID); err != nil {
				return &TurnSubmissionError{PlayerID: playerID, PlayerName: player.Name, Err: err}
			}
			if atWar(player, command.PlayerID) {
				continue
			}
			key := pair{min(playerID, command.PlayerID), max(playerID, command.PlayerID)}
			if declared[key] == nil {
				declared[key] = map[uint]bool{}
				order = append(order, key)
			}
			declared[key][playerID] = true
		}
	}
	for _, key := range order {
		a, _ := g.getPlayer(key.a)
		b, _ := g.getPlayer(key.b)
		for _, contact := range []*Contact{contactWith(a, b.ID), contactWith(b, a.ID)} {
			contact.AtWar = true
			contact.WarSince = g.CurrentTurn
		}
		mutual := declared[key][a.ID] && declared[key][b.ID]
		for _, side := range [][2]*Player{{a, b}, {b, a}} {
			self, other := side[0], side[1]
			var text string
			switch {
			case mutual:
				text = fmt.Sprintf("War with %s: both empires declared war", other.Name)
			case declared[key][self.ID]:
				text = fmt.Sprintf("You declared war on %s", other.Name)
			default:
				text = fmt.Sprintf("%s declared war on you", other.Name)
			}
			g.recordTurnEvent(TurnEvent{Owner: self.ID, Kind: TurnEventDeclaredWar, Text: text})
		}
	}
	return nil
}

// endWarsWith ends every war involving playerID (for example when that
// empire is eliminated). Contacts themselves are kept.
func (g *Game) endWarsWith(playerID uint) {
	for _, player := range g.Players {
		if player == nil {
			continue
		}
		for i := range player.Contacts {
			contact := &player.Contacts[i]
			if player.ID == playerID || contact.PlayerID == playerID {
				contact.AtWar = false
				contact.WarSince = 0
			}
		}
	}
}

// starPresenceOwners returns the sorted IDs of empires present at star: those
// owning a colony there or with a fleet in orbit. Fleets in transit and past
// exploration do not count.
func (g *Game) starPresenceOwners(star *Star) []uint {
	present := map[uint]struct{}{}
	for _, planet := range star.Planets {
		if planet != nil && planet.Owner != 0 {
			present[planet.Owner] = struct{}{}
		}
	}
	for _, fleet := range g.Fleets {
		if fleet == nil || fleet.Owner == 0 || fleet.StarID != star.ID || len(fleet.Ships) == 0 {
			continue
		}
		if fleet.DestinationStarID != 0 || fleet.TurnsRemaining != 0 {
			continue
		}
		present[fleet.Owner] = struct{}{}
	}
	owners := make([]uint, 0, len(present))
	for owner := range present {
		owners = append(owners, owner)
	}
	slices.Sort(owners)
	return owners
}

// resolveFirstContacts creates contacts between every pair of empires that
// share a star (colony or orbiting fleet), recording a first-contact turn
// event for both sides. It runs during turn resolution after fleet arrivals
// and colony founding, before battles. Caller holds the write lock.
func (g *Game) resolveFirstContacts() {
	for _, star := range g.Galaxy.Stars {
		if star == nil {
			continue
		}
		owners := g.starPresenceOwners(star)
		for i := 0; i < len(owners); i++ {
			for j := i + 1; j < len(owners); j++ {
				g.makeContact(owners[i], owners[j], star)
			}
		}
	}
}

func (g *Game) makeContact(aID, bID uint, star *Star) {
	a, errA := g.getPlayer(aID)
	b, errB := g.getPlayer(bID)
	if errA != nil || errB != nil || a == nil || b == nil {
		return
	}
	location := star.Name
	if location == "" {
		location = fmt.Sprintf("star %d", star.ID)
	}
	for _, pair := range [][2]*Player{{a, b}, {b, a}} {
		self, other := pair[0], pair[1]
		if hasContact(self, other.ID) {
			continue
		}
		self.Contacts = append(self.Contacts, Contact{PlayerID: other.ID, MetTurn: g.CurrentTurn, StarID: star.ID})
		g.recordTurnEvent(TurnEvent{
			Owner:  self.ID,
			Kind:   TurnEventFirstContact,
			Text:   fmt.Sprintf("First contact with %s at %s", other.Name, location),
			StarID: star.ID,
		})
	}
}
