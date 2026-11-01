package game

import (
	"os"
	"regexp"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

// The frontend's static editor catalog must agree with server-side validation.
func TestRaceTraitCatalogMatchesClient(t *testing.T) {
	data, err := os.ReadFile("../../client-solid/src/lib/race-design.ts")
	require.NoError(t, err)
	pattern := regexp.MustCompile(`\{ id: "([^"]+)"[^\n]*?pickCost: (-?\d+)[^\n]*?enabled: (true|false)`)
	matches := pattern.FindAllStringSubmatch(string(data), -1)
	require.Len(t, matches, len(raceTraitDefinitions))
	byID := raceTraitDefinitionsByID()
	for _, row := range matches {
		trait, ok := byID[row[1]]
		require.True(t, ok, row[1])
		cost, err := strconv.Atoi(row[2])
		require.NoError(t, err)
		enabled, err := strconv.ParseBool(row[3])
		require.NoError(t, err)
		require.Equal(t, trait.PickCost, cost, row[1])
		require.Equal(t, trait.Enabled, enabled, row[1])
	}
}
