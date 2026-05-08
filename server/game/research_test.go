package game

import (
	"testing"

	"ssg/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResearchMakesBuildingAvailable(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, "Ada", 4, model.DefaultPlayerColor())
	require.NoError(t, err)

	player := g.Players[0]
	g.research(player, "Automated Factory")
	g.research(player, "Automated Factory")

	assert.Contains(t, player.Available.Buildings, "Capitol")
	assert.Contains(t, player.Available.Buildings, "Colony Base")
	assert.Contains(t, player.Available.Buildings, "Star Base")
	assert.Contains(t, player.Available.Buildings, "Marine Barracks")
	assert.Contains(t, player.Available.Buildings, "Automated Factory")
	assert.Equal(t, 1, countStrings(player.Available.Buildings, "Automated Factory"))
}

func TestCapitalDoesNotStartWithColonyBaseBuilt(t *testing.T) {
	resources, err := NewResourceManager()
	require.NoError(t, err)

	g, err := NewGame(resources, "Ada", 4, model.DefaultPlayerColor())
	require.NoError(t, err)

	colonies, err := g.GetColonyList(g.Players[0].ID)
	require.NoError(t, err)
	require.NotEmpty(t, colonies)

	assert.NotContains(t, colonies[0].Buildings, "Colony Base")
	assert.Contains(t, colonies[0].Buildings, "Capitol")
	assert.Contains(t, colonies[0].Buildings, "Star Base")
}

func countStrings(values []string, needle string) int {
	count := 0
	for _, value := range values {
		if value == needle {
			count++
		}
	}
	return count
}
