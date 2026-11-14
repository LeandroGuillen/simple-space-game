package ai

import (
	"encoding/json"
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/require"
)

// setContactsJSON sets p's contacts from JSON so these tests also compile
// against code that predates the peace_treaty field.
func setContactsJSON(t *testing.T, p *model.Player, raw string) {
	t.Helper()
	p.Contacts = nil
	require.NoError(t, json.Unmarshal([]byte(raw), &p.Contacts))
}

func TestAIDoesNotAttackTreatyPartner(t *testing.T) {
	s, p, _ := militaryFixture()
	setContactsJSON(t, p, `[{"player_id":2,"peace_treaty":true}]`)
	d := Decision{}
	planFleetAttacks(s, p, &d)
	require.Empty(t, d.Commands.FleetMove, "no attack on an empire with a peace treaty")
}

func TestAIStillAttacksWithoutTreaty(t *testing.T) {
	for name, contacts := range map[string]string{
		"no contact entry": `[]`,
		"met, no treaty":   `[{"player_id":2}]`,
		"at war":           `[{"player_id":2,"at_war":true}]`,
	} {
		t.Run(name, func(t *testing.T) {
			s, p, _ := militaryFixture()
			setContactsJSON(t, p, contacts)
			d := Decision{}
			planFleetAttacks(s, p, &d)
			require.Len(t, d.Commands.FleetMove, 1, "the fight will auto-declare war")
			require.Equal(t, uint(2), d.Commands.FleetMove[0].DestinationStarID)
		})
	}
}

func TestAIDoesNotInvadeTreatyPartner(t *testing.T) {
	s, p, f := militaryFixture()
	f.StarID = 2
	f.Ships = append(f.Ships, model.FleetShip{Type: "transport"})
	setContactsJSON(t, p, `[{"player_id":2,"peace_treaty":true}]`)
	d := Decision{}
	issueInvasionsAtEnemyPlanets(s, 1, &d)
	require.Empty(t, d.Commands.Invade, "no invasion of an empire with a peace treaty")

	setContactsJSON(t, p, `[{"player_id":2}]`)
	issueInvasionsAtEnemyPlanets(s, 1, &d)
	require.Len(t, d.Commands.Invade, 1, "without a treaty the AI still invades (auto-declares war)")
}

func TestAIInvadesWithTreatyPartnerShipsInOrbit(t *testing.T) {
	// Player 3 (treaty partner) orbits player 2's colony; its ships do not
	// fight, so they do not stop the invasion of player 2.
	s, p, f := militaryFixture()
	f.StarID = 2
	f.Ships = append(f.Ships, model.FleetShip{Type: "transport"})
	s.Fleets = append(s.Fleets, &model.Fleet{ID: 3, Owner: 3, StarID: 2, Ships: []model.FleetShip{{Type: "ship_design"}}})
	setContactsJSON(t, p, `[{"player_id":2},{"player_id":3,"peace_treaty":true}]`)
	d := Decision{}
	issueInvasionsAtEnemyPlanets(s, 1, &d)
	require.Len(t, d.Commands.Invade, 1)
	require.Equal(t, uint(2), d.Commands.Invade[0].TargetPlanetID)
}
