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

// TurnEventDeclareWarFailed is the turn-report kind recorded for a declarer
// whose declaration of war was valid at submit time but could not be applied
// when the turn resolved.
const TurnEventDeclareWarFailed = "declare_war_failed"

// empireNameFor names otherID as viewer knows it: the empire's name once
// met, otherwise a neutral label.
func (g *Game) empireNameFor(viewer *Player, otherID uint) string {
	if other, err := g.getPlayer(otherID); err == nil && other != nil && hasContact(viewer, otherID) {
		return other.Name
	}
	return "an unknown empire"
}

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
// war. Both sides get a turn-report event. An order whose target is no longer
// valid is skipped with a declare_war_failed event for the declarer; it never
// fails the turn. Caller holds the write lock.
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
				// Valid at submit but not any more (for example the target
				// was eliminated meanwhile): skip this order and tell the
				// declarer, without failing the turn for everyone.
				g.recordTurnEvent(TurnEvent{
					Owner: playerID,
					Kind:  TurnEventDeclareWarFailed,
					Text:  fmt.Sprintf("Your declaration of war on %s could not be applied: %v", g.empireNameFor(player, command.PlayerID), err),
				})
				continue
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
		g.startWar(a, b)
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

// startWar puts both empires at war from this turn: at_war and war_since are
// set on both contacts and any peace treaty is cleared. Both contacts must
// exist. Turn events are the caller's job.
func (g *Game) startWar(a, b *Player) {
	for _, contact := range []*Contact{contactWith(a, b.ID), contactWith(b, a.ID)} {
		contact.AtWar = true
		contact.WarSince = g.CurrentTurn
		contact.PeaceTreaty = false
	}
}

// peaceTreatyBetween reports whether a peace treaty is in effect between the
// two empires (on either side's record). While it is, no battles or
// invasions happen between them. Slice 4 (Offer Peace) only has to set
// Contact.PeaceTreaty on both sides; declaring war clears it.
func (g *Game) peaceTreatyBetween(aID, bID uint) bool {
	a, _ := g.getPlayer(aID)
	b, _ := g.getPlayer(bID)
	return a.HasPeaceTreatyWith(bID) || b.HasPeaceTreatyWith(aID)
}

// autoDeclareWar is called when a fight or invasion between two empires is
// about to start. Empires already at war are left alone. Otherwise (the
// default no-treaty state) war is declared on the spot with the same
// bookkeeping as a declare_war order, and both sides get a declared_war turn
// event naming the cause. Missing contacts are created at starID first.
// Callers must have checked peaceTreatyBetween; a treaty blocks the fight.
func (g *Game) autoDeclareWar(aID, bID, starID uint, text func(self, other *Player) string) {
	a, errA := g.getPlayer(aID)
	b, errB := g.getPlayer(bID)
	if errA != nil || errB != nil || a == nil || b == nil || aID == bID {
		return
	}
	if !hasContact(a, bID) || !hasContact(b, aID) {
		if star, err := g.getStar(starID); err == nil && star != nil {
			g.makeContact(aID, bID, star)
		}
		if !hasContact(a, bID) || !hasContact(b, aID) {
			return
		}
	}
	if atWar(a, bID) && atWar(b, aID) {
		return
	}
	g.startWar(a, b)
	for _, side := range [][2]*Player{{a, b}, {b, a}} {
		g.recordTurnEvent(TurnEvent{Owner: side[0].ID, Kind: TurnEventDeclaredWar, Text: text(side[0], side[1]), StarID: starID})
	}
}

// starName returns a display name for starID.
func (g *Game) starName(starID uint) string {
	if star, err := g.getStar(starID); err == nil && star != nil && star.Name != "" {
		return star.Name
	}
	return fmt.Sprintf("star %d", starID)
}

// battleCombatants picks the fleets that fight at a shared star. Owners are
// taken in ID order and an owner with a peace treaty with an owner already
// taken sits the battle out (treaty partners never shoot at each other in
// the free-for-all). Returns nil when fewer than two owners would fight.
func (g *Game) battleCombatants(fleets []*Fleet) []*Fleet {
	owners := []uint{}
	seen := map[uint]bool{}
	for _, fleet := range fleets {
		if !seen[fleet.Owner] {
			seen[fleet.Owner] = true
			owners = append(owners, fleet.Owner)
		}
	}
	slices.Sort(owners)
	kept := map[uint]bool{}
	keptOrder := []uint{}
	for _, owner := range owners {
		blocked := false
		for _, other := range keptOrder {
			if g.peaceTreatyBetween(owner, other) {
				blocked = true
				break
			}
		}
		if !blocked {
			kept[owner] = true
			keptOrder = append(keptOrder, owner)
		}
	}
	if len(keptOrder) < 2 {
		return nil
	}
	out := make([]*Fleet, 0, len(fleets))
	for _, fleet := range fleets {
		if kept[fleet.Owner] {
			out = append(out, fleet)
		}
	}
	return out
}

// declareWarForBattle auto-declares war between every pair of combatant
// owners not yet at war, just before the battle at starID starts.
func (g *Game) declareWarForBattle(starID uint, fleets []*Fleet) {
	owners := []uint{}
	seen := map[uint]bool{}
	for _, fleet := range fleets {
		if !seen[fleet.Owner] {
			seen[fleet.Owner] = true
			owners = append(owners, fleet.Owner)
		}
	}
	slices.Sort(owners)
	where := g.starName(starID)
	for i := 0; i < len(owners); i++ {
		for j := i + 1; j < len(owners); j++ {
			g.autoDeclareWar(owners[i], owners[j], starID, func(_, other *Player) string {
				return fmt.Sprintf("War with %s: your fleets clashed at %s", other.Name, where)
			})
		}
	}
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
