package game

import (
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/require"
)

func TestValidateLoadedGameRejectsBadContacts(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)
	cases := map[string][]model.Contact{
		"self":      {{PlayerID: 1}},
		"unknown":   {{PlayerID: 42}},
		"duplicate": {{PlayerID: 2}, {PlayerID: 2}},
	}
	for name, contacts := range cases {
		t.Run(name, func(t *testing.T) {
			g := multiPlayerGame(t, 2, 30)
			g.Players[0].Contacts = contacts
			require.Error(t, ValidateLoadedGame(g, resources))
		})
	}
}

func TestTurnSnapshotRestoresContacts(t *testing.T) {
	g := multiPlayerGame(t, 2, 30)
	g.Players[0].Contacts = []model.Contact{{PlayerID: 2}}
	snap := snapshotGameState(g)
	g.Players[0].Contacts[0].Relation = 7
	g.Players[1].Contacts = append(g.Players[1].Contacts, model.Contact{PlayerID: 1})
	restoreGameState(g, snap)
	require.Equal(t, []model.Contact{{PlayerID: 2}}, g.Players[0].Contacts)
	require.Empty(t, g.Players[1].Contacts)
}
