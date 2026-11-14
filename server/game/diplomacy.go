package game

import (
	"fmt"
	"slices"

	. "ssg/model"
)

// TurnEventFirstContact is the turn-report kind recorded for both empires when
// they meet for the first time.
const TurnEventFirstContact = "first_contact"

// hasContact reports whether player has met otherID.
func hasContact(player *Player, otherID uint) bool {
	if player == nil {
		return false
	}
	for _, contact := range player.Contacts {
		if contact.PlayerID == otherID {
			return true
		}
	}
	return false
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
