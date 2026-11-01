package game_test

import (
	"crypto/sha256"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"ssg/game"

	"github.com/stretchr/testify/require"
)

func TestClientGalaxyCountsMatchServer(t *testing.T) {
	data, err := os.ReadFile("../../client-solid/src/routes/SetupPage.tsx")
	require.NoError(t, err)
	block := regexp.MustCompile(`(?s)const galaxySizeStars[^=]*=\s*\{([^}]*)\}`).FindStringSubmatch(string(data))
	require.Len(t, block, 2, "web setup galaxy-size count table")
	entries := regexp.MustCompile(`(tiny|medium|huge):\s*(\d+)`).FindAllStringSubmatch(block[1], -1)
	require.Len(t, entries, len(game.ClientGalaxyStarCounts))
	seen := map[string]bool{}
	for _, entry := range entries {
		require.False(t, seen[entry[1]], "duplicate galaxy size")
		seen[entry[1]] = true
		n, err := strconv.ParseUint(entry[2], 10, 64)
		require.NoError(t, err)
		require.Equal(t, game.ClientGalaxyStarCounts[entry[1]], uint(n), entry[1])
	}
}

func TestMediumAndHugeSeededPositionsRemainStable(t *testing.T) {
	for _, tc := range []struct {
		size  string
		count uint
		hash  string
	}{
		{"medium", 40, "19c592f6069a6da3cff840cf3f3d36c2968ae8a65a17eb106bc77fedd985dad9"},
		{"huge", 100, "33db9eb6f29ec58d775e95f0946247321c76eca7188adc77628bd6158d63ee0f"},
	} {
		t.Run(tc.size, func(t *testing.T) {
			game.SetRandomSeed(4242)
			galaxy := game.NewGalaxyForSize(tc.count, tc.size)
			var coords strings.Builder
			for _, star := range galaxy.Stars {
				fmt.Fprintf(&coords, "%d,%d;", star.X, star.Y)
			}
			require.Equal(t, tc.hash, fmt.Sprintf("%x", sha256.Sum256([]byte(coords.String()))))
		})
	}
}
